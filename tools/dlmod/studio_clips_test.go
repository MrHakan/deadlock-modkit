package main

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/MrHakan/deadlock-modkit/tools/dlmod/source2"
)

// Heroes keep their clips in clips/, clip/ or dmx/animation/ag2/ next to the skeleton; dlmod
// takes the folder with the most, and leaves out the derived non-additive copies.
func TestHeroClipFolderAndList(t *testing.T) {
	files := map[string][]byte{}
	for _, name := range []string{
		"models/heroes_staging/haze/clips/weapon_run_n.vnmclip_c",
		"models/heroes_staging/haze/clips/melee_quick_1.vnmclip_c",
		"models/heroes_staging/haze/clips/aim_idle.vnmclip_c",
		"models/heroes_staging/haze/clips/aim_idle.vnmclip+non_additive.vnmclip_c",
		"models/heroes_staging/haze/old/one.vnmclip_c",
		"models/heroes_staging/mirage_v2/clips/a.vnmclip_c",
		"models/heroes_staging/mirage_v2/clips/b.vnmclip_c",
		"models/heroes_staging/mirage_v2/dmx/animation/ag2/a.vnmclip_c",
	} {
		files[name] = []byte("x")
	}
	vpk := filepath.Join(t.TempDir(), "pak01_dir.vpk")
	if err := os.WriteFile(vpk, source2.WriteVPK(files), 0o644); err != nil {
		t.Fatal(err)
	}
	index, err := source2.OpenVPK(vpk)
	if err != nil {
		t.Fatal(err)
	}
	if got := heroClipFolder(index, "models/heroes_staging/haze/haze.vnmskel"); got != "models/heroes_staging/haze/clips/" {
		t.Errorf("haze: %q", got)
	}
	if got := heroClipFolder(index, "models/heroes_staging/mirage_v2/dmx/mirage.vnmskel"); got != "models/heroes_staging/mirage_v2/clips/" {
		t.Errorf("mirage: %q", got)
	}
	if got := heroClips(index, "models/heroes_staging/haze/clips/"); !reflect.DeepEqual(got, []string{"weapon_run_n", "melee_quick_1", "aim_idle"}) {
		t.Errorf("clips, the clips worth knowing first: %v", got)
	}
	masks := []string{"Cast_UpperBody", "Reload_Idle", "melee_UpperBody"}
	for clip, want := range map[string]string{"melee_quick_1": "melee_UpperBody", "reload_stand": "Reload_Idle", "ability_sleep_dagger": "Cast_UpperBody", "weapon_run_n": "", "throw_grenade": ""} {
		if got := suggestedMask(clip, masks); got != want {
			t.Errorf("suggestedMask(%q) = %q, want %q", clip, got, want)
		}
	}
}

// The rig check plays the movement, standing, a melee and a reload over the movement, a cast and
// a jump, each through the mask the game would use.
func TestRigCheckClips(t *testing.T) {
	clips := []string{"weapon_run_n", "weapon_idle", "melee_hit", "melee_quick_1", "reload_crouch_idle", "reload_stand_idle", "ability_sleep_dagger", "jump_forced"}
	masks := []string{"melee_UpperBody", "Reload_Idle", "Cast_UpperBody"}
	got := rigCheckClips(clips, masks)
	want := []checkClip{
		{Clip: "weapon_run_n"}, {Clip: "weapon_idle"},
		{Clip: "melee_quick_1", Over: "weapon_run_n", Mask: "melee_UpperBody"},
		{Clip: "reload_stand_idle", Over: "weapon_idle", Mask: "Reload_Idle"},
		{Clip: "ability_sleep_dagger", Over: "weapon_idle", Mask: "Cast_UpperBody"},
		{Clip: "jump_forced"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v\nwant %+v", got, want)
	}
}
