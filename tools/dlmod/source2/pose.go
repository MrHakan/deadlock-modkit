package source2

import (
	"errors"
	"math"
	"sort"
	"strings"
)

// mat34 is an affine transform: a row-major 3x4 matrix, as compiled models store inverse bind poses.
type mat34 [12]float64

func identity34() mat34 { return mat34{1, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1, 0} }

func (a mat34) mul(b mat34) mat34 {
	var out mat34
	for r := 0; r < 3; r++ {
		for c := 0; c < 4; c++ {
			v := a[r*4]*b[c] + a[r*4+1]*b[4+c] + a[r*4+2]*b[8+c]
			if c == 3 {
				v += a[r*4+3]
			}
			out[r*4+c] = v
		}
	}
	return out
}

func (a mat34) apply(p [3]float64) [3]float64 {
	return [3]float64{
		a[0]*p[0] + a[1]*p[1] + a[2]*p[2] + a[3],
		a[4]*p[0] + a[5]*p[1] + a[6]*p[2] + a[7],
		a[8]*p[0] + a[9]*p[1] + a[10]*p[2] + a[11],
	}
}

func (a mat34) rotate(v [3]float64) [3]float64 {
	return [3]float64{a[0]*v[0] + a[1]*v[1] + a[2]*v[2], a[4]*v[0] + a[5]*v[1] + a[6]*v[2], a[8]*v[0] + a[9]*v[1] + a[10]*v[2]}
}

func (a mat34) inverse() (mat34, bool) {
	det := a[0]*(a[5]*a[10]-a[6]*a[9]) - a[1]*(a[4]*a[10]-a[6]*a[8]) + a[2]*(a[4]*a[9]-a[5]*a[8])
	if math.Abs(det) < 1e-12 {
		return identity34(), false
	}
	inv := 1 / det
	var out mat34
	out[0] = (a[5]*a[10] - a[6]*a[9]) * inv
	out[1] = (a[2]*a[9] - a[1]*a[10]) * inv
	out[2] = (a[1]*a[6] - a[2]*a[5]) * inv
	out[4] = (a[6]*a[8] - a[4]*a[10]) * inv
	out[5] = (a[0]*a[10] - a[2]*a[8]) * inv
	out[6] = (a[2]*a[4] - a[0]*a[6]) * inv
	out[8] = (a[4]*a[9] - a[5]*a[8]) * inv
	out[9] = (a[1]*a[8] - a[0]*a[9]) * inv
	out[10] = (a[0]*a[5] - a[1]*a[4]) * inv
	t := [3]float64{a[3], a[7], a[11]}
	for r := 0; r < 3; r++ {
		out[r*4+3] = -(out[r*4]*t[0] + out[r*4+1]*t[1] + out[r*4+2]*t[2])
	}
	return out, true
}

func fromRotation(q [4]float64, p [3]float64, s float64) mat34 {
	x, y, z, w := q[0], q[1], q[2], q[3]
	return mat34{
		s * (1 - 2*(y*y+z*z)), s * 2 * (x*y - z*w), s * 2 * (x*z + y*w), p[0],
		s * 2 * (x*y + z*w), s * (1 - 2*(x*x+z*z)), s * 2 * (y*z - x*w), p[1],
		s * 2 * (x*z - y*w), s * 2 * (y*z + x*w), s * (1 - 2*(x*x+y*y)), p[2],
	}
}

// rotationAbout turns by angle degrees around axis through pivot.
func rotationAbout(axis [3]float64, angle float64, pivot [3]float64) mat34 {
	axis = normalize(axis)
	sin, cos := math.Sincos(angle * math.Pi / 180 / 2)
	r := fromRotation([4]float64{axis[0] * sin, axis[1] * sin, axis[2] * sin, cos}, [3]float64{}, 1)
	moved := r.apply(pivot)
	r[3], r[7], r[11] = pivot[0]-moved[0], pivot[1]-moved[1], pivot[2]-moved[2]
	return r
}

// Skeleton is a model's bones in their bind pose.
type Skeleton struct {
	Names  []string
	Parent []int
	World  []mat34
	index  map[string]int
}

// Skeleton reads the model's bones from its DATA block.
func (m *Model) Skeleton() (*Skeleton, error) {
	sk := m.Data.Get("m_modelSkeleton")
	if sk == nil || sk.Get("m_boneName") == nil {
		return nil, errors.New("the model has no skeleton")
	}
	names, parents := sk.Get("m_boneName").Items, sk.Get("m_nParent")
	positions, rotations, scales := sk.Get("m_bonePosParent"), sk.Get("m_boneRotParent"), sk.Get("m_boneScaleParent")
	s := &Skeleton{index: map[string]int{}}
	for i, n := range names {
		s.Names = append(s.Names, n.S)
		s.index[n.S] = i
		parent := -1
		if parents != nil && i < len(parents.Items) {
			parent = intOf(parents.Items[i])
		}
		s.Parent = append(s.Parent, parent)
		var p [3]float64
		q := [4]float64{0, 0, 0, 1}
		scale := 1.0
		if positions != nil && i < len(positions.Items) && len(positions.Items[i].Items) == 3 {
			for k := 0; k < 3; k++ {
				p[k] = positions.Items[i].Items[k].F
			}
		}
		if rotations != nil && i < len(rotations.Items) && len(rotations.Items[i].Items) == 4 {
			for k := 0; k < 4; k++ {
				q[k] = rotations.Items[i].Items[k].F
			}
		}
		if scales != nil && i < len(scales.Items) {
			scale = scales.Items[i].F
		}
		local := fromRotation(q, p, scale)
		if parent >= 0 && parent < i {
			local = s.World[parent].mul(local)
		}
		s.World = append(s.World, local)
	}
	return s, nil
}

