// Package source2 reads and writes the Source 2 resource formats modding needs: binary KeyValues3
// (KV3) and compiled resources such as sounds. The layouts follow ValveResourceFormat
// (github.com/ValveResourceFormat/ValveResourceFormat), which reads every Source 2 game's files.
package source2

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"
)

// Kind is the type of a KV3 value. Numbers keep their exact width, because the engine reads
// fields by type.
type Kind uint8

const (
	Null Kind = iota
	Bool
	Int16
	Uint16
	Int32
	Uint32
	Int64
	Uint64
	Float32
	Float64
	String
	Blob
	Array
	Object
)

// Value is one KV3 value. Objects keep their members in order.
type Value struct {
	Kind   Kind
	B      bool
	I      int64
	U      uint64
	F      float64
	S      string
	Blob   []byte
	Items  []*Value // arrays
	Keys   []string // objects
	Fields []*Value // objects, in the order of Keys
	Flag   byte
}

func NewObject() *Value               { return &Value{Kind: Object} }
func NewString(s string) *Value       { return &Value{Kind: String, S: s} }
func NewInt32(i int32) *Value         { return &Value{Kind: Int32, I: int64(i)} }
func NewUint32(u uint32) *Value       { return &Value{Kind: Uint32, U: uint64(u)} }
func NewFloat32(f float32) *Value     { return &Value{Kind: Float32, F: float64(f)} }
func NewBool(b bool) *Value           { return &Value{Kind: Bool, B: b} }
func NewArray(items ...*Value) *Value { return &Value{Kind: Array, Items: items} }

// Set adds or replaces an object member and returns the object.
func (v *Value) Set(key string, value *Value) *Value {
	for i, existing := range v.Keys {
		if existing == key {
			v.Fields[i] = value
			return v
		}
	}
	v.Keys = append(v.Keys, key)
	v.Fields = append(v.Fields, value)
	return v
}

// Get returns an object member, or nil.
func (v *Value) Get(key string) *Value {
	if v == nil {
		return nil
	}
	for i, existing := range v.Keys {
		if existing == key {
			return v.Fields[i]
		}
	}
	return nil
}

// Node types of binary KV3.
const (
	nodeNull           = 1
	nodeBoolean        = 2
	nodeInt64          = 3
	nodeUint64         = 4
	nodeDouble         = 5
	nodeString         = 6
	nodeBlob           = 7
	nodeArray          = 8
	nodeObject         = 9
	nodeArrayTyped     = 10
	nodeInt32          = 11
	nodeUint32         = 12
	nodeTrue           = 13
	nodeFalse          = 14
	nodeInt64Zero      = 15
	nodeInt64One       = 16
	nodeDoubleZero     = 17
	nodeDoubleOne      = 18
	nodeFloat          = 19
	nodeInt16          = 20
	nodeUint16         = 21
	nodeInt8           = 22
	nodeUint8          = 23
	nodeArrayByteLen   = 24
	nodeArrayAuxiliary = 25
	nodeUint32Zero     = 26
	nodeUint32One      = 27
)

const (
	magic4      = 0x4B563304 // "KV3" version 4
	magic5      = 0x4B563305 // version 5, which splits the data into two buffers
	trailer     = 0xFFEEDD00
	compression = 0 // uncompressed; the engine and every reader accept it
)

// FormatGeneric is the KV3 format id Valve's compilers write for resource blocks.
var FormatGeneric = [16]byte{0x7c, 0x16, 0x12, 0x74, 0xe9, 0x06, 0x98, 0x46, 0xaf, 0xf2, 0xe6, 0x3e, 0xb5, 0x90, 0x37, 0xe7}

// --- writing ---

type writer struct {
	bytes1, bytes2, bytes4, bytes8, types []byte
	strings                               []string
	stringIDs                             map[string]int32
	objects, arrays                       int
	blobs                                 [][]byte
}

func (w *writer) stringID(s string) int32 {
	if s == "" {
		return -1
	}
	if id, ok := w.stringIDs[s]; ok {
		return id
	}
	id := int32(len(w.strings))
	w.strings = append(w.strings, s)
	w.stringIDs[s] = id
	return id
}

func (w *writer) u32(v uint32) { w.bytes4 = binary.LittleEndian.AppendUint32(w.bytes4, v) }

