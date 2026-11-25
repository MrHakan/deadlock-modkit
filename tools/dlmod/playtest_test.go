package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPlayTestStartsTheSandboxWithTheHero(t *testing.T) {
	game := newFakeGame(t)
	var launched []string
	saved := launchWith
	launchWith = func(options string) error { launched = append(launched, options); return nil }
	defer func() { launchWith = saved }()

	if err := playTest(quietUI(""), game.Game, "nobody"); err == nil || len(launched) != 0 {
		t.Fatalf("an unknown hero: %v %v", err, launched)
	}
	if err := playTest(quietUI(""), game.Game, "haze"); err != nil {
		t.Fatal(err)
	}
	cfg, err := os.ReadFile(filepath.Join(game.Citadel(), "cfg", "dlmod_test.cfg"))
	if err != nil || !strings.Contains(string(cfg), "citadel_hero_testing_enabled 1") || !strings.Contains(string(cfg), "selecthero hero_haze") {
		t.Fatalf("cfg %q %v", cfg, err)
	}
	if len(launched) != 1 || !strings.Contains(launched[0], "+map new_player_basics") || !strings.Contains(launched[0], "+exec dlmod_test") {
		t.Fatalf("launched %v", launched)
	}
	if info, _ := os.ReadFile(game.GameInfo()); !strings.Contains(string(info), "citadel/addons") {
		t.Fatal("the game must load mods when it starts")
	}
}