// Position is where a bone is in the bind pose.
func (s *Skeleton) Position(name string) ([3]float64, bool) { return s.position(name) }

func (s *Skeleton) position(name string) ([3]float64, bool) {
	i, ok := s.index[name]
	if !ok {
		return [3]float64{}, false
	}
	return [3]float64{s.World[i][3], s.World[i][7], s.World[i][11]}, true
}

// Pose bends the hero's limbs away from its bind pose so they line up with an imported model's.
type Pose struct {
	Arms float64 // degrees the arms are lowered (negative raises them)
	Legs float64 // degrees the legs are spread (negative closes them)
}

// Limb groups a bone can belong to.
const (
	limbNone = iota
	limbArmL
	limbArmR
	limbLegL
	limbLegR
	limbCount
)

var limbRoots = [limbCount]string{"", "arm_upper_L", "arm_upper_R", "leg_upper_L", "leg_upper_R"}
var limbEnds = [limbCount]string{"", "hand_L", "hand_R", "ankle_L", "ankle_R"}

// limbs says which limb each bone is part of.
func (s *Skeleton) limbs() []int {
	out := make([]int, len(s.Names))
	for limb := limbArmL; limb < limbCount; limb++ {
		root, ok := s.index[limbRoots[limb]]
		if !ok {
			continue
		}
		for i := range s.Names {
			for b, depth := i, 0; b >= 0 && depth < 256; b, depth = s.Parent[b], depth+1 {
				if b == root {
					out[i] = limb
					break
				}
			}
		}
	}
	return out
}

// limbMatrices is the transform each limb gets in a pose: a turn about the forward (X) axis
// through its shoulder or hip.
func (s *Skeleton) limbMatrices(pose Pose) [limbCount]mat34 {
	var out [limbCount]mat34
	for limb := range out {
		out[limb] = identity34()
		if limb == limbNone {
			continue
		}
		pivot, ok1 := s.position(limbRoots[limb])
		end, ok2 := s.position(limbEnds[limb])
		if !ok1 || !ok2 {
			continue
		}
		angle := pose.Arms
		if limb >= limbLegL {
			angle = pose.Legs
		}
		// Find which way a positive angle lowers an arm or spreads a leg on this side.
		moved := rotationAbout([3]float64{1, 0, 0}, 10, pivot).apply(end)
		lower := moved[2] < end[2]
		spread := math.Abs(moved[1]) > math.Abs(end[1])
		if (limb <= limbArmR && !lower) || (limb >= limbLegL && !spread) {
			angle = -angle
		}
		out[limb] = rotationAbout([3]float64{1, 0, 0}, angle, pivot)
	}
	return out
}

// heroSample is a point of the hero's own meshes: where it is, which way it faces and the bones
// it follows.
type heroSample struct {
	position [3]float64
	normal   [3]float64
	bones    []BoneWeight
	limb     [limbCount]float64 // how much it follows each limb
}

// coreBone reports whether a bone belongs to the body itself rather than to cloth, hair, props
// or IK helpers, by the names Valve's hero skeletons use.
// CoreBone reports whether a hero bone belongs to the body (see coreBone), the bones worth
// offering when a model's part is put on a bone by hand.
func CoreBone(name string) bool { return coreBone(name) }

func coreBone(name string) bool {
	lower := strings.ToLower(name)
	for _, skip := range []string{"iktarget", "offset", "_end", "ponytail", "cloth", "coat", "cape", "skirt", "hair", "tail", "ear_", "card", "gun", "weapon", "magazine", "muzzle"} {
		if strings.Contains(lower, skip) {
			return false
		}
	}
	for _, prefix := range []string{"pelvis", "spine", "neck", "head", "jaw", "clavicle", "scap", "arm_", "hand_", "finger_", "leg_", "ankle_", "ball_", "toe", "hip", "knee", "elbow", "wrist", "shoulder"} {
		if strings.HasPrefix(lower, prefix) {
			return true
		}
	}
	return false
}

// coreAncestor is the closest bone at or above a bone that coreBone accepts.
func (s *Skeleton) coreAncestor(name string) string {
	i, ok := s.index[name]
	for depth := 0; ok && i >= 0 && depth < 256; depth++ {
		if coreBone(s.Names[i]) {
			return s.Names[i]
		}
		i = s.Parent[i]
	}
	return name
}

func (m *Model) heroSamples(opts ImportOptions, skeleton *Skeleton) ([]heroSample, error) {
	var limbs []int
	core := false
	if skeleton != nil {
		limbs = skeleton.limbs()
		for _, name := range limbRoots[1:] {
			_, found := skeleton.index[name]
			core = found
			if !found {
				break
			}
		}
	}
	var samples []heroSample
	for _, mesh := range m.replaced(opts) {
		for _, vb := range mesh.VertexBuffers {
			position, normal := vb.Attribute("position", 0), vb.Attribute("normal", 0)
			if position == nil || vb.Attribute("blendindices", 0) == nil {
				continue
			}
			positions, err := vb.Floats(position)
			if err != nil {
				return nil, err
			}
			var normals [][4]float32
			if normal != nil {
				normals, _ = vb.Floats(normal)
			}
			skin, err := vb.Skin()
			if err != nil {
				return nil, err
			}
			for i, p := range positions {
				s := heroSample{position: [3]float64{float64(p[0]), float64(p[1]), float64(p[2])}}
				if normals != nil {
					s.normal = [3]float64{float64(normals[i][0]), float64(normals[i][1]), float64(normals[i][2])}
				}
				var body, total float32
				for _, in := range skin[i] {
					if in.Bone >= len(mesh.Bones) {
						continue
					}
					name := mesh.Bones[in.Bone]
					total += in.Weight
					if core {
						// Weights on cloth, hair and props would make the new model flap like
						// them: they go to the body bone the cloth hangs from instead.
						if coreBone(name) {
							body += in.Weight
						} else {
							name = skeleton.coreAncestor(name)
						}
					}
					s.bones = append(s.bones, BoneWeight{name, in.Weight})
					if skeleton != nil {
						if b, ok := skeleton.index[name]; ok {
							s.limb[limbs[b]] += float64(in.Weight)
						}
					}
				}
				// Points that mostly follow cloth or hair (a coat's hem, a ponytail) say little
				// about the body underneath, so they are left out.
				if len(s.bones) > 0 && (!core || body >= total/2) {
					samples = append(samples, s)
				}
			}
		}
	}
	if len(samples) == 0 {
		return nil, errors.New("the replaced parts of the hero carry no bone weights to copy")
	}
	return samples, nil
}

