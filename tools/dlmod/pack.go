package main

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Character packs: a character made for a hero (the creator's folder, game/dlmod/characters/<hero>,
// or a workspace folder such as meshy/<hero>) in one file, <name>.dlmodchar. It is a ZIP of the
// workspace, its assets, the reference pictures and the creator's state, which dlmod can install (dlmod apply), unpack
// into the creator (dlmod unpack), or keep beside other characters made for the same hero
// (game/dlmod/characters/saved/<hero>).

const (
	packExt        = ".dlmodchar"
	packFormat     = 1
	packMaxBytes   = 1 << 30   // everything in a pack, unpacked
	packMaxFile    = 512 << 20 // one file in a pack, unpacked
	packMaxEntries = 4096
	packManifestN  = "pack.json"
)

type packManifest struct {
	Format  int    `json:"format"`
	Hero    string `json:"hero"`
	Name    string `json:"name"`
	Created string `json:"created"`
	DLMod   string `json:"dlmod"`
}

// packTop are the files and folders a pack may hold.
var packTop = []string{packManifestN, "workspace.json", "creator.json", "README.md", "prompts.md", "assets/", "gen/", "refs/"}

func (g Game) charactersDir() string { return filepath.Join(g.StateDir(), "characters") }

// savedDir keeps a hero's other characters, as packs.
func (g Game) savedDir(hero string) string { return filepath.Join(g.charactersDir(), "saved", hero) }

// keptAsset is where choose copies a kept generation's file in assets/, or "" (see choose).
func keptAsset(steps []workspaceStep, g *creatorGen, i int) string {
	switch {
	case g.Step == "turnaround":
		return fmt.Sprintf("assets/views_%d.png", i+1)
	case i > 0:
		return ""
	case g.Step == "rig":
		return "assets/model.glb"
	}
	for _, s := range steps {
		if s.ID == g.Step {
			return s.Output
		}
	}
	return ""
}

// packName is a character's name for its pack: the name on the hero select screen, else the
// first words of the description, else the hero's.
func packName(w *workspace) string {
	if w.Select.Name != "" {
		return w.Select.Name
	}
	if name, _, _ := strings.Cut(w.Concept, ","); strings.TrimSpace(name) != "" && !strings.Contains(name, conceptMark) {
		return strings.TrimSpace(name)
	}
	return w.Name
}

// exportPack packs a character's folder. Of the creator's generations only the kept ones go in,
// and a kept file that assets/ already holds unchanged goes in once.
func exportPack(dir string) ([]byte, packManifest, error) {
	w, err := readWorkspace(dir)
	if err != nil {
		return nil, packManifest{}, err
	}
	manifest := packManifest{Format: packFormat, Hero: w.Hero, Name: packName(w), Created: time.Now().UTC().Format(time.RFC3339), DLMod: version}
	var out bytes.Buffer
	z := zip.NewWriter(&out)
	add := func(name string, data []byte) error {
		f, err := z.CreateHeader(&zip.FileHeader{Name: name, Method: zip.Deflate, Modified: time.Now()})
		if err != nil {
			return err
		}
		_, err = f.Write(data)
		return err
	}
	addFile := func(name string) error {
		data, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(name)))
		if err != nil {
			return err
		}
		return add(name, data)
	}
	info, _ := json.MarshalIndent(manifest, "", "  ")
	if err := add(packManifestN, info); err != nil {
		return nil, manifest, err
	}
	for _, name := range []string{"workspace.json", "README.md", "prompts.md"} {
		if fileExists(filepath.Join(dir, name)) {
			if err := addFile(name); err != nil {
				return nil, manifest, err
			}
		}
	}
	var assets []string // the made files, and the reference pictures they were made from
	for _, folder := range []string{"assets", "refs"} {
		filepath.WalkDir(filepath.Join(dir, folder), func(p string, d os.DirEntry, err error) error {
			if err == nil && d.Type().IsRegular() && !strings.HasSuffix(d.Name(), "_keyed.png") { // keyed copies are made again
				rel, _ := filepath.Rel(dir, p)
				assets = append(assets, filepath.ToSlash(rel))
			}
			return nil
		})
	}
	sort.Strings(assets)
	for _, name := range assets {
		if err := addFile(name); err != nil {
			return nil, manifest, err
		}
	}
	if data, err := os.ReadFile(filepath.Join(dir, "creator.json")); err == nil {
		var state creatorState
		if err := json.Unmarshal(data, &state); err != nil {
			return nil, manifest, fmt.Errorf("creator.json: %v", err)
		}
		steps := creatorSteps(w)
		kept := map[string]bool{}
		for _, id := range state.Chosen {
			kept[id] = true
		}
		var gens []*creatorGen
		for _, g := range state.Gens {
			if !kept[g.ID] || g.Status != "SUCCEEDED" {
				continue
			}
			gens = append(gens, g)
			for i, f := range g.Files {
				own, err := os.ReadFile(filepath.Join(dir, "gen", f))
				if err != nil {
					continue
				}
				if asset := keptAsset(steps, g, i); asset != "" {
					if copied, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(asset))); err == nil && bytes.Equal(copied, own) {
						continue // unpacking copies it back from assets/
					}
				}
				if err := add("gen/"+f, own); err != nil {
					return nil, manifest, err
				}
			}
			if g.Thumb != "" {
				if thumb, err := os.ReadFile(filepath.Join(dir, "gen", g.Thumb)); err == nil {
					if err := add("gen/"+g.Thumb, thumb); err != nil {
						return nil, manifest, err
					}
				}
			}
		}
		chosen := map[string]string{}
		for step, id := range state.Chosen {
			for _, g := range gens {
				if g.ID == id {
					chosen[step] = id
				}
			}
		}
		state.Gens, state.Chosen = gens, chosen
		data, _ := json.MarshalIndent(state, "", "  ")
		if err := add("creator.json", data); err != nil {
			return nil, manifest, err
		}
	}
	if err := z.Close(); err != nil {
		return nil, manifest, err
	}
	return out.Bytes(), manifest, nil
}

