package source2

import (
	"errors"
	"math"
	"sort"
	"strings"
)

// Rig tuning and checking for what the game does to a hero's bones (see
// docs/custom-characters.md, "What we found out about Deadlock"): the arms are pulled to the gun
// by IK every frame, the spine leans and twists with every step, and a body bound to the hero's
// bind pose leans as far as the hero does.

// RigTuning adjusts a rig after its weights are made.
type RigTuning struct {
	SteadyBody bool // the spine's weight goes to the pelvis: the body neither leans nor twists
	NoArmIK    bool // the arms' weight goes to the clavicles: IK no longer bends them
	// UsualPose binds the body (pelvis, spine, neck, head) in the hero's usual pose instead of its
	// bind pose, so it looks as modelled while the hero moves rather than when it stands in its
	// bind pose: model-space matrices by bone, row-major 3x4.
	UsualPose map[string][12]float64
}

// usualPoseBones are the bones bound in the usual pose: the body, not the limbs, whose pose
// AutoRig matches already.
var usualPoseBones = []string{"pelvis", "spine_0", "spine_1", "spine_2", "spine_3", "neck_0", "head"}

func isArmBone(name string) (string, bool) {
	for _, side := range []string{"_L", "_R"} {
		if !strings.HasSuffix(name, side) {
			continue
		}
		base := strings.TrimSuffix(name, side)
		if base == "arm_upper" || base == "arm_lower" || base == "hand" || strings.HasPrefix(base, "finger_") ||
			strings.HasPrefix(base, "forearm_") || strings.HasPrefix(base, "weapon_hand") {
			return "clavicle" + side, true
		}
	}
	return "", false
}

// tuneRig applies the tuning to a finished rig.
func tuneRig(r *Rigged, skeleton *Skeleton, t RigTuning) {
	if skeleton == nil {
		return
	}
	move := func(from func(string) (string, bool)) {
		for i, list := range r.Skin {
			merged := map[string]float32{}
			for _, bw := range list {
				name := bw.Bone
				if to, ok := from(name); ok {
					if _, known := skeleton.index[to]; known {
						name = to
					}
				}
				merged[name] += bw.Weight
			}
			out := make([]BoneWeight, 0, len(merged))
			for name, w := range merged {
				out = append(out, BoneWeight{name, w})
			}
			sort.Slice(out, func(a, b int) bool {
				if out[a].Weight != out[b].Weight {
					return out[a].Weight > out[b].Weight
				}
				return out[a].Bone < out[b].Bone
			})
			r.Skin[i] = out
		}
	}
	if t.SteadyBody {
		move(func(name string) (string, bool) { return "pelvis", strings.HasPrefix(name, "spine_") })
	}
	if t.NoArmIK {
		move(isArmBone)
	}
	if len(t.UsualPose) > 0 && r.Mesh != nil {
		// v' = sum of w * Bind * Usual^-1 * v: the vertex where it must be at rest for the usual
		// pose to put it where it was modelled.
		rebind := map[string]mat34{}
		for _, name := range usualPoseBones {
			usual, ok := t.UsualPose[name]
			b, known := skeleton.index[name]
			if !ok || !known {
				continue
			}
			inv, ok := mat34(usual).inverse()
			if ok {
				rebind[name] = skeleton.World[b].mul(inv)
			}
		}
		if len(rebind) == 0 {
			return
		}
		// Bones below the body (clavicles, arms, the head's own) take their body bone's correction,
		// so the shoulders do not tear when the chest is rebound; the legs keep their bind.
		rebindOf := func(name string) mat34 {
			b, ok := skeleton.index[name]
			for ok && b >= 0 {
				n := skeleton.Names[b]
				if strings.HasPrefix(n, "leg_") || strings.HasPrefix(n, "ankle") || strings.HasPrefix(n, "ball") {
					break
				}
				if m, found := rebind[n]; found {
					return m
				}
				b = skeleton.Parent[b]
			}
			return identity34()
		}
		for i, list := range r.Skin {
			var p, n [3]float64
			var total float64
			for _, bw := range list {
				m := rebindOf(bw.Bone)
				w := float64(bw.Weight)
				q := m.apply(vec64(r.Mesh.Positions[i]))
				var nn [3]float64
				if i < len(r.Mesh.Normals) {
					nn = m.rotate(vec64(r.Mesh.Normals[i]))
				}
				for k := 0; k < 3; k++ {
					p[k] += q[k] * w
					n[k] += nn[k] * w
				}
				total += w
			}
			if total <= 0 {
				continue
			}
			for k := 0; k < 3; k++ {
				p[k] /= total
			}
			r.Mesh.Positions[i] = [3]float32{float32(p[0]), float32(p[1]), float32(p[2])}
			if i < len(r.Mesh.Normals) {
				n = normalize(n)
				r.Mesh.Normals[i] = [3]float32{float32(n[0]), float32(n[1]), float32(n[2])}
			}
		}
	}
}

