package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/MrHakan/deadlock-modkit/tools/dlmod/source2"
)

// paintResult summarises one hero repaint: every compiled replacement file ready for a VPK, plus
// counts and diagnostics for CLI output and reports.
type paintResult struct {
	Files             map[string][]byte
	ParticleFiles     int
	ParticleColors    int
	MaterialFiles     int
	MaterialVectors   int
	TextureFiles      int
	RepaintedTextures []paintedTexture
	Skipped           []string
}

type paintedTexture struct {
	Path          string
	Width, Height int
	Format        int
	Mips          int
	ChangedPixels int
	TotalPixels   int
	Original      image.Image
	Repainted     image.Image
}

// paintCommand is "dlmod paint <hero> [palette|restore] [-skip LIST]": repaints a hero's ability
// and weapon particles, hero-exclusive materials, and their colour textures (including ability
// prop model textures such as Ivy's bramble vines) with a multi-tone palette (default: cobalt),
// packing them into the "Custom effects" mod.
func paintCommand(ui prompter, store *Store, game Game, args []string, opts options) error {
	if len(args) < 1 {
		return errors.New("name a hero, for example: dlmod paint ivy (or: dlmod paint ivy restore)")
	}
	if opts.preview == "" && gameRunning() {
		return errRunning
	}
	hero, err := findHero(args[0])
	if err != nil {
		return err
	}

	// Preview mode: read-only against the game VPK, write before/after contact sheets, touch nothing in the game.
	if opts.preview != "" {
		palName := "cobalt"
		if len(args) >= 2 {
			palName = args[1]
		}
		pal, err := LookupPalette(palName)
		if err != nil {
			return err
		}
		var skipSubstrs []string
		for _, part := range strings.Split(opts.skip, ",") {
			if part = strings.TrimSpace(strings.ToLower(part)); part != "" {
				skipSubstrs = append(skipSubstrs, part)
			}
		}
		index, err := openPaintVPK(game)
		if err != nil {
			return err
		}
		res, err := paintHero(index, hero, pal, skipSubstrs, nil, true)
		if err != nil {
			return err
		}
		if err := writePaintContactSheets(opts.preview, hero, pal, res, index); err != nil {
			return err
		}
		for _, msg := range res.Skipped {
			ui.say("Skipped %s", msg)
		}
		var matPaths []string
		var totalBytes int64
		for p, b := range res.Files {
			totalBytes += int64(len(b))
			if strings.HasSuffix(p, ".vmat_c") {
				matPaths = append(matPaths, p)
			}
		}
		sort.Strings(matPaths)
		for _, m := range matPaths {
			ui.say("  material %s", m)
		}
		for _, tex := range res.RepaintedTextures {
			ui.say("  texture  %-68s %dx%d %s (%d mips, %.1f%% pixels changed)",
				tex.Path, tex.Width, tex.Height, source2.TextureFormatName(tex.Format), tex.Mips,
				100*float64(tex.ChangedPixels)/float64(max(1, tex.TotalPixels)))
		}
		ui.say("Preview for %s (%s): %d particles (%d colours), %d materials (%d vectors), %d textures (%s) -> %s",
			heroDisplayName(hero), pal.Name, res.ParticleFiles, res.ParticleColors,
			res.MaterialFiles, res.MaterialVectors, res.TextureFiles, formatSize(totalBytes), opts.preview)
		return nil
	}

	project, err := loadReplacements(game.effectsDir(), customEffectsName)
	if err != nil {
		return err
	}
	painted, err := loadPaintedManifest(game)
	if err != nil {
		return err
	}

	// Always start fresh for this hero: restore any previously painted paths and particle effects.
	if prev := painted[hero]; len(prev) > 0 {
		project.restore(prev)
		delete(painted, hero)
	}
	s := &studio{game: game}
	if abilities, weapon, err := s.heroParticles(hero); err == nil {
		project.restore(append(append([]string{}, abilities...), weapon...))
	}

	if len(args) >= 2 && strings.EqualFold(args[1], "restore") {
		if err := savePaintedManifest(game, painted); err != nil {
			return err
		}
		if all, err := loadEffectSettings(game); err == nil {
			delete(all, hero)
			_ = saveEffectSettings(game, all)
		}
		ui.say("%s's painted effects and textures are the game's own again.", heroDisplayName(hero))
		if err := store.installReplacements(ui, project); err != nil {
			return err
		}
		return ensurePatched(ui, game)
	}

	palName := "cobalt"
	if len(args) >= 2 {
		palName = args[1]
	}
	pal, err := LookupPalette(palName)
	if err != nil {
		return err
	}

	// Detect game paths already replaced by other enabled mods so we do not conflict with them
	// (for example, an installed custom prop mod replacing ivy_thornbomb_model).
	replacedByOther := map[string]string{}
	if entries, err := store.entries(); err == nil {
		for _, entry := range entries {
			if !entry.Enabled() || (entry.Mod != nil && entry.Mod.Name == customEffectsName) {
				continue
			}
			for _, p := range entry.Paths {
				replacedByOther[strings.ToLower(p)] = entry.Name()
			}
		}
	}
	var skipSubstrs []string
	for _, part := range strings.Split(opts.skip, ",") {
		if part = strings.TrimSpace(strings.ToLower(part)); part != "" {
			skipSubstrs = append(skipSubstrs, part)
		}
	}

	index, err := openPaintVPK(game)
	if err != nil {
		return err
	}
	res, err := paintHero(index, hero, pal, skipSubstrs, replacedByOther, false)
	if err != nil {
		return err
	}
	if len(res.Files) == 0 {
		return fmt.Errorf("%s has no ability effects or textures to repaint", heroDisplayName(hero))
	}
	label := "painted " + pal.Name
	paths := make([]string, 0, len(res.Files))
	for p := range res.Files {
		paths = append(paths, p)
	}
	sort.Strings(paths)
	for _, p := range paths {
		if err := project.add(p, label, res.Files[p]); err != nil {
			return err
		}
	}
	painted[hero] = paths
	if err := savePaintedManifest(game, painted); err != nil {
		return err
	}
	if all, err := loadEffectSettings(game); err == nil {
		all[hero] = effectSettings{Palette: pal.Name, Saturation: 1, Brightness: 1, Weapon: true}
		_ = saveEffectSettings(game, all)
	}
	for _, msg := range res.Skipped {
		ui.say("Skipped %s", msg)
	}
	ui.say("Painted %s (%s): %d particles (%d colours), %d materials (%d vectors), %d textures.",
		heroDisplayName(hero), pal.Name, res.ParticleFiles, res.ParticleColors,
		res.MaterialFiles, res.MaterialVectors, res.TextureFiles)
	if err := store.installReplacements(ui, project); err != nil {
		return err
	}
	return ensurePatched(ui, game)
}