// blend mixes the limb transforms by how much a point follows each limb.
func blend(weights [limbCount]float64, matrices [limbCount]mat34) mat34 {
	var out mat34
	total := 0.0
	for limb, w := range weights {
		if w == 0 {
			continue
		}
		for e := range out {
			out[e] += matrices[limb][e] * w
		}
		total += w
	}
	if total == 0 {
		return identity34()
	}
	for e := range out {
		out[e] /= total
	}
	return out
}

// unbend moves a point and its normal from a pose back into the bind pose: each limb's own turn
// is undone and the results are mixed by how much the point follows each limb. Undoing the mixed
// transform instead blows up where opposite limbs mix: averaging both arms lowered by 88 degrees
// leaves a nearly flat matrix, and its inverse throws a cape's points hundreds of units away.
func unbend(weights [limbCount]float64, inverses [limbCount]mat34, p, n [3]float64) ([3]float64, [3]float64) {
	var position, normal [3]float64
	total := 0.0
	for limb, w := range weights {
		if w == 0 {
			continue
		}
		q, r := inverses[limb].apply(p), inverses[limb].rotate(n)
		for k := 0; k < 3; k++ {
			position[k] += q[k] * w
			normal[k] += r[k] * w
		}
		total += w
	}
	if total == 0 {
		return p, n
	}
	for k := 0; k < 3; k++ {
		position[k] /= total
	}
	return position, normalize(normal)
}

func posedSamples(samples []heroSample, matrices [limbCount]mat34) []heroSample {
	out := make([]heroSample, len(samples))
	for i, s := range samples {
		m := blend(s.limb, matrices)
		out[i] = s
		out[i].position = m.apply(s.position)
		out[i].normal = normalize(m.rotate(s.normal))
	}
	return out
}

// matchPose finds the arm and leg angles that bring the hero's meshes closest to the model:
// the mean distance from a sample of the model's points to the hero's, leaving out the farthest
// fifth (backpacks, weapons and capes the hero has no match for).
func matchPose(samples []heroSample, skeleton *Skeleton, points [][3]float32, cell float64) Pose {
	step := max(1, len(points)/2500)
	var probe [][3]float32
	for i := 0; i < len(points); i += step {
		probe = append(probe, points[i])
	}
	if every := len(samples) / 20000; every > 1 { // a thinner hero is plenty to compare poses
		var thin []heroSample
		for i := 0; i < len(samples); i += every {
			thin = append(thin, samples[i])
		}
		samples = thin
	}
	score := func(pose Pose) float64 {
		posed := posedSamples(samples, skeleton.limbMatrices(pose))
		grid := newPointGrid(cell)
		for i, s := range posed {
			grid.add(i, [3]float32{float32(s.position[0]), float32(s.position[1]), float32(s.position[2])})
		}
		distances := make([]float64, 0, len(probe))
		for _, p := range probe {
			if hits := grid.nearest(p, 1); len(hits) > 0 {
				distances = append(distances, hits[0].distance)
			}
		}
		sort.Float64s(distances)
		keep := distances[:len(distances)*4/5]
		total := 0.0
		for _, d := range keep {
			total += d
		}
		return total / float64(max(1, len(keep)))
	}
	best := Pose{}
	bestScore := score(best)
	try := func(pose Pose) {
		if s := score(pose); s < bestScore {
			best, bestScore = pose, s
		}
	}
	for arms := -50.0; arms <= 80; arms += 10 {
		try(Pose{Arms: arms})
	}
	base := best.Arms
	for arms := base - 8; arms <= base+8; arms += 2 {
		try(Pose{Arms: arms})
	}
	for legs := -10.0; legs <= 20; legs += 5 {
		try(Pose{Arms: best.Arms, Legs: legs})
	}
	return best
}

// Rigged is an imported model moved into the hero's bind pose, with the bones each vertex follows.
type Rigged struct {
	Mesh     *ImportedMesh
	Skin     [][]BoneWeight
	Pose     Pose              // how the hero was bent to match the model
	JointMap map[string]string // a rigged model's joints and the hero bones they became
}

// onHeroBones reports whether a rigged model's joints are the hero's own bones: named like them,
// limbs among them, and every one within 3% of the hero's height of the bone of its name.
func onHeroBones(mesh *ImportedMesh, skeleton *Skeleton, height float64) bool {
	named, limb := heroJoints(mesh, skeleton, height)
	return named >= 4 && limb
}