// RigStrain is how much a rig suffers in a set of poses.
type RigStrain struct {
	Stretched  float64        `json:"stretched"`  // share of the surface (its edges, by length) 1.6 times longer or shorter than at rest, in the worst pose
	Worst      float64        `json:"worst"`      // the largest change of an edge's length, as a ratio (2 = twice as long or half as long)
	WorstFrame int            `json:"worstFrame"` // the pose with the most stretched edges
	Bones      map[string]int `json:"bones"`      // stretched edges in the worst pose, by the bone that moves their ends most
	BelowFloor float64        `json:"belowFloor"` // share of the vertices more than 2 units under the ground, in the worst pose for it
	Behind     float64        `json:"behind"`     // share of the vertices further behind the pelvis than a third of the height, where the camera is
}

// Strain skins the rig in each pose (model-space matrices by bone name; bones a pose leaves out
// follow their parent) and measures how far its surface stretches, how far it sinks into the
// ground and how much of it swings behind the hero, where the camera sits.
func (r *Rigged) Strain(skeleton *Skeleton, poses []map[string][12]float64) RigStrain {
	var out RigStrain
	if r.Mesh == nil || skeleton == nil || len(poses) == 0 {
		return out
	}
	type edge struct{ a, b uint32 }
	seen := map[edge]bool{}
	var edges []edge
	for _, part := range r.Mesh.Parts {
		for t := 0; t+2 < len(part.Indices); t += 3 {
			tri := part.Indices[t : t+3]
			for k := 0; k < 3; k++ {
				a, b := tri[k], tri[(k+1)%3]
				if a > b {
					a, b = b, a
				}
				if e := (edge{a, b}); a != b && !seen[e] && int(b) < len(r.Mesh.Positions) {
					seen[e] = true
					edges = append(edges, e)
				}
			}
		}
	}
	rest := make([]float64, len(edges))
	lo, hi := math.Inf(1), math.Inf(-1)
	for _, p := range r.Mesh.Positions {
		lo, hi = math.Min(lo, float64(p[2])), math.Max(hi, float64(p[2]))
	}
	height := math.Max(hi-lo, 1)
	shortest := height * 0.004
	for i, e := range edges {
		rest[i] = distance(vec64(r.Mesh.Positions[e.a]), vec64(r.Mesh.Positions[e.b]))
	}
	// The bone that moves each vertex most, for blaming.
	dominant := make([]string, len(r.Skin))
	for i, list := range r.Skin {
		best := float32(-1)
		for _, bw := range list {
			if bw.Weight > best {
				best, dominant[i] = bw.Weight, bw.Bone
			}
		}
	}
	inverses := make([]mat34, len(skeleton.World))
	for i, w := range skeleton.World {
		inverses[i], _ = w.inverse()
	}
	moved := make([][3]float64, len(r.Mesh.Positions))
	worstShare := -1.0
	for f, pose := range poses {
		skin := make([]mat34, len(skeleton.Names))
		for i, name := range skeleton.Names {
			switch m, ok := pose[name]; {
			case ok && skeleton.Parent[i] >= 0:
				skin[i] = mat34(m).mul(inverses[i])
			case skeleton.Parent[i] >= 0:
				skin[i] = skin[skeleton.Parent[i]]
			default:
				skin[i] = identity34()
			}
		}
		for i, p := range r.Mesh.Positions {
			v := vec64(p)
			var q [3]float64
			var total float64
			for _, bw := range r.Skin[i] {
				b, ok := skeleton.index[bw.Bone]
				if !ok {
					continue
				}
				s := skin[b].apply(v)
				for k := 0; k < 3; k++ {
					q[k] += s[k] * float64(bw.Weight)
				}
				total += float64(bw.Weight)
			}
			if total > 0 {
				for k := 0; k < 3; k++ {
					q[k] /= total
				}
			} else {
				q = v
			}
			moved[i] = q
		}
		// The share is of the surface's length, not of its edges: a dense mesh is not worse for it.
		stretched, considered := 0.0, 0.0
		bones := map[string]int{}
		for i, e := range edges {
			if rest[i] < shortest { // seams and tiny details stretch by huge ratios over nothing
				continue
			}
			considered += rest[i]
			ratio := distance(moved[e.a], moved[e.b]) / rest[i]
			if ratio < 1 {
				ratio = 1 / math.Max(ratio, 0.01) // shorter counts as much as longer
			}
			out.Worst = math.Max(out.Worst, ratio)
			if ratio > 1.6 {
				stretched += rest[i]
				bones[dominant[e.a]]++
			}
		}
		if share := stretched / math.Max(considered, 1e-9); share > worstShare {
			worstShare, out.Stretched, out.WorstFrame, out.Bones = share, share, f, bones
		}
		pelvis := [3]float64{}
		if m, ok := pose["pelvis"]; ok {
			pelvis = [3]float64{m[3], m[7], m[11]}
		}
		below, behind := 0, 0
		for _, q := range moved {
			if q[2] < -2 {
				below++
			}
			if q[0] < pelvis[0]-height/3 { // the hero faces +X; the camera sits behind it
				behind++
			}
		}
		count := float64(max(len(moved), 1))
		out.BelowFloor = math.Max(out.BelowFloor, float64(below)/count)
		out.Behind = math.Max(out.Behind, float64(behind)/count)
	}
	return out
}

