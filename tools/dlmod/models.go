package main

import (
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/MrHakan/deadlock-modkit/tools/dlmod/source2"
)

// Custom models: drop a .glb, .gltf or .obj on a hero and dlmod rebuilds the hero's model file
// around it (source2.ImportModel): the hero's skeleton, animations and hitboxes stay, the new
// mesh is fitted, rigged from the hero's own body and given its own materials.

// heroModels is where each hero's model lives in the game (build 6864, from the deadlock-api
// asset data). If a path moves, heroModelPath looks for the file by name instead.
var heroModels = map[string]string{
	"artist":      "models/heroes_wip/artist/artist.vmdl",
	"astro":       "models/heroes_staging/astro/astro.vmdl",
	"atlas":       "models/heroes_wip/abrams/abrams.vmdl",
	"baba":        "models/heroes_wip/baba/baba.vmdl",
	"bebop":       "models/heroes_staging/bebop/bebop.vmdl",
	"bookworm":    "models/heroes_wip/bookworm/bookworm.vmdl",
	"chessmaster": "models/heroes_wip/chessmaster/chessmaster.vmdl",
	"chrono":      "models/heroes_staging/chrono/chrono.vmdl",
	"deadpack":    "models/heroes_wip/deadpack/deadpack.vmdl",
	"doorman":     "models/heroes_wip/doorman_v2/doorman.vmdl",
	"drifter":     "models/heroes_wip/drifter/drifter.vmdl",
	"dynamo":      "models/heroes_wip/dynamo/dynamo.vmdl",
	"familiar":    "models/heroes_wip/familiar/familiar_wip.vmdl",
	"fencer":      "models/heroes_wip/fencer/fencer.vmdl",
	"forge":       "models/heroes_wip/mcginnis/mcginnis.vmdl",
	"frank":       "models/heroes_wip/frank/frank.vmdl",
	"ghost":       "models/heroes_wip/geist/geist.vmdl",
	"gigawatt":    "models/heroes_staging/gigawatt_prisoner/gigawatt_prisoner.vmdl",
	"haze":        "models/heroes_staging/haze/haze.vmdl",
	"hornet":      "models/heroes_staging/hornet_v3/hornet.vmdl",
	"inferno":     "models/heroes_wip/inferno/inferno.vmdl",
	"kelvin":      "models/heroes_staging/kelvin_v2/kelvin.vmdl",
	"krill":       "models/heroes_staging/digger/digger.vmdl",
	"lash":        "models/heroes_wip/lash/lash.vmdl",
	"magician":    "models/heroes_staging/magician_v2/magician.vmdl",
	"mirage":      "models/heroes_staging/mirage_v2/mirage.vmdl",
	"nano":        "models/heroes_staging/nano/nano_v2/nano.vmdl",
	"necro":       "models/heroes_wip/necro/necro.vmdl",
	"nurse":       "models/heroes_wip/nurse/nurse.vmdl",
	"orion":       "models/heroes_staging/archer/archer.vmdl",
	"priest":      "models/heroes_wip/priest/priest.vmdl",
	"punkgoat":    "models/heroes_wip/punkgoat/punkgoat.vmdl",
	"ratking":     "models/heroes_wip/ratking/ratking.vmdl",
	"shiv":        "models/heroes_staging/shiv/shiv.vmdl",
	"synth":       "models/heroes_wip/pocket/pocket.vmdl",
	"tengu":       "models/heroes_wip/ivy/ivy.vmdl",
	"unicorn":     "models/heroes_wip/unicorn/unicorn.vmdl",
	"vampirebat":  "models/heroes_wip/vampirebat/vampirebat.vmdl",
	"viper":       "models/heroes_staging/viper/viper.vmdl",
	"viscous":     "models/heroes_staging/viscous/viscous.vmdl",
	"warden":      "models/heroes_staging/warden/warden.vmdl",
	"werewolf":    "models/heroes_wip/werewolf/werewolf.vmdl",
	"wraith":      "models/heroes_wip/wraith/wraith.vmdl",
	"yamato":      "models/heroes_staging/yamato_v2/yamato.vmdl",
}

// isModelFile reports whether a file is a 3D model the studio reads.
func isModelFile(name string) bool {
	switch strings.ToLower(filepath.Ext(name)) {
	case ".glb", ".gltf", ".obj", ".fbx":
		return true
	}
	return false
}

