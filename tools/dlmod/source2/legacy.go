package source2

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"strings"
)

// Older compiled models (ability props such as Ivy's thorn bomb) keep each mesh's buffers in one
// MBUF block of the older VBIB layout instead of MVTX and MIDX blocks, and the mesh's description
// in an MDAT block that CTRL's embedded_meshes points at through data_block and vbib_block. The
// layout follows Source 2 Viewer's VBIB reader:
//
//	header:  vertex buffer offset (from the block's start), vertex buffer count,
//	         index buffer offset (from that field), index buffer count
//	buffer:  element count, element size (with flags in the top bits), attribute offset (from
//	         that field), attribute count, data offset (from that field), data size; 24 bytes
//	field:   semantic name (32 bytes), semantic index, DXGI format, offset, slot, slot type,
//	         instance step rate; 56 bytes
//
// Bit 0x4000000 of the element size is set when the data is NOT meshopt compressed; 0x8000000
// marks zstd.

const (
	vbibPlain = 0x4000000
	vbibZstd  = 0x8000000
	vbibSize  = 0x3FFFFFF
)

// legacyBuffer is one decoded buffer of a VBIB block.
type legacyBuffer struct {
	count, size int
	flags       uint32 // the element size word's top bits, as stored
	fields      []legacyField
	data        []byte
}

type legacyField struct {
	name                        string
	index, format, offset, slot int
	slotType                    uint32
	stepRate                    int
}

func decodeVBIB(block []byte) (vertex, index []legacyBuffer, err error) {
	if len(block) < 16 {
		return nil, nil, errors.New("the mesh buffer block is too short")
	}
	u32 := func(at int) (int, error) {
		if at < 0 || at+4 > len(block) {
			return 0, errors.New("the mesh buffer block ends early")
		}
		return int(binary.LittleEndian.Uint32(block[at:])), nil
	}
	read := func(at int, isVertex bool) (legacyBuffer, error) {
		var b legacyBuffer
		count, err := u32(at)
		if err != nil {
			return b, err
		}
		word, err := u32(at + 4)
		if err != nil {
			return b, err
		}
		b.count, b.size, b.flags = count, word&vbibSize, uint32(word)&^vbibSize
		attrOffset, _ := u32(at + 8)
		attrCount, _ := u32(at + 12)
		dataOffset, _ := u32(at + 16)
		total, err := u32(at + 20)
		if err != nil {
			return b, err
		}
		for i := 0; i < attrCount; i++ {
			f := at + 8 + attrOffset + 56*i
			if f+56 > len(block) {
				return b, errors.New("a vertex field runs past the block")
			}
			name := block[f : f+32]
			if n := bytes.IndexByte(name, 0); n >= 0 {
				name = name[:n]
			}
			field := legacyField{name: string(name)}
			vals := make([]int, 6)
			for k := range vals {
				vals[k] = int(int32(binary.LittleEndian.Uint32(block[f+32+4*k:])))
			}
			field.index, field.format, field.offset, field.slot = vals[0], vals[1], vals[2], vals[3]
			field.slotType, field.stepRate = uint32(vals[4]), vals[5]
			b.fields = append(b.fields, field)
		}
		start := at + 16 + dataOffset
		if start < 0 || start+total > len(block) {
			return b, errors.New("a buffer's data runs past the block")
		}
		raw := block[start : start+total]
		if b.flags&vbibZstd != 0 { // zstd first, then meshopt unless the plain bit is set
			if raw, err = DecodeZstd(raw, 0); err != nil {
				return b, err
			}
		}
		want := b.count * b.size
		switch {
		case b.flags&vbibPlain != 0 || len(raw) == want:
			b.data = append([]byte{}, raw...)
		case isVertex:
			b.data, err = DecodeVertexBuffer(b.count, b.size, raw)
		default:
			b.data, err = DecodeIndexBuffer(b.count, b.size, raw)
		}
		return b, err
	}
	vbOffset, _ := u32(0)
	vbCount, _ := u32(4)
	ibOffset, _ := u32(8)
	ibCount, _ := u32(12)
	for i := 0; i < vbCount; i++ {
		b, err := read(vbOffset+24*i, true)
		if err != nil {
			return nil, nil, fmt.Errorf("vertex buffer %d: %v", i, err)
		}
		vertex = append(vertex, b)
	}
	for i := 0; i < ibCount; i++ {
		b, err := read(8+ibOffset+24*i, false)
		if err != nil {
			return nil, nil, fmt.Errorf("index buffer %d: %v", i, err)
		}
		index = append(index, b)
	}
	return vertex, index, nil
}

