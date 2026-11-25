package source2

import (
	"archive/tar"
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// zstdSamples makes the test inputs. The files in testdata are these, put in a tar archive and
// compressed by libzstd 1.5 (Windows' tar): "tar --zstd --options zstd:compression-level=N".
// ZSTD_SAMPLES=<dir> writes them out to compress again.
func zstdSamples() map[string][]byte {
	seed := uint32(2463534242)
	next := func() uint32 { // xorshift32, so the samples never change
		seed ^= seed << 13
		seed ^= seed >> 17
		seed ^= seed << 5
		return seed
	}
	words := strings.Fields("the hero walks to the lane and casts a spell on the enemy tower while the " +
		"creeps push forward; Deadlock patch notes mention souls, items, abilities, cooldowns, " +
		"spirit power and weapon damage. 0123456789 ABCDEFGHIJKLMNOPQRSTUVWXYZ {}[]()<>=+-*/")
	var text bytes.Buffer
	for text.Len() < 150_000 {
		text.WriteString(words[next()%uint32(len(words))])
		if next()%9 == 0 {
			text.WriteByte('\n')
		} else {
			text.WriteByte(' ')
		}
	}
	random := make([]byte, 20_000)
	for i := range random {
		random[i] = byte(next())
	}
	nibbles := make([]byte, 24_000) // a few symbols: Huffman weights written 4 bits each
	for i := range nibbles {
		a, b := byte(next()%16), byte(next()%16)
		nibbles[i] = min(a, b)
	}
	var runs bytes.Buffer
	for i := 0; i < 40; i++ {
		runs.Write(bytes.Repeat([]byte{byte('a' + i%26)}, 1+int(next()%9000)))
		runs.WriteString("break")
	}
	return map[string][]byte{"text.txt": text.Bytes(), "random.bin": random, "runs.txt": runs.Bytes(), "small.txt": []byte("hello, zstd\n"),
		"nibbles.bin": nibbles}
}

func TestZstdDecodesLibzstdOutput(t *testing.T) {
	samples := zstdSamples()
	if dir := os.Getenv("ZSTD_SAMPLES"); dir != "" {
		for name, data := range samples {
			os.WriteFile(filepath.Join(dir, name), data, 0o644)
		}
	}
	archives, _ := filepath.Glob(filepath.Join("testdata", "*.tar.zst"))
	if len(archives) < 3 {
		t.Fatalf("test archives missing: %v", archives)
	}
	for _, archive := range archives {
		compressed, err := os.ReadFile(archive)
		if err != nil {
			t.Fatal(err)
		}
		decoded, err := DecodeZstd(compressed, 0)
		if err != nil {
			t.Fatalf("%s: %v", archive, err)
		}
		files := tar.NewReader(bytes.NewReader(decoded))
		seen := 0
		for {
			header, err := files.Next()
			if err == io.EOF {
				break
			}
			if err != nil {
				t.Fatalf("%s: %v", archive, err)
			}
			content, _ := io.ReadAll(files)
			want, ok := samples[header.Name]
			if !ok || !bytes.Equal(content, want) {
				t.Fatalf("%s: %s decoded wrongly (%d bytes, want %d)", archive, header.Name, len(content), len(want))
			}
			seen++
		}
		if seen != len(samples) {
			t.Fatalf("%s: %d files, want %d", archive, seen, len(samples))
		}
	}
}

func TestZstdFramesByHand(t *testing.T) {
	frame := func(descriptor byte, rest ...byte) []byte {
		return append([]byte{0x28, 0xB5, 0x2F, 0xFD, descriptor}, rest...)
	}
	// Single segment, 1-byte content size 5, one last raw block "hello".
	raw := frame(0x20, 5, 5<<3|1, 0, 0, 'h', 'e', 'l', 'l', 'o')
	// One last RLE block of 4 'z', no content size, window descriptor 0.
	rle := frame(0x00, 0, 4<<3|1<<1|1, 0, 0, 'z')
	skippable := []byte{0x50, 0x2A, 0x4D, 0x18, 2, 0, 0, 0, 9, 9}
	all := append(append(append([]byte{}, raw...), skippable...), rle...)
	got, err := DecodeZstd(all, 9)
	if err != nil || string(got) != "hellozzzz" {
		t.Fatalf("got %q, %v", got, err)
	}
	if _, err := DecodeZstd(raw, 6); err == nil {
		t.Fatal("a wrong expected size must fail")
	}
	if _, err := DecodeZstd(frame(0x20, 6, 5<<3|1, 0, 0, 'h', 'e', 'l', 'l', 'o'), 0); err == nil {
		t.Fatal("a frame shorter than its content size must fail")
	}
	if _, err := DecodeZstd(frame(0x01, 0, 7, 5<<3|1, 0, 0, 'h', 'e', 'l', 'l', 'o'), 0); err == nil {
		t.Fatal("a frame that needs a dictionary must fail")
	}
	if _, err := DecodeZstd(raw[:len(raw)-1], 0); err == nil {
		t.Fatal("a cut frame must fail")
	}
	// With a checksum: the low 32 bits of XXH64("hello").
	sum := uint32(xxhash64([]byte("hello")))
	checked := frame(0x24, 5, 5<<3|1, 0, 0, 'h', 'e', 'l', 'l', 'o', byte(sum), byte(sum>>8), byte(sum>>16), byte(sum>>24))
	if got, err := DecodeZstd(checked, 0); err != nil || string(got) != "hello" {
		t.Fatalf("checked frame: %q %v", got, err)
	}
	checked[len(checked)-1] ^= 1
	if _, err := DecodeZstd(checked, 0); err == nil || !strings.Contains(err.Error(), "checksum") {
		t.Fatalf("a wrong checksum must fail: %v", err)
	}
}

func TestXXHash64(t *testing.T) {
	for input, want := range map[string]uint64{"": 0xEF46DB3751D8E999, "abc": 0x44BC2CF5AD770999} {
		if got := xxhash64([]byte(input)); got != want {
			t.Errorf("XXH64(%q) = %#x, want %#x", input, got, want)
		}
	}
}

// Every cut of a real frame must fail cleanly rather than panic.
func TestZstdRejectsDamage(t *testing.T) {
	compressed, err := os.ReadFile(filepath.Join("testdata", "level19.tar.zst"))
	if err != nil {
		t.Fatal(err)
	}
	for cut := 0; cut < len(compressed); cut += 97 {
		if _, err := DecodeZstd(compressed[:cut], 0); err == nil && cut > 0 {
			t.Fatalf("a frame cut at %d decoded", cut)
		}
	}
	for i := 4; i < len(compressed); i += 89 {
		damaged := append([]byte{}, compressed...)
		damaged[i] ^= 0x5A
		DecodeZstd(damaged, 0) // must not panic; a damaged byte may still decode to something
	}
}

// Literals given as one byte repeated (a compressed block with RLE literals and no sequences).
func TestZstdRepeatedLiterals(t *testing.T) {
	block := []byte{5<<3 | 1, 'x', 0} // literals: RLE, 5 bytes of 'x'; then 0 sequences
	frame := append([]byte{0x28, 0xB5, 0x2F, 0xFD, 0x20, 5, byte(len(block))<<3 | 2<<1 | 1, 0, 0}, block...)
	if got, err := DecodeZstd(frame, 5); err != nil || string(got) != "xxxxx" {
		t.Fatalf("got %q, %v", got, err)
	}
}

// FuzzDecodeZstd checks that no input makes the decoder panic or run away. go test runs the
// seeds; "go test -fuzz FuzzDecodeZstd ./source2" searches further.
func FuzzDecodeZstd(f *testing.F) {
	archives, _ := filepath.Glob(filepath.Join("testdata", "*.tar.zst"))
	for _, archive := range archives {
		data, _ := os.ReadFile(archive)
		f.Add(data[:min(len(data), 4096)])
	}
	f.Add([]byte{0x28, 0xB5, 0x2F, 0xFD, 0x20, 5, 3<<3 | 2<<1 | 1, 0, 0, 5<<3 | 1, 'x', 0})
	f.Fuzz(func(t *testing.T, data []byte) {
		out, err := DecodeZstd(data, 0)
		if err == nil && len(out) > zstdMaxOutput {
			t.Fatalf("decoded %d bytes", len(out))
		}
	})
}