func openPaintVPK(game Game) (*source2.VPKIndex, error) {
	return source2.OpenVPKFiltered(filepath.Join(game.Citadel(), "pak01_dir.vpk"), func(name string) bool {
		return strings.HasSuffix(name, ".vpcf_c") ||
			strings.HasSuffix(name, ".vmdl_c") ||
			strings.HasSuffix(name, ".vmat_c") ||
			strings.HasSuffix(name, ".vtex_c")
	})
}

func loadPaintedManifest(game Game) (map[string][]string, error) {
	out := map[string][]string{}
	data, err := os.ReadFile(filepath.Join(game.effectsDir(), "painted.json"))
	if errors.Is(err, os.ErrNotExist) {
		return out, nil
	}
	if err == nil {
		err = json.Unmarshal(data, &out)
	}
	return out, err
}

func savePaintedManifest(game Game, painted map[string][]string) error {
	data, err := json.MarshalIndent(painted, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(game.effectsDir(), 0o755); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(game.effectsDir(), "painted.json"), data, 0o644)
}

func pathMatchesAny(p string, substrs []string) bool {
	low := strings.ToLower(p)
	for _, s := range substrs {
		if s != "" && strings.Contains(low, s) {
			return true
		}
	}
	return false
}

func pathContainsHero(p string, heroNames []string) bool {
	low := strings.ToLower(p)
	for _, name := range heroNames {
		if len(name) >= 3 && strings.Contains(low, name) {
			return true
		}
	}
	return false
}

