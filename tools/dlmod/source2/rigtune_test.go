package source2

import (
	"math"
	"testing"
)

func tuneSkeleton() *Skeleton {
	s := &Skeleton{
		Names:  []string{"root_motion", "pelvis", "spine_1", "clavicle_L", "hand_L"},
		Parent: []int{-1, 0, 1, 2, 3},
		World:  []mat34{identity34(), identity34(), identity34(), identity34(), identity34()},
		index:  map[string]int{},
	}
	for i, n := range s.Names {
		s.index[n] = i
	}
	return s
}

func TestTuneRigMovesWeights(t *testing.T) {
	r := &Rigged{Mesh: &ImportedMesh{Positions: make([][3]float32, 3)}, Skin: [][]BoneWeight{
		{{"spine_1", 0.7}, {"pelvis", 0.3}},
		{{"hand_L", 1}},
		{{"clavicle_L", 0.5}, {"hand_L", 0.5}},
	}}
	tuneRig(r, tuneSkeleton(), RigTuning{SteadyBody: true, NoArmIK: true})
	want := [][]BoneWeight{{{"pelvis", 1}}, {{"clavicle_L", 1}}, {{"clavicle_L", 1}}}
	for i := range want {
		if len(r.Skin[i]) != 1 || r.Skin[i][0].Bone != want[i][0].Bone || math.Abs(float64(r.Skin[i][0].Weight-1)) > 1e-6 {
			t.Errorf("vertex %d: %v, want %v", i, r.Skin[i], want[i])
		}
	}
}

// Bound in the usual pose, a vertex lands where it was modelled when the hero is in that pose.
func TestTuneRigBindsInTheUsualPose(t *testing.T) {
	usual := rotationAbout([3]float64{0, 0, 1}, 90, [3]float64{})
	r := &Rigged{Mesh: &ImportedMesh{Positions: [][3]float32{{1, 0, 0}}, Normals: [][3]float32{{1, 0, 0}}}, Skin: [][]BoneWeight{{{"pelvis", 1}}}}
	tuneRig(r, tuneSkeleton(), RigTuning{UsualPose: map[string][12]float64{"pelvis": usual}})
	if p := r.Mesh.Positions[0]; math.Abs(float64(p[0])) > 1e-5 || math.Abs(float64(p[1])+1) > 1e-5 {
		t.Fatalf("at rest the vertex is turned back: %v", p)
	}
	back := usual.apply(vec64(r.Mesh.Positions[0]))
	if math.Abs(back[0]-1) > 1e-5 || math.Abs(back[1]) > 1e-5 {
		t.Fatalf("the usual pose puts it back at (1,0,0): %v", back)
	}
}

func TestRigStrain(t *testing.T) {
	// A strip of two triangles across two bones: pulling the hand away stretches the edges between.
	r := &Rigged{
		Mesh: &ImportedMesh{Positions: [][3]float32{{0, 0, 0}, {0, 1, 0}, {1, 0, 0}, {1, 1, 0}},
			Parts: []ImportedPart{{Indices: []uint32{0, 1, 2, 1, 3, 2}}}},
		Skin: [][]BoneWeight{{{"pelvis", 1}}, {{"pelvis", 1}}, {{"hand_L", 1}}, {{"hand_L", 1}}},
	}
	still := map[string][12]float64{"pelvis": [12]float64(identity34()), "hand_L": [12]float64(identity34())}
	pulled := map[string][12]float64{"pelvis": [12]float64(identity34()), "hand_L": {1, 0, 0, 5, 0, 1, 0, 0, 0, 0, 1, -10}}
	strain := r.Strain(tuneSkeleton(), []map[string][12]float64{still, pulled})
	if strain.WorstFrame != 1 || strain.Stretched < 0.5 || strain.Worst < 5 || strain.Bones["pelvis"]+strain.Bones["hand_L"] == 0 {
		t.Fatalf("pulled apart: %+v", strain)
	}
	if strain.BelowFloor != 0.5 {
		t.Fatalf("the hand's two vertices sink 10 units: %+v", strain)
	}
	calm := r.Strain(tuneSkeleton(), []map[string][12]float64{still})
	if calm.Stretched != 0 || calm.Worst > 1.0001 || calm.BelowFloor != 0 {
		t.Fatalf("at rest nothing stretches: %+v", calm)
	}
}

// Bound in the usual pose, the arms move with the chest they hang from, and the legs stay.
func TestTuneRigUsualPoseCarriesTheArmsNotTheLegs(t *testing.T) {
	s := &Skeleton{
		Names:  []string{"root_motion", "pelvis", "spine_3", "clavicle_L", "hand_L", "leg_upper_L"},
		Parent: []int{-1, 0, 1, 2, 3, 1},
		World:  []mat34{identity34(), identity34(), identity34(), identity34(), identity34(), identity34()},
		index:  map[string]int{},
	}
	for i, n := range s.Names {
		s.index[n] = i
	}
	lean := rotationAbout([3]float64{0, 1, 0}, 30, [3]float64{})
	r := &Rigged{Mesh: &ImportedMesh{Positions: [][3]float32{{0, 0, 10}, {0, 0, 10}, {0, 0, 10}}},
		Skin: [][]BoneWeight{{{"spine_3", 1}}, {{"hand_L", 1}}, {{"leg_upper_L", 1}}}}
	tuneRig(r, s, RigTuning{UsualPose: map[string][12]float64{"spine_3": [12]float64(lean)}})
	chest, hand, leg := r.Mesh.Positions[0], r.Mesh.Positions[1], r.Mesh.Positions[2]
	if chest != hand {
		t.Errorf("the hand did not follow the chest: %v %v", chest, hand)
	}
	if chest == ([3]float32{0, 0, 10}) {
		t.Error("the chest was not rebound")
	}
	if leg != ([3]float32{0, 0, 10}) {
		t.Errorf("the leg moved: %v", leg)
	}
}