func (w *writer) typ(t byte, flag byte) {
	if flag != 0 {
		w.types = append(w.types, t|0x80, flag)
		return
	}
	w.types = append(w.types, t)
}

func (w *writer) value(v *Value) error {
	switch v.Kind {
	case Null:
		w.typ(nodeNull, v.Flag)
	case Bool:
		if v.B {
			w.typ(nodeTrue, v.Flag)
		} else {
			w.typ(nodeFalse, v.Flag)
		}
	case Int16:
		w.typ(nodeInt16, v.Flag)
		w.bytes2 = binary.LittleEndian.AppendUint16(w.bytes2, uint16(int16(v.I)))
	case Uint16:
		w.typ(nodeUint16, v.Flag)
		w.bytes2 = binary.LittleEndian.AppendUint16(w.bytes2, uint16(v.U))
	case Int32:
		w.typ(nodeInt32, v.Flag)
		w.u32(uint32(int32(v.I)))
	case Uint32:
		w.typ(nodeUint32, v.Flag)
		w.u32(uint32(v.U))
	case Int64:
		switch v.I {
		case 0:
			w.typ(nodeInt64Zero, v.Flag)
		case 1:
			w.typ(nodeInt64One, v.Flag)
		default:
			w.typ(nodeInt64, v.Flag)
			w.bytes8 = binary.LittleEndian.AppendUint64(w.bytes8, uint64(v.I))
		}
	case Uint64:
		w.typ(nodeUint64, v.Flag)
		w.bytes8 = binary.LittleEndian.AppendUint64(w.bytes8, v.U)
	case Float32:
		w.typ(nodeFloat, v.Flag)
		w.u32(math.Float32bits(float32(v.F)))
	case Float64:
		switch {
		case math.Float64bits(v.F) == 0:
			w.typ(nodeDoubleZero, v.Flag)
		case v.F == 1:
			w.typ(nodeDoubleOne, v.Flag)
		default:
			w.typ(nodeDouble, v.Flag)
			w.bytes8 = binary.LittleEndian.AppendUint64(w.bytes8, math.Float64bits(v.F))
		}
	case String:
		w.typ(nodeString, v.Flag)
		w.u32(uint32(w.stringID(v.S)))
	case Blob:
		w.typ(nodeBlob, v.Flag)
		w.blobs = append(w.blobs, v.Blob)
	case Array:
		w.typ(nodeArray, v.Flag)
		w.arrays++
		w.u32(uint32(len(v.Items)))
		for _, item := range v.Items {
			if err := w.value(item); err != nil {
				return err
			}
		}
	case Object:
		w.typ(nodeObject, v.Flag)
		w.objects++
		w.u32(uint32(len(v.Keys)))
		for i, key := range v.Keys {
			w.u32(uint32(w.stringID(key)))
			if err := w.value(v.Fields[i]); err != nil {
				return err
			}
		}
	default:
		return fmt.Errorf("kv3: cannot write %v", v.Kind)
	}
	return nil
}

func pad(buffer []byte, alignment int) []byte {
	for len(buffer)%alignment != 0 {
		buffer = append(buffer, 0)
	}
	return buffer
}

