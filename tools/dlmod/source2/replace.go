package source2

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"regexp"
	"sort"
	"strings"
)

// ModelPart is one of a hero model's meshes as the importer offers it: replaced by the new
// model (and hidden) or kept as it is, like the gun.
type ModelPart struct {
	Name     string // without the _lod1/_lod2 suffix
	Vertices int
	Keep     bool // kept unless the user says otherwise: weapons
}

var lodSuffix = regexp.MustCompile(`_lod\d+$`)

func baseName(name string) string { return lodSuffix.ReplaceAllString(name, "") }

func (m *Model) lodMask(mesh *Mesh) int {
	masks := m.Data.Get("m_refLODGroupMasks")
	if masks == nil || mesh.MeshIndex >= len(masks.Items) {
		return 1
	}
	return intOf(masks.Items[mesh.MeshIndex])
}

func vertexCount(mesh *Mesh) int {
	n := 0
	for _, vb := range mesh.VertexBuffers {
		n += vb.Count
	}
	return n
}

// Parts lists the model's meshes as they appear at full detail.
func (m *Model) Parts() []ModelPart {
	var out []ModelPart
	for _, mesh := range m.Meshes {
		if m.lodMask(mesh)&1 == 0 {
			continue
		}
		lower := strings.ToLower(mesh.Name)
		out = append(out, ModelPart{
			Name:     baseName(mesh.Name),
			Vertices: vertexCount(mesh),
			Keep:     strings.Contains(lower, "gun") || strings.Contains(lower, "weapon"),
		})
	}
	return out
}

// ImportOptions says how a model is placed on a hero.
type ImportOptions struct {
	Keep      map[string]bool   // part names (see Parts) left as they are
	Scale     float64           // on top of fitting the model to the hero's height; 0 means 1
	Yaw       float64           // degrees to turn the model around the up axis
	Offset    [3]float64        // moved after fitting, in game units (Z is up)
	ZUp       bool              // the file is Z-up (Blender's axes) rather than Y-up
	Materials map[string]string // imported part name to the material it is drawn with

	// Hand edits from the studio's rig tools, applied over the computed weights.
	JointMap map[string]string // a rigged model's joint (by name) -> hero bone, over the automatic choice
	Pins     map[int]string    // a piece of the model (see Pieces) -> the one hero bone it follows
	Paint    []PaintStroke     // painted weights, in order

	// Kept parts moved, by part name, in the model's space at rest (game units): a weapon pulled
	// out of a wider body. The part still follows its bones, so the move turns with them.
	KeptShift map[string][3]float64
	// Bones moved at rest, by name, in the model's space: only bones the animations leave alone
	// (a muzzle, which follows its parent as it rests), to go with a moved part.
	BoneShift map[string][3]float64

	// Tuning adjusts the finished rig for what the game does to the hero's bones.
	Tuning RigTuning
}

// moveBones shifts bones at rest in the model's skeleton and in its animation group's decode
// key, keeping their parents; the game places a bone no clip animates from these.
func (m *Model) moveBones(blocks []Block, shift map[string][3]float64) error {
	if len(shift) == 0 {
		return nil
	}
	sk, err := m.Skeleton()
	if err != nil {
		return err
	}
	local := map[string][3]float64{} // each move in its parent's space
	for name, s := range shift {
		i, ok := sk.index[name]
		if !ok {
			return fmt.Errorf("there is no bone %s to move", name)
		}
		if p := sk.Parent[i]; p >= 0 {
			r := sk.World[p]
			s = [3]float64{
				r[0]*s[0] + r[4]*s[1] + r[8]*s[2],
				r[1]*s[0] + r[5]*s[1] + r[9]*s[2],
				r[2]*s[0] + r[6]*s[1] + r[10]*s[2],
			}
		}
		local[name] = s
	}
	rewrite := func(kind string, edit func(root *Value) int) error {
		for i := range blocks {
			if blocks[i].Type != kind {
				continue
			}
			root, format, err := ReadKV3(blocks[i].Data)
			if err != nil {
				return err
			}
			if edit(root) == 0 {
				continue
			}
			data, err := WriteKV3(root, format)
			if err != nil {
				return err
			}
			blocks[i].Data = data
		}
		return nil
	}
	moved := 0
	err = rewrite("DATA", func(root *Value) int {
		skel := root.Get("m_modelSkeleton")
		if skel == nil || skel.Get("m_boneName") == nil || skel.Get("m_bonePosParent") == nil {
			return 0
		}
		n := 0
		for i, name := range skel.Get("m_boneName").Items {
			if s, ok := local[name.S]; ok && i < len(skel.Get("m_bonePosParent").Items) {
				pos := skel.Get("m_bonePosParent").Items[i]
				for k := 0; k < 3 && k < len(pos.Items); k++ {
					pos.Items[k].F += s[k]
				}
				n++
			}
		}
		moved = n
		return n
	})
	if err != nil {
		return err
	}
	if moved != len(local) {
		return fmt.Errorf("moved %d of %d bones in the skeleton", moved, len(local))
	}
	return rewrite("AGRP", func(root *Value) int {
		key := root.Get("m_decodeKey")
		if key == nil || key.Get("m_boneArray") == nil {
			return 0
		}
		n := 0
		for _, bone := range key.Get("m_boneArray").Items {
			if name := bone.Get("m_name"); name != nil {
				if s, ok := local[name.S]; ok && bone.Get("m_pos") != nil {
					for k := 0; k < 3 && k < len(bone.Get("m_pos").Items); k++ {
						bone.Get("m_pos").Items[k].F += s[k]
					}
					n++
				}
			}
		}
		return n
	})
}