// encodeVBIB writes buffers in the VBIB layout, uncompressed.
func encodeVBIB(vertex, index []legacyBuffer) []byte {
	all := append(append([]legacyBuffer{}, vertex...), index...)
	headers := 16 + 24*len(all)
	fieldsAt := make([]int, len(all))
	at := headers
	for i, b := range all {
		fieldsAt[i] = at
		at += 56 * len(b.fields)
	}
	dataAt := make([]int, len(all))
	for i, b := range all {
		at = (at + 15) &^ 15
		dataAt[i] = at
		at += len(b.data)
	}
	out := make([]byte, at)
	put := func(p, v int) { binary.LittleEndian.PutUint32(out[p:], uint32(v)) }
	put(0, 16)
	put(4, len(vertex))
	put(8, 16+24*len(vertex)-8)
	put(12, len(index))
	for i, b := range all {
		h := 16 + 24*i
		put(h, b.count)
		binary.LittleEndian.PutUint32(out[h+4:], uint32(b.size)|(b.flags&^(vbibZstd))|vbibPlain)
		put(h+8, fieldsAt[i]-(h+8))
		put(h+12, len(b.fields))
		put(h+16, dataAt[i]-(h+16))
		put(h+20, len(b.data))
		for k, f := range b.fields {
			p := fieldsAt[i] + 56*k
			copy(out[p:p+31], f.name)
			for j, v := range []int{f.index, f.format, f.offset, f.slot, int(f.slotType), f.stepRate} {
				binary.LittleEndian.PutUint32(out[p+32+4*j:], uint32(int32(v)))
			}
		}
		copy(out[dataAt[i]:], b.data)
	}
	return out
}

func (b legacyBuffer) vertexBuffer() *VertexBuffer {
	vb := &VertexBuffer{Count: b.count, Stride: b.size, Data: b.data}
	for _, f := range b.fields {
		vb.Attributes = append(vb.Attributes, VertexAttribute{Name: f.name, Index: f.index, Format: f.format, Offset: f.offset})
	}
	return vb
}

// LegacyProp is a prop model in the older layout, opened for its mesh to be replaced.
type LegacyProp struct {
	Resource *Resource
	Data     *Value
	Bones    []string // the mesh's skeleton, which blend indices point into
	BoneRest [][12]float64

	meshBlock, bufferBlock int
	render                 *Value
	vertex, index          []legacyBuffer
}