// localPath reports whether a slash-separated path stays inside its folder on every system: not
// absolute, no "..", no backslash, no drive or stream colon, no Windows device name.
func localPath(p string) bool {
	if p == "" || strings.ContainsAny(p, `\:`) || path.Clean(p) != p || path.IsAbs(p) || !filepath.IsLocal(filepath.FromSlash(p)) {
		return false
	}
	for _, part := range strings.Split(p, "/") {
		if part == ".." || windowsDevice(part) { // packs travel between systems: refuse NUL on Linux too
			return false
		}
	}
	return true
}

// windowsDevice reports whether a file name is one Windows keeps for a device (NUL, COM1...).
func windowsDevice(name string) bool {
	base, _, _ := strings.Cut(strings.ToUpper(name), ".")
	switch base {
	case "CON", "PRN", "AUX", "NUL":
		return true
	}
	return len(base) == 4 && (strings.HasPrefix(base, "COM") || strings.HasPrefix(base, "LPT")) && base[3] >= '0' && base[3] <= '9'
}

// packEntryAllowed reports whether a name in a pack is one of a pack's own files.
func packEntryAllowed(name string) bool {
	if !localPath(name) {
		return false
	}
	for _, top := range packTop {
		if name == top || (strings.HasSuffix(top, "/") && strings.HasPrefix(name, top) && len(name) > len(top)) {
			return true
		}
	}
	return false
}