// heroJoints counts a rigged model's skinned joints that are the hero's bones where they stand, and
// tells whether a limb is among them; none when any skinned joint is not.
func heroJoints(mesh *ImportedMesh, skeleton *Skeleton, height float64) (int, bool) {
	if len(mesh.Joints) != len(mesh.Positions) || len(mesh.JointPositions) != len(mesh.JointNames) || height <= 0 {
		return 0, false
	}
	skinned := map[int]bool{}
	for _, joints := range mesh.Joints {
		for _, j := range joints {
			if j.Weight > 0 {
				skinned[j.Joint] = true
			}
		}
	}
	limbs := skeleton.limbs()
	named, limb := 0, false
	for joint := range skinned {
		if joint < 0 || joint >= len(mesh.JointNames) {
			return 0, false
		}
		bone, ok := skeleton.index[mesh.JointNames[joint]]
		if !ok {
			return 0, false
		}
		p, _ := skeleton.position(mesh.JointNames[joint])
		q := vec64(mesh.JointPositions[joint])
		if math.Sqrt(dot(sub(p, q), sub(p, q))) > 0.03*height {
			return 0, false
		}
		named++
		limb = limb || limbs[bone] != limbNone
	}
	return named, limb
}

// PaintStroke sets how much vertices follow a bone, as painted in the studio: weight 1 makes
// them follow it alone, 0 takes the bone away, and the rest share what is left.
type PaintStroke struct {
	Bone     string    `json:"bone"`
	Vertices []int32   `json:"vertices"`
	Weights  []float32 `json:"weights"`
}

// applyRigEdits lays the studio's hand edits over the computed weights: pinned pieces follow one
// bone, then painted strokes in order.
func applyRigEdits(mesh *ImportedMesh, skin [][]BoneWeight, skeleton *Skeleton, opts ImportOptions, maxBones int) {
	if len(opts.Pins) > 0 {
		pieces := Pieces(mesh)
		for i, piece := range pieces {
			if bone, ok := opts.Pins[piece]; ok {
				if _, known := skeleton.index[bone]; known {
					skin[i] = []BoneWeight{{bone, 1}}
				}
			}
		}
	}
	for _, stroke := range opts.Paint {
		if _, known := skeleton.index[stroke.Bone]; !known {
			continue
		}
		for k, vertex := range stroke.Vertices {
			if vertex < 0 || int(vertex) >= len(skin) || k >= len(stroke.Weights) {
				continue
			}
			w := float32(math.Max(0, math.Min(1, float64(stroke.Weights[k]))))
			var others []BoneWeight
			var rest float32
			for _, b := range skin[vertex] {
				if b.Bone != stroke.Bone {
					others = append(others, b)
					rest += b.Weight
				}
			}
			if rest == 0 { // nothing else to share with: the bone keeps it all, or nothing changes
				if w > 0 {
					skin[vertex] = []BoneWeight{{stroke.Bone, 1}}
				}
				continue
			}
			list := make([]BoneWeight, 0, len(others)+1)
			if w > 0 {
				list = append(list, BoneWeight{stroke.Bone, w})
			}
			for _, b := range others {
				list = append(list, BoneWeight{b.Bone, b.Weight / rest * (1 - w)})
			}
			sort.Slice(list, func(a, b int) bool {
				if list[a].Weight != list[b].Weight {
					return list[a].Weight > list[b].Weight
				}
				return list[a].Bone < list[b].Bone
			})
			list = list[:min(len(list), maxBones)]
			var sum float32
			for _, b := range list {
				sum += b.Weight
			}
			for j := range list {
				list[j].Weight /= sum
			}
			skin[vertex] = list
		}
	}
}

// Pieces numbers the separate pieces of a mesh (a hat, a bag, the body), as the studio picks
// them for pinning: triangles sharing vertices, or vertices at the same spot (texture seams),
// are one piece. Pieces are numbered in the order of their first vertex.
func Pieces(mesh *ImportedMesh) []int {
	n := len(mesh.Positions)
	parent := make([]int, n)
	for i := range parent {
		parent[i] = i
	}
	find := func(i int) int {
		for parent[i] != i {
			parent[i] = parent[parent[i]]
			i = parent[i]
		}
		return i
	}
	union := func(a, b int) {
		if a < 0 || b < 0 || a >= n || b >= n {
			return
		}
		if ra, rb := find(a), find(b); ra != rb {
			if ra < rb {
				parent[rb] = ra
			} else {
				parent[ra] = rb
			}
		}
	}
	for _, part := range mesh.Parts {
		for t := 0; t+2 < len(part.Indices); t += 3 {
			union(int(part.Indices[t]), int(part.Indices[t+1]))
			union(int(part.Indices[t]), int(part.Indices[t+2]))
		}
	}
	at := map[[3]int32]int{}
	for i, p := range mesh.Positions {
		key := [3]int32{int32(math.Round(float64(p[0]) * 1000)), int32(math.Round(float64(p[1]) * 1000)), int32(math.Round(float64(p[2]) * 1000))}
		if j, ok := at[key]; ok {
			union(i, j)
		} else {
			at[key] = i
		}
	}
	ids := map[int]int{}
	out := make([]int, n)
	for i := range out {
		root := find(i)
		id, ok := ids[root]
		if !ok {
			id = len(ids)
			ids[root] = id
		}
		out[i] = id
	}
	return out
}

