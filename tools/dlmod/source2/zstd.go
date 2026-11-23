package source2

// A Zstandard decoder (RFC 8878), for the game files KV3 stores zstd-compressed, such as
// scripts/abilities.vdata_c and scripts/heroes.vdata_c. It decodes whole frames into memory and
// follows the RFC's reference ("educational") decoder: Huffman-coded literals, FSE-coded
// sequences with repeat offsets, and the frame's xxHash64 checksum. Dictionaries are not
// supported; the game does not use them.

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math/bits"
)

const (
	zstdMagic        = 0xFD2FB528
	zstdSkippableLow = 0x184D2A50 // skippable frames use 0x184D2A50 to 0x184D2A5F
	zstdMaxBlock     = 128 << 10
	zstdMaxOutput    = 1 << 30 // refuse to make more than this from one call
)

var errZstd = errors.New("zstd: corrupt data")

// DecodeZstd decodes every frame in src. size, when positive, is the expected decoded size,
// which sizes the output and is checked at the end.
func DecodeZstd(src []byte, size int) ([]byte, error) {
	capacity := size
	if capacity <= 0 || capacity > zstdMaxOutput {
		capacity = 0
	}
	out := make([]byte, 0, capacity)
	for len(src) > 0 {
		if len(src) < 4 {
			return nil, errZstd
		}
		magic := binary.LittleEndian.Uint32(src)
		if magic&0xFFFFFFF0 == zstdSkippableLow {
			if len(src) < 8 {
				return nil, errZstd
			}
			skip := uint64(binary.LittleEndian.Uint32(src[4:])) + 8
			if skip > uint64(len(src)) {
				return nil, errZstd
			}
			src = src[skip:]
			continue
		}
		if magic != zstdMagic {
			return nil, errors.New("zstd: not a Zstandard frame")
		}
		var err error
		if out, src, err = decodeZstdFrame(out, src[4:]); err != nil {
			return nil, err
		}
	}
	if size > 0 && len(out) != size {
		return nil, fmt.Errorf("zstd: decoded %d bytes, expected %d", len(out), size)
	}
	return out, nil
}

// zstdFrame holds what one frame's blocks share: the previous Huffman table and FSE tables
// (blocks may say "repeat") and the three most recent offsets.
type zstdFrame struct {
	huffman           *huffmanTable
	literalLengths    *fseTable
	offsets           *fseTable
	matchLengths      *fseTable
	recent            [3]int
	start             int // where this frame's output starts in out
	literals, scratch []byte
}