// openPack checks a pack's list of files and reads its manifest, unpacking nothing else: every
// name must be one of a pack's own (pack.json, the workspace, assets/, gen/), and the number of
// files and their sizes are bounded.
func openPack(data []byte) (*zip.Reader, packManifest, error) {
	var manifest packManifest
	z, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, manifest, fmt.Errorf("not a character pack: %v", err)
	}
	if len(z.File) > packMaxEntries {
		return nil, manifest, errors.New("the pack holds too many files")
	}
	var total uint64
	var info, work *zip.File
	for _, f := range z.File {
		if f.FileInfo().IsDir() {
			continue
		}
		if !packEntryAllowed(f.Name) {
			return nil, manifest, fmt.Errorf("the pack holds a file dlmod does not unpack: %q", f.Name)
		}
		if f.UncompressedSize64 > packMaxFile {
			return nil, manifest, fmt.Errorf("%s is too large", f.Name)
		}
		if total += f.UncompressedSize64; total > packMaxBytes {
			return nil, manifest, errors.New("the pack is too large")
		}
		switch f.Name {
		case packManifestN:
			info = f
		case "workspace.json":
			work = f
		}
	}
	if info == nil {
		return nil, manifest, errors.New("not a character pack: pack.json is missing")
	}
	r, err := info.Open()
	if err != nil {
		return nil, manifest, err
	}
	raw, err := io.ReadAll(io.LimitReader(r, 1<<16))
	r.Close()
	if err != nil || json.Unmarshal(raw, &manifest) != nil {
		return nil, manifest, errors.New("not a character pack: pack.json is broken")
	}
	if manifest.Format > packFormat {
		return nil, manifest, fmt.Errorf("the pack was made by a newer dlmod (format %d); update dlmod", manifest.Format)
	}
	if manifest.Hero, err = findHero(manifest.Hero); err != nil {
		return nil, manifest, err
	}
	if work == nil {
		return nil, manifest, errors.New("the pack has no workspace.json")
	}
	return z, manifest, nil
}

// packManifestOf reads a pack's manifest, after checking its list of files.
func packManifestOf(data []byte) (packManifest, error) {
	_, manifest, err := openPack(data)
	return manifest, err
}

// extractPack writes a checked pack's files into a new folder, one at a time, holding the sizes
// to what the pack declared.
func extractPack(z *zip.Reader, dir string) error {
	var total int64
	for _, f := range z.File {
		if f.FileInfo().IsDir() || f.Name == packManifestN {
			continue
		}
		if !packEntryAllowed(f.Name) { // openPack refused these already
			return fmt.Errorf("bad file %q", f.Name)
		}
		target := filepath.Join(dir, filepath.FromSlash(f.Name))
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		r, err := f.Open()
		if err != nil {
			return err
		}
		out, err := os.Create(target)
		if err != nil {
			r.Close()
			return err
		}
		n, err := io.Copy(out, io.LimitReader(r, packMaxFile+1))
		r.Close()
		if closeErr := out.Close(); err == nil {
			err = closeErr
		}
		if err != nil {
			return fmt.Errorf("%s: %v", f.Name, err)
		}
		if total += n; n > packMaxFile || uint64(n) > f.UncompressedSize64 || total > packMaxBytes {
			return errors.New("the pack is larger than it says")
		}
	}
	return nil
}