// AutoRig gives each vertex of a fitted model the bones of the nearest points of the hero's own
// meshes, so it moves with the hero's animations. When the model stands in another pose (arms
// down where the hero's are raised), the hero is first bent to match it, unless pose says how;
// the model is then bent back into the hero's bind pose by its new weights.
func (m *Model) AutoRig(fitted *ImportedMesh, opts ImportOptions, pose *Pose, maxBones int) (*Rigged, error) {
	skeleton, _ := m.Skeleton()
	samples, err := m.heroSamples(opts, skeleton)
	if err != nil {
		return nil, err
	}
	lo, hi, err := m.Bounds(opts)
	if err != nil {
		return nil, err
	}
	cell := math.Max(hi[2]-lo[2], 1) / 48
	chosen := Pose{}
	matrices := [limbCount]mat34{}
	for i := range matrices {
		matrices[i] = identity34()
	}
	var limbs []int
	if skeleton != nil {
		limbs = skeleton.limbs()
		switch {
		case pose != nil:
			chosen = *pose
		case onHeroBones(fitted, skeleton, hi[2]-lo[2]):
			// Rigged on the hero's own skeleton, standing where its bones stand: already in the
			// hero's bind pose, which guessing from the shape could only get wrong.
		default:
			chosen = matchPose(samples, skeleton, fitted.Positions, cell)
		}
		matrices = skeleton.limbMatrices(chosen)
		samples = posedSamples(samples, matrices)
	}
	// One grid per limb: a vertex takes its weights only from hero points of the limb it is on,
	// so a hand resting on a thigh does not follow the leg.
	var grids [limbCount]*pointGrid
	var segments []limbSegment
	if skeleton != nil {
		segments = skeleton.segments(samples, matrices, limbs)
	}
	for limb := range grids {
		grids[limb] = newPointGrid(cell)
	}
	all := newPointGrid(cell)
	for i, s := range samples {
		p := [3]float32{float32(s.position[0]), float32(s.position[1]), float32(s.position[2])}
		all.add(i, p)
		total := 0.0
		for _, w := range s.limb {
			total += w
		}
		// A limb takes points that follow it even partly, so shoulders and hips blend; the body
		// takes only points that mostly follow it, or the sides of a chest would follow the arms
		// hanging beside them.
		for limb, w := range s.limb {
			share := 0.3
			if limb == limbNone {
				share = 0.7
			}
			if total > 0 && w >= share*total {
				grids[limb].add(i, p)
			}
		}
	}
	const neighbours = 6
	out := &Rigged{Pose: chosen, Skin: make([][]BoneWeight, len(fitted.Positions))}
	labels := make([]int, len(fitted.Positions))
	if len(segments) > 0 {
		labels = surfaceLabels(fitted, segments)
	}
	distances := make([]float64, len(fitted.Positions))
	bind := &ImportedMesh{Parts: fitted.Parts, UVs: fitted.UVs,
		Positions: make([][3]float32, len(fitted.Positions)), Normals: make([][3]float32, len(fitted.Positions))}
	for i, p := range fitted.Positions {
		var normal [3]float64
		if i < len(fitted.Normals) {
			normal = vec64(fitted.Normals[i])
		}
		grid := all
		if len(segments) > 0 {
			if g := grids[labels[i]]; len(g.pts) > 0 {
				grid = g
			}
		}
		near := grid.nearest(p, neighbours)
		if len(near) > 0 {
			distances[i] = near[0].distance
		}
		totals := map[string]float64{}
		closest := 0.0
		for rank, hit := range near {
			s := samples[hit.index]
			if rank == 0 {
				closest = hit.distance
			}
			// Points much further than the closest one say little; points facing the other way
			// (the inside of the other leg, the back of an arm) say less.
			w := 1 / math.Pow(hit.distance+closest*0.25+cell*0.05, 2)
			w *= math.Max(0.15, (1+dot(normal, s.normal))/2)
			for _, b := range s.bones {
				totals[b.Bone] += w * float64(b.Weight)
			}
		}
		list := make([]BoneWeight, 0, len(totals))
		for bone, w := range totals {
			list = append(list, BoneWeight{bone, float32(w)})
		}
		sort.Slice(list, func(a, b int) bool {
			if list[a].Weight != list[b].Weight {
				return list[a].Weight > list[b].Weight
			}
			return list[a].Bone < list[b].Bone
		})
		list = list[:min(len(list), maxBones)]
		var sum float32
		for _, b := range list {
			sum += b.Weight
		}
		kept := list[:0]
		for _, b := range list {
			b.Weight /= sum
			if b.Weight >= 0.02 || len(kept) == 0 { // a bone holding under 2% only adds jitter
				kept = append(kept, b)
			}
		}
		out.Skin[i] = kept
	}
	if len(fitted.Joints) == len(fitted.Positions) && skeleton != nil {
		out.JointMap = mapJoints(fitted, out.Skin, distances, skeleton, maxBones, math.Max(hi[2]-lo[2], 1), opts.JointMap)
	} else {
		fixed := rigidParts(fitted, out.Skin, labels, distances, math.Max(hi[2]-lo[2], 1), skeleton)
		smoothWeights(fitted, out.Skin, fixed, 8, maxBones)
	}
	if skeleton != nil {
		applyRigEdits(fitted, out.Skin, skeleton, opts, maxBones)
	}
	var inverses [limbCount]mat34
	for limb := range inverses {
		inverses[limb], _ = matrices[limb].inverse() // turns, which always have one
	}
	for i, p := range fitted.Positions {
		var normal [3]float64
		if i < len(fitted.Normals) {
			normal = vec64(fitted.Normals[i])
		}
		kept := out.Skin[i]
		// Bend the vertex back from the matched pose into the hero's bind pose.
		position, n := vec64(p), normal
		if limbs != nil {
			var follow [limbCount]float64
			for _, b := range kept {
				if bone, ok := skeleton.index[b.Bone]; ok {
					follow[limbs[bone]] += float64(b.Weight)
				}
			}
			position, n = unbend(follow, inverses, position, n)
		}
		bind.Positions[i] = [3]float32{float32(position[0]), float32(position[1]), float32(position[2])}
		bind.Normals[i] = [3]float32{float32(n[0]), float32(n[1]), float32(n[2])}
	}
	out.Mesh = bind
	tuneRig(out, skeleton, opts.Tuning)
	return out, nil
}

// limbSegment is a bone drawn from its parent, in the matched pose, and the limb it lies in.
type limbSegment struct {
	from, to [3]float64
	limb     int
	seed     bool // a vertex nearest this segment surely lies on its limb (see surfaceLabels)
}

