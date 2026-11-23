package source2

import (
	"encoding/binary"
	"hash/crc32"
	"math"
	"testing"
)

func kv3Resource(t *testing.T, root *Value) []byte {
	t.Helper()
	data, err := WriteKV3(root, FormatGeneric)
	if err != nil {
		t.Fatal(err)
	}
	return (&Resource{Blocks: []Block{{"DATA", data}}}).Encode()
}

func floatArray(values ...float64) *Value {
	a := NewArray()
	for _, v := range values {
		a.Items = append(a.Items, &Value{Kind: Float32, F: v})
	}
	return a
}

func entity(class string, origin ...float64) *Value {
	values := NewObject().Set("classname", NewString(class)).Set("scales", floatArray(1, 1, 1)).Set("origin", floatArray(origin...))
	return NewObject().Set("keyValues3Data", NewObject().Set("values", values))
}

func TestPanoramaTextRoundTrip(t *testing.T) {
	original := (&Resource{Blocks: []Block{{"RED2", []byte("x")}, {"DATA", []byte{0, 0, 0, 0, 0, 0, '<', 's', 'v', 'g', '/', '>'}}}}).Encode()
	text, err := PanoramaText(original)
	if err != nil || text != "<svg/>" {
		t.Fatalf("read %q, %v", text, err)
	}
	out, err := ReplacePanoramaText(original, "<svg>NAME</svg>")
	if err != nil {
		t.Fatal(err)
	}
	r, err := DecodeResource(out)
	if err != nil {
		t.Fatal(err)
	}
	data := r.Block("DATA")
	if string(data[6:]) != "<svg>NAME</svg>" || binary.LittleEndian.Uint32(data) != crc32.ChecksumIEEE([]byte("<svg>NAME</svg>")) {
		t.Fatalf("DATA is %q", data)
	}
	if string(r.Block("RED2")) != "x" {
		t.Fatal("other blocks were not kept")
	}
}

func TestHideHeroScene(t *testing.T) {
	ents := NewObject().Set("m_entityKeyValues", NewArray(
		entity("prop_dynamic", 1, 2, 3),
		entity("light_barn", 4, 5, 6),
		entity("prop_dynamic", 0, 0, 10),
	))
	identity := NewArray(floatArray(1, 0, 0, 7), floatArray(0, 1, 0, 8), floatArray(0, 0, 1, 9))
	node := NewObject().Set("m_sceneObjects", NewArray(NewObject().Set("m_vTransform", identity)))
	files := map[string][]byte{
		"maps/ui/hero_prefabs/x/entities/default_ents.vents_c": kv3Resource(t, ents),
		"maps/ui/hero_prefabs/x/worldnodes/n0.vwnod_c":         kv3Resource(t, node),
		"maps/ui/hero_prefabs/x/lightmaps/a.vtex_c":            []byte("untouched"),
	}
	props, objects, err := HideHeroScene(files)
	if err != nil || props != 2 || objects != 1 {
		t.Fatalf("hid %d props and %d objects, %v", props, objects, err)
	}
	read := func(name string) *Value {
		r, err := DecodeResource(files[name])
		if err != nil {
			t.Fatal(err)
		}
		v, _, err := ReadKV3(r.Block("DATA"))
		if err != nil {
			t.Fatal(err)
		}
		return v
	}
	list := read("maps/ui/hero_prefabs/x/entities/default_ents.vents_c").Get("m_entityKeyValues").Items
	hero := list[0].Get("keyValues3Data").Get("values")
	if s := hero.Get("scales").Items[0].F; s > 0.01 {
		t.Errorf("the hero kept scale %v", s)
	}
	if z := hero.Get("origin").Items[2].F; z != 3-hiddenDrop {
		t.Errorf("the hero is at z %v", z)
	}
	if light := list[1].Get("keyValues3Data").Get("values"); light.Get("scales").Items[0].F != 1 || light.Get("origin").Items[2].F != 6 {
		t.Error("the light moved")
	}
	m := read("maps/ui/hero_prefabs/x/worldnodes/n0.vwnod_c").Get("m_sceneObjects").Items[0].Get("m_vTransform").Items
	if math.Abs(m[0].Items[0].F-0.001) > 1e-6 || m[2].Items[3].F != -hiddenDrop || m[0].Items[3].F != 0 {
		t.Errorf("world object transform %v %v %v", m[0].Items[0].F, m[2].Items[3].F, m[0].Items[3].F)
	}
	if string(files["maps/ui/hero_prefabs/x/lightmaps/a.vtex_c"]) != "untouched" {
		t.Error("other files changed")
	}
}