// WriteKV3 encodes a value as uncompressed binary KV3 version 4, the layout of ValveResourceFormat's
// serializer.
func WriteKV3(root *Value, format [16]byte) ([]byte, error) {
	w := &writer{stringIDs: map[string]int32{}}
	w.u32(0) // the string count, filled in below
	if err := w.value(root); err != nil {
		return nil, err
	}
	binary.LittleEndian.PutUint32(w.bytes4, uint32(len(w.strings)))
	var buffer []byte
	buffer = append(buffer, w.bytes1...)
	if len(w.bytes2) > 0 {
		buffer = append(pad(buffer, 2), w.bytes2...)
	}
	buffer = append(pad(buffer, 4), w.bytes4...)
	buffer = pad(buffer, 8) // versions before 5 align for the 8-byte lane even when it is empty
	buffer = append(buffer, w.bytes8...)
	stringsStart := len(buffer)
	for _, s := range w.strings {
		buffer = append(append(buffer, s...), 0)
	}
	buffer = append(buffer, w.types...)
	countTypes := len(buffer) - stringsStart
	var blobBytes []byte
	for _, blob := range w.blobs {
		buffer = binary.LittleEndian.AppendUint32(buffer, uint32(len(blob)))
		blobBytes = append(blobBytes, blob...)
	}
	buffer = binary.LittleEndian.AppendUint32(buffer, trailer)

	out := binary.LittleEndian.AppendUint32(nil, magic4)
	out = append(out, format[:]...)
	out = binary.LittleEndian.AppendUint32(out, compression)
	out = binary.LittleEndian.AppendUint16(out, 0) // compression dictionary
	out = binary.LittleEndian.AppendUint16(out, 0) // compression frame size
	for _, field := range []int{len(w.bytes1), len(w.bytes4) / 4, len(w.bytes8) / 8, countTypes} {
		out = binary.LittleEndian.AppendUint32(out, uint32(field))
	}
	out = binary.LittleEndian.AppendUint16(out, uint16(min(w.objects, math.MaxUint16)))
	out = binary.LittleEndian.AppendUint16(out, uint16(min(w.arrays, math.MaxUint16)))
	for _, field := range []int{len(buffer), len(buffer), len(w.blobs), len(blobBytes), len(w.bytes2) / 2, 0} {
		out = binary.LittleEndian.AppendUint32(out, uint32(field))
	}
	out = append(out, buffer...)
	if len(w.blobs) > 0 {
		// Uncompressed blobs follow the buffer, closed by the trailer like ValveResourceFormat writes them.
		out = binary.LittleEndian.AppendUint32(append(out, blobBytes...), trailer)
	}
	return out, nil
}

// --- reading ---

type lanes struct{ b1, b2, b4, b8 []byte }

type reader struct {
	version int
	main    lanes
	aux     *lanes // the lanes ARRAY_TYPE_AUXILIARY_BUFFER elements come from
	objects []byte // version 5 keeps object member counts apart
	types   []byte
	strings []string
	blobs   [][]byte
}

// stringLanes is where the string count is: the auxiliary buffer in version 5, which is the main
// buffer before it.
func (r *reader) stringLanes() *lanes { return r.aux }

var errShort = errors.New("kv3: data ends early")

func take(lane *[]byte, n int) ([]byte, error) {
	if len(*lane) < n {
		return nil, errShort
	}
	value := (*lane)[:n]
	*lane = (*lane)[n:]
	return value, nil
}

func (r *reader) typ() (byte, byte, error) {
	if len(r.types) == 0 {
		return 0, 0, errShort
	}
	t, flag := r.types[0], byte(0)
	r.types = r.types[1:]
	if t&0x80 != 0 {
		if len(r.types) == 0 {
			return 0, 0, errShort
		}
		flag, r.types = r.types[0], r.types[1:]
	}
	if t&0x40 != 0 && len(r.types) > 0 {
		r.types = r.types[1:]
	}
	return t & 0x3F, flag, nil
}

func (r *reader) str(id int32) string {
	if id >= 0 && int(id) < len(r.strings) {
		return r.strings[id]
	}
	return ""
}

