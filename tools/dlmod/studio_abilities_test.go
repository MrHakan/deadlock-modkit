package main

import (
	"image/png"
	"net/http"
	"reflect"
	"testing"

	"github.com/MrHakan/deadlock-modkit/tools/dlmod/source2"
)

// The Abilities tab shows each ability from the game's data with its icon, effect colours and
// model, and once a mod recolours an effect, the colours the game will show.
func TestStudioAbilities(t *testing.T) {
	icon, err := source2.CompileTexture(source2.SolidImage([4]float32{1, 1, 1, 1}), 16, [4]float32{1, 1, 1, 1})
	if err != nil {
		t.Fatal(err)
	}
	slots := source2.NewObject()
	for i, key := range []string{"ability_card_trick", "ability_two", "ability_three", "ability_four"} {
		slots.Set("ESlot_Signature_"+string(rune('1'+i)), source2.NewString(key))
	}
	cooldown := source2.NewObject().Set("m_strValue", source2.NewString("12")).Set("m_eDisplayUnits", source2.NewString("EDisplayUnit_Seconds"))
	unused := source2.NewObject().Set("m_strValue", source2.NewString("-1"))
	card := source2.NewObject().
		Set("m_strAbilityImage", flagged("file://{images}/hud/abilities/wraith/card_trick.psd", 3)).
		Set("m_ExplodeParticle", flagged("particles/abilities/wraith/card_trick.vpcf", 2)).
		Set("m_strModel", flagged("models/abilities/wraith_card.vmdl", 2)).
		Set("m_strCastSound", flagged("Wraith.Card.Cast", 4)).
		Set("m_mapAbilityProperties", source2.NewObject().Set("AbilityCooldown", cooldown).Set("AbilityCastRange", unused))
	game, post, base := studioServer(t, map[string][]byte{
		heroesVdata:    vdataFile(t, source2.NewObject().Set("hero_wraith", source2.NewObject().Set("m_mapBoundAbilities", slots))),
		abilitiesVdata: vdataFile(t, source2.NewObject().Set("ability_card_trick", card)),
		"particles/abilities/wraith/card_trick.vpcf_c":               source2.SampleParticle(),
		"panorama/images/hud/abilities/wraith/card_trick_psd.vtex_c": icon,
		"models/abilities/wraith_card.vmdl_c":                        source2.SampleHero(),
	})
	type listing struct {
		Abilities []abilityView
		Error     string
	}
	var got listing
	if code := post("abilities", "application/json", []byte(`{"hero":"wraith"}`), &got); code != 200 || len(got.Abilities) != 4 {
		t.Fatalf("abilities: %d %+v", code, got)
	}
	first := got.Abilities[0]
	if first.Key != "ability_card_trick" || first.Icon != "panorama/images/hud/abilities/wraith/card_trick_psd.vtex_c" || first.IconMod != "" {
		t.Fatalf("first ability: %+v", first)
	}
	if !reflect.DeepEqual(first.Values, []map[string]string{{"label": "Cooldown", "value": "12s"}}) {
		t.Fatalf("values (an unused -1 is left out): %v", first.Values)
	}
	if first.Particles != 1 || first.Recoloured != 0 || len(first.Before) != 3 || !reflect.DeepEqual(first.Before, first.After) {
		t.Fatalf("effects before any mod: %+v", first)
	}
	if !reflect.DeepEqual(first.Models, []abilityModel{{Path: "models/abilities/wraith_card.vmdl"}}) || !reflect.DeepEqual(first.Sounds, []string{"Wraith.Card.Cast"}) {
		t.Fatalf("models and sounds: %+v %+v", first.Models, first.Sounds)
	}

	var applied struct{ Error string }
	if code := post("effects/apply", "application/json", []byte(`{"hero":"wraith","settings":{"hue":120,"saturation":1,"brightness":1}}`), &applied); code != 200 {
		t.Fatalf("apply: %d %+v", code, applied)
	}
	post("abilities", "application/json", []byte(`{"hero":"wraith"}`), &got)
	first = got.Abilities[0]
	if first.Recoloured != 1 || !reflect.DeepEqual(first.EffectMods, []string{customEffectsName}) || reflect.DeepEqual(first.Before, first.After) {
		t.Fatalf("effects with the mod on: %+v", first)
	}

	for _, address := range []string{
		"/api/abilities/icon?path=panorama/images/hud/abilities/wraith/card_trick_psd.vtex_c&which=game&t=secret",
		"/api/abilities/model?path=models/abilities/wraith_card.vmdl&which=game&yaw=30&t=secret",
	} {
		response, err := http.Get(base + address)
		if err != nil {
			t.Fatal(err)
		}
		_, decodeErr := png.Decode(response.Body)
		response.Body.Close()
		if response.StatusCode != 200 || decodeErr != nil {
			t.Fatalf("%s: %d %v", address, response.StatusCode, decodeErr)
		}
	}
	response, _ := http.Get(base + "/api/abilities/model?path=models/abilities/wraith_card.vmdl&which=yours&t=secret")
	if response.StatusCode == 200 {
		t.Fatal("no mod replaces the model, so there is no picture of yours")
	}
	response.Body.Close()
	_ = game
}

// When two switched-on mods replace one file, the game takes it from the lower slot.
func TestInstalledFilesLowestSlotWins(t *testing.T) {
	game := newFakeGame(t)
	store, _ := openStore(game.Game)
	for _, name := range []string{"First", "Second"} { // different files, so the second is not the same mod again
		file := game.write("downloads/"+name+".vpk", buildVPK(2, "sounds/shared.vsnd_c", "sounds/"+name+".vsnd_c"))
		if err := store.install(quietUI(""), file, name); err != nil {
			t.Fatal(err)
		}
	}
	files, err := installedFiles(game.Game)
	if err != nil {
		t.Fatal(err)
	}
	if got := files["sounds/shared.vsnd_c"].mod; got != "First" {
		t.Fatalf("got %q", got)
	}
	entries, _ := store.entries()
	for _, e := range entries {
		if e.Name() == "First" {
			if err := store.disable(e); err != nil {
				t.Fatal(err)
			}
		}
	}
	files, _ = installedFiles(game.Game)
	if got := files["sounds/shared.vsnd_c"].mod; got != "Second" {
		t.Fatalf("with the first switched off: %q", got)
	}
}

func TestAbilityValueText(t *testing.T) {
	for name, want := range map[string]string{"AbilityCooldown": "Cooldown", "AbilityCastRange": "Cast range", "Radius": "Radius", "TechDPS": "Tech DPS"} {
		if got := valueLabel(name); got != want {
			t.Errorf("valueLabel(%q) = %q, want %q", name, got, want)
		}
	}
	for _, c := range []struct {
		v    abilityValue
		want string
	}{{abilityValue{Value: "12", Units: "Seconds"}, "12s"}, {abilityValue{Value: "6m", Units: "Meters"}, "6m"}, {abilityValue{Value: "30", Units: "Percent"}, "30%"}, {abilityValue{Value: "2"}, "2"}} {
		if got := valueText(c.v); got != c.want {
			t.Errorf("valueText(%+v) = %q, want %q", c.v, got, c.want)
		}
	}
	for value, want := range map[string]bool{"12": true, "0": false, "0m": false, "-1": false, "-1.0": false, "0.2": true, "": false, "yes": true} {
		if got := shownValue(value); got != want {
			t.Errorf("shownValue(%q) = %v", value, got)
		}
	}
}