// LoadLegacyProp opens a compiled model of the older layout with one embedded mesh.
func LoadLegacyProp(raw []byte) (*LegacyProp, error) {
	res, err := DecodeResource(raw)
	if err != nil {
		return nil, err
	}
	ctrl, _, err := ReadKV3(res.Block("CTRL"))
	if err != nil {
		return nil, fmt.Errorf("model CTRL: %v", err)
	}
	meshes := ctrl.Get("embedded_meshes")
	if meshes == nil || len(meshes.Items) != 1 {
		return nil, errors.New("only props with one embedded mesh are supported")
	}
	desc := meshes.Items[0]
	p := &LegacyProp{Resource: res, meshBlock: intOf(desc.Get("data_block")), bufferBlock: intOf(desc.Get("vbib_block"))}
	if desc.Get("vbib_block") == nil || p.bufferBlock < 0 || p.bufferBlock >= len(res.Blocks) || p.meshBlock < 0 || p.meshBlock >= len(res.Blocks) {
		return nil, errors.New("this model is not in the older layout")
	}
	if p.Data, _, err = ReadKV3(res.Block("DATA")); err != nil {
		return nil, fmt.Errorf("model DATA: %v", err)
	}
	if p.render, _, err = ReadKV3(res.Blocks[p.meshBlock].Data); err != nil {
		return nil, fmt.Errorf("mesh data: %v", err)
	}
	if p.vertex, p.index, err = decodeVBIB(res.Blocks[p.bufferBlock].Data); err != nil {
		return nil, err
	}
	if len(p.vertex) == 0 || len(p.index) == 0 {
		return nil, errors.New("the mesh has no buffers")
	}
	if sk := p.render.Get("m_skeleton"); sk != nil && sk.Get("m_bones") != nil {
		for _, bone := range sk.Get("m_bones").Items {
			name := ""
			if n := bone.Get("m_boneName"); n != nil {
				name = n.S
			}
			p.Bones = append(p.Bones, name)
			var inv [12]float64
			if m := bone.Get("m_invBindPose"); m != nil && len(m.Items) == 12 {
				for k := range inv {
					inv[k] = numberOf(m.Items[k])
				}
			}
			p.BoneRest = append(p.BoneRest, inv)
		}
	}
	return p, nil
}

// Layout describes the mesh's vertex layout, for reports.
func (p *LegacyProp) Layout() string {
	var parts []string
	for _, f := range p.vertex[0].fields {
		parts = append(parts, fmt.Sprintf("%s%d:%d@%d", f.name, f.index, f.format, f.offset))
	}
	return fmt.Sprintf("%d vertices of %d bytes [%s], %d indices of %d bytes", p.vertex[0].count, p.vertex[0].size,
		strings.Join(parts, " "), p.index[0].count, p.index[0].size)
}

// Bounds is the box the current mesh fills.
func (p *LegacyProp) Bounds() (lo, hi [3]float64) {
	lo = [3]float64{math.Inf(1), math.Inf(1), math.Inf(1)}
	hi = [3]float64{math.Inf(-1), math.Inf(-1), math.Inf(-1)}
	vb := p.vertex[0].vertexBuffer()
	at := vb.Attribute("position", 0)
	if at == nil {
		return
	}
	pos, err := vb.Floats(at)
	if err != nil {
		return
	}
	for _, q := range pos {
		for k := 0; k < 3; k++ {
			lo[k], hi[k] = math.Min(lo[k], float64(q[k])), math.Max(hi[k], float64(q[k]))
		}
	}
	return
}