// OwnRig is the hero's own replaced meshes with the hero's own weights, as a Rigged: the yardstick
// a new rig is measured against, since even the hero's own surface stretches in some clips.
func (m *Model) OwnRig(opts ImportOptions) (*Rigged, error) {
	r := &Rigged{Mesh: &ImportedMesh{}}
	for _, mesh := range m.replaced(opts) {
		for vbi, vb := range mesh.VertexBuffers {
			position := vb.Attribute("position", 0)
			if position == nil || vb.Attribute("blendindices", 0) == nil {
				continue
			}
			values, err := vb.Floats(position)
			if err != nil {
				return nil, err
			}
			skin, err := vb.Skin()
			if err != nil {
				return nil, err
			}
			base := uint32(len(r.Mesh.Positions))
			for i, v := range values {
				r.Mesh.Positions = append(r.Mesh.Positions, [3]float32{v[0], v[1], v[2]})
				var list []BoneWeight
				for _, in := range skin[i] {
					if in.Bone < len(mesh.Bones) && in.Weight > 0 {
						list = append(list, BoneWeight{mesh.Bones[in.Bone], in.Weight})
					}
				}
				r.Skin = append(r.Skin, list)
			}
			var part ImportedPart
			for _, draw := range mesh.DrawCalls {
				if draw.VertexBuffer != vbi || draw.IndexBuffer >= len(mesh.IndexBuffers) {
					continue
				}
				indices := mesh.IndexBuffers[draw.IndexBuffer].Indices()
				end := min(len(indices), draw.StartIndex+draw.IndexCount)
				for _, i := range indices[min(draw.StartIndex, end):end] {
					if v := base + i + uint32(draw.BaseVertex); int(v) < len(r.Mesh.Positions) {
						part.Indices = append(part.Indices, v)
					}
				}
			}
			part.Indices = part.Indices[:len(part.Indices)/3*3]
			r.Mesh.Parts = append(r.Mesh.Parts, part)
		}
	}
	if len(r.Mesh.Positions) == 0 {
		return nil, errors.New("the hero's replaced parts carry no bone weights")
	}
	return r, nil
}