// segments are the body bones the hero's points follow, as line segments from each bone's
// parent. A segment belongs to its parent's limb: the upper arm runs from arm_upper to arm_lower,
// while clavicle to arm_upper is still the shoulder.
func (s *Skeleton) segments(samples []heroSample, matrices [limbCount]mat34, limbs []int) []limbSegment {
	used := map[int]bool{}
	for _, sample := range samples {
		for _, b := range sample.bones {
			if i, ok := s.index[b.Bone]; ok {
				used[i] = true
			}
		}
	}
	posed := func(i int) [3]float64 {
		p := [3]float64{s.World[i][3], s.World[i][7], s.World[i][11]}
		return matrices[limbs[i]].apply(p)
	}
	var out []limbSegment
	for i := range s.Names {
		parent := s.Parent[i]
		if !used[i] || parent < 0 || !coreBone(s.Names[i]) || !coreBone(s.Names[parent]) {
			continue
		}
		// Hands and forearms can rest anywhere (on a hip, across the chest), and the hip joint and
		// shoulders sit where limbs meet, so only the spine, the upper arms and the legs decide.
		name := strings.ToLower(s.Names[parent])
		seed := strings.HasPrefix(name, "spine") || strings.HasPrefix(name, "neck") || strings.HasPrefix(name, "head") ||
			strings.HasPrefix(name, "arm_upper") || strings.HasPrefix(name, "leg_")
		out = append(out, limbSegment{posed(parent), posed(i), limbs[parent], seed})
	}
	return out
}

// limbDistances is how far a point is from the nearest bone of each limb, and whether the
// nearest bone of all is one that decides a limb by itself.
func limbDistances(segments []limbSegment, p [3]float64) ([limbCount]float64, bool) {
	var out [limbCount]float64
	for i := range out {
		out[i] = math.Inf(1)
	}
	nearest, seed := math.Inf(1), false
	for _, seg := range segments {
		d := sub(seg.to, seg.from)
		t := 0.0
		if l := dot(d, d); l > 0 {
			t = math.Max(0, math.Min(1, dot(sub(p, seg.from), d)/l))
		}
		closest := [3]float64{seg.from[0] + d[0]*t, seg.from[1] + d[1]*t, seg.from[2] + d[2]*t}
		distance := math.Sqrt(dot(sub(p, closest), sub(p, closest)))
		out[seg.limb] = math.Min(out[seg.limb], distance)
		if distance < nearest {
			// An arm decides only near the shoulder: a lowered elbow sits beside the hips, where
			// wide trousers or a coat flare out.
			nearest, seed = distance, seg.seed && (seg.limb != limbArmL && seg.limb != limbArmR || t <= 0.5)
		}
	}
	return out, seed
}

func nearestLimb(segments []limbSegment, p [3]float64) int {
	distances, _ := limbDistances(segments, p)
	best := limbNone
	for limb, d := range distances {
		if d < distances[best] {
			best = limb
		}
	}
	return best
}

// surfaceLabels says which limb each vertex is on. Vertices plainly closer to one limb's bones
// than to any other's decide for themselves; the rest take the limb of the nearest such vertex
// along the surface. A hand resting on a hip is close to the hip in space but far from it across
// the body, so the hip stays with the body. Loose pieces with no plain vertex use the nearest bone.
func surfaceLabels(mesh *ImportedMesh, segments []limbSegment) []int {
	n := len(mesh.Positions)
	labels := make([]int, n)
	reach := make([]float64, n)
	var queue labelQueue
	for i, p := range mesh.Positions {
		distances, decisive := limbDistances(segments, vec64(p))
		best, second := limbNone, -1
		for limb := range distances {
			if distances[limb] < distances[best] {
				best = limb
			}
		}
		for limb := range distances {
			if limb != best && (second < 0 || distances[limb] < distances[second]) {
				second = limb
			}
		}
		labels[i] = best
		reach[i] = math.Inf(1)
		if decisive && (second < 0 || distances[best] < 0.5*distances[second]) {
			reach[i] = 0
			queue.push(labelItem{0, i})
		}
	}
	neighbours := surfaceNeighbours(mesh)
	for queue.Len() > 0 {
		item := queue.pop()
		if item.cost > reach[item.vertex] {
			continue
		}
		p := vec64(mesh.Positions[item.vertex])
		for _, j := range neighbours[item.vertex] {
			q := vec64(mesh.Positions[j])
			cost := item.cost + math.Sqrt(dot(sub(p, q), sub(p, q)))
			if cost < reach[j] {
				reach[j], labels[j] = cost, labels[item.vertex]
				queue.push(labelItem{cost, int(j)})
			}
		}
	}
	return labels
}

type labelItem struct {
	cost   float64
	vertex int
}

// labelQueue is a binary min-heap of vertices by cost.
type labelQueue []labelItem

func (q labelQueue) Len() int { return len(q) }

func (q *labelQueue) push(item labelItem) {
	*q = append(*q, item)
	for i := len(*q) - 1; i > 0; {
		parent := (i - 1) / 2
		if (*q)[parent].cost <= (*q)[i].cost {
			break
		}
		(*q)[parent], (*q)[i] = (*q)[i], (*q)[parent]
		i = parent
	}
}

func (q *labelQueue) pop() labelItem {
	old := *q
	top := old[0]
	last := len(old) - 1
	old[0] = old[last]
	*q = old[:last]
	for i := 0; ; {
		left, smallest := 2*i+1, i
		if left < last && (*q)[left].cost < (*q)[smallest].cost {
			smallest = left
		}
		if left+1 < last && (*q)[left+1].cost < (*q)[smallest].cost {
			smallest = left + 1
		}
		if smallest == i {
			break
		}
		(*q)[i], (*q)[smallest] = (*q)[smallest], (*q)[i]
		i = smallest
	}
	return top
}