// Replace puts a new mesh in the prop: positions in the prop's space, every vertex on one bone,
// drawn with material. The vertex layout stays the prop's own; dlmod writes it uncompressed.
func (p *LegacyProp) Replace(positions, normals [][3]float32, uvs [][2]float32, indices []uint32, bone int, material string) ([]byte, error) {
	if len(positions) == 0 || len(indices)%3 != 0 {
		return nil, errors.New("the new mesh is empty")
	}
	layout := p.vertex[0]
	vb := layout.vertexBuffer()
	data := VertexData{Positions: positions, Normals: normals, UVs: uvs, Skin: make([][]Influence, len(positions))}
	for i := range data.Skin {
		data.Skin[i] = []Influence{{bone, 1}}
	}
	encoded, err := vb.Encode(data)
	if err != nil {
		return nil, err
	}
	size := 2
	if len(positions) > 65535 {
		size = 4
	}
	idx := make([]byte, 0, len(indices)*size)
	for _, i := range indices {
		if size == 2 {
			idx = append(idx, byte(i), byte(i>>8))
		} else {
			idx = append(idx, byte(i), byte(i>>8), byte(i>>16), byte(i>>24))
		}
	}
	vertex := []legacyBuffer{{count: len(positions), size: layout.size, flags: layout.flags, fields: layout.fields, data: encoded}}
	index := []legacyBuffer{{count: len(indices), size: size, flags: p.index[0].flags, data: idx}}
	// The draw call: all of it, with the new material; the bounds of the new mesh.
	lo := [3]float64{math.Inf(1), math.Inf(1), math.Inf(1)}
	hi := [3]float64{math.Inf(-1), math.Inf(-1), math.Inf(-1)}
	for _, q := range positions {
		for k := 0; k < 3; k++ {
			lo[k], hi[k] = math.Min(lo[k], float64(q[k])), math.Max(hi[k], float64(q[k]))
		}
	}
	objects := p.render.Get("m_sceneObjects")
	if objects == nil || len(objects.Items) == 0 {
		return nil, errors.New("the mesh has no scene objects")
	}
	for o, object := range objects.Items {
		calls := object.Get("m_drawCalls")
		if calls == nil || len(calls.Items) == 0 {
			continue
		}
		if o == 0 {
			call := calls.Items[0]
			calls.Items = calls.Items[:1]
			setNumber(call, "m_nBaseVertex", 0)
			setNumber(call, "m_nVertexCount", len(positions))
			setNumber(call, "m_nStartIndex", 0)
			setNumber(call, "m_nIndexCount", len(indices))
			if m := call.Get("m_material"); m != nil {
				m.S = material
			}
			for key, bound := range map[string][3]float64{"m_vMinBounds": lo, "m_vMaxBounds": hi} {
				if v := object.Get(key); v != nil && len(v.Items) == 3 {
					for k := 0; k < 3; k++ {
						v.Items[k].F = bound[k]
					}
				}
			}
		} else {
			calls.Items = nil
		}
	}
	blocks := make([]Block, len(p.Resource.Blocks))
	copy(blocks, p.Resource.Blocks)
	format := [16]byte{}
	copy(format[:], blocks[p.meshBlock].Data[4:20])
	mdat, err := WriteKV3(p.render, format)
	if err != nil {
		return nil, err
	}
	blocks[p.meshBlock].Data = mdat
	blocks[p.bufferBlock].Data = encodeVBIB(vertex, index)
	for i, b := range blocks {
		if b.Type != "RERL" {
			continue
		}
		refs, err := ReadReferences(b.Data)
		if err != nil {
			return nil, err
		}
		have := false
		for _, r := range refs {
			have = have || strings.EqualFold(r.Name, material)
		}
		if !have {
			refs = append(refs, Reference{ResourceID(material), material})
		}
		blocks[i].Data = WriteReferences(refs)
	}
	out := &Resource{Version: p.Resource.Version, Blocks: blocks, Streaming: p.Resource.Streaming}
	return out.Encode(), nil
}

// Triangles collects the prop's geometry, for previews.
func (p *LegacyProp) Triangles() (Triangles, error) {
	vb := p.vertex[0].vertexBuffer()
	position := vb.Attribute("position", 0)
	if position == nil {
		return Triangles{}, errors.New("the mesh has no positions")
	}
	values, err := vb.Floats(position)
	if err != nil {
		return Triangles{}, err
	}
	var out Triangles
	for _, v := range values {
		out.Positions = append(out.Positions, [3]float32{v[0], v[1], v[2]})
	}
	ib := p.index[0]
	if (ib.size != 2 && ib.size != 4) || len(ib.data) < ib.count*ib.size {
		return Triangles{}, errors.New("the mesh's index buffer cannot be read")
	}
	buffer := IndexBuffer{Count: ib.count, Size: ib.size, Data: ib.data}
	indices := buffer.Indices()
	for t := 0; t+3 <= len(indices); t += 3 {
		a, b, c := indices[t], indices[t+1], indices[t+2]
		if n := uint32(len(out.Positions)); a < n && b < n && c < n {
			out.Indices = append(out.Indices, a, b, c)
		}
	}
	return out, nil
}