// unpackTo writes a checked pack into a new folder, and puts the creator's kept files back in
// gen/ from assets/ where the pack left them out. The names in the pack's creator.json and
// workspace.json are checked like the pack's own: a result whose files would lie outside gen/
// or assets/ is dropped.
func unpackTo(z *zip.Reader, dir string) error {
	if fileExists(dir) {
		return fmt.Errorf("%s is already there", dir)
	}
	if err := extractPack(z, dir); err != nil {
		return err
	}
	w, err := readWorkspace(dir) // refuses steps that point outside the folder
	if err != nil {
		return err
	}
	data, err := os.ReadFile(filepath.Join(dir, "creator.json"))
	if err != nil {
		return nil // a workspace without the creator
	}
	var state creatorState
	if err := json.Unmarshal(data, &state); err != nil {
		return fmt.Errorf("creator.json: %v", err)
	}
	steps := creatorSteps(w)
	genName := func(f string) bool { return localPath(f) && path.Base(f) == f }
	var gens []*creatorGen
	for _, g := range state.Gens {
		whole := g != nil && len(g.Files) > 0
		for i := 0; whole && i < len(g.Files); i++ {
			f := g.Files[i]
			if !genName(f) {
				whole = false
				break
			}
			own := filepath.Join(dir, "gen", f)
			if fileExists(own) {
				continue
			}
			asset := keptAsset(steps, g, i)
			if !localPath(asset) || !strings.HasPrefix(asset, "assets/") {
				whole = false
				break
			}
			copied, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(asset)))
			if err != nil {
				whole = false
				break
			}
			os.MkdirAll(filepath.Dir(own), 0o755)
			if err := os.WriteFile(own, copied, 0o644); err != nil {
				return err
			}
		}
		if !whole {
			continue
		}
		if g.Thumb != "" && (!genName(g.Thumb) || !fileExists(filepath.Join(dir, "gen", g.Thumb))) {
			g.Thumb = ""
		}
		gens = append(gens, g)
	}
	chosen := map[string]string{}
	for step, id := range state.Chosen {
		for _, g := range gens {
			if g.ID == id {
				chosen[step] = id
			}
		}
	}
	state.Gens, state.Chosen = gens, chosen
	data, _ = json.MarshalIndent(state, "", "  ")
	return os.WriteFile(filepath.Join(dir, "creator.json"), data, 0o644)
}

// savePack keeps a hero's current character as a pack among its saved ones, and returns the file.
func savePack(game Game, dir string) (string, packManifest, error) {
	data, manifest, err := exportPack(dir)
	if err != nil {
		return "", manifest, err
	}
	folder := game.savedDir(manifest.Hero)
	if err := os.MkdirAll(folder, 0o755); err != nil {
		return "", manifest, err
	}
	base := slugify(manifest.Name)
	if base == "" {
		base = "character"
	}
	file := filepath.Join(folder, base+packExt)
	for n := 2; fileExists(file); n++ {
		// The same character saved again replaces its pack; another one of the same name gets a number.
		if old, err := os.ReadFile(file); err == nil {
			if m, err := packManifestOf(old); err == nil && m.Name == manifest.Name {
				break
			}
		}
		file = filepath.Join(folder, fmt.Sprintf("%s-%d%s", base, n, packExt))
	}
	return file, manifest, os.WriteFile(file, data, 0o644)
}

// worthKeeping reports whether a character folder holds anything of the user's: a description,
// a generation or a file in assets/. A folder the Create tab only opened is not saved.
func worthKeeping(dir string) bool {
	if !fileExists(filepath.Join(dir, "workspace.json")) {
		return false
	}
	if refs, _ := os.ReadDir(filepath.Join(dir, "refs")); len(refs) > 0 {
		return true
	}
	entries, _ := os.ReadDir(filepath.Join(dir, "assets"))
	for _, e := range entries {
		if e.Name() != "README.md" { // the workspace's own note on what goes there
			return true
		}
	}
	var state creatorState
	if data, err := os.ReadFile(filepath.Join(dir, "creator.json")); err == nil && json.Unmarshal(data, &state) == nil {
		return state.Concept != "" || len(state.Gens) > 0
	}
	return false
}