// rigidParts makes small loose pieces of the model that reach from a hand into a leg (a gun held
// at the hip, a sword along the thigh) follow one bone, the one the piece is closest to, instead
// of being pulled apart between the hand and the thigh. Pieces are triangles sharing vertices or
// positions.
func rigidParts(mesh *ImportedMesh, skin [][]BoneWeight, labels []int, distances []float64, height float64, skeleton *Skeleton) []bool {
	// A held prop rides on the hand, not on the finger that happens to touch it.
	holder := func(bone string) string {
		if skeleton == nil {
			return bone
		}
		i, ok := skeleton.index[bone]
		for depth := 0; ok && i >= 0 && depth < 64 && strings.HasPrefix(strings.ToLower(skeleton.Names[i]), "finger"); depth++ {
			i = skeleton.Parent[i]
		}
		if ok && i >= 0 {
			return skeleton.Names[i]
		}
		return bone
	}
	n := len(mesh.Positions)
	groups := map[int][]int{}
	for i, piece := range Pieces(mesh) {
		groups[piece] = append(groups[piece], i)
	}
	fixed := make([]bool, n)
	for _, members := range groups {
		seen := [limbCount]bool{}
		for _, i := range members {
			seen[labels[i]] = true
		}
		// A held prop reaches from a hand down past a hip. Clothing spans both arms or both legs,
		// and the body is most of the model; those keep their weights vertex by vertex.
		arm, leg := seen[limbArmL] || seen[limbArmR], seen[limbLegL] || seen[limbLegR]
		pair := (seen[limbArmL] && seen[limbArmR]) || (seen[limbLegL] && seen[limbLegR])
		if !arm || !leg || pair || len(members) > n*15/100 {
			continue
		}
		votes := map[string]float64{}
		for _, i := range members {
			closeness := 1 / math.Pow(distances[i]+height*0.002, 2)
			for _, b := range skin[i] {
				votes[holder(b.Bone)] += closeness * float64(b.Weight)
			}
		}
		best, bestVote := "", -1.0
		for bone, v := range votes {
			if v > bestVote || (v == bestVote && bone < best) {
				best, bestVote = bone, v
			}
		}
		for _, i := range members {
			skin[i] = []BoneWeight{{best, 1}}
			fixed[i] = true
		}
	}
	return fixed
}

// mapJoints keeps a rigged model's own weights and only swaps its joints for the hero's bones:
// the bone of the same name when the hero has one, else the bone the skeleton's shape puts it on
// (structuralJoints), else the hero bone its vertices voted for most (closer vertices count
// more). Props stay on the joint the artist put them on, and the weights stay as smooth as the
// artist painted them. forced (joint name -> bone, from the studio) overrides all of that. It
// returns each skinned joint's bone, by joint name.
func mapJoints(mesh *ImportedMesh, skin [][]BoneWeight, distances []float64, skeleton *Skeleton, maxBones int, height float64,
	forced map[string]string) map[string]string {
	votes := map[int]map[string]float64{}
	for i, joints := range mesh.Joints {
		closeness := 1 / math.Pow(distances[i]+height*0.002, 2)
		for _, j := range joints {
			if votes[j.Joint] == nil {
				votes[j.Joint] = map[string]float64{}
			}
			for _, b := range skin[i] {
				votes[j.Joint][b.Bone] += closeness * float64(j.Weight) * float64(b.Weight)
			}
		}
	}
	byName := map[string]string{}
	for _, name := range skeleton.Names {
		byName[strings.ToLower(name)] = name
	}
	nameOf := func(joint int) string {
		if joint >= 0 && joint < len(mesh.JointNames) {
			return mesh.JointNames[joint]
		}
		return ""
	}
	isProp := func(name string) bool {
		lower := strings.ToLower(name)
		for _, word := range []string{"weapon", "gun", "prop", "sword", "shield", "item", "attach"} {
			if strings.Contains(lower, word) {
				return true
			}
		}
		return false
	}
	// Joints named like the hero's bones (models from Valve's games) map by name. When most are
	// not, a humanoid skeleton maps by its shape (see structuralJoints), and the joints off its
	// main chains follow their parent.
	named := 0
	for joint := range votes {
		if _, ok := byName[strings.ToLower(nameOf(joint))]; ok {
			named++
		}
	}
	var structural map[int]string
	if named*2 < len(votes) {
		structural = structuralJoints(mesh, skeleton)
	}
	mapping := map[int]string{}
	for joint, name := range mesh.JointNames {
		if bone, ok := forced[name]; ok {
			if _, known := skeleton.index[bone]; known {
				mapping[joint] = bone
			}
		}
	}
	var resolve func(joint, depth int) string
	resolve = func(joint, depth int) string {
		if bone, ok := mapping[joint]; ok {
			return bone
		}
		name := nameOf(joint)
		// A bone of the same name (models from Valve's games share their naming) wins.
		if bone, ok := byName[strings.ToLower(name)]; ok {
			mapping[joint] = bone
			return bone
		}
		if structural != nil {
			if bone, ok := structural[joint]; ok {
				mapping[joint] = bone
				return bone
			}
			if depth < 64 && joint < len(mesh.JointParents) && mesh.JointParents[joint] >= 0 {
				if bone := resolve(mesh.JointParents[joint], depth+1); bone != "" {
					mapping[joint] = bone
					return bone
				}
			}
		}
		// A weapon or prop joint the hero lacks rides on whatever its parent joint becomes.
		if isProp(name) && depth < 64 && joint < len(mesh.JointParents) && mesh.JointParents[joint] >= 0 {
			if bone := resolve(mesh.JointParents[joint], depth+1); bone != "" {
				mapping[joint] = bone
				return bone
			}
		}
		best, bestVote := "", -1.0
		for bone, v := range votes[joint] {
			if v > bestVote || (v == bestVote && bone < best) {
				best, bestVote = bone, v
			}
		}
		mapping[joint] = best
		return best
	}
	for joint := range votes {
		resolve(joint, 0)
	}
	for i, joints := range mesh.Joints {
		if len(joints) == 0 {
			continue // a vertex the file left unskinned keeps its guessed weights
		}
		totals := map[string]float32{}
		for _, j := range joints {
			if bone := mapping[j.Joint]; bone != "" {
				totals[bone] += j.Weight
			}
		}
		if len(totals) == 0 {
			continue
		}
		list := make([]BoneWeight, 0, len(totals))
		for bone, w := range totals {
			list = append(list, BoneWeight{bone, w})
		}
		sort.Slice(list, func(a, b int) bool {
			if list[a].Weight != list[b].Weight {
				return list[a].Weight > list[b].Weight
			}
			return list[a].Bone < list[b].Bone
		})
		list = list[:min(len(list), maxBones)]
		var sum float32
		for _, b := range list {
			sum += b.Weight
		}
		for k := range list {
			list[k].Weight /= sum
		}
		skin[i] = list
	}
	out := map[string]string{}
	for joint := range votes {
		if name := nameOf(joint); name != "" && mapping[joint] != "" {
			out[name] = mapping[joint]
		}
	}
	return out
}