func isNonColorTexturePath(p string) bool {
	low := strings.ToLower(p)
	if strings.HasPrefix(low, "materials/default/") ||
		strings.HasPrefix(low, "materials/dev/") ||
		strings.HasPrefix(low, "materials/stone/") ||
		strings.HasPrefix(low, "materials/models/statuseffects/") ||
		strings.HasPrefix(low, "models/heroes_") {
		return true
	}
	base := path.Base(low)
	for _, kw := range []string{
		"normal", "rough", "_ao_", "_ao.", "metal", "mask", "warp", "distort",
		"noise_flow", "flow_", "noise", "colorwarp", "ui_common", "selfillum", "flare",
		"blood", "fire_particle", "us_shield", "sphere_hotblue",
	} {
		if strings.Contains(base, kw) || strings.Contains(low, "/"+kw+"/") {
			return true
		}
	}
	return false
}

func isSkippableMaterialPath(p string, directParticleOverride bool) bool {
	low := strings.ToLower(p)
	if strings.HasPrefix(low, "materials/default/") || strings.HasPrefix(low, "materials/dev/") {
		return true
	}
	if strings.HasPrefix(low, "models/heroes_") && !directParticleOverride {
		return true
	}
	return false
}

// paintHero discovers and repaints a hero's ability particles, materials, and colour textures
// using pal. Set keepImages to retain decoded before/after images (for contact sheets).
func paintHero(index *source2.VPKIndex, hero string, pal ColorPalette, skipSubstrs []string, replacedByOther map[string]string, keepImages bool) (*paintResult, error) {
	names := heroFileNames(hero)
	res := &paintResult{Files: map[string][]byte{}}

	// 1. Seed particles: abilities, weapon_fx, and status_fx belonging to the hero.
	particleSet := map[string]bool{}
	materialSet := map[string]bool{}
	directOverrideMat := map[string]bool{}
	modelSet := map[string]bool{}
	texSet := map[string]bool{}

	for file := range index.Files {
		low := strings.ToLower(file)
		for _, name := range names {
			if strings.HasSuffix(low, ".vpcf_c") {
				if strings.HasPrefix(low, "particles/abilities/"+name+"/") ||
					strings.HasPrefix(low, "particles/weapon_fx/"+name+"/") ||
					(strings.HasPrefix(low, "particles/status_fx/") && strings.Contains(path.Base(low), name)) {
					particleSet[file] = true
					break
				}
			}
			// Include hero-specific cable materials (e.g. tengu_tether_cable.vmat_c).
			if strings.HasSuffix(low, ".vmat_c") && strings.HasPrefix(low, "materials/particle/cables/") && strings.Contains(path.Base(low), name) {
				materialSet[strings.TrimSuffix(low, "_c")] = true
				directOverrideMat[strings.TrimSuffix(low, "_c")] = true
				break
			}
		}
	}

	// 2. Walk particle graph to collect active models, materials, and particle textures.
	var pQueue []string
	for p := range particleSet {
		pQueue = append(pQueue, p)
	}
	sort.Strings(pQueue)
	visitedPcf := map[string]bool{}
	for len(pQueue) > 0 {
		p := pQueue[0]
		pQueue = pQueue[1:]
		if visitedPcf[p] {
			continue
		}
		visitedPcf[p] = true
		data, err := index.Read(p)
		if err != nil {
			continue
		}
		refs, err := source2.ParticleAssetRefs(data)
		if err != nil {
			continue
		}
		for _, ch := range refs.Children {
			chc := ch + "_c"
			if _, ok := index.Files[chc]; ok && !visitedPcf[chc] {
				if pathContainsHero(chc, names) {
					particleSet[chc] = true
					pQueue = append(pQueue, chc)
				}
			}
		}
		for _, m := range refs.Models {
			modelSet[m] = true
		}
		for _, mat := range refs.Materials {
			materialSet[mat] = true
			directOverrideMat[mat] = true
		}
		for _, tex := range refs.Textures {
			texSet[tex] = true
		}
	}

	// 3. Collect materials from ability prop models (skipping hero body models, -skip matches,
	// and props already replaced by another installed mod).
	var models []string
	for m := range modelSet {
		models = append(models, m)
	}
	sort.Strings(models)
	for _, m := range models {
		mc := m + "_c"
		if strings.HasPrefix(m, "models/heroes_") {
			continue
		}
		if modName, ok := replacedByOther[mc]; ok {
			res.Skipped = append(res.Skipped, fmt.Sprintf("%s (already replaced by %q)", m, modName))
			continue
		}
		if pathMatchesAny(m, skipSubstrs) {
			res.Skipped = append(res.Skipped, fmt.Sprintf("%s (matched -skip)", m))
			continue
		}
		data, err := index.Read(mc)
		if err != nil {
			continue
		}
		mats, err := source2.ModelMaterials(data)
		if err != nil {
			continue
		}
		for _, mat := range mats {
			materialSet[mat] = true
		}
	}

	// 4. Filter materials and extract their colour texture parameters.
	var materials []string
	for m := range materialSet {
		if isSkippableMaterialPath(m, directOverrideMat[m]) {
			continue
		}
		if pathMatchesAny(m, skipSubstrs) {
			res.Skipped = append(res.Skipped, fmt.Sprintf("%s (matched -skip)", m))
			continue
		}
		if modName, ok := replacedByOther[m+"_c"]; ok {
			res.Skipped = append(res.Skipped, fmt.Sprintf("%s (already replaced by %q)", m, modName))
			continue
		}
		materials = append(materials, m)
	}
	sort.Strings(materials)

	// Track which textures belong to hero-exclusive materials so we know whether a texture like
	// materials/blends/ground_plants_01_color_psd_ac95d917.vtex is pulled via a hero-only material.
	texFromMat := map[string][]string{}
	for _, m := range materials {
		data, err := index.Read(m + "_c")
		if err != nil {
			continue
		}
		_, params, err := source2.MaterialTextureParams(data)
		if err != nil {
			continue
		}
		for paramName, tex := range params {
			if !source2.IsMaterialColorParam(paramName) {
				continue
			}
			texSet[tex] = true
			texFromMat[tex] = append(texFromMat[tex], m)
		}
	}

	// 5. Check whether non-hero-named materials/textures are shared with other heroes' abilities.
	checkShare := map[string]bool{}
	for _, m := range materials {
		if !pathContainsHero(m, names) {
			checkShare[m] = true
		}
	}
	for t := range texSet {
		if !isNonColorTexturePath(t) && !pathContainsHero(t, names) {
			checkShare[t] = true
		}
	}
	sharedByOtherHero := findSharedAssets(index, names, checkShare)

	// 6. Repaint particles.
	var particles []string
	for p := range particleSet {
		particles = append(particles, p)
	}
	sort.Strings(particles)
	for _, p := range particles {
		if modName, ok := replacedByOther[strings.ToLower(p)]; ok {
			res.Skipped = append(res.Skipped, fmt.Sprintf("%s (already replaced by %q)", p, modName))
			continue
		}
		data, err := index.Read(p)
		if err != nil {
			continue
		}
		out, changed, err := source2.RecolorParticle(data, pal.Recolor)
		if err != nil {
			return nil, fmt.Errorf("%s: %v", p, err)
		}
		if changed == 0 {
			continue
		}
		res.Files[p] = out
		res.ParticleFiles++
		res.ParticleColors += changed
	}

	// 7. Repaint non-shared materials that have chromatic colour/tint vectors.
	activeMaterials := map[string]bool{}
	for _, m := range materials {
		if other := sharedByOtherHero[m]; other != "" {
			res.Skipped = append(res.Skipped, fmt.Sprintf("%s (shared with %s)", m, other))
			continue
		}
		activeMaterials[m] = true
		data, err := index.Read(m + "_c")
		if err != nil {
			continue
		}
		out, changed, err := source2.RecolorMaterial(data, pal.Recolor)
		if err != nil {
			continue
		}
		if changed == 0 {
			continue
		}
		res.Files[m+"_c"] = out
		res.MaterialFiles++
		res.MaterialVectors += changed
	}

	// 8. Repaint non-shared colour textures that actually carry chromatic pixels.
	var textures []string
	for t := range texSet {
		textures = append(textures, t)
	}
	sort.Strings(textures)
	for _, t := range textures {
		tc := t + "_c"
		if isNonColorTexturePath(t) {
			continue
		}
		if pathMatchesAny(t, skipSubstrs) {
			res.Skipped = append(res.Skipped, fmt.Sprintf("%s (matched -skip)", t))
			continue
		}
		if modName, ok := replacedByOther[tc]; ok {
			res.Skipped = append(res.Skipped, fmt.Sprintf("%s (already replaced by %q)", t, modName))
			continue
		}
		if other := sharedByOtherHero[t]; other != "" {
			res.Skipped = append(res.Skipped, fmt.Sprintf("%s (shared with %s)", t, other))
			continue
		}
		// If this texture came only from materials that were all skipped as shared, skip it too.
		if mats := texFromMat[t]; len(mats) > 0 {
			anyActive := false
			for _, m := range mats {
				if activeMaterials[m] {
					anyActive = true
					break
				}
			}
			if !anyActive {
				continue
			}
		}
		data, err := index.Read(tc)
		if err != nil {
			continue
		}
		img, info, err := source2.DecodeTexture(data)
		if err != nil {
			res.Skipped = append(res.Skipped, fmt.Sprintf("%s (%v)", t, err))
			continue
		}
		repaintedImg, changedPx := RepaintImage(img, pal.Recolor)
		totalPx := info.Width * info.Height
		// Skip essentially monochrome/grayscale textures (tinted at runtime by particle colours).
		if totalPx == 0 || changedPx*100 < totalPx {
			continue
		}
		compiled, err := source2.RepaintTexture(data, repaintedImg)
		if err != nil {
			res.Skipped = append(res.Skipped, fmt.Sprintf("%s (%v)", t, err))
			continue
		}
		res.Files[tc] = compiled
		res.TextureFiles++
		entry := paintedTexture{
			Path:          tc,
			Width:         info.Width,
			Height:        info.Height,
			Format:        info.Format,
			Mips:          info.Mips,
			ChangedPixels: changedPx,
			TotalPixels:   totalPx,
		}
		if keepImages {
			entry.Original = img
			entry.Repainted = repaintedImg
		}
		res.RepaintedTextures = append(res.RepaintedTextures, entry)
	}

	sort.Strings(res.Skipped)
	return res, nil
}