func (r *reader) read(t, flag byte, lane *lanes) (*Value, error) {
	get := func(l *[]byte, n int) ([]byte, error) { return take(l, n) }
	v := &Value{Flag: flag}
	switch t {
	case nodeNull:
		v.Kind = Null
	case nodeTrue, nodeFalse:
		v.Kind, v.B = Bool, t == nodeTrue
	case nodeBoolean:
		b, err := get(&lane.b1, 1)
		if err != nil {
			return nil, err
		}
		v.Kind, v.B = Bool, b[0] != 0
	case nodeInt64Zero, nodeInt64One:
		v.Kind, v.I = Int64, int64(t-nodeInt64Zero)
	case nodeDoubleZero, nodeDoubleOne:
		v.Kind, v.F = Float64, float64(t-nodeDoubleZero)
	case nodeUint32Zero, nodeUint32One:
		v.Kind, v.U = Uint32, uint64(t-nodeUint32Zero)
	case nodeInt8, nodeUint8:
		b, err := get(&lane.b1, 1)
		if err != nil {
			return nil, err
		}
		if t == nodeInt8 {
			v.Kind, v.I = Int32, int64(int8(b[0]))
		} else {
			v.Kind, v.U = Uint32, uint64(b[0])
		}
	case nodeInt16, nodeUint16:
		b, err := get(&lane.b2, 2)
		if err != nil {
			return nil, err
		}
		if t == nodeInt16 {
			v.Kind, v.I = Int16, int64(int16(binary.LittleEndian.Uint16(b)))
		} else {
			v.Kind, v.U = Uint16, uint64(binary.LittleEndian.Uint16(b))
		}
	case nodeInt32, nodeUint32, nodeFloat:
		b, err := get(&lane.b4, 4)
		if err != nil {
			return nil, err
		}
		bits := binary.LittleEndian.Uint32(b)
		switch t {
		case nodeInt32:
			v.Kind, v.I = Int32, int64(int32(bits))
		case nodeUint32:
			v.Kind, v.U = Uint32, uint64(bits)
		default:
			v.Kind, v.F = Float32, float64(math.Float32frombits(bits))
		}
	case nodeInt64, nodeUint64, nodeDouble:
		b, err := get(&lane.b8, 8)
		if err != nil {
			return nil, err
		}
		bits := binary.LittleEndian.Uint64(b)
		switch t {
		case nodeInt64:
			v.Kind, v.I = Int64, int64(bits)
		case nodeUint64:
			v.Kind, v.U = Uint64, bits
		default:
			v.Kind, v.F = Float64, math.Float64frombits(bits)
		}
	case nodeBlob:
		if len(r.blobs) == 0 {
			return nil, errShort
		}
		v.Kind, v.Blob, r.blobs = Blob, r.blobs[0], r.blobs[1:]
	case nodeString:
		b, err := get(&r.main.b4, 4)
		if err != nil {
			return nil, err
		}
		v.Kind, v.S = String, r.str(int32(binary.LittleEndian.Uint32(b)))
	case nodeArray:
		b, err := get(&r.main.b4, 4)
		if err != nil {
			return nil, err
		}
		v.Kind = Array
		for n := binary.LittleEndian.Uint32(b); n > 0; n-- {
			et, ef, err := r.typ()
			if err != nil {
				return nil, err
			}
			item, err := r.read(et, ef, &r.main)
			if err != nil {
				return nil, err
			}
			v.Items = append(v.Items, item)
		}
	case nodeArrayTyped, nodeArrayByteLen, nodeArrayAuxiliary:
		var count int
		if t == nodeArrayTyped {
			b, err := get(&r.main.b4, 4)
			if err != nil {
				return nil, err
			}
			count = int(binary.LittleEndian.Uint32(b))
		} else {
			b, err := get(&r.main.b1, 1)
			if err != nil {
				return nil, err
			}
			count = int(b[0])
		}
		et, ef, err := r.typ()
		if err != nil {
			return nil, err
		}
		v.Kind = Array
		elements := &r.main
		if t == nodeArrayAuxiliary {
			elements = r.aux
		}
		v.Items = make([]*Value, 0, count)
		for i := 0; i < count; i++ {
			item, err := r.read(et, ef, elements)
			if err != nil {
				return nil, err
			}
			v.Items = append(v.Items, item)
		}
	case nodeObject:
		counts := &r.main.b4
		if r.version >= 5 {
			counts = &r.objects
		}
		b, err := get(counts, 4)
		if err != nil {
			return nil, err
		}
		v.Kind = Object
		for n := binary.LittleEndian.Uint32(b); n > 0; n-- {
			nb, err := get(&r.main.b4, 4)
			if err != nil {
				return nil, err
			}
			et, ef, err := r.typ()
			if err != nil {
				return nil, err
			}
			field, err := r.read(et, ef, &r.main)
			if err != nil {
				return nil, err
			}
			v.Keys = append(v.Keys, r.str(int32(binary.LittleEndian.Uint32(nb))))
			v.Fields = append(v.Fields, field)
		}
	default:
		return nil, fmt.Errorf("kv3: node type %d is not supported", t)
	}
	return v, nil
}