// heroDisplayName is the name players know a hero by.
func heroDisplayName(id string) string {
	if name, ok := heroNames[id]; ok {
		return name
	}
	return id
}

// heroIDs lists the heroes dlmod knows a model for, by display name.
func heroIDs() []string {
	ids := make([]string, 0, len(heroModels))
	for id := range heroModels {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(a, b int) bool { return heroDisplayName(ids[a]) < heroDisplayName(ids[b]) })
	return ids
}

// findHero matches what the user typed against internal and display names, whole or as a prefix.
func findHero(text string) (string, error) {
	want := strings.ToLower(strings.TrimSpace(text))
	if want == "" {
		return "", errors.New("name a hero, for example: dlmod model wraith model.glb")
	}
	var prefixed []string
	for _, id := range heroIDs() {
		names := []string{id, strings.ToLower(heroDisplayName(id)), path.Base(strings.TrimSuffix(heroModels[id], ".vmdl"))}
		for _, name := range names {
			if name == want {
				return id, nil
			}
			if strings.HasPrefix(name, want) {
				prefixed = append(prefixed, id)
				break
			}
		}
	}
	if len(prefixed) == 1 {
		return prefixed[0], nil
	}
	if len(prefixed) > 1 {
		var names []string
		for _, id := range prefixed {
			names = append(names, heroDisplayName(id))
		}
		return "", fmt.Errorf("%q could be %s", text, strings.Join(names, ", "))
	}
	return "", fmt.Errorf("no hero is called %q", text)
}

// openHeroModels indexes the hero model files of the game's archive.
func openHeroModels(game Game) (*source2.VPKIndex, error) {
	index, err := source2.OpenVPKFiltered(filepath.Join(game.Citadel(), "pak01_dir.vpk"), func(name string) bool {
		return strings.HasPrefix(name, "models/heroes") && strings.HasSuffix(name, ".vmdl_c")
	})
	if err != nil {
		return nil, fmt.Errorf("cannot read the game's models: %v", err)
	}
	return index, nil
}

// heroModelPath finds a hero's model in the archive: where it was, or else a model file of the
// same name anywhere under models/heroes.
func heroModelPath(index *source2.VPKIndex, id string) (string, error) {
	known := heroModels[id]
	if _, ok := index.Files[known+"_c"]; ok {
		return known, nil
	}
	base := path.Base(known) + "_c"
	var found []string
	for name := range index.Files {
		if path.Base(name) == base {
			found = append(found, strings.TrimSuffix(name, "_c"))
		}
	}
	sort.Strings(found)
	if len(found) == 0 {
		return "", fmt.Errorf("the game has no model for %s (looked for %s)", heroDisplayName(id), known)
	}
	return found[0], nil
}

// modelModName is the name a hero's custom model mod gets, so a new import replaces the last.
func modelModName(id string) string { return "Custom model: " + heroDisplayName(id) }

// installModel packs an import's files into a VPK and installs it as the hero's custom model, in
// place of the hero's previous one. That one is removed by its name, not by its files: a model
// with other materials has other files, and was left installed beside the new one, the two
// fighting over the hero's model.
func installModel(p prompter, store *Store, id string, result *source2.ImportResult) error {
	folder := filepath.Join(store.game.StateDir(), "models")
	if err := os.MkdirAll(folder, 0o755); err != nil {
		return err
	}
	vpk := filepath.Join(folder, "pak01_dir.vpk")
	if err := os.WriteFile(vpk, source2.WriteVPK(result.Files), 0o644); err != nil {
		return err
	}
	defer os.Remove(vpk)
	entries, err := store.entries()
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if entry.Mod != nil && entry.Mod.Name == modelModName(id) {
			if err := store.remove(entry); err != nil {
				return err
			}
			p.say("Removed the previous '%s'.", entry.Name())
		}
	}
	return store.install(p, vpk, modelModName(id))
}