func decodeZstdFrame(out, src []byte) ([]byte, []byte, error) {
	if len(src) < 1 {
		return nil, nil, errZstd
	}
	descriptor := src[0]
	src = src[1:]
	contentSizeFlag := descriptor >> 6
	singleSegment := descriptor&0x20 != 0
	if descriptor&0x08 != 0 {
		return nil, nil, errors.New("zstd: reserved frame header bit set")
	}
	hasChecksum := descriptor&0x04 != 0
	dictionaryBytes := [4]int{0, 1, 2, 4}[descriptor&3]
	if !singleSegment {
		if len(src) < 1 {
			return nil, nil, errZstd
		}
		src = src[1:] // window descriptor: the whole output stays in memory, so it is not needed
	}
	if len(src) < dictionaryBytes {
		return nil, nil, errZstd
	}
	dictionary := uint32(0)
	for i := dictionaryBytes - 1; i >= 0; i-- {
		dictionary = dictionary<<8 | uint32(src[i])
	}
	if dictionary != 0 {
		return nil, nil, errors.New("zstd: frames that need a dictionary are not supported")
	}
	src = src[dictionaryBytes:]
	contentBytes := [4]int{0, 2, 4, 8}[contentSizeFlag]
	if contentSizeFlag == 0 && singleSegment {
		contentBytes = 1
	}
	if len(src) < contentBytes {
		return nil, nil, errZstd
	}
	contentSize := uint64(0)
	for i := contentBytes - 1; i >= 0; i-- {
		contentSize = contentSize<<8 | uint64(src[i])
	}
	if contentBytes == 2 {
		contentSize += 256
	}
	src = src[contentBytes:]
	if contentBytes > 0 && contentSize > zstdMaxOutput {
		return nil, nil, errors.New("zstd: frame too large")
	}

	frame := &zstdFrame{recent: [3]int{1, 4, 8}, start: len(out)}
	for {
		if len(src) < 3 {
			return nil, nil, errZstd
		}
		header := uint32(src[0]) | uint32(src[1])<<8 | uint32(src[2])<<16
		src = src[3:]
		last, kind, blockSize := header&1 != 0, (header>>1)&3, int(header>>3)
		if blockSize > zstdMaxBlock {
			return nil, nil, errZstd
		}
		switch kind {
		case 0: // raw
			if len(src) < blockSize {
				return nil, nil, errZstd
			}
			out = append(out, src[:blockSize]...)
			src = src[blockSize:]
		case 1: // one byte repeated
			if len(src) < 1 {
				return nil, nil, errZstd
			}
			for i := 0; i < blockSize; i++ {
				out = append(out, src[0])
			}
			src = src[1:]
		case 2:
			if len(src) < blockSize {
				return nil, nil, errZstd
			}
			var err error
			if out, err = frame.block(out, src[:blockSize]); err != nil {
				return nil, nil, err
			}
			src = src[blockSize:]
		default:
			return nil, nil, errors.New("zstd: reserved block type")
		}
		if len(out)-frame.start > zstdMaxOutput {
			return nil, nil, errors.New("zstd: frame too large")
		}
		if last {
			break
		}
	}
	content := out[frame.start:]
	if contentBytes > 0 && uint64(len(content)) != contentSize {
		return nil, nil, fmt.Errorf("zstd: frame decoded to %d bytes, its header says %d", len(content), contentSize)
	}
	if hasChecksum {
		if len(src) < 4 {
			return nil, nil, errZstd
		}
		if uint32(xxhash64(content)) != binary.LittleEndian.Uint32(src) {
			return nil, nil, errors.New("zstd: checksum mismatch")
		}
		src = src[4:]
	}
	return out, src, nil
}

// block decodes one compressed block: its literals, then the sequences that copy literals and
// earlier output.
func (f *zstdFrame) block(out, src []byte) ([]byte, error) {
	literals, src, err := f.readLiterals(src)
	if err != nil {
		return nil, err
	}
	return f.sequences(out, literals, src)
}

