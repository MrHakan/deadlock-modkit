package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
)

const appID = "1422450"

// gameRunning reports whether Deadlock is open. Tests replace it, so they pass on a computer
// where the game is running.
var gameRunning = deadlockRunning

// Game is a Deadlock installation; Root holds the game folder.
type Game struct{ Root string }

func (g Game) Citadel() string  { return filepath.Join(g.Root, "game", "citadel") }
func (g Game) Addons() string   { return filepath.Join(g.Citadel(), "addons") }
func (g Game) GameInfo() string { return filepath.Join(g.Citadel(), "gameinfo.gi") }

// StateDir holds dlmod's list of mods, switched-off mods and the gameinfo.gi backup.
// It is outside every game search path, so the game never reads it.
func (g Game) StateDir() string { return filepath.Join(g.Root, "game", "dlmod") }

// Version returns the client build from steam.inf, or 0 when it cannot be read.
func (g Game) Version() int {
	data, err := os.ReadFile(filepath.Join(g.Citadel(), "steam.inf"))
	if err != nil {
		return 0
	}
	for _, line := range strings.Split(string(data), "\n") {
		if value, ok := strings.CutPrefix(strings.TrimSpace(line), "ClientVersion="); ok {
			build, _ := strconv.Atoi(strings.TrimSpace(value))
			return build
		}
	}
	return 0
}

func isFile(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.Mode().IsRegular()
}

// gameRoot walks up from dir to the folder that contains game/citadel/gameinfo.gi.
func gameRoot(dir string) (string, bool) {
	dir, err := filepath.Abs(dir)
	if err != nil {
		return "", false
	}
	for {
		if isFile(filepath.Join(dir, "game", "citadel", "gameinfo.gi")) {
			return dir, true
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", false
		}
		dir = parent
	}
}

func findGame(explicit string) (Game, error) {
	if explicit != "" {
		if root, ok := gameRoot(explicit); ok {
			return Game{root}, nil
		}
		return Game{}, fmt.Errorf("%s is not a Deadlock folder (game/citadel/gameinfo.gi is missing)", explicit)
	}
	var candidates []string
	if dir := os.Getenv("DEADLOCK_DIR"); dir != "" {
		candidates = append(candidates, dir)
	}
	if exe, err := os.Executable(); err == nil {
		candidates = append(candidates, filepath.Dir(exe))
	}
	if dir, err := os.Getwd(); err == nil {
		candidates = append(candidates, dir)
	}
	for _, candidate := range candidates {
		if root, ok := gameRoot(candidate); ok {
			return Game{root}, nil
		}
	}
	for _, steam := range steamRoots() {
		for _, library := range steamLibraries(steam) {
			if root := deadlockIn(library); root != "" {
				return Game{root}, nil
			}
		}
	}
	return Game{}, errors.New("Deadlock was not found. Pass its folder with -game, for example " +
		`-game "D:\SteamLibrary\steamapps\common\Deadlock"`)
}

func steamRoots() []string {
	home, _ := os.UserHomeDir()
	switch runtime.GOOS {
	case "windows":
		var roots []string
		for _, key := range [][3]string{
			{"HKCU", `Software\Valve\Steam`, "SteamPath"},
			{"HKLM", `SOFTWARE\WOW6432Node\Valve\Steam`, "InstallPath"},
			{"HKLM", `SOFTWARE\Valve\Steam`, "InstallPath"},
		} {
			if value := registryValue(key[0], key[1], key[2]); value != "" {
				roots = append(roots, filepath.FromSlash(value))
			}
		}
		return append(roots, `C:\Program Files (x86)\Steam`, `C:\Program Files\Steam`)
	case "darwin":
		return []string{filepath.Join(home, "Library", "Application Support", "Steam")}
	default:
		return []string{
			filepath.Join(home, ".steam", "steam"),
			filepath.Join(home, ".steam", "root"),
			filepath.Join(home, ".local", "share", "Steam"),
			filepath.Join(home, ".var", "app", "com.valvesoftware.Steam", ".local", "share", "Steam"),
			filepath.Join(home, "snap", "steam", "common", ".local", "share", "Steam"),
		}
	}
}

var (
	vdfPath       = regexp.MustCompile(`"path"\s+"((?:[^"\\]|\\.)*)"`)
	acfInstallDir = regexp.MustCompile(`"installdir"\s+"([^"]+)"`)
)

// steamLibraries returns the Steam folder and every library listed in its libraryfolders.vdf.
func steamLibraries(root string) []string {
	libraries := []string{root}
	data, err := os.ReadFile(filepath.Join(root, "steamapps", "libraryfolders.vdf"))
	if err != nil {
		return libraries
	}
	for _, match := range vdfPath.FindAllStringSubmatch(string(data), -1) {
		libraries = append(libraries, filepath.FromSlash(strings.ReplaceAll(match[1], `\\`, `\`)))
	}
	return libraries
}

func deadlockIn(library string) string {
	apps := filepath.Join(library, "steamapps")
	folder := "Deadlock"
	if data, err := os.ReadFile(filepath.Join(apps, "appmanifest_"+appID+".acf")); err == nil {
		if match := acfInstallDir.FindStringSubmatch(string(data)); match != nil {
			folder = match[1]
		}
	}
	root := filepath.Join(apps, "common", folder)
	if isFile(filepath.Join(root, "game", "citadel", "gameinfo.gi")) {
		return root
	}
	return ""
}

func launchGame() error { return startURL("steam://rungameid/" + appID) }

// startURL opens an address with the system's handler (Steam for steam:// addresses).
func startURL(url string) error {
	switch runtime.GOOS {
	case "windows":
		return exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		return exec.Command("open", url).Start()
	default:
		return exec.Command("xdg-open", url).Start()
	}
}

// exeNameIs compares a NUL-terminated UTF-16 process name with a lowercase ASCII name without
// converting it, so checking every running process allocates nothing.
func exeNameIs(name []uint16, want string) bool {
	if len(name) <= len(want) || name[len(want)] != 0 {
		return false
	}
	for i := 0; i < len(want); i++ {
		char := name[i]
		if char >= 'A' && char <= 'Z' {
			char += 'a' - 'A'
		}
		if char != uint16(want[i]) {
			return false
		}
	}
	return true
}