// unpackCharacter makes a pack the hero's current character in the creator. A character already
// there is saved as a pack first, so switching characters loses nothing.
func unpackCharacter(game Game, data []byte) (string, packManifest, string, error) {
	z, manifest, err := openPack(data)
	if err != nil {
		return "", manifest, "", err
	}
	dir := filepath.Join(game.charactersDir(), manifest.Hero)
	saved := ""
	if worthKeeping(dir) {
		if saved, _, err = savePack(game, dir); err != nil {
			return "", manifest, "", fmt.Errorf("the character already there could not be saved first: %v", err)
		}
	}
	staging := dir + ".unpacking"
	os.RemoveAll(staging)
	if err := unpackTo(z, staging); err != nil {
		os.RemoveAll(staging)
		return "", manifest, "", err
	}
	// The old folder moves aside before the new one moves in, and is deleted only then: a file
	// Windows holds open can stop a move, but never leaves the character half deleted.
	old := dir + ".old"
	os.RemoveAll(old)
	if fileExists(dir) {
		if err := os.Rename(dir, old); err != nil {
			os.RemoveAll(staging)
			return "", manifest, "", fmt.Errorf("the character there could not be moved aside (is a file of it open?): %v", err)
		}
	}
	if err := os.Rename(staging, dir); err != nil {
		os.Rename(old, dir)
		os.RemoveAll(staging)
		return "", manifest, "", err
	}
	os.RemoveAll(old)
	return dir, manifest, saved, nil
}

// packCommand is "dlmod pack <hero|folder> [file]": pack the creator's character for a hero, or
// a workspace folder, into one file.
func packCommand(ui prompter, game Game, args []string) error {
	if len(args) < 1 || len(args) > 2 {
		return errors.New("name a hero or a workspace folder, for example: dlmod pack haze, or dlmod pack meshy/haze rook.dlmodchar")
	}
	dir := args[0]
	if !fileExists(filepath.Join(dir, "workspace.json")) {
		hero, err := findHero(args[0])
		if err != nil {
			return fmt.Errorf("%s is neither a workspace folder nor a hero", args[0])
		}
		dir = filepath.Join(game.charactersDir(), hero)
		if !fileExists(filepath.Join(dir, "workspace.json")) {
			return fmt.Errorf("there is no character for %s in the creator yet", heroDisplayName(hero))
		}
	}
	data, manifest, err := exportPack(dir)
	if err != nil {
		return err
	}
	out := ""
	if len(args) == 2 {
		out = args[1]
	} else {
		out = slugify(manifest.Name) + "-" + manifest.Hero + packExt
	}
	if !strings.HasSuffix(strings.ToLower(out), packExt) {
		out += packExt
	}
	if err := os.WriteFile(out, data, 0o644); err != nil {
		return err
	}
	ui.say("Packed %s for %s into %s (%.1f MB).", manifest.Name, heroDisplayName(manifest.Hero), out, float64(len(data))/(1<<20))
	return nil
}

// unpackCommand is "dlmod unpack <file.dlmodchar>": make a pack the hero's character in the
// creator (dlmod studio, Create tab).
func unpackCommand(ui prompter, game Game, args []string) error {
	if len(args) != 1 {
		return errors.New("name a character pack, for example: dlmod unpack rook-haze.dlmodchar")
	}
	data, err := os.ReadFile(args[0])
	if err != nil {
		return err
	}
	dir, manifest, saved, err := unpackCharacter(game, data)
	if err != nil {
		return err
	}
	if saved != "" {
		ui.say("The character that was there is saved in %s.", saved)
	}
	ui.say("%s is now %s's character in the creator (%s). Install it with: dlmod apply %s", manifest.Name, heroDisplayName(manifest.Hero), dir, args[0])
	return nil
}

// applyPack installs a pack as dlmod apply installs a workspace folder, from a copy unpacked in a
// temporary folder.
func applyPack(ui *UI, store *Store, game Game, file string, opts options) error {
	data, err := os.ReadFile(file)
	if err != nil {
		return err
	}
	z, _, err := openPack(data)
	if err != nil {
		return err
	}
	tmp, err := os.MkdirTemp("", "dlmod-pack-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmp)
	dir := filepath.Join(tmp, "character")
	if err := unpackTo(z, dir); err != nil {
		return err
	}
	return applyCommand(ui, store, game, []string{dir}, opts)
}