func (f *zstdFrame) readLiterals(src []byte) ([]byte, []byte, error) {
	if len(src) < 1 {
		return nil, nil, errZstd
	}
	kind, sizeFormat := src[0]&3, (src[0]>>2)&3
	if kind < 2 { // raw or one byte repeated
		var size, headerBytes int
		switch sizeFormat {
		case 0, 2:
			size, headerBytes = int(src[0]>>3), 1
		case 1:
			if len(src) < 2 {
				return nil, nil, errZstd
			}
			size, headerBytes = int(src[0]>>4)|int(src[1])<<4, 2
		case 3:
			if len(src) < 3 {
				return nil, nil, errZstd
			}
			size, headerBytes = int(src[0]>>4)|int(src[1])<<4|int(src[2])<<12, 3
		}
		src = src[headerBytes:]
		if kind == 0 {
			if len(src) < size {
				return nil, nil, errZstd
			}
			return src[:size], src[size:], nil
		}
		if len(src) < 1 {
			return nil, nil, errZstd
		}
		f.literals = f.literals[:0]
		for i := 0; i < size; i++ {
			f.literals = append(f.literals, src[0])
		}
		return f.literals, src[1:], nil
	}

	// Huffman-coded, with a new table (2) or the previous block's (3).
	streams, headerBytes, sizeBits := 4, 3, 10
	switch sizeFormat {
	case 0:
		streams = 1
	case 2:
		headerBytes, sizeBits = 4, 14
	case 3:
		headerBytes, sizeBits = 5, 18
	}
	if len(src) < headerBytes {
		return nil, nil, errZstd
	}
	var header uint64
	for i := headerBytes - 1; i >= 0; i-- {
		header = header<<8 | uint64(src[i])
	}
	mask := uint64(1)<<sizeBits - 1
	regenerated := int(header >> 4 & mask)
	compressed := int(header >> (4 + sizeBits) & mask)
	src = src[headerBytes:]
	if len(src) < compressed || regenerated > zstdMaxBlock {
		return nil, nil, errZstd
	}
	data, rest := src[:compressed], src[compressed:]
	if kind == 2 {
		table, used, err := readHuffmanTable(data)
		if err != nil {
			return nil, nil, err
		}
		f.huffman, data = table, data[used:]
	} else if f.huffman == nil {
		return nil, nil, errors.New("zstd: literals reuse a Huffman table that was never sent")
	}
	if cap(f.literals) < regenerated {
		f.literals = make([]byte, regenerated)
	}
	literals := f.literals[:regenerated]
	if streams == 1 {
		if err := f.huffman.decode(literals, data); err != nil {
			return nil, nil, err
		}
		return literals, rest, nil
	}
	if len(data) < 6 {
		return nil, nil, errZstd
	}
	sizes := [4]int{int(binary.LittleEndian.Uint16(data)), int(binary.LittleEndian.Uint16(data[2:])), int(binary.LittleEndian.Uint16(data[4:]))}
	data = data[6:]
	sizes[3] = len(data) - sizes[0] - sizes[1] - sizes[2]
	if sizes[3] < 0 {
		return nil, nil, errZstd
	}
	segment := (regenerated + 3) / 4
	for i := 0; i < 4; i++ {
		from, to := i*segment, (i+1)*segment
		if to > regenerated {
			to = regenerated
		}
		if from > to {
			from = to
		}
		if err := f.huffman.decode(literals[from:to], data[:sizes[i]]); err != nil {
			return nil, nil, err
		}
		data = data[sizes[i]:]
	}
	return literals, rest, nil
}

// Sequence codes: each literal length and match length code stands for a baseline plus a number
// of extra bits read from the stream.
var (
	literalLengthBase = [36]int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15,
		16, 18, 20, 22, 24, 28, 32, 40, 48, 64, 128, 256, 512, 1024, 2048, 4096, 8192, 16384, 32768, 65536}
	literalLengthBits = [36]uint8{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		1, 1, 1, 1, 2, 2, 3, 3, 4, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}
	matchLengthBase = [53]int{3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23,
		24, 25, 26, 27, 28, 29, 30, 31, 32, 33, 34, 35, 37, 39, 41, 43, 47, 51, 59, 67, 83, 99, 131, 259,
		515, 1027, 2051, 4099, 8195, 16387, 32771, 65539}
	matchLengthBits = [53]uint8{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 1, 2, 2, 3, 3, 4, 4, 5, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}

	// The predefined distributions, used when a block does not send its own.
	literalLengthDefault = []int16{4, 3, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 1, 1, 1, 2, 2, 2, 2, 2, 2, 2, 2,
		2, 3, 2, 1, 1, 1, 1, 1, -1, -1, -1, -1}
	matchLengthDefault = []int16{1, 4, 3, 2, 2, 2, 2, 2, 2, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1,
		1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, -1, -1, -1, -1, -1, -1, -1}
	offsetDefault = []int16{1, 1, 1, 1, 1, 1, 2, 2, 2, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, -1,
		-1, -1, -1, -1}
)

// sequenceTable reads one of the three FSE tables of a block according to its mode.
func sequenceTable(mode byte, src []byte, previous *fseTable, defaults []int16, defaultLog, maxLog, maxSymbol int) (*fseTable, int, error) {
	switch mode {
	case 0:
		table, err := newFSETable(defaults, defaultLog)
		return table, 0, err
	case 1: // one symbol, no bits
		if len(src) < 1 || int(src[0]) > maxSymbol {
			return nil, 0, errZstd
		}
		return &fseTable{log: 0, symbols: []uint8{src[0]}, bits: []uint8{0}, base: []uint16{0}}, 1, nil
	case 2:
		in := &forwardBits{src: src}
		table, err := readFSETable(in, maxLog, maxSymbol)
		return table, in.bytesUsed(), err
	default:
		if previous == nil {
			return nil, 0, errors.New("zstd: sequences reuse a table that was never sent")
		}
		return previous, 0, nil
	}
}

