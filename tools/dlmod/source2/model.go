package source2

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"strings"
)

// DXGI formats compiled meshes use for their vertex attributes.
const (
	FormatR32G32B32A32Float = 2
	FormatR32G32B32Float    = 6
	FormatR16G16B16A16Uint  = 12
	FormatR16G16B16A16Sint  = 14
	FormatR8G8B8A8Unorm     = 28
	FormatR8G8B8A8Uint      = 30
	FormatR16G16Float       = 34
	FormatR16G16Unorm       = 35
	FormatR16G16Snorm       = 37
	FormatR16G16Sint        = 38
	FormatR32Float          = 41
	FormatR32Uint           = 42
	FormatR32G32Float       = 16
)

// VertexAttribute is one field of a vertex, such as its position or bone weights.
type VertexAttribute struct {
	Name   string // as the file spells it: "POSITION", "texcoord", "blendweight", ... (see semantic)
	Index  int    // the semantic index: texcoord 1 is a second UV set
	Format int    // a DXGI format number
	Offset int
}

// VertexBuffer is one decoded vertex buffer of a mesh.
type VertexBuffer struct {
	Block      int // the resource block holding it
	Count      int
	Stride     int
	Attributes []VertexAttribute
	Data       []byte
	Desc       *Value // its description in the CTRL block
}

// IndexBuffer is one decoded triangle list.
type IndexBuffer struct {
	Block int
	Count int
	Size  int // 2 or 4 bytes per index
	Data  []byte
	Desc  *Value
}

// DrawCall is a range of triangles drawn with one material.
type DrawCall struct {
	Material     string
	VertexBuffer int // index into the mesh's VertexBuffers
	IndexBuffer  int
	BaseVertex   int
	VertexCount  int
	StartIndex   int
	IndexCount   int
	Desc         *Value
}

// Mesh is one embedded mesh of a model: its buffers and the CRenderMesh describing how to draw
// them with which bones.
type Mesh struct {
	Name          string
	MeshIndex     int
	DataBlock     int
	VertexBuffers []*VertexBuffer
	IndexBuffers  []*IndexBuffer
	DrawCalls     []DrawCall
	Bones         []string // the mesh's own skeleton, which blend indices point into
	Render        *Value   // the MDAT block
	Desc          *Value   // the entry in CTRL's embedded_meshes
}

// Model is a decoded .vmdl_c: the resource, its DATA and CTRL blocks and its embedded meshes.
type Model struct {
	Resource *Resource
	Data     *Value
	Ctrl     *Value
	Meshes   []*Mesh
}

func intOf(v *Value) int {
	if v == nil {
		return 0
	}
	switch v.Kind {
	case Uint16, Uint32, Uint64:
		return int(v.U)
	case Float32, Float64:
		return int(v.F)
	case Bool:
		if v.B {
			return 1
		}
		return 0
	}
	return int(v.I)
}

func boolOf(v *Value) bool { return v != nil && (v.B || intOf(v) != 0) }

