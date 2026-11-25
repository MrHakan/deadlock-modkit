package main

import (
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/MrHakan/deadlock-modkit/tools/dlmod/source2"
)

// propCommand is "dlmod prop <game model> <file> [particle ...]": put an imported model in place
// of an ability prop's mesh (an older-layout model such as a thrown bomb). With particle systems
// named, the prop's original stays and a copy made from the file is written beside it, and those
// particle systems are pointed at the copy; this way one ability can get its own model when
// several share one. -scale sizes the new mesh against the old one.
func propCommand(ui prompter, store *Store, game Game, args []string, opts options) error {
	if len(args) < 2 {
		return errors.New("name a game model and a model file, for example: dlmod prop models/abilities/ivy_thornbomb_model.vmdl bomb.glb")
	}
	if gameRunning() {
		return errRunning
	}
	target := strings.TrimSuffix(filepath.ToSlash(args[0]), "_c")
	particles := args[2:]
	want := map[string]bool{target + "_c": true}
	for i, p := range particles {
		particles[i] = strings.TrimSuffix(filepath.ToSlash(p), "_c")
		want[particles[i]+"_c"] = true
	}
	index, err := source2.OpenVPKFiltered(filepath.Join(game.Citadel(), "pak01_dir.vpk"), func(name string) bool { return want[name] })
	if err != nil {
		return err
	}
	raw, err := index.Read(target + "_c")
	if err != nil {
		return fmt.Errorf("the game has no model %s", target)
	}
	mesh, err := source2.LoadMeshFile(args[1])
	if err != nil {
		return err
	}
	newPath := target
	if len(particles) > 0 {
		base := strings.TrimSuffix(filepath.Base(args[1]), filepath.Ext(args[1]))
		newPath = "models/dlmod/props/" + strings.ToLower(strings.ReplaceAll(base, " ", "_")) + ".vmdl"
	}
	var files map[string][]byte
	if model, lerr := source2.LoadModel(raw); lerr == nil && len(model.Meshes) > 0 && len(particles) == 0 {
		// A model in the newer layout (Ivy's stone form): the import is fitted to it as for a hero,
		// and every piece of it pinned to the skeleton's root, so it stays whole like a statue.
		sk, err := model.Skeleton()
		if err != nil {
			return err
		}
		root := sk.Names[0]
		pins := map[int]string{}
		for _, piece := range source2.Pieces(mesh) {
			pins[piece] = root
		}
		result, err := source2.ImportModel(raw, target, mesh, source2.ImportOptions{Keep: map[string]bool{}, Scale: opts.scale, Pins: pins}, nil)
		if err != nil {
			return err
		}
		files = result.Files
	} else {
		files, err = source2.BuildProp(raw, newPath, mesh, opts.scale)
		if err != nil {
			return err
		}
	}
	ui.say("%s: %s (%d triangles) at %s.", path.Base(target), filepath.Base(args[1]), mesh.TriangleCount(), newPath)
	for _, p := range particles {
		original, err := index.Read(p + "_c")
		if err != nil {
			return fmt.Errorf("the game has no particle system %s", p)
		}
		repointed, n, err := source2.RepointParticle(original, target, newPath)
		if err != nil {
			return fmt.Errorf("%s: %v", p, err)
		}
		files[p+"_c"] = repointed
		ui.say("  %s now uses it (%d places).", path.Base(p), n)
	}
	name := "Custom prop: " + path.Base(newPath)
	folder := filepath.Join(store.game.StateDir(), "models")
	if err := os.MkdirAll(folder, 0o755); err != nil {
		return err
	}
	vpk := filepath.Join(folder, "pak01_dir.vpk")
	if err := os.WriteFile(vpk, source2.WriteVPK(files), 0o644); err != nil {
		return err
	}
	defer os.Remove(vpk)
	entries, err := store.entries()
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if entry.Mod != nil && entry.Mod.Name == name {
			if err := store.remove(entry); err != nil {
				return err
			}
			ui.say("Removed the previous '%s'.", entry.Name())
		}
	}
	if err := store.install(ui, vpk, name); err != nil {
		return err
	}
	return ensurePatched(ui, game)
}