// Bend moves a rigged model's vertices as the hero's limbs would in a pose, to check the weights
// before going in game: the arms lifted or lowered, the legs spread or closed.
func (m *Model) Bend(rigged *Rigged, pose Pose) [][3]float32 {
	out := make([][3]float32, len(rigged.Mesh.Positions))
	skeleton, err := m.Skeleton()
	if err != nil {
		copy(out, rigged.Mesh.Positions)
		return out
	}
	limbs := skeleton.limbs()
	matrices := skeleton.limbMatrices(pose)
	for i, p := range rigged.Mesh.Positions {
		var follow [limbCount]float64
		for _, b := range rigged.Skin[i] {
			if bone, ok := skeleton.index[b.Bone]; ok {
				follow[limbs[bone]] += float64(b.Weight)
			}
		}
		q := blend(follow, matrices).apply(vec64(p))
		out[i] = [3]float32{float32(q[0]), float32(q[1]), float32(q[2])}
	}
	return out
}

// surfaceNeighbours lists the vertices each vertex shares an edge with, counting vertices split
// along texture seams (at the same spot) as one.
func surfaceNeighbours(mesh *ImportedMesh) [][]int32 {
	neighbours := make([][]int32, len(mesh.Positions))
	link := func(a, b uint32) {
		neighbours[a] = append(neighbours[a], int32(b))
		neighbours[b] = append(neighbours[b], int32(a))
	}
	for _, part := range mesh.Parts {
		for t := 0; t+2 < len(part.Indices); t += 3 {
			a, b, c := part.Indices[t], part.Indices[t+1], part.Indices[t+2]
			link(a, b)
			link(b, c)
			link(c, a)
		}
	}
	at := map[[3]int32]int{}
	for i, p := range mesh.Positions {
		key := [3]int32{int32(math.Round(float64(p[0]) * 1000)), int32(math.Round(float64(p[1]) * 1000)), int32(math.Round(float64(p[2]) * 1000))}
		if j, ok := at[key]; ok {
			link(uint32(i), uint32(j))
		} else {
			at[key] = i
		}
	}
	return neighbours
}

// smoothWeights blends each vertex's bone weights with its neighbours' a few times, so weights
// fade across a joint instead of jumping where one bone's area ends. Vertices in fixed (rigid
// pieces) keep theirs.
func smoothWeights(mesh *ImportedMesh, skin [][]BoneWeight, fixed []bool, rounds, maxBones int) {
	neighbours := surfaceNeighbours(mesh)
	ids := map[string]int{}
	var names []string
	type weight struct {
		bone int
		w    float32
	}
	current := make([][]weight, len(skin))
	for i, list := range skin {
		for _, b := range list {
			id, ok := ids[b.Bone]
			if !ok {
				id = len(names)
				ids[b.Bone] = id
				names = append(names, b.Bone)
			}
			current[i] = append(current[i], weight{id, b.Weight})
		}
	}
	sum := make([]float32, len(names))
	var touched []int
	for round := 0; round < rounds; round++ {
		next := make([][]weight, len(current))
		for i := range current {
			if fixed[i] || len(neighbours[i]) == 0 {
				next[i] = current[i]
				continue
			}
			touched = touched[:0]
			add := func(list []weight, scale float32) {
				for _, w := range list {
					if sum[w.bone] == 0 {
						touched = append(touched, w.bone)
					}
					sum[w.bone] += w.w * scale
				}
			}
			add(current[i], 0.5)
			share := 0.5 / float32(len(neighbours[i]))
			for _, j := range neighbours[i] {
				add(current[j], share)
			}
			list := make([]weight, 0, len(touched))
			for _, bone := range touched {
				list = append(list, weight{bone, sum[bone]})
				sum[bone] = 0
			}
			next[i] = list
		}
		current = next
	}
	for i, list := range current {
		if fixed[i] {
			continue
		}
		sort.Slice(list, func(a, b int) bool {
			if list[a].w != list[b].w {
				return list[a].w > list[b].w
			}
			return list[a].bone < list[b].bone
		})
		list = list[:min(len(list), maxBones)]
		var total float32
		for _, w := range list {
			total += w.w
		}
		out := make([]BoneWeight, 0, len(list))
		for _, w := range list {
			if share := w.w / total; share >= 0.02 || len(out) == 0 {
				out = append(out, BoneWeight{names[w.bone], share})
			}
		}
		var kept float32
		for _, b := range out {
			kept += b.Weight
		}
		for k := range out {
			out[k].Weight /= kept
		}
		skin[i] = out
	}
}