func (f *zstdFrame) sequences(out, literals, src []byte) ([]byte, error) {
	if len(src) < 1 {
		return nil, errZstd
	}
	count := int(src[0])
	switch {
	case count == 0:
		return append(out, literals...), nil
	case count < 128:
		src = src[1:]
	case count < 255:
		if len(src) < 2 {
			return nil, errZstd
		}
		count, src = (count-128)<<8|int(src[1]), src[2:]
	default:
		if len(src) < 3 {
			return nil, errZstd
		}
		count, src = int(src[1])|int(src[2])<<8+0x7F00, src[3:]
	}
	if len(src) < 1 {
		return nil, errZstd
	}
	modes := src[0]
	if modes&3 != 0 {
		return nil, errors.New("zstd: reserved sequence mode bits set")
	}
	src = src[1:]
	var used int
	var err error
	if f.literalLengths, used, err = sequenceTable(modes>>6, src, f.literalLengths, literalLengthDefault, 6, 9, 35); err != nil {
		return nil, err
	}
	src = src[used:]
	if f.offsets, used, err = sequenceTable(modes>>4&3, src, f.offsets, offsetDefault, 5, 8, 31); err != nil {
		return nil, err
	}
	src = src[used:]
	if f.matchLengths, used, err = sequenceTable(modes>>2&3, src, f.matchLengths, matchLengthDefault, 6, 9, 52); err != nil {
		return nil, err
	}
	src = src[used:]

	stream, err := newBackwardBits(src)
	if err != nil {
		return nil, err
	}
	literalState := stream.read(f.literalLengths.log)
	offsetState := stream.read(f.offsets.log)
	matchState := stream.read(f.matchLengths.log)
	for i := 0; i < count; i++ {
		offsetCode := int(f.offsets.symbols[offsetState])
		matchCode := int(f.matchLengths.symbols[matchState])
		literalCode := int(f.literalLengths.symbols[literalState])
		if offsetCode > 31 || matchCode > 52 || literalCode > 35 {
			return nil, errZstd
		}
		offsetValue := 1<<offsetCode + int(stream.read(offsetCode))
		matchLength := matchLengthBase[matchCode] + int(stream.read(int(matchLengthBits[matchCode])))
		literalLength := literalLengthBase[literalCode] + int(stream.read(int(literalLengthBits[literalCode])))
		if i < count-1 {
			literalState = f.literalLengths.next(literalState, stream)
			matchState = f.matchLengths.next(matchState, stream)
			offsetState = f.offsets.next(offsetState, stream)
		}

		offset := f.offset(offsetValue, literalLength)
		if literalLength > len(literals) {
			return nil, errZstd
		}
		out = append(out, literals[:literalLength]...)
		literals = literals[literalLength:]
		if offset <= 0 || offset > len(out)-f.start {
			return nil, errZstd
		}
		if len(out)-f.start+matchLength > zstdMaxOutput {
			return nil, errors.New("zstd: frame too large")
		}
		from := len(out) - offset
		if offset >= matchLength {
			out = append(out, out[from:from+matchLength]...)
		} else {
			for j := 0; j < matchLength; j++ { // the match overlaps what it writes
				out = append(out, out[from+j])
			}
		}
	}
	if stream.offset != 0 {
		return nil, errZstd
	}
	return append(out, literals...), nil
}

