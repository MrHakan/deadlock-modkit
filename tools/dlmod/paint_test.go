package main

import (
	"bytes"
	"image/color"
	"os"
	"path/filepath"
	"testing"

	"github.com/MrHakan/deadlock-modkit/tools/dlmod/source2"
)

func makeTestMaterial(t *testing.T, name, colorTex string, selfIllum [4]float32) []byte {
	t.Helper()
	mat, err := source2.CompileMaterial(name, colorTex, 4, false)
	if err != nil {
		t.Fatal(err)
	}
	if selfIllum == ([4]float32{}) {
		return mat
	}
	r, err := source2.DecodeResource(mat)
	if err != nil {
		t.Fatal(err)
	}
	v, format, err := source2.ReadKV3(r.Block("DATA"))
	if err != nil {
		t.Fatal(err)
	}
	vp := v.Get("m_vectorParams")
	vp.Items = append(vp.Items, source2.NewObject().
		Set("m_name", source2.NewString("g_vSelfIllumTint1")).
		Set("m_value", source2.NewArray(
			source2.NewFloat32(selfIllum[0]),
			source2.NewFloat32(selfIllum[1]),
			source2.NewFloat32(selfIllum[2]),
			source2.NewFloat32(selfIllum[3]),
		)))
	enc, err := source2.WriteKV3(v, format)
	if err != nil {
		t.Fatal(err)
	}
	for i := range r.Blocks {
		if r.Blocks[i].Type == "DATA" {
			r.Blocks[i].Data = enc
		}
	}
	return r.Encode()
}

func makeTestParticleWithRef(t *testing.T, ref string, col [3]int64) []byte {
	t.Helper()
	data := source2.NewObject().
		Set("m_ConstantColor", source2.NewArray(
			source2.NewInt32(int32(col[0])),
			source2.NewInt32(int32(col[1])),
			source2.NewInt32(int32(col[2])),
			source2.NewInt32(255),
		)).
		Set("m_hTexture", source2.NewString(ref))
	dataBlock, err := source2.WriteKV3(data, source2.FormatGeneric)
	if err != nil {
		t.Fatal(err)
	}
	refs := []source2.Reference{{ID: source2.ResourceID(ref), Name: ref}}
	return (&source2.Resource{
		Version: 1,
		Blocks: []source2.Block{
			{Type: "RERL", Data: source2.WriteReferences(refs)},
			{Type: "DATA", Data: dataBlock},
		},
	}).Encode()
}

func makeTestModelWithMat(t *testing.T, mat string) []byte {
	t.Helper()
	refs := []source2.Reference{{ID: source2.ResourceID(mat), Name: mat}}
	return (&source2.Resource{
		Version: 1,
		Blocks: []source2.Block{
			{Type: "RERL", Data: source2.WriteReferences(refs)},
		},
	}).Encode()
}