// shiftKept moves a kept mesh's vertices at rest, and its bounds, by shift.
func shiftKept(mesh *Mesh, shift [3]float64, setBlock func(int, []byte) error, plain func(*Value, int, int)) error {
	for _, vb := range mesh.VertexBuffers {
		at := vb.Attribute("position", 0)
		if at == nil {
			continue
		}
		if at.Format != FormatR32G32B32Float && at.Format != FormatR32G32B32A32Float {
			return fmt.Errorf("%s: positions in format %d cannot be moved", mesh.Name, at.Format)
		}
		data := append([]byte{}, vb.Data...)
		for i := 0; i < vb.Count; i++ {
			p := data[i*vb.Stride+at.Offset:]
			for k := 0; k < 3; k++ {
				v := math.Float32frombits(binary.LittleEndian.Uint32(p[4*k:])) + float32(shift[k])
				binary.LittleEndian.PutUint32(p[4*k:], math.Float32bits(v))
			}
		}
		plain(vb.Desc, vb.Count, vb.Stride)
		if err := setBlock(vb.Block, data); err != nil {
			return err
		}
	}
	if objects := mesh.Render.Get("m_sceneObjects"); objects != nil {
		for _, object := range objects.Items {
			for _, key := range []string{"m_vMinBounds", "m_vMaxBounds"} {
				if v := object.Get(key); v != nil && len(v.Items) == 3 {
					for k := 0; k < 3; k++ {
						v.Items[k].F += shift[k]
					}
				}
			}
		}
	}
	return nil
}

func (o ImportOptions) kept(name string) bool { return o.Keep[baseName(name)] }

// replaced are the full-detail meshes the new model stands in for.
func (m *Model) replaced(opts ImportOptions) []*Mesh {
	var out []*Mesh
	for _, mesh := range m.Meshes {
		if m.lodMask(mesh)&1 != 0 && !opts.kept(mesh.Name) {
			out = append(out, mesh)
		}
	}
	return out
}

// Bounds is the box around the meshes the new model replaces, in game units.
func (m *Model) Bounds(opts ImportOptions) (lo, hi [3]float64, err error) {
	lo = [3]float64{math.Inf(1), math.Inf(1), math.Inf(1)}
	hi = [3]float64{math.Inf(-1), math.Inf(-1), math.Inf(-1)}
	for _, mesh := range m.replaced(opts) {
		tris, err := mesh.Triangles()
		if err != nil {
			return lo, hi, err
		}
		for _, p := range tris.Positions {
			for k := 0; k < 3; k++ {
				lo[k], hi[k] = math.Min(lo[k], float64(p[k])), math.Max(hi[k], float64(p[k]))
			}
		}
	}
	if math.IsInf(lo[0], 1) {
		return lo, hi, errors.New("every part of the hero is kept, so there is nothing to replace")
	}
	return lo, hi, nil
}

