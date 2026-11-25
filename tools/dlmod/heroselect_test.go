package main

import (
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/MrHakan/deadlock-modkit/tools/dlmod/source2"
)

func TestSceneFrame(t *testing.T) {
	// A model twice as wide as the hero, standing 5 units lower and off to one side.
	scale, offset := sceneFrame([3]float64{-10, -10, 0}, [3]float64{10, 10, 80}, [3]float64{-10, -20, -5}, [3]float64{30, 20, 75})
	if math.Abs(scale-0.5) > 1e-9 { // 20 wide against 40 wide
		t.Fatalf("scale %v", scale)
	}
	// Its centre (10, 0) scaled lands on the hero's (0, 0); its feet on the hero's.
	if math.Abs(offset[0]+5) > 1e-9 || math.Abs(offset[1]) > 1e-9 || math.Abs(offset[2]-2.5) > 1e-9 {
		t.Fatalf("offset %v", offset)
	}
	if scale, _ := sceneFrame([3]float64{}, [3]float64{10, 10, 80}, [3]float64{}, [3]float64{1, 1, 1}); scale != 2.5 {
		t.Fatalf("a tiny model is enlarged at most 2.5 times: %v", scale)
	}
}

// scene=frame keeps the hero's prop in the hero select scene for the installed model, hides the
// hero's decor, and needs a model installed.
func TestSelectFramesTheInstalledModel(t *testing.T) {
	game := newFakeGame(t)
	model := heroModels["wraith"]
	game.write("game/citadel/pak01_dir.vpk", source2.WriteVPK(map[string][]byte{model + "_c": source2.SampleHero()}))
	prop := func(model string) *source2.Value {
		values := source2.NewObject().Set("classname", source2.NewString("prop_dynamic")).Set("model", source2.NewString(model))
		values.Set("origin", source2.NewArray(source2.NewFloat32(0), source2.NewFloat32(0), source2.NewFloat32(0)))
		values.Set("angles", source2.NewArray(source2.NewFloat32(0), source2.NewFloat32(291), source2.NewFloat32(0)))
		values.Set("scales", source2.NewArray(source2.NewFloat32(1), source2.NewFloat32(1), source2.NewFloat32(1)))
		return source2.NewObject().Set("keyValues3Data", source2.NewObject().Set("values", values))
	}
	ents := source2.NewObject().Set("m_entityKeyValues", source2.NewArray(prop(model), prop("models/heroes_wip/wraith/wraith_card_fx_model.vmdl")))
	game.write("game/citadel/maps/ui/hero_prefabs/wraith.vpk", source2.WriteVPK(map[string][]byte{
		"maps/ui/hero_prefabs/wraith/entities/default_ents.vents_c": vdataFile(t, ents),
	}))

	err := selectCommand(quietUI(""), game.Game, []string{"wraith", "scene=frame"})
	if err == nil || !strings.Contains(err.Error(), "no switched-on mod replaces") {
		t.Fatalf("framing without a model installed: %v", err)
	}
	store, _ := openStore(game.Game)
	if err := store.install(quietUI(""), game.write("downloads/model.vpk", source2.WriteVPK(map[string][]byte{model + "_c": source2.SampleHero()})), "Custom model: Wraith"); err != nil {
		t.Fatal(err)
	}
	if err := selectCommand(quietUI(""), game.Game, []string{"wraith", "scene=frame"}); err != nil {
		t.Fatal(err)
	}
	data, err := game.installed(t, heroSelectName).Read("maps/ui/hero_prefabs/wraith.vpk")
	if err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(t.TempDir(), "wraith.vpk")
	os.WriteFile(file, data, 0o644)
	scene, err := source2.OpenVPK(file)
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := scene.Read("maps/ui/hero_prefabs/wraith/entities/default_ents.vents_c")
	r, _ := source2.DecodeResource(raw)
	root, _, _ := source2.ReadKV3(r.Block("DATA"))
	list := root.Get("m_entityKeyValues").Items
	hero, decor := list[0].Get("keyValues3Data").Get("values"), list[1].Get("keyValues3Data").Get("values")
	if s := hero.Get("scales").Items[0].F; math.Abs(s-1) > 1e-6 { // the same model fits as it is
		t.Errorf("the hero's prop was scaled %v", s)
	}
	if s := decor.Get("scales").Items[0].F; s > 0.01 {
		t.Errorf("the hero's decor stays at scale %v", s)
	}
}