// ReadKV3 decodes binary KV3 version 4 or 5, uncompressed, LZ4- or zstd-compressed, with binary
// blobs (not yet with zstd).
// Version 5 is what current compilers write for models, materials and textures: it splits the
// data into two buffers, the strings with an auxiliary lane for packed arrays, and the rest.
func ReadKV3(data []byte) (*Value, [16]byte, error) {
	var format [16]byte
	if len(data) < 72 {
		return nil, format, errors.New("kv3: not a binary KV3 block")
	}
	version, headerSize := 0, 0
	switch binary.LittleEndian.Uint32(data) {
	case magic4:
		version, headerSize = 4, 72
	case magic5:
		version, headerSize = 5, 120
	default:
		return nil, format, errors.New("kv3: not a version 4 or 5 binary KV3 block")
	}
	if len(data) < headerSize {
		return nil, format, errShort
	}
	copy(format[:], data[4:20])
	method := binary.LittleEndian.Uint32(data[20:])
	if method > 2 {
		return nil, format, fmt.Errorf("kv3: compression method %d is not supported", method)
	}
	field := func(offset int) int { return int(binary.LittleEndian.Uint32(data[offset:])) }
	count1, count4, count8, countTypes := field(28), field(32), field(36), field(40)
	blocks, blobBytes, count2 := field(56), field(60), field(64)
	body := data[headerSize:]
	unpack := func(compressed, size int) ([]byte, error) {
		stored := size
		if method != 0 {
			stored = compressed
		}
		if stored < 0 || len(body) < stored {
			return nil, errShort
		}
		raw := body[:stored]
		body = body[stored:]
		switch method {
		case 0:
			return raw, nil
		case 2:
			return DecodeZstd(raw, size)
		}
		return decodeLZ4Block(raw, size)
	}

	r := &reader{}
	r.aux = &r.main // before version 5 the auxiliary lane is the main buffer
	var rest []byte // what follows the types: blob lengths, trailer, LZ4 frame sizes
	readStrings := func(source *[]byte) error {
		countBytes, err := take(&r.stringLanes().b4, 4)
		if err != nil {
			return err
		}
		for n := binary.LittleEndian.Uint32(countBytes); n > 0; n-- {
			end := 0
			for end < len(*source) && (*source)[end] != 0 {
				end++
			}
			if end >= len(*source) {
				return errShort
			}
			r.strings = append(r.strings, string((*source)[:end]))
			*source = (*source)[end+1:]
		}
		return nil
	}
	if version == 4 {
		buffer, err := unpack(field(52), field(48))
		if err != nil {
			return nil, format, err
		}
		offset, err := splitLanes(buffer, 0, count1, count2, count4, count8, true, &r.main)
		if err != nil {
			return nil, format, err
		}
		tail := buffer[offset:]
		if err := readStrings(&tail); err != nil {
			return nil, format, err
		}
		typesLength := countTypes - (len(buffer) - offset - len(tail))
		if typesLength < 0 || typesLength > len(tail) {
			return nil, format, errShort
		}
		r.types, rest = tail[:typesLength], tail[typesLength:]
	} else {
		buffer1, err := unpack(field(76), field(72))
		if err != nil {
			return nil, format, err
		}
		buffer2, err := unpack(field(84), field(80))
		if err != nil {
			return nil, format, err
		}
		r.aux = &lanes{}
		if _, err := splitLanes(buffer1, 0, count1, count2, count4, count8, false, r.aux); err != nil {
			return nil, format, err
		}
		if err := readStrings(&r.aux.b1); err != nil {
			return nil, format, err
		}
		objects := field(108) * 4
		if objects > len(buffer2) {
			return nil, format, errShort
		}
		r.objects = buffer2[:objects]
		offset, err := splitLanes(buffer2, objects, field(88), field(92), field(96), field(100), false, &r.main)
		if err != nil {
			return nil, format, err
		}
		if offset+countTypes > len(buffer2) {
			return nil, format, errShort
		}
		r.types, rest = buffer2[offset:offset+countTypes], buffer2[offset+countTypes:]
	}

	if blocks > 0 {
		// Each blob's length, the trailer, then (LZ4) each compressed frame's size. The blobs
		// follow the buffers, each split into frames that may refer back to earlier ones.
		var lengths []int
		for i := 0; i < blocks; i++ {
			b, err := take(&rest, 4)
			if err != nil {
				return nil, format, err
			}
			lengths = append(lengths, int(binary.LittleEndian.Uint32(b)))
		}
		if _, err := take(&rest, 4); err != nil {
			return nil, format, err
		}
		if method == 2 {
			return nil, format, errors.New("kv3: zstd-compressed binary blobs are not supported")
		}
		var raw []byte
		if method == 0 {
			if len(body) < blobBytes {
				return nil, format, errShort
			}
			raw = body[:blobBytes]
		} else {
			var frames []int
			for len(rest) >= 2 {
				frames = append(frames, int(binary.LittleEndian.Uint16(rest)))
				rest = rest[2:]
			}
			frameSize := int(binary.LittleEndian.Uint16(data[26:]))
			var err error
			if raw, err = decodeLZ4Chain(body, frames, frameSize, blobBytes); err != nil {
				return nil, format, err
			}
		}
		for _, length := range lengths {
			if length > len(raw) {
				return nil, format, errShort
			}
			r.blobs, raw = append(r.blobs, raw[:length]), raw[length:]
		}
	}
	r.version = version
	t, flag, err := r.typ()
	if err != nil {
		return nil, format, err
	}
	root, err := r.read(t, flag, &r.main)
	return root, format, err
}

