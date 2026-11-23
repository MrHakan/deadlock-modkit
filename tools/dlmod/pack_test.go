package main

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"
)

// characterFolder makes a creation: a design sheet, a model and the rigged model that replaced
// it in assets/, and a result that was not kept.
func characterFolder(t *testing.T, dir, name string) {
	t.Helper()
	w := &workspace{Hero: "haze", Name: "Haze", Game: &heroAssets{ID: "haze", Name: "Haze"}, Select: workspaceSelect{Name: name}, Steps: []workspaceStep{
		{ID: "design_sheet", Output: "assets/design_sheet.png", Kind: ""},
		{ID: "model", Output: "assets/model.glb", Kind: "model"},
		{ID: "rig", Output: "assets/model.glb"},
	}}
	if err := writeWorkspace(dir, w); err != nil {
		t.Fatal(err)
	}
	write := func(name, content string) {
		os.MkdirAll(filepath.Dir(filepath.Join(dir, name)), 0o755)
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("assets/design_sheet.png", "sheet")
	write("assets/model.glb", "rigged")
	write("gen/sheet.png", "sheet")
	write("gen/plain.glb", "plain")
	write("gen/rigged.glb", "rigged")
	write("gen/thumb.png", "thumb")
	write("gen/old.png", "old")
	state := creatorState{Hero: "haze", Concept: name + ", a test", Gens: []*creatorGen{
		{ID: "s1", Step: "design_sheet", Status: "SUCCEEDED", Files: []string{"sheet.png"}},
		{ID: "s0", Step: "design_sheet", Status: "SUCCEEDED", Files: []string{"old.png"}},
		{ID: "m1", Step: "model", Status: "SUCCEEDED", Files: []string{"plain.glb"}, Thumb: "thumb.png"},
		{ID: "r1", Step: "rig", Status: "SUCCEEDED", Files: []string{"rigged.glb"}},
	}, Chosen: map[string]string{"design_sheet": "s1", "model": "m1", "rig": "r1"}}
	data, _ := json.Marshal(state)
	write("creator.json", string(data))
}

func zipNames(t *testing.T, data []byte) []string {
	t.Helper()
	z, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, f := range z.File {
		names = append(names, f.Name)
	}
	sort.Strings(names)
	return names
}

func TestCharacterPackRoundTrip(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "haze")
	characterFolder(t, dir, "Rook")
	data, manifest, err := exportPack(dir)
	if err != nil {
		t.Fatal(err)
	}
	if manifest.Hero != "haze" || manifest.Name != "Rook" {
		t.Fatalf("manifest %+v", manifest)
	}
	// Kept files that assets/ holds unchanged go in once; the plain model the rig replaced in
	// assets/ goes in as itself; the result that was not kept stays out.
	want := []string{"README.md", "assets/README.md", "assets/design_sheet.png", "assets/model.glb", "creator.json", "gen/plain.glb", "gen/thumb.png", "pack.json", "prompts.md", "workspace.json"}
	if got := zipNames(t, data); !reflect.DeepEqual(got, want) {
		t.Fatalf("pack holds %v\nwant %v", got, want)
	}
	z, _, err := openPack(data)
	if err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(t.TempDir(), "unpacked")
	if err := unpackTo(z, out); err != nil {
		t.Fatal(err)
	}
	for name, content := range map[string]string{"gen/sheet.png": "sheet", "gen/rigged.glb": "rigged", "gen/plain.glb": "plain", "assets/model.glb": "rigged"} {
		if got, err := os.ReadFile(filepath.Join(out, name)); err != nil || string(got) != content {
			t.Errorf("%s: %q %v", name, got, err)
		}
	}
	var state creatorState
	raw, _ := os.ReadFile(filepath.Join(out, "creator.json"))
	json.Unmarshal(raw, &state)
	if len(state.Gens) != 3 || state.Chosen["rig"] != "r1" || state.Chosen["design_sheet"] != "s1" {
		t.Fatalf("creator state %+v", state)
	}
}

