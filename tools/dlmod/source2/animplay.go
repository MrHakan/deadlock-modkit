package source2

import (
	"math"
)

// Playing clips the way the hero graph does, roughly, for previews: a clip such as a melee swing
// is laid over the movement through a bone mask from the skeleton, and two-bone IK pulls the arms
// to the gun (weaponHand_L/R) and the feet to their targets (leg_*_IKTARGET). The real graph
// blends many more layers; this is close enough to see how a rig bends in game.

// readMasks reads the skeleton's bone masks: each bone's weight in the mask, 0 for bones it does
// not list.
func readMasks(v *Value) map[string]map[string]float64 {
	defs := v.Get("m_maskDefinitions")
	if defs == nil {
		return nil
	}
	masks := map[string]map[string]float64{}
	for _, def := range defs.Items {
		id, list := def.Get("m_ID"), def.Get("m_primaryWeightList")
		if id == nil || list == nil {
			continue
		}
		bones, weights := list.Get("m_boneIDs"), list.Get("m_weights")
		if bones == nil || weights == nil || len(bones.Items) != len(weights.Items) {
			continue
		}
		mask := map[string]float64{}
		for i, bone := range bones.Items {
			mask[bone.S] = numberOf(weights.Items[i])
		}
		masks[id.S] = mask
	}
	return masks
}

func (s *AnimSkeleton) bone(name string) int {
	for i, b := range s.Bones {
		if b == name {
			return i
		}
	}
	return -1
}

func slerp(a, b [4]float64, t float64) [4]float64 {
	d := a[0]*b[0] + a[1]*b[1] + a[2]*b[2] + a[3]*b[3]
	if d < 0 { // the shorter way round
		b, d = [4]float64{-b[0], -b[1], -b[2], -b[3]}, -d
	}
	var out [4]float64
	if d > 0.9995 {
		for k := range out {
			out[k] = a[k] + (b[k]-a[k])*t
		}
	} else {
		angle := math.Acos(d)
		sa, sb := math.Sin((1-t)*angle)/math.Sin(angle), math.Sin(t*angle)/math.Sin(angle)
		for k := range out {
			out[k] = a[k]*sa + b[k]*sb
		}
	}
	n := math.Sqrt(out[0]*out[0] + out[1]*out[1] + out[2]*out[2] + out[3]*out[3])
	for k := range out {
		out[k] /= n
	}
	return out
}

// BlendPoses lays layer over base: each bone takes the layer by its weight in the mask, or
// wholly without one. A missing mask leaves the base.
func (s *AnimSkeleton) BlendPoses(base, layer []BonePose, mask string) []BonePose {
	weights, ok := s.Masks[mask]
	out := make([]BonePose, len(base))
	for i := range base {
		w := 1.0
		if mask != "" {
			w = 0
			if ok && i < len(s.Bones) {
				w = weights[s.Bones[i]]
			}
		}
		if i >= len(layer) || w <= 0 {
			out[i] = base[i]
			continue
		}
		a, b := base[i], layer[i]
		out[i] = BonePose{Rotation: slerp(a.Rotation, b.Rotation, w), Scale: a.Scale + (b.Scale-a.Scale)*w}
		for k := 0; k < 3; k++ {
			out[i].Position[k] = a.Position[k] + (b.Position[k]-a.Position[k])*w
		}
	}
	return out
}

// ikChains are the hero graph's two-bone IK chains: upper bone, lower bone, end, and its target.
var ikChains = [][4]string{
	{"arm_upper_L", "arm_lower_L", "hand_L", "weaponHand_L"},
	{"arm_upper_R", "arm_lower_R", "hand_R", "weaponHand_R"},
	{"leg_upper_L", "leg_lower_L", "ankle_L", "leg_L_IKTARGET"},
	{"leg_upper_R", "leg_lower_R", "ankle_R", "leg_R_IKTARGET"},
}

// PlayPose places a pose in model space for a preview: the root stays at its rest, so the hero
// moves on the spot, and with ik the arms and legs reach for their IK targets. The matrices are
// row-major 3x4, one per skeleton bone.
func (s *AnimSkeleton) PlayPose(pose []BonePose, ik bool) [][12]float64 {
	local := make([]BonePose, len(s.Bones))
	for i := range s.Bones {
		local[i] = s.Rest[i]
		if i < len(pose) && s.Parent[i] >= 0 {
			local[i] = pose[i]
		}
	}
	model := s.ModelSpace(local)
	if ik {
		for _, chain := range ikChains {
			s.solveTwoBone(model, s.bone(chain[0]), s.bone(chain[1]), s.bone(chain[2]), s.bone(chain[3]))
		}
	}
	out := make([][12]float64, len(model))
	for i, m := range model {
		out[i] = [12]float64(m)
	}
	return out
}

