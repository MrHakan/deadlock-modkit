package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func installVPK(t *testing.T, game fakeGame, store *Store, name, path string) {
	t.Helper()
	file := game.write("downloads/"+name+".vpk", buildVPK(2, path))
	if err := store.install(quietUI(""), file, ""); err != nil {
		t.Fatal(err)
	}
}

func enabledNames(t *testing.T, store *Store) []string {
	t.Helper()
	entries, err := store.entries()
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, entry := range entries {
		if entry.Enabled() {
			names = append(names, entry.Name())
		}
	}
	return names
}

func TestProfilesSwitchBetweenSetsOfMods(t *testing.T) {
	game := newFakeGame(t)
	store, _ := openStore(game.Game)
	installVPK(t, game, store, "Alpha", "sounds/alpha.vsnd_c")
	installVPK(t, game, store, "Bravo", "sounds/bravo.vsnd_c")
	game.write("game/citadel/addons/pak03_dir.vpk", buildVPK(2, "sounds/hand.vsnd_c")) // added by hand

	if count, err := store.saveProfile("All"); err != nil || count != 3 {
		t.Fatalf("save All: %d %v", count, err)
	}
	entries, _ := store.entries()
	if err := store.disable(entries[1]); err != nil { // Bravo off
		t.Fatal(err)
	}
	if _, err := store.saveProfile("Ranked"); err != nil {
		t.Fatal(err)
	}
	var log bytes.Buffer
	ui := quietUI("")
	ui.out = &log
	if err := store.useProfile(ui, "all"); err != nil {
		t.Fatal(err)
	}
	if got := enabledNames(t, store); !reflect.DeepEqual(got, []string{"Alpha", "Bravo", "pak03 (added outside dlmod)"}) {
		t.Fatalf("after All: %v", got)
	}
	entries, _ = store.entries()
	if store.activeProfile(entries) != "All" || !strings.Contains(log.String(), "1 switched on, 0 switched off") {
		t.Fatalf("active %q, log %q", store.activeProfile(entries), log.String())
	}
	if err := store.useProfile(ui, "Ranked"); err != nil {
		t.Fatal(err)
	}
	if got := enabledNames(t, store); !reflect.DeepEqual(got, []string{"Alpha", "pak03 (added outside dlmod)"}) {
		t.Fatalf("after Ranked: %v", got)
	}
	if err := store.useProfile(ui, "Ranked"); err != nil || !strings.Contains(log.String(), "already in use") {
		t.Fatalf("using the same profile twice: %v %q", err, log.String())
	}

	// Removing a mod drops it from every profile; deleting a profile keeps the mods.
	entries, _ = store.entries()
	if err := store.remove(entries[0]); err != nil {
		t.Fatal(err)
	}
	store, _ = openStore(game.Game)
	if got := store.manifest.Profiles["All"]; len(got) != 2 {
		t.Fatalf("All should keep 2 mods, has %v", got)
	}
	if err := store.deleteProfile("ALL"); err != nil || !reflect.DeepEqual(store.profileNames(), []string{"Ranked"}) {
		t.Fatalf("delete: %v %v", err, store.profileNames())
	}
	if err := store.useProfile(ui, "nope"); err == nil {
		t.Fatal("an unknown profile must be an error")
	}
	if _, err := store.saveProfile("  "); err == nil {
		t.Fatal("a profile needs a name")
	}
}

// newRelease builds a GitLab release with its files.
func newRelease(tag, date string, links ...releaseLink) release {
	r := release{Tag: tag, ReleasedAt: date}
	r.Assets.Links = links
	return r
}

// releaseFile links a file the fake server serves, through the release's own address, which
// redirects to the file as GitLab's does.
func releaseFile(base, tag, name string) releaseLink {
	return releaseLink{Name: name, URL: base + "/download/" + name, DirectURL: base + "/-/releases/" + tag + "/downloads/" + name}
}