// Framing keeps the prop showing the hero's model, resized and moved in its own turned frame,
// and hides the scene's other props and world meshes.
func TestFrameHeroScene(t *testing.T) {
	hero := entity("prop_dynamic", 10, 20, 0)
	values := hero.Get("keyValues3Data").Get("values")
	values.Set("model", NewString("models/heroes_wip/wraith/wraith.vmdl")).Set("angles", floatArray(0, 90, 0))
	ents := NewObject().Set("m_entityKeyValues", NewArray(hero, entity("prop_dynamic", 1, 2, 3), entity("light_barn", 4, 5, 6)))
	identity := NewArray(floatArray(1, 0, 0, 7), floatArray(0, 1, 0, 8), floatArray(0, 0, 1, 9))
	node := NewObject().Set("m_sceneObjects", NewArray(NewObject().Set("m_vTransform", identity)))
	files := map[string][]byte{
		"maps/ui/hero_prefabs/x/entities/default_ents.vents_c": kv3Resource(t, ents),
		"maps/ui/hero_prefabs/x/worldnodes/n0.vwnod_c":         kv3Resource(t, node),
	}
	n, err := FrameHeroScene(files, "models/heroes_wip/Wraith/wraith.vmdl", 0.5, [3]float64{4, 0, 2})
	if err != nil || n != 1 {
		t.Fatalf("framed %d, %v", n, err)
	}
	r, _ := DecodeResource(files["maps/ui/hero_prefabs/x/entities/default_ents.vents_c"])
	root, _, _ := ReadKV3(r.Block("DATA"))
	list := root.Get("m_entityKeyValues").Items
	framed := list[0].Get("keyValues3Data").Get("values")
	if s := framed.Get("scales").Items[0].F; math.Abs(s-0.5) > 1e-6 {
		t.Errorf("scale %v", s)
	}
	// The offset (4,0,2) in the prop's frame, turned 90° about Z, is (0,4,2) in the scene.
	o := framed.Get("origin").Items
	if math.Abs(o[0].F-10) > 1e-4 || math.Abs(o[1].F-24) > 1e-4 || math.Abs(o[2].F-2) > 1e-4 {
		t.Errorf("origin %v %v %v", o[0].F, o[1].F, o[2].F)
	}
	if other := list[1].Get("keyValues3Data").Get("values"); other.Get("scales").Items[0].F > 0.01 {
		t.Error("the scene's other props stay")
	}
	if light := list[2].Get("keyValues3Data").Get("values"); light.Get("origin").Items[2].F != 6 {
		t.Error("the light moved")
	}
	if _, err := FrameHeroScene(files, "models/other.vmdl", 1, [3]float64{}); err == nil {
		t.Error("a scene without the model cannot be framed")
	}
}

// A scene whose hero_root shows an older copy of the hero is pointed at the hero's model.
func TestFrameHeroSceneByName(t *testing.T) {
	hero := entity("prop_dynamic", 0, 0, 0)
	hero.Get("keyValues3Data").Get("values").Set("model", NewString("models/heroes_wip/yamato/yamato.vmdl")).Set("targetname", NewString("[PR#]hero_root"))
	files := map[string][]byte{"e.vents_c": kv3Resource(t, NewObject().Set("m_entityKeyValues", NewArray(hero)))}
	if n, err := FrameHeroScene(files, "models/heroes_staging/yamato_v2/yamato.vmdl", 1, [3]float64{}); err != nil || n != 1 {
		t.Fatalf("framed %d, %v", n, err)
	}
	r, _ := DecodeResource(files["e.vents_c"])
	root, _, _ := ReadKV3(r.Block("DATA"))
	if m := root.Get("m_entityKeyValues").Items[0].Get("keyValues3Data").Get("values").Get("model").S; m != "models/heroes_staging/yamato_v2/yamato.vmdl" {
		t.Fatalf("model %q", m)
	}
}