// findSharedAssets checks whether any candidate path (lowercase without "_c") is referenced in
// RERL by another hero's ability particles (particles/abilities/<other>/*). It reads only the
// first 4 KB of each particle file so the scan finishes quickly.
func findSharedAssets(index *source2.VPKIndex, heroNames []string, candidates map[string]bool) map[string]string {
	shared := map[string]string{}
	if len(candidates) == 0 {
		return shared
	}
	isOwn := func(bucket string) bool {
		for _, n := range heroNames {
			if bucket == n {
				return true
			}
		}
		return false
	}
	for file := range index.Files {
		if !strings.HasSuffix(file, ".vpcf_c") || !strings.HasPrefix(file, "particles/abilities/") {
			continue
		}
		rest := strings.TrimPrefix(file, "particles/abilities/")
		bucket, _, ok := strings.Cut(rest, "/")
		if !ok || isOwn(bucket) {
			continue
		}
		head, err := index.ReadHead(file, 4096)
		if err != nil {
			continue
		}
		refs, err := readReferencesFromHead(head)
		if err != nil {
			continue
		}
		for _, ref := range refs {
			p := strings.ToLower(strings.TrimSuffix(ref.Name, "_c"))
			if candidates[p] && shared[p] == "" {
				shared[p] = bucket
			}
		}
	}
	return shared
}