func TestPackRefusesStrangeFiles(t *testing.T) {
	for _, name := range []string{"../evil.txt", "assets/../../evil.txt", "/abs.txt", "dlmod.exe", "assets/x:y.png", "assets/nul", "gen/./a.png", "assets" + string(rune(92)) + "x.png"} { // 92: a backslash
		var buf bytes.Buffer
		z := zip.NewWriter(&buf)
		for _, n := range []string{"pack.json", "workspace.json", name} {
			f, _ := z.Create(n)
			if n == "pack.json" {
				f.Write([]byte(`{"format":1,"hero":"haze","name":"x"}`))
			} else {
				f.Write([]byte("{}"))
			}
		}
		z.Close()
		if _, _, err := openPack(buf.Bytes()); err == nil {
			t.Errorf("%q was accepted", name)
		}
	}
}

// Opening a pack keeps the character that was there as a saved pack, so switching loses nothing.
func TestUnpackCharacterSavesTheCurrentOne(t *testing.T) {
	game := newFakeGame(t)
	current := filepath.Join(game.charactersDir(), "haze")
	characterFolder(t, current, "Rook")
	other := filepath.Join(t.TempDir(), "other")
	characterFolder(t, other, "Shade")
	pack, _, err := exportPack(other)
	if err != nil {
		t.Fatal(err)
	}
	dir, manifest, saved, err := unpackCharacter(game.Game, pack)
	if err != nil {
		t.Fatal(err)
	}
	if dir != current || manifest.Name != "Shade" || filepath.Base(saved) != "rook.dlmodchar" {
		t.Fatalf("unpacked to %s as %q, saved %s", dir, manifest.Name, saved)
	}
	w, _ := readWorkspace(current)
	if w.Select.Name != "Shade" {
		t.Fatalf("the current character is %q", w.Select.Name)
	}
	back, _ := os.ReadFile(saved)
	if _, _, saved2, err := unpackCharacter(game.Game, back); err != nil || filepath.Base(saved2) != "shade.dlmodchar" {
		t.Fatalf("switching back: %v, saved %s", err, saved2)
	}
	if w, _ := readWorkspace(current); w.Select.Name != "Rook" {
		t.Fatalf("switched back to %q", w.Select.Name)
	}
	// A folder the Create tab only opened is not worth a pack.
	empty := filepath.Join(t.TempDir(), "empty")
	writeWorkspace(empty, &workspace{Hero: "haze", Game: &heroAssets{ID: "haze", Name: "Haze"}})
	if worthKeeping(empty) {
		t.Error("an untouched workspace was worth keeping")
	}
}

// A pack's workspace.json and creator.json are checked like its file names: a step or a result
// pointing outside the folder is refused or dropped.
func TestPackRefusesStrangePathsInside(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "haze")
	characterFolder(t, dir, "Rook")
	raw, _ := os.ReadFile(filepath.Join(dir, "creator.json"))
	var state creatorState
	json.Unmarshal(raw, &state)
	state.Gens[0].Files = []string{"../../outside.png"} // the kept design sheet
	raw, _ = json.Marshal(state)
	os.WriteFile(filepath.Join(dir, "creator.json"), raw, 0o644)
	data, _, err := exportPack(dir)
	if err != nil {
		t.Fatal(err)
	}
	z, _, err := openPack(data)
	if err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(t.TempDir(), "x", "haze")
	if err := unpackTo(z, out); err != nil {
		t.Fatal(err)
	}
	if fileExists(filepath.Join(out, "..", "outside.png")) || fileExists(filepath.Join(filepath.Dir(out), "..", "outside.png")) {
		t.Fatal("a result wrote outside the folder")
	}
	json.Unmarshal(mustRead(t, filepath.Join(out, "creator.json")), &state)
	for _, g := range state.Gens {
		if g.ID == "s1" {
			t.Fatal("the result with a strange file name was kept")
		}
	}

	w, _ := readWorkspace(dir)
	w.Steps[0].Output = "../escape.png"
	writeWorkspace(dir, w)
	if _, err := readWorkspace(dir); err == nil {
		t.Fatal("a step writing outside assets/ was accepted")
	}
}

func mustRead(t *testing.T, file string) []byte {
	t.Helper()
	data, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func TestLocalPath(t *testing.T) {
	for p, want := range map[string]bool{"assets/a.png": true, "gen/x.glb": true, "a/../b": false, "../a": false, "/a": false,
		"assets/nul.png": false, "assets/COM1": false, "assets/com10": true, "c:/x": false, "a" + string(rune(92)) + "b": false, "": false, "./a": false} { // 92: a backslash
		if got := localPath(p); got != want {
			t.Errorf("localPath(%q) = %v", p, got)
		}
	}
}
