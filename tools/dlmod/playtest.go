package main

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// A quick test in the game: dlmod play <hero> starts Deadlock in the Hero Sandbox
// (new_player_basics) with the hero picked, so a new model, pictures or effects can be seen
// without queueing for a match. The game reads cfg/dlmod_test.cfg, which dlmod writes: it turns
// on the sandbox's testing tools (citadel_hero_testing_enabled), which let selecthero change the
// hero outside the hero select screen, and picks the hero.

const (
	testMap    = "new_player_basics"
	testConfig = "dlmod_test" // cfg/dlmod_test.cfg, in the game's citadel folder
)

// testCommands are what dlmod_test.cfg runs.
func testCommands(hero string) string {
	return strings.Join([]string{
		"// Written by dlmod play " + hero + ": the sandbox's testing tools, and the hero to test.",
		"citadel_hero_testing_enabled 1",
		"selecthero hero_" + hero,
		"",
	}, "\n")
}

// testLaunchOptions start the sandbox and run the test config once it is loading.
func testLaunchOptions() string {
	return "-novid +citadel_hero_testing_enabled 1 +map " + testMap + " +exec " + testConfig
}

// writeTestConfig writes cfg/dlmod_test.cfg for a hero.
func writeTestConfig(game Game, hero string) (string, error) {
	file := filepath.Join(game.Citadel(), "cfg", testConfig+".cfg")
	if err := os.MkdirAll(filepath.Dir(file), 0o755); err != nil {
		return "", err
	}
	return file, os.WriteFile(file, []byte(testCommands(hero)), 0o644)
}

// playTest patches the game if needed and starts the sandbox with the hero. Steam asks before it
// starts a game with launch options.
func playTest(p prompter, game Game, hero string) error {
	if gameRunning() {
		return errRunning
	}
	hero, err := findHero(hero)
	if err != nil {
		return err
	}
	if err := ensurePatched(p, game); err != nil {
		return err
	}
	if _, err := writeTestConfig(game, hero); err != nil {
		return err
	}
	p.say("Starting Deadlock in the Hero Sandbox as %s; Steam asks before it uses the launch options.", heroDisplayName(hero))
	p.say("If the hero is not %s once you are in, open the console (F7) and type: exec %s", heroDisplayName(hero), testConfig)
	return launchWith(testLaunchOptions())
}

// launchWith starts the game with launch options; tests replace it.
var launchWith = launchGameWith

// launchGameWith starts Deadlock through Steam with launch options (steam://run/<app>//<options>).
func launchGameWith(options string) error {
	return startURL(fmt.Sprintf("steam://run/%s//%s/", appID, strings.ReplaceAll(options, " ", "%20")))
}

// playTest is the studio's "Test in the game".
func (s *studio) playTest(r *http.Request) (any, error) {
	var request struct {
		Hero string `json:"hero"`
	}
	if err := readJSON(r, &request); err != nil {
		return nil, err
	}
	p := &webPrompter{}
	if err := playTest(p, s.game, request.Hero); err != nil {
		return nil, err
	}
	return map[string]any{"messages": p.messages}, nil
}