// trimmedCentre is the middle of points across X and Y: the average of each, leaving out the
// outer 5% on either side, so a gun held out or a cape does not pull a model off its feet.
func trimmedCentre(points [][3]float64) [2]float64 {
	var out [2]float64
	if len(points) == 0 {
		return out
	}
	values := make([]float64, len(points))
	for k := 0; k < 2; k++ {
		for i, p := range points {
			values[i] = p[k]
		}
		sort.Float64s(values)
		cut := len(values) * 5 / 100
		middle := values[cut : len(values)-cut]
		sum := 0.0
		for _, v := range middle {
			sum += v
		}
		out[k] = sum / float64(len(middle))
	}
	return out
}

// Anchor is the point between the hero's feet that a model stands on, across X and Y: the pelvis
// bone when the skeleton has one, else the middle of the replaced parts.
func (m *Model) Anchor(opts ImportOptions) ([2]float64, error) {
	if skeleton, err := m.Skeleton(); err == nil {
		if p, ok := skeleton.position("pelvis"); ok {
			return [2]float64{p[0], p[1]}, nil
		}
	}
	var points [][3]float64
	for _, mesh := range m.replaced(opts) {
		tris, err := mesh.Triangles()
		if err != nil {
			return [2]float64{}, err
		}
		for _, p := range tris.Positions {
			points = append(points, vec64(p))
		}
	}
	return trimmedCentre(points), nil
}

// Fit turns an imported model into the hero's space: Z up, facing +X, as tall as the parts it
// replaces, its middle over the hero's pelvis and its feet on the hero's floor.
func (m *Model) Fit(in *ImportedMesh, opts ImportOptions) (*ImportedMesh, error) {
	lo, hi, err := m.Bounds(opts)
	if err != nil {
		return nil, err
	}
	sin, cos := math.Sincos(opts.Yaw * math.Pi / 180)
	turn := func(v [3]float32) [3]float64 {
		x, y, z := float64(v[0]), float64(v[1]), float64(v[2])
		if opts.ZUp {
			x, y = -y, x // Blender's front faces -Y
		} else {
			x, y, z = z, x, y // glTF's front faces +Z, up is +Y
		}
		return [3]float64{x*cos - y*sin, x*sin + y*cos, z}
	}
	out := &ImportedMesh{Parts: in.Parts, UVs: in.UVs, Joints: in.Joints, JointNames: in.JointNames, JointParents: in.JointParents}
	low := [3]float64{math.Inf(1), math.Inf(1), math.Inf(1)}
	high := [3]float64{math.Inf(-1), math.Inf(-1), math.Inf(-1)}
	turned := make([][3]float64, len(in.Positions))
	for i, p := range in.Positions {
		turned[i] = turn(p)
		for k := 0; k < 3; k++ {
			low[k], high[k] = math.Min(low[k], turned[i][k]), math.Max(high[k], turned[i][k])
		}
	}
	height := high[2] - low[2]
	if height <= 0 {
		return nil, errors.New("the model is flat: it has no height")
	}
	if (opts.Scale == 0 || opts.Scale == 1) && m.inHeroSpace(in, turn, hi[2]-lo[2]) {
		// Rigged on the hero's own bones where they stand: already in the hero's space, where a
		// rig made for the hero's motion must stay. Fitting would move it off its bones.
		out.Positions = make([][3]float32, len(turned))
		for i, p := range turned {
			for k := 0; k < 3; k++ {
				out.Positions[i][k] = float32(p[k] + opts.Offset[k])
			}
		}
		out.Normals = make([][3]float32, len(in.Normals))
		for i, n := range in.Normals {
			t := normalize(turn(n))
			out.Normals[i] = [3]float32{float32(t[0]), float32(t[1]), float32(t[2])}
		}
		out.JointPositions = make([][3]float32, len(in.JointPositions))
		for i, p := range in.JointPositions {
			t := turn(p)
			for k := 0; k < 3; k++ {
				out.JointPositions[i][k] = float32(t[k] + opts.Offset[k])
			}
		}
		return out, nil
	}
	scale := (hi[2] - lo[2]) / height
	if opts.Scale > 0 {
		scale *= opts.Scale
	}
	middle := trimmedCentre(turned)
	centre := [3]float64{middle[0], middle[1], low[2]}
	anchor, err := m.Anchor(opts)
	if err != nil {
		return nil, err
	}
	target := [3]float64{anchor[0], anchor[1], lo[2]}
	out.Positions = make([][3]float32, len(turned))
	for i, p := range turned {
		for k := 0; k < 3; k++ {
			out.Positions[i][k] = float32((p[k]-centre[k])*scale + target[k] + opts.Offset[k])
		}
	}
	out.Normals = make([][3]float32, len(in.Normals))
	for i, n := range in.Normals {
		t := normalize(turn(n))
		out.Normals[i] = [3]float32{float32(t[0]), float32(t[1]), float32(t[2])}
	}
	if in.JointPositions != nil {
		out.JointPositions = make([][3]float32, len(in.JointPositions))
		for i, p := range in.JointPositions {
			t := turn(p)
			for k := 0; k < 3; k++ {
				out.JointPositions[i][k] = float32((t[k]-centre[k])*scale + target[k] + opts.Offset[k])
			}
		}
	}
	return out, nil
}