// LoadModel decodes a compiled model with embedded meshes stored the current way: MVTX and MIDX
// blocks described by CTRL, and an MDAT block per mesh.
func LoadModel(data []byte) (*Model, error) {
	resource, err := DecodeResource(data)
	if err != nil {
		return nil, err
	}
	m := &Model{Resource: resource}
	if m.Data, _, err = ReadKV3(resource.Block("DATA")); err != nil {
		return nil, fmt.Errorf("model DATA: %v", err)
	}
	if m.Ctrl, _, err = ReadKV3(resource.Block("CTRL")); err != nil {
		return nil, fmt.Errorf("model CTRL: %v", err)
	}
	embedded := m.Ctrl.Get("embedded_meshes")
	if embedded == nil {
		return m, nil
	}
	for _, desc := range embedded.Items {
		if desc.Get("m_nDataBlock") == nil {
			return nil, errors.New("this model stores its meshes in an older layout")
		}
		mesh := &Mesh{
			Name:      desc.Get("m_Name").S,
			MeshIndex: intOf(desc.Get("m_nMeshIndex")),
			DataBlock: intOf(desc.Get("m_nDataBlock")),
			Desc:      desc,
		}
		if mesh.DataBlock < 0 || mesh.DataBlock >= len(resource.Blocks) {
			return nil, fmt.Errorf("mesh %s points at a missing block", mesh.Name)
		}
		if mesh.Render, _, err = ReadKV3(resource.Blocks[mesh.DataBlock].Data); err != nil {
			return nil, fmt.Errorf("mesh %s: %v", mesh.Name, err)
		}
		for _, vb := range desc.Get("m_vertexBuffers").Items {
			buffer := &VertexBuffer{
				Block:  intOf(vb.Get("m_nBlockIndex")),
				Count:  intOf(vb.Get("m_nElementCount")),
				Stride: intOf(vb.Get("m_nElementSizeInBytes")),
				Desc:   vb,
			}
			for _, field := range vb.Get("m_inputLayoutFields").Items {
				buffer.Attributes = append(buffer.Attributes, VertexAttribute{
					Name:   field.Get("m_pSemanticName").S,
					Index:  intOf(field.Get("m_nSemanticIndex")),
					Format: intOf(field.Get("m_Format")),
					Offset: intOf(field.Get("m_nOffset")),
				})
			}
			if buffer.Data, err = m.bufferData(vb, buffer.Count, buffer.Stride, true); err != nil {
				return nil, fmt.Errorf("mesh %s vertices: %v", mesh.Name, err)
			}
			mesh.VertexBuffers = append(mesh.VertexBuffers, buffer)
		}
		for _, ib := range desc.Get("m_indexBuffers").Items {
			buffer := &IndexBuffer{
				Block: intOf(ib.Get("m_nBlockIndex")),
				Count: intOf(ib.Get("m_nElementCount")),
				Size:  intOf(ib.Get("m_nElementSizeInBytes")),
				Desc:  ib,
			}
			if buffer.Data, err = m.bufferData(ib, buffer.Count, buffer.Size, false); err != nil {
				return nil, fmt.Errorf("mesh %s indices: %v", mesh.Name, err)
			}
			mesh.IndexBuffers = append(mesh.IndexBuffers, buffer)
		}
		if objects := mesh.Render.Get("m_sceneObjects"); objects != nil {
			for _, object := range objects.Items {
				for _, call := range object.Get("m_drawCalls").Items {
					draw := DrawCall{
						Material:    call.Get("m_material").S,
						BaseVertex:  intOf(call.Get("m_nBaseVertex")),
						VertexCount: intOf(call.Get("m_nVertexCount")),
						StartIndex:  intOf(call.Get("m_nStartIndex")),
						IndexCount:  intOf(call.Get("m_nIndexCount")),
						IndexBuffer: intOf(call.Get("m_indexBuffer").Get("m_hBuffer")),
						Desc:        call,
					}
					if vbs := call.Get("m_vertexBuffers"); vbs != nil && len(vbs.Items) > 0 {
						draw.VertexBuffer = intOf(vbs.Items[0].Get("m_hBuffer"))
					}
					mesh.DrawCalls = append(mesh.DrawCalls, draw)
				}
			}
		}
		if skeleton := mesh.Render.Get("m_skeleton"); skeleton != nil && skeleton.Get("m_bones") != nil {
			for _, bone := range skeleton.Get("m_bones").Items {
				mesh.Bones = append(mesh.Bones, bone.Get("m_boneName").S)
			}
		}
		m.Meshes = append(m.Meshes, mesh)
	}
	return m, nil
}

func (m *Model) bufferData(desc *Value, count, size int, vertex bool) ([]byte, error) {
	block := intOf(desc.Get("m_nBlockIndex"))
	if block < 0 || block >= len(m.Resource.Blocks) {
		return nil, errors.New("buffer points at a missing block")
	}
	raw := m.Resource.Blocks[block].Data
	if boolOf(desc.Get("m_bCompressedZ3")) {
		return nil, errors.New("buffer is compressed with Z3, which dlmod does not read")
	}
	if boolOf(desc.Get("m_bCompressedZSTD")) { // zstd first, then meshopt if that is set too
		decoded, err := DecodeZstd(raw, 0)
		if err != nil {
			return nil, err
		}
		raw = decoded
	}
	switch {
	case !boolOf(desc.Get("m_bMeshoptCompressed")):
		if len(raw) != count*size {
			return nil, fmt.Errorf("buffer is %d bytes, expected %d", len(raw), count*size)
		}
		return raw, nil
	case vertex:
		return DecodeVertexBuffer(count, size, raw)
	default:
		return DecodeIndexBuffer(count, size, raw)
	}
}

// semantic is an attribute name the way dlmod compares it. Compilers spell them differently:
// Deadlock's own hero models say POSITION, TEXCOORD, BLENDINDICES and blendweight, other
// compiled models say position, texcoord, blendindices and blendweights.
func semantic(name string) string {
	name = strings.ToLower(name)
	if name == "blendweight" {
		return "blendweights"
	}
	return name
}

// Attribute finds a vertex attribute by name (any spelling, see semantic) and semantic index.
func (b *VertexBuffer) Attribute(name string, index int) *VertexAttribute {
	want := semantic(name)
	for i := range b.Attributes {
		if semantic(b.Attributes[i].Name) == want && b.Attributes[i].Index == index {
			return &b.Attributes[i]
		}
	}
	return nil
}