// offset turns an offset value into a distance, keeping the three most recent distances: values
// 1 to 3 pick one of them (shifted by one when the sequence has no literals).
func (f *zstdFrame) offset(value, literalLength int) int {
	if value > 3 {
		offset := value - 3
		f.recent = [3]int{offset, f.recent[0], f.recent[1]}
		return offset
	}
	index := value - 1
	if literalLength == 0 {
		index++
	}
	if index == 0 {
		return f.recent[0]
	}
	var offset int
	if index < 3 {
		offset = f.recent[index]
	} else {
		offset = f.recent[0] - 1
	}
	if index > 1 {
		f.recent[2] = f.recent[1]
	}
	f.recent[1], f.recent[0] = f.recent[0], offset
	return offset
}

// forwardBits reads bits from the start of a buffer, lowest bit first.
type forwardBits struct {
	src    []byte
	offset int // in bits
	err    error
}

func (b *forwardBits) read(n int) int {
	if b.offset+n > len(b.src)*8 {
		b.err = errZstd
		return 0
	}
	value := readBitsLE(b.src, b.offset, n)
	b.offset += n
	return int(value)
}

func (b *forwardBits) bytesUsed() int { return (b.offset + 7) / 8 }

// backwardBits reads a stream written backwards: from its last byte down to its first, after a
// final 1 bit that marks where it ends. Reading past its start gives zero bits.
type backwardBits struct {
	src    []byte
	offset int // in bits; negative once the stream is used up
}

func newBackwardBits(src []byte) (*backwardBits, error) {
	if len(src) == 0 || src[len(src)-1] == 0 {
		return nil, errZstd
	}
	padding := 8 - (bits.Len8(src[len(src)-1]) - 1)
	return &backwardBits{src: src, offset: len(src)*8 - padding}, nil
}

func (b *backwardBits) read(n int) uint64 {
	if n == 0 {
		return 0
	}
	b.offset -= n
	from, count := b.offset, n
	if from < 0 {
		count += from
		from = 0
	}
	if count <= 0 {
		return 0
	}
	value := readBitsLE(b.src, from, count)
	if b.offset < 0 {
		value <<= uint(-b.offset)
	}
	return value
}

// readBitsLE reads n bits (at most 64) starting at a bit offset, lowest bit first.
func readBitsLE(src []byte, offset, n int) uint64 {
	var value uint64
	shift := 0
	for n > 0 {
		bit := offset & 7
		take := 8 - bit
		if take > n {
			take = n
		}
		chunk := uint64(src[offset>>3]>>bit) & (1<<take - 1)
		value |= chunk << shift
		shift += take
		n -= take
		offset += take
	}
	return value
}

// fseTable decodes FSE (tANS) symbols: each state gives a symbol, and the next state is base
// plus the given number of bits from the stream.
type fseTable struct {
	log     int
	symbols []uint8
	bits    []uint8
	base    []uint16
}

func (t *fseTable) next(state uint64, stream *backwardBits) uint64 {
	return uint64(t.base[state]) + stream.read(int(t.bits[state]))
}

// readFSETable reads a table's normalized symbol counts as RFC 8878 4.1.1 describes.
func readFSETable(in *forwardBits, maxLog, maxSymbol int) (*fseTable, error) {
	log := 5 + in.read(4)
	if log > maxLog {
		return nil, errZstd
	}
	remaining := 1 << log
	var counts []int16
	for remaining > 0 && len(counts) <= maxSymbol && in.err == nil {
		width := bits.Len(uint(remaining + 1))
		value := in.read(width)
		lowerMask := 1<<(width-1) - 1
		threshold := 1<<width - 1 - (remaining + 1)
		if value&lowerMask < threshold {
			in.offset-- // a small value uses one bit less
			value &= lowerMask
		} else if value > lowerMask {
			value -= threshold
		}
		probability := int16(value - 1)
		if probability < 0 {
			remaining--
		} else {
			remaining -= int(probability)
		}
		counts = append(counts, probability)
		if probability == 0 {
			for {
				repeat := in.read(2)
				for i := 0; i < repeat && len(counts) <= maxSymbol; i++ {
					counts = append(counts, 0)
				}
				if repeat != 3 || in.err != nil {
					break
				}
			}
		}
	}
	if in.err != nil || remaining != 0 || len(counts) > maxSymbol+1 {
		return nil, errZstd
	}
	return newFSETable(counts, log)
}

