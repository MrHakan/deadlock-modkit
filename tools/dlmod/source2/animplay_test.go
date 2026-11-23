package source2

import (
	"math"
	"testing"
)

func testArm(target [3]float64) *AnimSkeleton {
	still := [4]float64{0, 0, 0, 1}
	return &AnimSkeleton{
		Bones:  []string{"root_motion", "arm_upper_L", "arm_lower_L", "hand_L", "weaponHand_L"},
		Parent: []int{-1, 0, 1, 2, 0},
		Rest: []BonePose{{Rotation: still, Scale: 1}, {Rotation: still, Scale: 1}, {Rotation: still, Position: [3]float64{10, 0, 0}, Scale: 1},
			{Rotation: still, Position: [3]float64{10, 0, 0}, Scale: 1}, {Rotation: still, Position: target, Scale: 1}},
		Masks: map[string]map[string]float64{"arm": {"arm_lower_L": 0.5}},
	}
}

func at(m [12]float64) [3]float64 { return [3]float64{m[3], m[7], m[11]} }

func TestTwoBoneIKReachesTheTarget(t *testing.T) {
	for _, target := range [][3]float64{{12, 8, 0}, {0, 5, 15}, {-6, -6, 3}} {
		s := testArm(target)
		model := s.PlayPose(nil, true)
		if d := distance(at(model[3]), target); d > 1e-6 {
			t.Errorf("target %v: the hand is %.4f away (at %v)", target, d, at(model[3]))
		}
		if upper, lower := distance(at(model[1]), at(model[2])), distance(at(model[2]), at(model[3])); math.Abs(upper-10) > 1e-9 || math.Abs(lower-10) > 1e-9 {
			t.Errorf("target %v: the bones changed length: %v %v", target, upper, lower)
		}
	}
	// Out of reach: the arm points straight at it.
	s := testArm([3]float64{0, 40, 0})
	model := s.PlayPose(nil, true)
	if hand := at(model[3]); math.Abs(hand[0]) > 0.01 || math.Abs(hand[1]-20) > 0.01 {
		t.Errorf("an unreachable target: the hand is at %v", hand)
	}
	// Without IK the pose is the clip's.
	if hand := at(s.PlayPose(nil, false)[3]); hand != [3]float64{20, 0, 0} {
		t.Errorf("without IK the hand stays at %v", hand)
	}
}

func TestBlendPosesFollowsTheMask(t *testing.T) {
	s := testArm([3]float64{0, 0, 0})
	base := append([]BonePose{}, s.Rest...)
	layer := append([]BonePose{}, s.Rest...)
	half := math.Sqrt(0.5)
	layer[2].Rotation = [4]float64{0, 0, half, half} // 90° about Z
	layer[1].Position = [3]float64{4, 0, 0}
	blended := s.BlendPoses(base, layer, "arm")
	q := blended[2].Rotation
	if angle := 2 * math.Acos(q[3]) * 180 / math.Pi; math.Abs(angle-45) > 1e-6 {
		t.Errorf("half the mask gives half the turn: %v°", angle)
	}
	if blended[1].Position != base[1].Position {
		t.Errorf("a bone outside the mask keeps the base: %v", blended[1].Position)
	}
	if whole := s.BlendPoses(base, layer, ""); whole[1].Position != layer[1].Position {
		t.Errorf("without a mask the layer replaces the base: %v", whole[1].Position)
	}
}