func halfToFloat(h uint16) float32 {
	sign := uint32(h>>15) << 31
	exponent := int(h>>10) & 31
	mantissa := uint32(h & 1023)
	switch {
	case exponent == 0 && mantissa == 0:
		return math.Float32frombits(sign)
	case exponent == 0: // subnormal
		for mantissa&1024 == 0 {
			mantissa <<= 1
			exponent--
		}
		exponent++
		mantissa &= 1023
	case exponent == 31:
		return math.Float32frombits(sign | 0xFF<<23 | mantissa<<13)
	}
	return math.Float32frombits(sign | uint32(exponent+112)<<23 | mantissa<<13)
}

// Floats reads an attribute as up to four floats per vertex, whatever its storage format.
// Compressed normals come back as the decoded normal.
func (b *VertexBuffer) Floats(attribute *VertexAttribute) ([][4]float32, error) {
	out := make([][4]float32, b.Count)
	for i := range out {
		p := b.Data[i*b.Stride+attribute.Offset:]
		v := &out[i]
		switch attribute.Format {
		case FormatR32G32B32A32Float, FormatR32G32B32Float, FormatR32G32Float, FormatR32Float:
			n := map[int]int{FormatR32G32B32A32Float: 4, FormatR32G32B32Float: 3, FormatR32G32Float: 2, FormatR32Float: 1}[attribute.Format]
			for k := 0; k < n; k++ {
				v[k] = math.Float32frombits(binary.LittleEndian.Uint32(p[4*k:]))
			}
		case FormatR16G16Float:
			v[0] = halfToFloat(binary.LittleEndian.Uint16(p))
			v[1] = halfToFloat(binary.LittleEndian.Uint16(p[2:]))
		case FormatR16G16Unorm:
			v[0] = float32(binary.LittleEndian.Uint16(p)) / 65535
			v[1] = float32(binary.LittleEndian.Uint16(p[2:])) / 65535
		case FormatR16G16Snorm:
			v[0] = float32(int16(binary.LittleEndian.Uint16(p))) / 32767
			v[1] = float32(int16(binary.LittleEndian.Uint16(p[2:]))) / 32767
		case FormatR8G8B8A8Unorm:
			for k := 0; k < 4; k++ {
				v[k] = float32(p[k]) / 255
			}
		case FormatR8G8B8A8Uint:
			for k := 0; k < 4; k++ {
				v[k] = float32(p[k])
			}
		case FormatR32Uint:
			normal := DecodeNormal(binary.LittleEndian.Uint32(p))
			v[0], v[1], v[2] = normal[0], normal[1], normal[2]
		default:
			return nil, fmt.Errorf("attribute %s uses format %d, which dlmod does not read", attribute.Name, attribute.Format)
		}
	}
	return out, nil
}

// DecodeNormal unpacks the normal of a compressed normal and tangent frame: 10 bits each for X
// and Y of an octahedral encoding, 11 bits of tangent angle and a bitangent sign.
func DecodeNormal(packed uint32) [3]float32 {
	x := float32((packed>>12)&0x3ff)/1023*2 - 1
	y := float32((packed>>22)&0x3ff)/1023*2 - 1
	z := 1 - float32(math.Abs(float64(x))) - float32(math.Abs(float64(y)))
	if z < 0 {
		t := -z
		if x >= 0 {
			x -= t
		} else {
			x += t
		}
		if y >= 0 {
			y -= t
		} else {
			y += t
		}
	}
	length := float32(math.Sqrt(float64(x*x + y*y + z*z)))
	if length == 0 {
		return [3]float32{0, 0, 1}
	}
	return [3]float32{x / length, y / length, z / length}
}

// EncodeNormal packs a unit normal the way DecodeNormal reads it, with a tangent angle of zero and
// a positive bitangent sign.
func EncodeNormal(n [3]float32) uint32 {
	sum := float32(math.Abs(float64(n[0])) + math.Abs(float64(n[1])) + math.Abs(float64(n[2])))
	if sum == 0 {
		n, sum = [3]float32{0, 0, 1}, 1
	}
	x, y, z := n[0]/sum, n[1]/sum, n[2]/sum
	if z < 0 { // fold the lower half over the diagonals, as octahedral encoding does
		ox, oy := x, y
		x = (1 - float32(math.Abs(float64(oy)))) * sign(ox)
		y = (1 - float32(math.Abs(float64(ox)))) * sign(oy)
	}
	quantize := func(f float32) uint32 {
		q := math.Round(float64((f + 1) / 2 * 1023))
		return uint32(math.Max(0, math.Min(1023, q)))
	}
	return quantize(x)<<12 | quantize(y)<<22 | 1
}

func sign(f float32) float32 {
	if f < 0 {
		return -1
	}
	return 1
}

// Indices returns the triangle list as vertex numbers.
func (b *IndexBuffer) Indices() []uint32 {
	out := make([]uint32, b.Count)
	for i := range out {
		if b.Size == 2 {
			out[i] = uint32(binary.LittleEndian.Uint16(b.Data[2*i:]))
		} else {
			out[i] = binary.LittleEndian.Uint32(b.Data[4*i:])
		}
	}
	return out
}
