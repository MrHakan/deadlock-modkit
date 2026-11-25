package main

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/MrHakan/deadlock-modkit/tools/dlmod/source2"
)

// vdataFile compiles a KV3 object into a .vdata_c resource.
func vdataFile(t *testing.T, root *source2.Value) []byte {
	t.Helper()
	data, err := source2.WriteKV3(root, source2.FormatGeneric)
	if err != nil {
		t.Fatal(err)
	}
	return (&source2.Resource{Version: 1, Blocks: []source2.Block{{Type: "DATA", Data: data}}}).Encode()
}

func flagged(s string, flag byte) *source2.Value {
	v := source2.NewString(s)
	v.Flag = flag
	return v
}

func TestGameDataReadsHeroesAndAbilities(t *testing.T) {
	slots := source2.NewObject()
	for i, key := range []string{"ability_one", "ability_two", "ability_three", "ability_four"} {
		slots.Set("ESlot_Signature_"+string(rune('1'+i)), source2.NewString(key))
	}
	slots.Set("ESlot_Weapon_Melee", source2.NewString("ability_melee"))
	heroes := source2.NewObject().
		Set("hero_base", source2.NewObject()).
		Set("hero_rook", source2.NewObject().Set("m_mapBoundAbilities", slots))
	cooldown := source2.NewObject().Set("m_strValue", source2.NewString("12")).Set("m_eDisplayUnits", source2.NewString("EDisplayUnit_Seconds"))
	unused := source2.NewObject().Set("m_strValue", source2.NewString("0"))
	one := source2.NewObject().
		Set("m_strAbilityImage", flagged("file://{images}/hud/abilities/rook/rook_hammer.psd", 3)).
		Set("m_ExplodeParticle", flagged("particles/abilities/rook/hammer_explode.vpcf", 2)).
		Set("m_mapWeaponInfos", source2.NewObject().Set("m_szTracer", flagged("particles/abilities/rook/hammer_trail.vpcf", 2))).
		Set("m_strProjectileModel", flagged("models/abilities/rook_hammer.vmdl", 2)).
		Set("m_strCastSound", flagged("Rook.Hammer.Cast", 4)).
		Set("m_mapAbilityProperties", source2.NewObject().Set("AbilityCooldown", cooldown).Set("AbilityUnused", unused))
	abilities := source2.NewObject().Set("ability_one", one).Set("ability_two", source2.NewObject())
	const icon = "panorama/images/hud/abilities/rook/rook_hammer_psd.vtex_c"
	dir := t.TempDir()
	vpk := filepath.Join(dir, "pak01_dir.vpk")
	if err := os.WriteFile(vpk, source2.WriteVPK(map[string][]byte{
		heroesVdata:    vdataFile(t, heroes),
		abilitiesVdata: vdataFile(t, abilities),
		icon:           []byte("icon"),
	}), 0o644); err != nil {
		t.Fatal(err)
	}
	index, err := source2.OpenVPK(vpk)
	if err != nil {
		t.Fatal(err)
	}
	data, err := loadGameData(index)
	if err != nil {
		t.Fatal(err)
	}
	if got := data.heroes["rook"]; got != [4]string{"ability_one", "ability_two", "ability_three", "ability_four"} {
		t.Fatalf("slots %v", got)
	}
	if _, ok := data.heroes["base"]; ok {
		t.Fatal("hero_base has no abilities and must be left out")
	}
	want := &gameAbility{
		Key:       "ability_one",
		Icon:      icon,
		Particles: []string{"particles/abilities/rook/hammer_explode.vpcf", "particles/abilities/rook/hammer_trail.vpcf"},
		Models:    []string{"models/abilities/rook_hammer.vmdl"},
		Sounds:    []string{"Rook.Hammer.Cast"},
		Values:    []abilityValue{{Name: "AbilityCooldown", Value: "12", Units: "Seconds"}},
	}
	if got := data.abilities["ability_one"]; !reflect.DeepEqual(got, want) {
		t.Fatalf("ability\n got %+v\nwant %+v", got, want)
	}
	if again, _ := loadGameData(index); again != data {
		t.Fatal("the data must be read once per index")
	}

	keys, icons, _, source := heroAbilityKeys(index, "rook")
	if source != abilitiesFromGame || keys[0] != "ability_one" || icons[0] != icon || icons[1] != "" {
		t.Fatalf("from the game: %v %v %q", keys, icons, source)
	}
	// A hero the game's data does not know falls back to the table.
	keys, icons, names, source := heroAbilityKeys(index, "haze")
	if source != abilitiesFromTable || keys[0] != "ability_sleep_dagger" || icons[0] != "haze_sleep_dagger" || names[0] != "Sleep Dagger" {
		t.Fatalf("from the table: %v %v %v %q", keys, icons, names, source)
	}
}

func TestCompiledPanoramaImage(t *testing.T) {
	for in, want := range map[string]string{
		"file://{images}/hud/abilities/tengu/tengu_storm_flask.psd": "panorama/images/hud/abilities/tengu/tengu_storm_flask_psd.vtex_c",
		"file://{images}/heroes/haze_card.png":                      "panorama/images/heroes/haze_card_png.vtex_c",
		"s2r://panorama/images/x.vtex":                              "",
		"file://{images}/noextension":                               "",
	} {
		if got := compiledPanoramaImage(in); got != want {
			t.Errorf("%q: got %q, want %q", in, got, want)
		}
	}
}