// inHeroSpace tells whether a rigged model's joints, turned into the hero's axes, sit on the
// hero's bones of the same names, as they do in a model rigged for this hero.
func (m *Model) inHeroSpace(in *ImportedMesh, turn func([3]float32) [3]float64, height float64) bool {
	if len(in.JointPositions) == 0 || len(in.Joints) != len(in.Positions) {
		return false
	}
	skeleton, err := m.Skeleton()
	if err != nil {
		return false
	}
	probe := &ImportedMesh{Joints: in.Joints, JointNames: in.JointNames, Positions: in.Positions,
		JointPositions: make([][3]float32, len(in.JointPositions))}
	for i, p := range in.JointPositions {
		t := turn(p)
		probe.JointPositions[i] = [3]float32{float32(t[0]), float32(t[1]), float32(t[2])}
	}
	// No limb needed here: a rig on the body's bones alone is still placed on the hero.
	named, _ := heroJoints(probe, skeleton, height)
	return named >= 4
}

// BoneWeight is a bone, by name, and how much a vertex follows it.
type BoneWeight struct {
	Bone   string
	Weight float32
}

type pointGrid struct {
	cell  float64
	cells map[[3]int][]int
	pts   [][3]float32

	// The cells again, laid out by freeze over the box the points fill, x fastest: cell c holds
	// items[starts[c]:starts[c+1]]. Nil until the first search, and after any add.
	lo, size [3]int
	starts   []int32
	items    []int32
}

// maxFrozenCells caps the array freeze builds; a grid spread wider keeps searching its map.
const maxFrozenCells = 1 << 22

type gridHit struct {
	index    int
	distance float64
}

func newPointGrid(cell float64) *pointGrid {
	return &pointGrid{cell: cell, cells: map[[3]int][]int{}}
}

func (g *pointGrid) key(p [3]float32) [3]int {
	return [3]int{int(math.Floor(float64(p[0]) / g.cell)), int(math.Floor(float64(p[1]) / g.cell)), int(math.Floor(float64(p[2]) / g.cell))}
}

func (g *pointGrid) add(index int, p [3]float32) {
	for len(g.pts) <= index {
		g.pts = append(g.pts, [3]float32{})
	}
	g.pts[index] = p
	k := g.key(p)
	g.cells[k] = append(g.cells[k], index)
	g.starts = nil
}

func (g *pointGrid) cellIndex(k [3]int) int {
	return ((k[2]-g.lo[2])*g.size[1]+(k[1]-g.lo[1]))*g.size[0] + (k[0] - g.lo[0])
}

// freeze lays the cells out as one array over the box the points fill. A search far from the
// points (a round model's belly, nowhere near the hero's thin arm) walks dozens of rings of
// empty cells; as map lookups that took most of a minute per model, as array reads it does not,
// and cells outside the box are not visited at all.
func (g *pointGrid) freeze() {
	var hi [3]int
	first := true
	for k := range g.cells {
		for a := 0; a < 3; a++ {
			if first || k[a] < g.lo[a] {
				g.lo[a] = k[a]
			}
			if first || k[a] > hi[a] {
				hi[a] = k[a]
			}
		}
		first = false
	}
	total := 1
	for a := 0; a < 3; a++ {
		g.size[a] = hi[a] - g.lo[a] + 1
		total *= g.size[a]
		if total > maxFrozenCells {
			return
		}
	}
	starts := make([]int32, total+1)
	for k, list := range g.cells {
		starts[g.cellIndex(k)+1] += int32(len(list))
	}
	for c := 1; c <= total; c++ {
		starts[c] += starts[c-1]
	}
	items := make([]int32, starts[total])
	next := append([]int32(nil), starts[:total]...)
	for k, list := range g.cells {
		c := g.cellIndex(k)
		for _, i := range list { // in the order added, so ties go the same way as before
			items[next[c]] = int32(i)
			next[c]++
		}
	}
	g.starts, g.items = starts, items
}