// newFSETable spreads the symbols over the table's states (RFC 8878 4.1.1).
func newFSETable(counts []int16, log int) (*fseTable, error) {
	size := 1 << log
	t := &fseTable{log: log, symbols: make([]uint8, size), bits: make([]uint8, size), base: make([]uint16, size)}
	next := make([]int, len(counts))
	high := size
	for symbol, count := range counts {
		if count == -1 { // "less than 1": one state at the end of the table
			high--
			t.symbols[high] = uint8(symbol)
			next[symbol] = 1
		}
	}
	step := size>>1 + size>>3 + 3
	position := 0
	for symbol, count := range counts {
		if count <= 0 {
			continue
		}
		next[symbol] = int(count)
		for i := 0; i < int(count); i++ {
			t.symbols[position] = uint8(symbol)
			for {
				position = (position + step) & (size - 1)
				if position < high {
					break
				}
			}
		}
	}
	if position != 0 {
		return nil, errZstd
	}
	for state := 0; state < size; state++ {
		symbol := t.symbols[state]
		n := next[symbol]
		next[symbol]++
		t.bits[state] = uint8(log - (bits.Len(uint(n)) - 1))
		t.base[state] = uint16(n<<t.bits[state] - size)
	}
	return t, nil
}

// huffmanTable decodes literals: each state (the next maxBits bits) gives a symbol and how many
// of those bits its code uses.
type huffmanTable struct {
	maxBits int
	symbols []uint8
	bits    []uint8
}

// readHuffmanTable reads a Huffman tree description (RFC 8878 4.2.1) and returns the table and
// how many bytes it took.
func readHuffmanTable(src []byte) (*huffmanTable, int, error) {
	if len(src) < 1 {
		return nil, 0, errZstd
	}
	header := int(src[0])
	var weights []uint8
	used := 1
	if header < 128 { // FSE-compressed weights
		if len(src) < 1+header {
			return nil, 0, errZstd
		}
		data := src[1 : 1+header]
		used += header
		in := &forwardBits{src: data}
		table, err := readFSETable(in, 7, 255)
		if err != nil {
			return nil, 0, err
		}
		stream, err := newBackwardBits(data[in.bytesUsed():])
		if err != nil {
			return nil, 0, err
		}
		// Two states take turns; when the stream runs out, the other state's symbol is the last.
		state1, state2 := stream.read(table.log), stream.read(table.log)
		for len(weights) < 255 {
			weights = append(weights, table.symbols[state1])
			state1 = table.next(state1, stream)
			if stream.offset < 0 {
				weights = append(weights, table.symbols[state2])
				break
			}
			weights = append(weights, table.symbols[state2])
			state2 = table.next(state2, stream)
			if stream.offset < 0 {
				weights = append(weights, table.symbols[state1])
				break
			}
		}
	} else { // weights in 4 bits each
		count := header - 127
		used += (count + 1) / 2
		if len(src) < used {
			return nil, 0, errZstd
		}
		for i := 0; i < count; i++ {
			b := src[1+i/2]
			if i%2 == 0 {
				weights = append(weights, b>>4)
			} else {
				weights = append(weights, b&15)
			}
		}
	}
	if len(weights) > 255 {
		return nil, 0, errZstd
	}

	// The last symbol's weight is implied: the weights must fill a power of two.
	total := 0
	for _, w := range weights {
		if w > 11 {
			return nil, 0, errZstd
		}
		if w > 0 {
			total += 1 << (w - 1)
		}
	}
	if total == 0 {
		return nil, 0, errZstd
	}
	maxBits := bits.Len(uint(total))
	leftover := 1<<maxBits - total
	if leftover&(leftover-1) != 0 {
		return nil, 0, errZstd
	}
	weights = append(weights, uint8(bits.Len(uint(leftover))))
	if maxBits > 11 {
		return nil, 0, errZstd
	}

	codeBits := make([]int, len(weights))
	rankCount := make([]int, maxBits+2)
	for symbol, w := range weights {
		if w > 0 {
			codeBits[symbol] = maxBits + 1 - int(w)
			rankCount[codeBits[symbol]]++
		}
	}
	size := 1 << maxBits
	t := &huffmanTable{maxBits: maxBits, symbols: make([]uint8, size), bits: make([]uint8, size)}
	// Codes are handed out from the longest to the shortest; each code fills the states that
	// start with it.
	rankStart := make([]int, maxBits+2)
	for n := maxBits; n >= 1; n-- {
		rankStart[n-1] = rankStart[n] + rankCount[n]<<(maxBits-n)
		for i := rankStart[n]; i < rankStart[n-1]; i++ {
			t.bits[i] = uint8(n)
		}
	}
	if rankStart[0] != size {
		return nil, 0, errZstd
	}
	for symbol, n := range codeBits {
		if n == 0 {
			continue
		}
		length := 1 << (maxBits - n)
		for i := rankStart[n]; i < rankStart[n]+length; i++ {
			t.symbols[i] = uint8(symbol)
		}
		rankStart[n] += length
	}
	return t, used, nil
}