// splitLanes cuts a buffer into its 1, 2, 4 and 8 byte lanes from offset, each aligned to its
// width, and returns where they end. Version 4 aligns to 8 even when the 8 byte lane is empty.
func splitLanes(buffer []byte, offset, count1, count2, count4, count8 int, alignEmpty bool, out *lanes) (int, error) {
	section := func(count, size int) ([]byte, error) {
		if count == 0 {
			return nil, nil
		}
		for offset%size != 0 {
			offset++
		}
		end := offset + count*size
		if count < 0 || end > len(buffer) {
			return nil, errShort
		}
		part := buffer[offset:end]
		offset = end
		return part, nil
	}
	var err error
	if out.b1, err = section(count1, 1); err != nil {
		return 0, err
	}
	if out.b2, err = section(count2, 2); err != nil {
		return 0, err
	}
	if out.b4, err = section(count4, 4); err != nil {
		return 0, err
	}
	if out.b8, err = section(count8, 8); err != nil {
		return 0, err
	}
	if count8 == 0 && alignEmpty {
		for offset%8 != 0 {
			offset++
		}
	}
	return offset, nil
}

// decodeLZ4Chain decodes chained LZ4 frames: each holds up to frameSize bytes and may refer back
// to the bytes the earlier frames produced.
func decodeLZ4Chain(src []byte, frames []int, frameSize, size int) ([]byte, error) {
	out := make([]byte, 0, size)
	for _, length := range frames {
		if length > len(src) {
			return nil, errShort
		}
		before := len(out)
		var err error
		if out, err = lz4Append(out, src[:length]); err != nil {
			return nil, err
		}
		if len(out)-before > frameSize && frameSize > 0 {
			return nil, errors.New("lz4: frame larger than the frame size")
		}
		src = src[length:]
	}
	if len(out) != size {
		return nil, fmt.Errorf("lz4: decoded %d blob bytes, expected %d", len(out), size)
	}
	return out, nil
}

// decodeLZ4Block decodes one raw LZ4 block (no frame header) of a known decoded size.
func decodeLZ4Block(src []byte, size int) ([]byte, error) {
	out, err := lz4Append(make([]byte, 0, size), src)
	if err == nil && len(out) != size {
		err = fmt.Errorf("lz4: decoded %d bytes, expected %d", len(out), size)
	}
	return out, err
}

// lz4Append decodes one LZ4 block onto out; matches may reach into what out already holds.
func lz4Append(out, src []byte) ([]byte, error) {
	for i := 0; i < len(src); {
		token := src[i]
		i++
		literals := int(token >> 4)
		if literals == 15 {
			for i < len(src) {
				b := src[i]
				i++
				literals += int(b)
				if b != 255 {
					break
				}
			}
		}
		if i+literals > len(src) {
			return nil, errors.New("lz4: literals run past the input")
		}
		out = append(out, src[i:i+literals]...)
		i += literals
		if i >= len(src) {
			break // the last sequence has literals only
		}
		if i+2 > len(src) {
			return nil, errors.New("lz4: missing match offset")
		}
		distance := int(src[i]) | int(src[i+1])<<8
		i += 2
		length := int(token&15) + 4
		if token&15 == 15 {
			for i < len(src) {
				b := src[i]
				i++
				length += int(b)
				if b != 255 {
					break
				}
			}
		}
		if distance == 0 || distance > len(out) {
			return nil, errors.New("lz4: bad match offset")
		}
		start := len(out) - distance
		for k := 0; k < length; k++ {
			out = append(out, out[start+k])
		}
	}
	return out, nil
}