// nearest finds up to n points closest to p, searching outward ring by ring and keeping only the
// best n seen so far.
func (g *pointGrid) nearest(p [3]float32, n int) []gridHit {
	if g.starts == nil && len(g.cells) > 0 {
		g.freeze()
	}
	center := g.key(p)
	best := make([]gridHit, 0, n) // ordered by squared distance until the end
	consider := func(i int) {
		q := g.pts[i]
		d := sq(p[0]-q[0]) + sq(p[1]-q[1]) + sq(p[2]-q[2])
		if len(best) == n && d >= best[n-1].distance {
			return
		}
		if len(best) < n {
			best = append(best, gridHit{})
		}
		at := len(best) - 1
		for at > 0 && best[at-1].distance > d {
			best[at] = best[at-1]
			at--
		}
		best[at] = gridHit{i, d}
	}
	frozen := g.starts != nil
	visit := func(k [3]int) {
		if !frozen {
			for _, i := range g.cells[k] {
				consider(i)
			}
			return
		}
		c := g.cellIndex(k)
		for _, i := range g.items[g.starts[c]:g.starts[c+1]] {
			consider(int(i))
		}
	}
	// span is the part of a ring's side along one axis that lies in the box (all of it unfrozen).
	span := func(axis, ring int) (int, int) {
		if !frozen {
			return -ring, ring
		}
		return max(-ring, g.lo[axis]-center[axis]), min(ring, g.lo[axis]+g.size[axis]-1-center[axis])
	}
	for ring := 0; ring < 64; ring++ {
		x0, x1 := span(0, ring)
		y0, y1 := span(1, ring)
		z0, z1 := span(2, ring)
		for dx := x0; dx <= x1; dx++ {
			for dy := y0; dy <= y1; dy++ {
				edge := abs(dx) == ring || abs(dy) == ring
				for dz := z0; dz <= z1; dz++ {
					if !edge && abs(dz) != ring {
						dz = ring - 1 // inside the shell: jump to its far face
						continue
					}
					visit([3]int{center[0] + dx, center[1] + dy, center[2] + dz})
				}
			}
		}
		// Every point within ring cells has been seen once n points lie within that distance.
		if reach := float64(ring) * g.cell; len(best) == n && best[n-1].distance <= reach*reach {
			break
		}
		// Once the ring holds the whole box, every point has been seen.
		if frozen && x0 == g.lo[0]-center[0] && x1 == g.lo[0]+g.size[0]-1-center[0] &&
			y0 == g.lo[1]-center[1] && y1 == g.lo[1]+g.size[1]-1-center[1] &&
			z0 == g.lo[2]-center[2] && z1 == g.lo[2]+g.size[2]-1-center[2] {
			break
		}
	}
	for k := range best {
		best[k].distance = math.Sqrt(best[k].distance)
	}
	return best
}

func abs(i int) int {
	if i < 0 {
		return -i
	}
	return i
}

func sq(f float32) float64 { return float64(f) * float64(f) }

// --- writing ---

func setNumber(v *Value, key string, n int) {
	field := v.Get(key)
	if field == nil {
		v.Set(key, NewInt32(int32(n)))
		return
	}
	switch field.Kind {
	case Uint16, Uint32, Uint64:
		field.U, field.I = uint64(n), 0
	case Float32, Float64:
		field.F = float64(n)
	case Bool:
		field.B = n != 0
	default:
		if field.Kind == Null {
			field.Kind = Int32
		}
		field.I = int64(n)
	}
}

func setBool(v *Value, key string, b bool) {
	if field := v.Get(key); field != nil {
		switch field.Kind {
		case Bool:
			field.B = b
		default:
			n := 0
			if b {
				n = 1
			}
			setNumber(v, key, n)
		}
	}
}

func cloneValue(v *Value) *Value {
	if v == nil {
		return nil
	}
	c := *v
	c.Items = nil
	for _, item := range v.Items {
		c.Items = append(c.Items, cloneValue(item))
	}
	c.Keys = append([]string(nil), v.Keys...)
	c.Fields = nil
	for _, f := range v.Fields {
		c.Fields = append(c.Fields, cloneValue(f))
	}
	return &c
}