func TestPaintCommandAndRestore(t *testing.T) {
	greenTex, err := source2.CompileTexture(source2.SolidImage([4]float32{0.35, 0.72, 0.12, 1}), 16, [4]float32{1, 1, 1, 1})
	if err != nil {
		t.Fatal(err)
	}
	grayTex, err := source2.CompileTexture(source2.SolidImage([4]float32{0.6, 0.6, 0.6, 1}), 16, [4]float32{1, 1, 1, 1})
	if err != nil {
		t.Fatal(err)
	}
	sharedTex, err := source2.CompileTexture(source2.SolidImage([4]float32{0.3, 0.8, 0.2, 1}), 16, [4]float32{1, 1, 1, 1})
	if err != nil {
		t.Fatal(err)
	}
	brambleMat := makeTestMaterial(t, "models/particle/materials/ivy_bramble_fx.vmat", "materials/models/particle/ivy/ivy_bramble_color.vtex", [4]float32{0.3, 0.8, 0.2, 1})
	bombMat := makeTestMaterial(t, "models/abilities/materials/ivy_thornbomb.vmat", "models/abilities/materials/ivy_thornbomb_color.vtex", [4]float32{})

	vpkFiles := map[string][]byte{
		"particles/abilities/tengu/tengu_vines.vpcf_c":           makeTestParticleWithRef(t, "models/abilities/ivy_bramble_model_fx.vmdl", [3]int64{80, 180, 40}),
		"particles/abilities/tengu/tengu_bomb.vpcf_c":            makeTestParticleWithRef(t, "models/abilities/ivy_thornbomb_model.vmdl", [3]int64{90, 170, 50}),
		"particles/abilities/tengu/tengu_creeper.vpcf_c":         makeTestParticleWithRef(t, "materials/particle/beams/beam_creeper.vtex", [3]int64{100, 190, 60}),
		"particles/abilities/tengu/tengu_gray.vpcf_c":            makeTestParticleWithRef(t, "materials/particle/beams/beam_gray.vtex", [3]int64{100, 190, 60}),
		"particles/abilities/tengu/tengu_shared.vpcf_c":          makeTestParticleWithRef(t, "materials/particle/beams/beam_shared.vtex", [3]int64{100, 190, 60}),
		"particles/abilities/haze/haze_shared.vpcf_c":            makeTestParticleWithRef(t, "materials/particle/beams/beam_shared.vtex", [3]int64{100, 190, 60}),
		"models/abilities/ivy_bramble_model_fx.vmdl_c":           makeTestModelWithMat(t, "models/particle/materials/ivy_bramble_fx.vmat"),
		"models/abilities/ivy_thornbomb_model.vmdl_c":            makeTestModelWithMat(t, "models/abilities/materials/ivy_thornbomb.vmat"),
		"models/particle/materials/ivy_bramble_fx.vmat_c":        brambleMat,
		"models/abilities/materials/ivy_thornbomb.vmat_c":        bombMat,
		"materials/models/particle/ivy/ivy_bramble_color.vtex_c": greenTex,
		"models/abilities/materials/ivy_thornbomb_color.vtex_c":  greenTex,
		"materials/particle/beams/beam_creeper.vtex_c":           greenTex,
		"materials/particle/beams/beam_gray.vtex_c":              grayTex,
		"materials/particle/beams/beam_shared.vtex_c":            sharedTex,
	}

	game, _, _ := studioServer(t, vpkFiles)
	store, err := openStore(game.Game)
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	ui := &UI{out: &buf}

	// Run paint with -skip ivy_thornbomb_model
	if err := paintCommand(ui, store, game.Game, []string{"ivy"}, options{skip: "ivy_thornbomb_model"}); err != nil {
		t.Fatalf("paintCommand: %v\n%s", err, buf.String())
	}

	vpk := game.installed(t, customEffectsName)
	// Repainted: bramble texture, bramble material, creeper texture, tengu particles.
	for _, want := range []string{
		"particles/abilities/tengu/tengu_vines.vpcf_c",
		"models/particle/materials/ivy_bramble_fx.vmat_c",
		"materials/models/particle/ivy/ivy_bramble_color.vtex_c",
		"materials/particle/beams/beam_creeper.vtex_c",
	} {
		if _, ok := vpk.Files[want]; !ok {
			t.Errorf("expected %s in installed mod, files=%v", want, vpk.Files)
		}
	}
	// Skipped: thornbomb (via -skip), beam_gray (monochrome), beam_shared (shared with haze).
	for _, unwanted := range []string{
		"models/abilities/materials/ivy_thornbomb_color.vtex_c",
		"materials/particle/beams/beam_gray.vtex_c",
		"materials/particle/beams/beam_shared.vtex_c",
	} {
		if _, ok := vpk.Files[unwanted]; ok {
			t.Errorf("did not expect %s in installed mod", unwanted)
		}
	}
	// Verify the repainted bramble texture decoded back is cobalt blue (B > G, B > R).
	rawTex, err := vpk.Read("materials/models/particle/ivy/ivy_bramble_color.vtex_c")
	if err != nil {
		t.Fatal(err)
	}
	img, _, err := source2.DecodeTexture(rawTex)
	if err != nil {
		t.Fatal(err)
	}
	px := img.At(1, 1).(color.NRGBA)
	if px.B <= px.G || px.B <= px.R {
		t.Fatalf("repainted bramble pixel = %v, want cobalt blue", px)
	}

	// Restore removes all painted files and the mod.
	if err := paintCommand(ui, store, game.Game, []string{"ivy", "restore"}, options{}); err != nil {
		t.Fatalf("paint restore: %v", err)
	}
	if game.hasMod(t, customEffectsName) {
		t.Fatal("expected Custom effects mod to be removed after restore")
	}
}

func TestEffectsCommandPaletteMode(t *testing.T) {
	game, _, _ := studioServer(t, map[string][]byte{
		"particles/abilities/tengu/tengu_test.vpcf_c": makeTestParticleWithRef(t, "materials/particle/beams/beam_vine.vtex", [3]int64{80, 180, 40}),
	})
	store, err := openStore(game.Game)
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	ui := &UI{out: &buf}
	if err := effectsCommand(ui, store, game.Game, []string{"ivy", "palette", "cobalt"}); err != nil {
		t.Fatalf("effects palette: %v", err)
	}
	vpk := game.installed(t, customEffectsName)
	raw, err := vpk.Read("particles/abilities/tengu/tengu_test.vpcf_c")
	if err != nil {
		t.Fatal(err)
	}
	cols, err := source2.ParticleColors(raw)
	if err != nil || len(cols) != 1 {
		t.Fatalf("colors = %v, err = %v", cols, err)
	}
	if cols[0][2] <= cols[0][1] || cols[0][2] <= cols[0][0] {
		t.Fatalf("expected green particle colour to map to cobalt blue, got %v", cols[0])
	}
	if _, err := os.Stat(filepath.Join(game.Game.effectsDir(), "settings.json")); err != nil {
		t.Fatalf("settings.json missing: %v", err)
	}
}

func TestPaletteFromColors(t *testing.T) {
	p, err := LookupPalette("#2467FF, #FF3B3B ,#7AB6FF")
	if err != nil || p.Primary != [3]float64{0x24, 0x67, 0xFF} || p.Accent != [3]float64{0xFF, 0x3B, 0x3B} || p.Glow != [3]float64{0x7A, 0xB6, 0xFF} {
		t.Fatalf("three colours: %+v %v", p, err)
	}
	one, err := LookupPalette("#112233")
	if err != nil || one.Accent != one.Primary || one.Glow[0] <= one.Primary[0] {
		t.Fatalf("one colour: the accent repeats it and the glow is lighter: %+v %v", one, err)
	}
	for _, bad := range []string{"#12345", "#112233,#zzzzzz", "#1,#2,#3,#4", "neon"} {
		if _, err := LookupPalette(bad); err == nil {
			t.Errorf("%q was accepted", bad)
		}
	}
}