// fakeGitLab serves a release list and the files it names.
func fakeGitLab(t *testing.T, files map[string][]byte, releases func(base string) []release) *httptest.Server {
	t.Helper()
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("User-Agent") == "" {
			http.Error(w, "a User-Agent is required", http.StatusForbidden)
			return
		}
		if r.URL.Path == "/releases" {
			json.NewEncoder(w).Encode(releases(server.URL))
			return
		}
		if strings.HasPrefix(r.URL.Path, "/-/releases/") {
			name := r.URL.Path[strings.LastIndex(r.URL.Path, "/")+1:]
			http.Redirect(w, r, "/download/"+name, http.StatusFound)
			return
		}
		if data, ok := files[strings.TrimPrefix(r.URL.Path, "/download/")]; ok {
			w.Write(data)
			return
		}
		http.NotFound(w, r)
	}))
	saved := releasesURL
	releasesURL = server.URL + "/releases"
	t.Cleanup(func() { releasesURL = saved; server.Close() })
	return server
}

func sha256Of(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func TestUpdateInstallsTheHUDForTheGameBuild(t *testing.T) {
	game := newFakeGame(t) // build 6864
	hud := writeZip(t, filepath.Join(t.TempDir(), "hud.zip"), map[string][]byte{
		"pak01_dir.vpk": buildVPK(2, "panorama/layout/hud.vxml_c", "panorama/scripts/dci_hud.vts_c"),
	})
	zipData, _ := os.ReadFile(hud)
	const name6864 = "deadlock-counter-intelligence-6864-hud-en.zip"
	const name6870 = "deadlock-counter-intelligence-6870-hud-en.zip"
	// The checksum as sha256sum writes it.
	sidecar := []byte(sha256Of(zipData) + "  " + name6864 + "\n")
	fakeGitLab(t, map[string][]byte{name6864: zipData, name6864 + ".sha256": sidecar, name6870: []byte("newer")}, func(base string) []release {
		upcoming := newRelease("v0.5.0", "2099-01-01T00:00:00.000Z", releaseFile(base, "v0.5.0", name6864))
		upcoming.Upcoming = true
		// Listed out of order: the newest release by date wins.
		return []release{
			newRelease("v0.3.0", "2026-03-01T10:00:00.000Z", releaseFile(base, "v0.3.0", name6864), releaseFile(base, "v0.3.0", name6864+".sha256")),
			upcoming,
			newRelease("v0.4.1-beta", "2026-05-01T10:00:00.000Z", releaseFile(base, "v0.4.1-beta", name6864)),
			newRelease("v0.4.0", "2026-04-01T10:00:00.000Z", releaseFile(base, "v0.4.0", name6870), releaseFile(base, "v0.4.0", "dlmod-windows-amd64.exe")),
			newRelease("v0.2.0", "2026-02-01T10:00:00.000Z", releaseFile(base, "v0.2.0", name6864)),
		}
	})
	store, _ := openStore(game.Game)
	var log bytes.Buffer
	ui := quietUI("")
	ui.out = &log
	if err := store.updateHUD(ui); err != nil {
		t.Fatal(err)
	}
	if got := enabledNames(t, store); !reflect.DeepEqual(got, []string{"Deadlock Counter Intelligence v0.3.0"}) {
		t.Fatalf("installed %v; log %q", got, log.String())
	}
	if err := store.updateHUD(ui); err != nil || !strings.Contains(log.String(), "already installed") || len(game.addons()) != 1 {
		t.Fatalf("updating twice: %v %q %v", err, log.String(), game.addons())
	}

	os.WriteFile(filepath.Join(game.Citadel(), "steam.inf"), []byte("ClientVersion=6999\n"), 0o644)
	err := store.updateHUD(ui)
	if err == nil || !strings.Contains(err.Error(), "no HUD for game build 6999 yet; the newest release (v0.4.0) is for build 6870") {
		t.Fatalf("missing build: %v", err)
	}
}

func TestUpdateRejectsADownloadThatFailsItsChecksum(t *testing.T) {
	const name = "deadlock-counter-intelligence-6864-hud-en.zip"
	for _, sidecar := range map[string][]byte{"mismatch": []byte(strings.Repeat("0", 64)), "missing": nil} {
		game := newFakeGame(t)
		files := map[string][]byte{name: []byte("tampered")}
		fakeGitLab(t, files, func(base string) []release {
			links := []releaseLink{releaseFile(base, "v0.3.0", name)}
			if sidecar != nil {
				files[name+".sha256"] = sidecar
				links = append(links, releaseFile(base, "v0.3.0", name+".sha256"))
			}
			return []release{newRelease("v0.3.0", "2026-03-01T10:00:00Z", links...)}
		})
		store, _ := openStore(game.Game)
		err := store.updateHUD(quietUI(""))
		if sidecar == nil && (err == nil || !strings.Contains(err.Error(), "has no "+name+".sha256")) {
			t.Fatalf("without a checksum: %v", err)
		}
		if sidecar != nil && (err == nil || !strings.Contains(err.Error(), "checksum")) {
			t.Fatalf("with a wrong checksum: %v", err)
		}
		if len(game.addons()) != 0 {
			t.Fatalf("nothing may be installed: %v", game.addons())
		}
		if leftovers, _ := os.ReadDir(filepath.Join(game.StateDir(), "downloads")); len(leftovers) != 0 {
			t.Fatalf("the download must be deleted: %v", leftovers)
		}
	}
}

func TestUpdateExplainsAMissingProject(t *testing.T) {
	game := newFakeGame(t)
	server := httptest.NewServer(http.NotFoundHandler())
	defer server.Close()
	saved := releasesURL
	releasesURL = server.URL + "/releases"
	defer func() { releasesURL = saved }()
	store, _ := openStore(game.Game)
	if err := store.updateHUD(quietUI("")); err == nil || !strings.Contains(err.Error(), "may not be public") {
		t.Fatalf("got %v", err)
	}
}

func TestParseChecksum(t *testing.T) {
	const name = "hud.zip"
	hash := sha256Of([]byte("hud"))
	upper := strings.ToUpper(hash)
	utf16le := func(text string) []byte {
		out := []byte{0xFF, 0xFE}
		for _, r := range text {
			out = append(out, byte(r), byte(r>>8))
		}
		return out
	}
	good := map[string][]byte{
		"bare":            []byte(hash),
		"sha256sum":       []byte(hash + "  " + name + "\n"),
		"upper and CRLF":  []byte(upper + "\r\n"),
		"UTF-8 BOM":       append([]byte{0xEF, 0xBB, 0xBF}, hash...),
		"UTF-16":          utf16le(upper + "\r\n"),
		"Get-FileHash":    []byte("\r\nAlgorithm       Hash                                                                   Path\r\n---------       ----                                                                   ----\r\nSHA256          " + upper + "       C:\\dist\\" + name + "\r\n"),
		"several, named":  []byte(sha256Of([]byte("other")) + "  other.zip\n" + hash + "  " + name + "\n"),
		"star for binary": []byte(hash + " *" + name),
	}
	for label, data := range good {
		if got, err := parseChecksum(data, name); err != nil || got != hash {
			t.Errorf("%s: %q, %v", label, got, err)
		}
	}
	bad := map[string][]byte{
		"empty":            nil,
		"too short":        []byte(hash[:63]),
		"several, no name": []byte(hash + "\n" + sha256Of([]byte("other")) + "\n"),
	}
	for label, data := range bad {
		if got, err := parseChecksum(data, name); err == nil {
			t.Errorf("%s: accepted %q", label, got)
		}
	}
}

func TestProfilesInTheFullScreenView(t *testing.T) {
	game := newFakeGame(t)
	store, _ := openStore(game.Game)
	installVPK(t, game, store, "Alpha", "sounds/alpha.vsnd_c")
	installVPK(t, game, store, "Bravo", "sounds/bravo.vsnd_c")
	// f opens the dialog; Down + Enter picks "Save the mods that are on now…"; then type a name.
	ui := newTestTUI(t, game.Game, "\x1b[B", "\r")
	ui.handle([]byte("f"))
	if ui.mode != modePrompt || ui.purpose != "profile" {
		t.Fatalf("expected the profile name prompt, mode %d purpose %q", ui.mode, ui.purpose)
	}
	ui.handle([]byte("Ranked"))
	ui.handle([]byte("\r"))
	screen := ui.screen()
	if !strings.Contains(screen, "Saved profile 'Ranked' with 2 mods") || !strings.Contains(screen, "profile Ranked") {
		t.Fatalf("after saving:\n%s", screen)
	}
}

func TestOnlySecureAddressesAreDownloaded(t *testing.T) {
	for address, ok := range map[string]bool{
		"https://github.com/MrHakan/deadlock-modkit/releases/download/v0.3.0/x.zip": true,
		"http://127.0.0.1:8080/releases":                                            true,
		"http://example.com/x.zip":                                                  false,
		"file:///etc/passwd":                                                        false,
		"ftp://example.com/x.zip":                                                   false,
	} {
		if err := allowedURL(address); (err == nil) != ok {
			t.Errorf("allowedURL(%q) = %v", address, err)
		}
	}
}