// boneLimit is the highest bone number a mesh's first vertex buffer can store: one byte per bone
// in most layouts, though Wraith's body lists 347 bones (the ones past 255 are cloth, which only
// her other buffers with 16-bit indices use).
func boneLimit(mesh *Mesh) int {
	if len(mesh.VertexBuffers) == 0 {
		return 0
	}
	if a := mesh.VertexBuffers[0].Attribute("blendindices", 0); a != nil {
		switch a.Format {
		case FormatR8G8B8A8Uint, FormatR16G16B16A16Uint:
			return 255
		}
	}
	return 65535
}

// boneIndex maps bone names to a mesh's own bone numbers. A bone the mesh lacks, or cannot store
// (see boneLimit), falls back to its nearest ancestor the mesh has, through the model's skeleton.
func (m *Model) boneIndex(mesh *Mesh) func(string) int {
	own := map[string]int{}
	limit := boneLimit(mesh)
	for i, name := range mesh.Bones {
		if i <= limit {
			own[name] = i
		}
	}
	parents := map[string]string{}
	if skeleton := m.Data.Get("m_modelSkeleton"); skeleton != nil {
		names, links := skeleton.Get("m_boneName"), skeleton.Get("m_nParent")
		if names != nil && links != nil {
			for i, n := range names.Items {
				if i < len(links.Items) {
					if p := intOf(links.Items[i]); p >= 0 && p < len(names.Items) {
						parents[n.S] = names.Items[p].S
					}
				}
			}
		}
	}
	return func(name string) int {
		for depth := 0; depth < 256 && name != ""; depth++ {
			if i, ok := own[name]; ok {
				return i
			}
			name = parents[name]
		}
		return 0
	}
}

