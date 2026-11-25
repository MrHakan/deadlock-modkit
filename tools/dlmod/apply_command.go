package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"image/png"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/MrHakan/deadlock-modkit/tools/dlmod/source2"
)

// applyCommand installs what a hero's Meshy workspace folder holds (dlmod meshy writes it): each
// step whose file is in assets/ goes in, in the order model, hero select screen, pictures, ability
// icons, ability models. Cards drawn on green are keyed first; the rest is done by the commands
// that install each kind by hand (dlmod model, select, image, icons, prop).
func applyCommand(ui *UI, store *Store, game Game, args []string, opts options) error {
	if len(args) != 1 {
		return errors.New("name a hero's workspace folder, for example: dlmod apply meshy/ivy")
	}
	if gameRunning() {
		return errRunning
	}
	dir := args[0]
	w, err := readWorkspace(dir)
	if err != nil {
		return err
	}
	hero, err := findHero(w.Hero)
	if err != nil {
		return err
	}
	done, missing := 0, []string{}
	for _, s := range w.Steps {
		file := filepath.Join(dir, filepath.FromSlash(s.Output))
		if s.Kind != "" && !fileExists(file) {
			missing = append(missing, s.Output)
			continue
		}
		if s.Kind == "model" {
			ui.say("Model: %s", s.Output)
			if err := modelCommand(ui, store, game, []string{hero, file}, opts); err != nil {
				return err
			}
			done++
		}
	}
	pictures, err := installPictures(ui, store, game, w, dir)
	if err != nil {
		return err
	}
	done += pictures
	for _, s := range w.Steps {
		file := filepath.Join(dir, filepath.FromSlash(s.Output))
		if s.Kind != "prop" || !fileExists(file) {
			continue
		}
		if err := propCommand(ui, store, game, []string{s.Target, file}, opts); err != nil {
			return err
		}
		done++
	}
	ui.say("%s: %d pieces installed.", w.Name, done)
	if len(missing) > 0 {
		ui.say("Not generated yet: %s", strings.Join(missing, ", "))
	}
	return nil
}

func readWorkspace(dir string) (*workspace, error) {
	data, err := os.ReadFile(filepath.Join(dir, "workspace.json"))
	if err != nil {
		return nil, fmt.Errorf("%s has no workspace.json (make one with dlmod meshy <hero> <folder>): %v", dir, err)
	}
	var w workspace
	if err := json.Unmarshal(data, &w); err != nil {
		return nil, fmt.Errorf("workspace.json: %v", err)
	}
	// A workspace can come from someone else (a character pack): its steps may only name files
	// in its own assets/ and files of the game.
	for _, s := range w.Steps {
		if s.Output != "" && (!localPath(s.Output) || !strings.HasPrefix(s.Output, "assets/")) {
			return nil, fmt.Errorf("workspace.json: step %s writes %q, outside the folder's assets/", s.ID, s.Output)
		}
		if s.Target != "" && !localPath(s.Target) {
			return nil, fmt.Errorf("workspace.json: step %s replaces %q, which is not a game file", s.ID, s.Target)
		}
	}
	return &w, nil
}

// installPictures installs the hero select screen, the cards and portraits and the ability icons
// a workspace folder holds, and returns how many pieces went in. dlmod apply and the studio's
// character creator both use it.
func installPictures(ui prompter, store *Store, game Game, w *workspace, dir string) (int, error) {
	hero, err := findHero(w.Hero)
	if err != nil {
		return 0, err
	}
	done := 0
	var images, icons []string
	selectArgs := []string{hero}
	for _, s := range w.Steps {
		file := filepath.Join(dir, filepath.FromSlash(s.Output))
		if s.Kind == "" || !fileExists(file) {
			continue
		}
		switch s.Kind {
		case "select":
			selectArgs = append(selectArgs, "bg="+file)
		case "image":
			keyed, err := keyedCopy(file)
			if err != nil {
				return done, err
			}
			images = append(images, s.Target, keyed)
		case "icon":
			icons = append(icons, strings.TrimSuffix(path.Base(s.Target), "_psd.vtex_c")+"="+file)
		}
	}
	sel := w.Select
	if logo := filepath.Join(dir, "assets", "logo.svg"); fileExists(logo) {
		selectArgs = append(selectArgs, "logo="+logo) // a logo drawn by hand wins over the built-in font
	}
	if sel.Name != "" {
		selectArgs = append(selectArgs, "name="+sel.Name)
	}
	if w.Game != nil && len(sel.Tags) > 0 && !sameStrings(sel.Tags, w.Game.Tags) {
		selectArgs = append(selectArgs, "tags="+strings.Join(sel.Tags, ","))
	}
	if len(sel.Colors) > 0 {
		selectArgs = append(selectArgs, "colors="+strings.Join(sel.Colors, ","))
	}
	if len(selectArgs) > 1 {
		switch {
		case sel.Scene == "frame":
			selectArgs = append(selectArgs, "scene=frame")
		case sel.HideScene && hasPrefix(selectArgs, "bg="):
			selectArgs = append(selectArgs, "scene=hide")
		}
		ui.say("Hero select screen: %s", strings.Join(selectArgs[1:], " "))
		if err := selectCommand(ui, game, selectArgs); err != nil {
			return done, err
		}
		done++
	}
	if len(images) > 0 {
		if err := imageCommand(ui, store, game, images); err != nil {
			return done, err
		}
		done += len(images) / 2
	}
	if len(icons) > 0 {
		if err := iconsCommand(ui, store, game, append([]string{hero}, icons...)); err != nil {
			return done, err
		}
		done += len(icons)
	}
	return done, nil
}

// keyedCopy returns a picture drawn on green as a copy with the green keyed out (next to it,
// *_keyed.png), or the picture itself when it has no green background.
func keyedCopy(file string) (string, error) {
	data, err := os.ReadFile(file)
	if err != nil {
		return "", err
	}
	img, err := source2.DecodeImage(data)
	if err != nil {
		return "", fmt.Errorf("%s: %v", file, err)
	}
	if !hasGreenBackground(img) {
		return file, nil
	}
	out := strings.TrimSuffix(file, filepath.Ext(file)) + "_keyed.png"
	f, err := os.Create(out)
	if err != nil {
		return "", err
	}
	defer f.Close()
	return out, png.Encode(f, keyGreen(img))
}

func sameStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func hasPrefix(list []string, prefix string) bool {
	for _, s := range list {
		if strings.HasPrefix(s, prefix) {
			return true
		}
	}
	return false
}

func fileExists(name string) bool {
	_, err := os.Stat(name)
	return err == nil
}