// readReferencesFromHead reads just the RERL block from a truncated resource header without
// needing the full resource file.
func readReferencesFromHead(head []byte) ([]source2.Reference, error) {
	if len(head) < 16 {
		return nil, errors.New("short header")
	}
	if r, err := source2.DecodeResource(head); err == nil {
		return source2.ReadReferences(r.Block("RERL"))
	}
	// If the full resource extends past head, parse only the block table for RERL.
	tableStart := 8 + int(uint32(head[8])|uint32(head[9])<<8|uint32(head[10])<<16|uint32(head[11])<<24)
	count := int(uint32(head[12]) | uint32(head[13])<<8 | uint32(head[14])<<16 | uint32(head[15])<<24)
	if tableStart < 0 || tableStart+12*count > len(head) {
		return nil, errors.New("table past head")
	}
	for i := 0; i < count; i++ {
		entry := tableStart + 12*i
		if string(head[entry:entry+4]) != "RERL" {
			continue
		}
		off := entry + 4 + int(uint32(head[entry+4])|uint32(head[entry+5])<<8|uint32(head[entry+6])<<16|uint32(head[entry+7])<<24)
		length := int(uint32(head[entry+8]) | uint32(head[entry+9])<<8 | uint32(head[entry+10])<<16 | uint32(head[entry+11])<<24)
		if off < 0 || off+length > len(head) {
			return nil, errors.New("rerl past head")
		}
		return source2.ReadReferences(head[off : off+length])
	}
	return nil, nil
}