// Replace writes the model with the fitted, rigged mesh in place of the parts it replaces: the
// mesh is drawn by one replaced part at every level of detail and the other replaced
// parts are emptied. Skeleton, animations, hitboxes, physics and attachments stay the hero's.
func (m *Model) Replace(raw []byte, fitted *ImportedMesh, skin [][]BoneWeight, opts ImportOptions) ([]byte, error) {
	fresh, err := LoadModel(raw) // edit a copy so the caller's model stays as it was
	if err != nil {
		return nil, err
	}
	// A level of detail is drawn by the part whose own bones carry most of the new skin's weight,
	// then the biggest. A bone a mesh lacks falls back to an ancestor and stops moving on its own,
	// and no part has them all: Wraith's body has no forearm or finger bones, her arms no neck.
	// Picking by size chose her body at LOD0 but her arms at LOD1, so the model changed with distance.
	used := map[string]float64{}
	for _, bones := range skin {
		for _, b := range bones {
			used[b.Bone] += float64(b.Weight)
		}
	}
	covers := map[*Mesh]float64{}
	for _, mesh := range fresh.Meshes {
		limit := boneLimit(mesh)
		for i, name := range mesh.Bones {
			if i <= limit {
				covers[mesh] += used[name]
			}
		}
	}
	targets := map[*Mesh]bool{}
	covered := 0
	for bit := 0; bit < 8; bit++ {
		if covered&(1<<bit) != 0 {
			continue
		}
		var best *Mesh
		for _, mesh := range fresh.Meshes {
			if fresh.lodMask(mesh)&(1<<bit) == 0 || opts.kept(mesh.Name) || len(mesh.VertexBuffers) == 0 {
				continue
			}
			vb := mesh.VertexBuffers[0]
			if vb.Attribute("position", 0) == nil || vb.Attribute("blendindices", 0) == nil || vb.Attribute("normal", 0) == nil {
				continue
			}
			if best == nil || covers[mesh] > covers[best] || (covers[mesh] == covers[best] && vertexCount(mesh) > vertexCount(best)) {
				best = mesh
			}
		}
		if best != nil {
			targets[best] = true
			covered |= fresh.lodMask(best)
		}
	}
	if len(targets) == 0 {
		return nil, errors.New("none of the replaced parts can hold a skinned mesh")
	}
	blocks := make([]Block, len(fresh.Resource.Blocks))
	copy(blocks, fresh.Resource.Blocks)
	setBlock := func(index int, data []byte) error {
		if index < 0 || index >= len(blocks) {
			return fmt.Errorf("block %d is missing", index)
		}
		blocks[index].Data = data
		return nil
	}
	plain := func(desc *Value, count, size int) {
		setNumber(desc, "m_nElementCount", count)
		setNumber(desc, "m_nElementSizeInBytes", size)
		setBool(desc, "m_bMeshoptCompressed", false)
		setBool(desc, "m_bMeshoptIndexSequence", false)
		setBool(desc, "m_bCompressedZSTD", false)
		setBool(desc, "m_bCompressedZ3", false)
	}
	empty := func(mesh *Mesh, from int) error {
		for _, vb := range mesh.VertexBuffers[min(from, len(mesh.VertexBuffers)):] {
			plain(vb.Desc, 3, vb.Stride)
			if err := setBlock(vb.Block, make([]byte, 3*vb.Stride)); err != nil {
				return err
			}
		}
		for _, ib := range mesh.IndexBuffers[min(from, len(mesh.IndexBuffers)):] {
			plain(ib.Desc, 3, 2)
			if err := setBlock(ib.Block, make([]byte, 6)); err != nil {
				return err
			}
		}
		return nil
	}
	for _, mesh := range fresh.Meshes {
		if opts.kept(mesh.Name) {
			shift, ok := opts.KeptShift[baseName(mesh.Name)]
			if !ok || shift == ([3]float64{}) {
				continue
			}
			if err := shiftKept(mesh, shift, setBlock, plain); err != nil {
				return nil, err
			}
			format := [16]byte{}
			copy(format[:], blocks[mesh.DataBlock].Data[4:20])
			encoded, err := WriteKV3(mesh.Render, format)
			if err != nil {
				return nil, err
			}
			blocks[mesh.DataBlock].Data = encoded
			continue
		}
		setNumber(mesh.Desc, "m_nMorphBlock", -1)
		objects := mesh.Render.Get("m_sceneObjects")
		if !targets[mesh] {
			if err := empty(mesh, 0); err != nil {
				return nil, err
			}
			if objects != nil {
				for _, object := range objects.Items {
					for _, call := range object.Get("m_drawCalls").Items {
						setNumber(call, "m_nBaseVertex", 0)
						setNumber(call, "m_nVertexCount", 3)
						setNumber(call, "m_nStartIndex", 0)
						setNumber(call, "m_nIndexCount", 3)
						for _, h := range call.Get("m_vertexBuffers").Items {
							setNumber(h, "m_hBuffer", min(intOf(h.Get("m_hBuffer")), len(mesh.VertexBuffers)-1))
						}
					}
				}
			}
		} else {
			if err := fresh.writeTarget(mesh, fitted, skin, opts, setBlock, plain); err != nil {
				return nil, err
			}
			if err := empty(mesh, 1); err != nil {
				return nil, err
			}
		}
		format := [16]byte{}
		copy(format[:], blocks[mesh.DataBlock].Data[4:20])
		encoded, err := WriteKV3(mesh.Render, format)
		if err != nil {
			return nil, err
		}
		blocks[mesh.DataBlock].Data = encoded
	}
	ctrlIndex := -1
	for i, b := range blocks {
		if b.Type == "CTRL" {
			ctrlIndex = i
		}
	}
	format := [16]byte{}
	copy(format[:], blocks[ctrlIndex].Data[4:20])
	ctrl, err := WriteKV3(fresh.Ctrl, format)
	if err != nil {
		return nil, err
	}
	blocks[ctrlIndex].Data = ctrl
	if err := fresh.moveBones(blocks, opts.BoneShift); err != nil {
		return nil, err
	}
	if err := fresh.addReferences(blocks, opts); err != nil {
		return nil, err
	}
	out := &Resource{Version: fresh.Resource.Version, Blocks: blocks, Streaming: fresh.Resource.Streaming}
	return out.Encode(), nil
}