// modelCommand is "dlmod model <hero> <file>": import and install without the studio.
func modelCommand(ui *UI, store *Store, game Game, args []string, opts options) error {
	if len(args) == 0 {
		return runStudio(ui, game)
	}
	if len(args) == 1 && isModelFile(args[0]) {
		return runStudio(ui, game, args[0])
	}
	if len(args) != 2 {
		return errors.New("name a hero and a model file, for example: dlmod model wraith model.glb (or dlmod studio)")
	}
	id, err := findHero(args[0])
	if err != nil {
		return err
	}
	mesh, err := source2.LoadMeshFile(args[1])
	if err != nil {
		return err
	}
	index, err := openHeroModels(game)
	if err != nil {
		return err
	}
	modelPath, err := heroModelPath(index, id)
	if err != nil {
		return err
	}
	raw, err := index.Read(modelPath + "_c")
	if err != nil {
		return err
	}
	model, err := source2.LoadModel(raw)
	if err != nil {
		return fmt.Errorf("%s's model: %v", heroDisplayName(id), err)
	}
	importOpts := source2.ImportOptions{Keep: map[string]bool{}, Scale: opts.scale, Yaw: opts.yaw, ZUp: opts.zup}
	importOpts.Tuning = source2.RigTuning{SteadyBody: opts.tuning.SteadyBody, NoArmIK: opts.tuning.NoArmIK}
	if opts.tuning.UsualPose {
		h, err := (&studio{game: game}).heroAnimation(id)
		if err == nil {
			importOpts.Tuning.UsualPose, err = h.usualPose()
		}
		if err != nil {
			return fmt.Errorf("the hero's usual pose could not be read: %v", err)
		}
	}
	if len(opts.material) > 0 {
		importOpts.Materials = map[string]string{}
		for part, mat := range opts.material {
			importOpts.Materials[part] = mat
			ui.say("  part %s is drawn with %s", part, mat)
		}
	}
	keepList := strings.Split(opts.keep, ",")
	for _, part := range model.Parts() {
		keep := part.Keep
		if opts.keep != "" {
			keep = false
			for _, k := range keepList {
				if k = strings.TrimSpace(strings.ToLower(k)); k != "" && strings.Contains(strings.ToLower(part.Name), k) {
					keep = true
				}
			}
		}
		importOpts.Keep[part.Name] = keep
		for name, shift := range opts.move {
			if keep && strings.Contains(strings.ToLower(part.Name), name) {
				if importOpts.KeptShift == nil {
					importOpts.KeptShift = map[string][3]float64{}
				}
				importOpts.KeptShift[part.Name] = shift
			}
		}
		state := "replaced"
		if keep {
			state = "kept"
			if s, ok := importOpts.KeptShift[part.Name]; ok {
				state = fmt.Sprintf("kept, moved %.1f %.1f %.1f", s[0], s[1], s[2])
			}
		}
		ui.say("  %-32s %8d vertices  %s", part.Name, part.Vertices, state)
	}
	// A -move naming a bone moves that bone at rest: a muzzle, to stay at the end of a moved gun.
	if len(opts.move) > 0 {
		if sk, err := model.Skeleton(); err == nil {
			for _, bone := range sk.Names {
				if shift, ok := opts.move[strings.ToLower(bone)]; ok {
					if importOpts.BoneShift == nil {
						importOpts.BoneShift = map[string][3]float64{}
					}
					importOpts.BoneShift[bone] = shift
					ui.say("  bone %-27s moved %.1f %.1f %.1f", bone, shift[0], shift[1], shift[2])
				}
			}
		}
		for name := range opts.move {
			used := false
			for part := range importOpts.KeptShift {
				used = used || strings.Contains(strings.ToLower(part), name)
			}
			for bone := range importOpts.BoneShift {
				used = used || strings.ToLower(bone) == name
			}
			if !used {
				return fmt.Errorf("-move %s: no kept part or bone of that name", name)
			}
		}
	}
	ui.say("Fitting and rigging %s (%d triangles) onto %s...", filepath.Base(args[1]), mesh.TriangleCount(), heroDisplayName(id))
	result, err := source2.ImportModel(raw, modelPath, mesh, importOpts, nil)
	if err != nil {
		return err
	}
	for _, note := range result.Notes {
		ui.say("Note: %s", note)
	}
	ui.say("Matched the model's pose: arms %+.0f°, legs %+.0f° from %s's.", result.Pose.Arms, result.Pose.Legs, heroDisplayName(id))
	if err := installModel(ui, store, id, result); err != nil {
		return err
	}
	return ensurePatched(ui, game)
}