func position(m mat34) [3]float64 { return [3]float64{m[3], m[7], m[11]} }

func distance(a, b [3]float64) float64 {
	d := sub(a, b)
	return math.Sqrt(dot(d, d))
}

func angleBetween(a, b [3]float64) float64 {
	la, lb := math.Sqrt(dot(a, a)), math.Sqrt(dot(b, b))
	if la == 0 || lb == 0 {
		return 0
	}
	return math.Acos(math.Max(-1, math.Min(1, dot(a, b)/(la*lb)))) * 180 / math.Pi
}

// turnSubtree turns a bone and everything below it about an axis through a point.
func (s *AnimSkeleton) turnSubtree(model []mat34, root int, axis [3]float64, degrees float64, pivot [3]float64) {
	if degrees == 0 || dot(axis, axis) < 1e-12 {
		return
	}
	turn := rotationAbout(axis, degrees, pivot)
	inside := make([]bool, len(model))
	for i := root; i < len(model); i++ {
		if i == root || (s.Parent[i] >= 0 && inside[s.Parent[i]]) {
			inside[i] = true
			model[i] = turn.mul(model[i])
		}
	}
}

// solveTwoBone bends upper-lower-end so that end reaches target: first the middle joint opens or
// closes in its own plane until the chain is as long as the distance to the target, then the
// whole chain turns to point at it.
func (s *AnimSkeleton) solveTwoBone(model []mat34, upper, lower, end, target int) {
	if upper < 0 || lower < 0 || end < 0 || target < 0 {
		return
	}
	a, b, c, t := position(model[upper]), position(model[lower]), position(model[end]), position(model[target])
	lab, lbc := distance(a, b), distance(b, c)
	if lab == 0 || lbc == 0 {
		return
	}
	reach := math.Max(math.Abs(lab-lbc)+1e-3, math.Min(distance(a, t), lab+lbc-1e-3))
	// The joint's angle for that reach (law of cosines), turned about the chain's bend axis.
	want := math.Acos(math.Max(-1, math.Min(1, (lab*lab+lbc*lbc-reach*reach)/(2*lab*lbc)))) * 180 / math.Pi
	have := angleBetween(sub(a, b), sub(c, b))
	axis := cross(sub(c, b), sub(a, b))
	if dot(axis, axis) < 1e-12 { // a straight limb: bend it the way the bone's own frame says
		axis = [3]float64{model[lower][1], model[lower][5], model[lower][9]}
	}
	s.turnSubtree(model, lower, axis, have-want, b)
	c = position(model[end])
	s.turnSubtree(model, upper, cross(sub(c, a), sub(t, a)), angleBetween(sub(c, a), sub(t, a)), a)
}

// AveragePose is the mean of several poses, bone by bone: the hero's usual pose over its clips.
func (s *AnimSkeleton) AveragePose(poses [][]BonePose) []BonePose {
	out := make([]BonePose, len(s.Bones))
	copy(out, s.Rest)
	if len(poses) == 0 {
		return out
	}
	for i := range s.Bones {
		var q [4]float64
		var p [3]float64
		var scale float64
		n := 0
		for _, pose := range poses {
			if i >= len(pose) {
				continue
			}
			r := pose[i].Rotation
			if q[0]*r[0]+q[1]*r[1]+q[2]*r[2]+q[3]*r[3] < 0 { // the same turn either way round
				r = [4]float64{-r[0], -r[1], -r[2], -r[3]}
			}
			for k := 0; k < 4; k++ {
				q[k] += r[k]
			}
			for k := 0; k < 3; k++ {
				p[k] += pose[i].Position[k]
			}
			scale += pose[i].Scale
			n++
		}
		length := math.Sqrt(q[0]*q[0] + q[1]*q[1] + q[2]*q[2] + q[3]*q[3])
		if n == 0 || length == 0 {
			continue
		}
		for k := 0; k < 4; k++ {
			q[k] /= length
		}
		for k := 0; k < 3; k++ {
			p[k] /= float64(n)
		}
		out[i] = BonePose{Rotation: q, Position: p, Scale: scale / float64(n)}
	}
	return out
}

// ByName gives a pose's model-space matrices (as PlayPose returns them) by bone name.
func (s *AnimSkeleton) ByName(model [][12]float64) map[string][12]float64 {
	out := make(map[string][12]float64, len(model))
	for i, m := range model {
		if i < len(s.Bones) {
			out[s.Bones[i]] = m
		}
	}
	return out
}