func (m *Model) writeTarget(mesh *Mesh, fitted *ImportedMesh, skin [][]BoneWeight, opts ImportOptions,
	setBlock func(int, []byte) error, plain func(*Value, int, int)) error {
	vb := mesh.VertexBuffers[0]
	ib := mesh.IndexBuffers[0]
	index := m.boneIndex(mesh)
	positions, normals := fitted.Positions, fitted.Normals
	if sk, err := m.Skeleton(); err == nil {
		positions, normals = rootUnturn(sk, positions, normals, skin)
	}
	data := VertexData{Positions: positions, Normals: normals, UVs: fitted.UVs, Skin: make([][]Influence, len(skin))}
	for i, bones := range skin {
		for _, b := range bones {
			data.Skin[i] = append(data.Skin[i], Influence{index(b.Bone), b.Weight})
		}
	}
	vertices, err := vb.Encode(data)
	if err != nil {
		return fmt.Errorf("%s: %v", mesh.Name, err)
	}
	size := 2
	if len(fitted.Positions) > 65535 {
		size = 4
	}
	var indices []byte
	type rangeOf struct{ start, count int }
	var ranges []rangeOf
	for _, part := range fitted.Parts {
		ranges = append(ranges, rangeOf{len(indices) / size, len(part.Indices)})
		for _, i := range part.Indices {
			if size == 2 {
				indices = append(indices, byte(i), byte(i>>8))
			} else {
				indices = append(indices, byte(i), byte(i>>8), byte(i>>16), byte(i>>24))
			}
		}
	}
	plain(vb.Desc, len(fitted.Positions), vb.Stride)
	plain(ib.Desc, len(indices)/size, size)
	if err := setBlock(vb.Block, vertices); err != nil {
		return err
	}
	if err := setBlock(ib.Block, indices); err != nil {
		return err
	}
	objects := mesh.Render.Get("m_sceneObjects")
	if objects == nil || len(objects.Items) == 0 || len(mesh.DrawCalls) == 0 {
		return fmt.Errorf("%s has no draw calls to copy", mesh.Name)
	}
	template := mesh.DrawCalls[0].Desc
	for _, d := range mesh.DrawCalls {
		if d.VertexBuffer == 0 {
			template = d.Desc
			break
		}
	}
	var calls []*Value
	for p, part := range fitted.Parts {
		call := cloneValue(template)
		setNumber(call, "m_nBaseVertex", 0)
		setNumber(call, "m_nVertexCount", len(fitted.Positions))
		setNumber(call, "m_nStartIndex", ranges[p].start)
		setNumber(call, "m_nIndexCount", ranges[p].count)
		setNumber(call.Get("m_indexBuffer"), "m_hBuffer", 0)
		for _, h := range call.Get("m_vertexBuffers").Items {
			setNumber(h, "m_hBuffer", 0)
		}
		if material := opts.Materials[part.Name]; material != "" {
			call.Get("m_material").S = material
		}
		calls = append(calls, call)
	}
	lo := [3]float32{float32(math.Inf(1)), float32(math.Inf(1)), float32(math.Inf(1))}
	hi := [3]float32{float32(math.Inf(-1)), float32(math.Inf(-1)), float32(math.Inf(-1))}
	for _, p := range fitted.Positions {
		for k := 0; k < 3; k++ {
			lo[k], hi[k] = min(lo[k], p[k]), max(hi[k], p[k])
		}
	}
	for o, object := range objects.Items {
		if o == 0 {
			object.Get("m_drawCalls").Items = calls
			for key, bound := range map[string][3]float32{"m_vMinBounds": lo, "m_vMaxBounds": hi} {
				if v := object.Get(key); v != nil && len(v.Items) == 3 {
					for k := 0; k < 3; k++ {
						v.Items[k].F = float64(bound[k])
					}
				}
			}
			continue
		}
		for _, call := range object.Get("m_drawCalls").Items {
			setNumber(call, "m_nVertexCount", 3)
			setNumber(call, "m_nIndexCount", 3)
			setNumber(call, "m_nStartIndex", 0)
		}
	}
	return nil
}

// PartGeometry is a part's full-detail triangles, for previews.
type PartGeometry struct {
	ModelPart
	Positions [][3]float32
	Indices   []uint32
}

// Geometry returns every part at full detail, in the order Parts lists them.
func (m *Model) Geometry() ([]PartGeometry, error) {
	var out []PartGeometry
	parts := m.Parts()
	i := 0
	for _, mesh := range m.Meshes {
		if m.lodMask(mesh)&1 == 0 {
			continue
		}
		tris, err := mesh.Triangles()
		if err != nil {
			return nil, fmt.Errorf("%s: %v", mesh.Name, err)
		}
		out = append(out, PartGeometry{ModelPart: parts[i], Positions: tris.Positions, Indices: tris.Indices})
		i++
	}
	return out, nil
}