// decode fills out from one backward stream, which must end exactly.
func (t *huffmanTable) decode(out, src []byte) error {
	if len(out) == 0 && len(src) == 0 {
		return nil
	}
	stream, err := newBackwardBits(src)
	if err != nil {
		return err
	}
	mask := uint64(1)<<t.maxBits - 1
	state := stream.read(t.maxBits)
	for i := range out {
		out[i] = t.symbols[state]
		n := int(t.bits[state])
		state = (state<<n | stream.read(n)) & mask
	}
	if stream.offset != -t.maxBits {
		return errZstd
	}
	return nil
}

// xxhash64 is XXH64 with seed 0, which zstd frames use for their checksum.
func xxhash64(data []byte) uint64 {
	const (
		prime1 uint64 = 11400714785074694791
		prime2 uint64 = 14029467366897019727
		prime3 uint64 = 1609587929392839161
		prime4 uint64 = 9650029242287828579
		prime5 uint64 = 2870177450012600261
	)
	round := func(acc, input uint64) uint64 {
		return bits.RotateLeft64(acc+input*prime2, 31) * prime1
	}
	n := uint64(len(data))
	var h, seed uint64 // seed 0; a variable, so the sums below wrap at run time
	if len(data) >= 32 {
		v1, v2, v3, v4 := seed+prime1+prime2, seed+prime2, seed, seed-prime1
		for len(data) >= 32 {
			v1 = round(v1, binary.LittleEndian.Uint64(data))
			v2 = round(v2, binary.LittleEndian.Uint64(data[8:]))
			v3 = round(v3, binary.LittleEndian.Uint64(data[16:]))
			v4 = round(v4, binary.LittleEndian.Uint64(data[24:]))
			data = data[32:]
		}
		h = bits.RotateLeft64(v1, 1) + bits.RotateLeft64(v2, 7) + bits.RotateLeft64(v3, 12) + bits.RotateLeft64(v4, 18)
		for _, v := range [4]uint64{v1, v2, v3, v4} {
			h = (h^round(0, v))*prime1 + prime4
		}
	} else {
		h = seed + prime5
	}
	h += n
	for len(data) >= 8 {
		h ^= round(0, binary.LittleEndian.Uint64(data))
		h = bits.RotateLeft64(h, 27)*prime1 + prime4
		data = data[8:]
	}
	if len(data) >= 4 {
		h ^= uint64(binary.LittleEndian.Uint32(data)) * prime1
		h = bits.RotateLeft64(h, 23)*prime2 + prime3
		data = data[4:]
	}
	for _, c := range data {
		h ^= uint64(c) * prime5
		h = bits.RotateLeft64(h, 11) * prime1
	}
	h ^= h >> 33
	h *= prime2
	h ^= h >> 29
	h *= prime3
	h ^= h >> 32
	return h
}
