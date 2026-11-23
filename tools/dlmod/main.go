// dlmod installs Deadlock mods without a mod manager: it copies .vpk mods into
// game/citadel/addons under free pakNN_dir.vpk names and makes gameinfo.gi load that folder.
package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"

	"github.com/MrHakan/deadlock-modkit/tools/dlmod/source2"
)

var version = "dev"

const usage = `dlmod - installs Deadlock mods without a mod manager

Usage:
  dlmod                        open the full-screen menu
  dlmod <mod.zip|.rar|.7z|.vpk>  install mods (on Windows, drag files onto dlmod.exe)
  dlmod install <file> [-name NAME]
  dlmod list                   show mods, conflicts and whether the game loads mods
  dlmod off <mod>              keep a mod but stop loading it
  dlmod on <mod>               load a switched-off mod again
  dlmod remove <mod>           delete a mod
  dlmod patch                  make the game load mods again after a game update
  dlmod unpatch                undo dlmod's gameinfo.gi change
  dlmod play                   patch if needed, then start Deadlock through Steam
  dlmod update                 download the Counter Intelligence HUD for your game build (GitHub)
  dlmod search <words> [-hero H] [-category C]
                               search the online mod catalog (deadlockmods.app, from GameBanana)
  dlmod get <id|GameBanana URL>  download and install a mod from GameBanana
  dlmod extract <path prefix> [folder]
                               copy files out of the game's archive, for example
                               dlmod extract models/heroes_staging/haze
  dlmod studio                 open the model studio in your browser: drop a .glb, .gltf or .obj
                               on a hero, check the fit and the rig, install
  dlmod model <hero> <file> [-keep gun] [-move gun=X,Y,Z] [-scale 1] [-yaw 0] [-zup] [-steady] [-no-arm-ik] [-usual-pose]
  dlmod image <game image> <picture> [...]  put pictures over game images (panorama/images/...vtex_c)
  dlmod text KEY=VALUE [...]           change lines of the English hero text (KEY= restores one)
  dlmod select <hero> [bg=PIC] [name=TEXT] [logo=SVG] [tags=A,B,C] [colors=#rrggbb,...] [scene=hide|frame]
                               the hero select screen: background, name logo, tags, tag colours, 3D scene
  dlmod icons <hero> [ICON=PIC ...]  list or replace ability icons (pictures become white glyphs)
  dlmod meshy <hero|all> [folder] [concept=...]
                               write a Meshy workspace: per-hero prompts for a full replacement character
  dlmod apply <workspace folder|pack.dlmodchar>  install the generated files of a hero's Meshy workspace or a character pack
  dlmod pack <hero|folder> [file]  pack a character (the creator's for a hero, or a workspace folder) into one .dlmodchar file
  dlmod unpack <file.dlmodchar>  make a character pack the hero's character in the creator
  dlmod prop <game model> <file> [particle ...] [-scale 1]  put a model in place of an ability prop (a thrown bomb)
  dlmod effects <hero> <#rrggbb|palette> [strength|name]  recolour a hero's ability and weapon particles (or: restore)
  dlmod paint <hero> [palette|#a,#b,#c|restore] [-skip NAME]  repaint a hero's ability particles, materials and textures
  dlmod profile                list profiles (named sets of mods)
  dlmod profile save <name>    remember which mods are on now
  dlmod profile use <name>     switch on exactly the mods in a profile
  dlmod profile delete <name>  forget a profile

Only "update", "search" and "get" (and b in the full-screen view) use the network, through
the curl that comes with Windows 10 and later. .rar and .7z archives are opened with Windows'
tar.exe or 7-Zip.

<mod> is the number shown by "dlmod list", a slot such as pak03, or part of the name.

Options:
  -game DIR   the Deadlock folder (found through Steam when omitted; DEADLOCK_DIR also works)
  -plain      use the simple numbered menu instead of the full-screen one
  -keep LIST  for model: hero parts to keep, by part of their names (default: weapons)
  -move P=X,Y,Z for model: move a kept part at rest, in game units (a weapon out of a wider body)
  -material PART=MAT for model: draw a part of the file with a game material (models/.../x.vmat)
  -scale N    for model: size on top of fitting the hero's height
  -yaw DEG    for model: turn the model around before fitting it
  -zup        for model: the file is Z-up (as Blender's .obj exports can be)
  -steady     for model: the body follows the hips, not the spine's lean and twist
  -no-arm-ik  for model: the arms follow the shoulders instead of being pulled to the gun by IK
  -usual-pose for model: bind the body in the hero's usual pose over its clips (a model made upright)
  -skip LIST  for paint: ability prop names to skip (comma-separated, e.g. ivy_thornbomb_model)
  -preview DIR  for paint: write before/after PNG contact sheets into DIR without installing
`

type options struct {
	game, name     string
	hero, category string
	help, plain    bool
	keep           string
	skip           string
	preview        string
	move           map[string][3]float64 // kept parts moved at rest, by part of their names
	material       map[string]string     // imported parts drawn with a game material, by part name
	scale, yaw     float64
	zup            bool
	tuning         studioOptions // for model: -steady, -no-arm-ik, -usual-pose
}

func parseArgs(argv []string) ([]string, options, error) {
	var opts options
	var rest []string
	for i := 0; i < len(argv); i++ {
		arg := argv[i]
		if !strings.HasPrefix(arg, "-") || arg == "-" {
			rest = append(rest, arg)
			continue
		}
		key, value, hasValue := strings.Cut(strings.TrimLeft(arg, "-"), "=")
		switch key {
		case "game", "name", "hero", "category", "keep", "skip", "preview", "move", "material", "scale", "yaw":
			if !hasValue {
				if i+1 >= len(argv) {
					return nil, opts, fmt.Errorf("%s needs a value", arg)
				}
				i++
				value = argv[i]
			}
			switch key {
			case "game":
				opts.game = value
			case "name":
				opts.name = value
			case "hero":
				opts.hero = value
			case "keep":
				opts.keep = value
			case "skip":
				if opts.skip != "" {
					opts.skip += "," + value
				} else {
					opts.skip = value
				}
			case "preview":
				opts.preview = value
			case "material":
				part, mat, ok := strings.Cut(value, "=")
				if !ok || part == "" || mat == "" {
					return nil, opts, fmt.Errorf("%s needs PART=MATERIAL", arg)
				}
				if opts.material == nil {
					opts.material = map[string]string{}
				}
				opts.material[part] = strings.TrimSuffix(mat, "_c")
			case "move":
				part, xyz, ok := strings.Cut(value, "=")
				nums := strings.Split(xyz, ",")
				if !ok || part == "" || len(nums) != 3 {
					return nil, opts, fmt.Errorf("%s needs PART=X,Y,Z", arg)
				}
				var v [3]float64
				for k, s := range nums {
					n, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
					if err != nil {
						return nil, opts, fmt.Errorf("%s needs numbers", arg)
					}
					v[k] = n
				}
				if opts.move == nil {
					opts.move = map[string][3]float64{}
				}
				opts.move[strings.ToLower(part)] = v
			case "scale", "yaw":
				number, err := strconv.ParseFloat(value, 64)
				if err != nil {
					return nil, opts, fmt.Errorf("%s needs a number", arg)
				}
				if key == "scale" {
					opts.scale = number
				} else {
					opts.yaw = number
				}
			default:
				opts.category = value
			}
		case "plain":
			opts.plain = true
		case "zup":
			opts.zup = true
		case "steady":
			opts.tuning.SteadyBody = true
		case "no-arm-ik":
			opts.tuning.NoArmIK = true
		case "usual-pose":
			opts.tuning.UsualPose = true
		case "h", "help":
			opts.help = true
		case "v", "version":
			fmt.Println("dlmod", version)
			os.Exit(0)
		default:
			return nil, opts, fmt.Errorf("unknown option %s", arg)
		}
	}
	return rest, opts, nil
}

var commands = map[string]bool{
	"install": true, "list": true, "status": true, "on": true, "enable": true, "off": true,
	"disable": true, "remove": true, "uninstall": true, "patch": true, "repair": true,
	"unpatch": true, "play": true, "help": true, "update": true, "profile": true, "profiles": true,
	"search": true, "get": true, "extract": true, "model": true, "image": true, "prop": true, "effects": true, "paint": true, "text": true, "select": true, "icons": true, "meshy": true, "apply": true, "studio": true, "pack": true, "unpack": true,
}

// palette holds escape sequences; every field is empty when colours are off (NO_COLOR, old consoles).
type palette struct {
	reset, bold, dim, green, red, yellow, cyan, invert, row string
	okBadge, offBadge, warnBadge, infoBadge                 string
}

func newPalette(color bool) palette {
	if !color {
		return palette{}
	}
	return palette{
		reset: "\x1b[0m", bold: "\x1b[1m", dim: "\x1b[90m", green: "\x1b[32m", red: "\x1b[91m",
		yellow: "\x1b[33m", cyan: "\x1b[36m", invert: "\x1b[7m", row: "\x1b[48;5;237m",
		okBadge: "\x1b[30;42m", offBadge: "\x1b[97;41m", warnBadge: "\x1b[30;43m", infoBadge: "\x1b[30;46m",
	}
}

func (p palette) paint(style, text string) string {
	if style == "" || p.reset == "" {
		return text
	}
	return style + text + p.reset
}

// UI is the plain console interface. Without a console every question takes its default.
type UI struct {
	in          *bufio.Reader
	out         io.Writer
	interactive bool
	eof         bool
	p           palette
	progressed  bool
}

func newUI() *UI {
	info, err := os.Stdin.Stat()
	interactive := err == nil && info.Mode()&os.ModeCharDevice != 0
	return &UI{in: bufio.NewReader(os.Stdin), out: os.Stdout, interactive: interactive, p: newPalette(colorSupported())}
}

func (u *UI) say(format string, args ...any) {
	u.endProgress()
	text := fmt.Sprintf(format, args...)
	switch {
	case strings.HasPrefix(text, "Conflict") || strings.HasPrefix(text, "Skipped") || strings.HasPrefix(text, "Warning"):
		text = u.p.paint(u.p.yellow, text)
	case strings.HasPrefix(text, "Installed") || strings.HasPrefix(text, "Updated"):
		text = u.p.paint(u.p.green, text)
	}
	fmt.Fprintln(u.out, text)
}

func (u *UI) ask(question string) string {
	u.endProgress()
	fmt.Fprint(u.out, question)
	line, err := u.in.ReadString('\n')
	if err != nil {
		u.eof = true
	}
	return cleanInput(line)
}

// cleanInput removes the quotes Windows adds around dragged paths.
func cleanInput(text string) string {
	text = strings.TrimSpace(text)
	text = strings.TrimSpace(strings.TrimPrefix(text, "& "))
	return strings.Trim(text, `"'`)
}

func (u *UI) confirm(question string, fallback bool) bool {
	if !u.interactive {
		return fallback
	}
	hint := " [y/N] "
	if fallback {
		hint = " [Y/n] "
	}
	return answerYes(u.ask(question+hint), fallback)
}

func answerYes(answer string, fallback bool) bool {
	switch strings.ToLower(strings.TrimSpace(answer)) {
	case "":
		return fallback
	case "y", "yes", "e", "evet":
		return true
	default:
		return false
	}
}

func (u *UI) choose(question string, options []string) (int, bool, bool) {
	if !u.interactive {
		return 0, true, true
	}
	u.say("%s", question)
	for i, option := range options {
		u.say("  %d  %s", i+1, option)
	}
	answer := u.ask("Enter a number, or \"all\": ")
	if strings.EqualFold(answer, "all") {
		return 0, true, true
	}
	index, err := strconv.Atoi(answer)
	if err != nil || index < 1 || index > len(options) {
		return 0, false, false
	}
	return index - 1, false, true
}

// progress draws one updating line for copies large enough to take a moment.
func (u *UI) progress(label string, done, total int64) {
	if !u.interactive || total < 8<<20 {
		return
	}
	u.progressed = true
	fmt.Fprintf(u.out, "\r  Copying %s  %3d%%  %s / %s ", truncate(label, 40), done*100/total, formatSize(done), formatSize(total))
}

func (u *UI) endProgress() {
	if u.progressed {
		fmt.Fprintln(u.out)
		u.progressed = false
	}
}

func main() {
	args, opts, err := parseArgs(os.Args[1:])
	ui := newUI()
	code := 0
	if err != nil {
		ui.say("Error: %v\n\n%s", err, usage)
		code = 2
	} else {
		code = run(ui, args, opts)
	}
	// A window opened by dragging files onto dlmod.exe, or by double-clicking it when something
	// went wrong, would close before the message can be read.
	dragged := len(args) > 0 && !commands[args[0]]
	if runtime.GOOS == "windows" && ui.interactive && !ui.eof && (dragged || (len(args) == 0 && code != 0)) {
		ui.ask("\nPress Enter to close.")
	}
	os.Exit(code)
}

func run(ui *UI, args []string, opts options) int {
	if opts.help || (len(args) > 0 && args[0] == "help") {
		fmt.Fprint(ui.out, usage)
		return 0
	}
	game, err := findGame(opts.game)
	for err != nil && ui.interactive && opts.game == "" && (len(args) == 0 || !commands[args[0]]) {
		ui.say("%v", err)
		folder := ui.ask("Paste the Deadlock folder (Steam > Deadlock > Manage > Browse local files), or press Enter to quit: ")
		if folder == "" {
			return 1
		}
		game, err = findGame(folder)
	}
	if err != nil {
		ui.say("Error: %v", err)
		return 1
	}
	if len(args) == 0 {
		if ui.interactive && !opts.plain {
			if err := runTUI(game); err == nil {
				return 0
			} else if !errors.Is(err, errNoTUI) {
				ui.say("Error: %v", err)
				return 1
			}
		}
		return menu(ui, game)
	}
	command, rest := args[0], args[1:]
	if !commands[command] {
		command, rest = "install", args
		if len(args) == 1 && isModelFile(args[0]) { // a model dropped on dlmod opens the studio with it
			command = "studio"
		}
		if len(args) == 1 && strings.HasSuffix(strings.ToLower(args[0]), packExt) { // a character pack dropped on dlmod installs it
			command = "apply"
		}
	}
	if err := execute(ui, game, command, rest, opts); err != nil {
		ui.say("%s", ui.p.paint(ui.p.red, "Error: "+err.Error()))
		return 1
	}
	return 0
}

var errRunning = errors.New("Deadlock is running; close it first")

func execute(ui *UI, game Game, command string, args []string, opts options) error {
	store, err := openStore(game)
	if err != nil {
		return err
	}
	if command != "list" && command != "status" && command != "studio" && !(command == "paint" && opts.preview != "") && gameRunning() {
		return errRunning
	}
	switch command {
	case "list", "status":
		return showStatus(ui, game, store)
	case "install":
		if len(args) == 0 {
			return errors.New("name the .zip or .vpk file to install")
		}
		for _, file := range args {
			if err := store.install(ui, file, opts.name); err != nil {
				return err
			}
		}
		return ensurePatched(ui, game)
	case "on", "enable", "off", "disable", "remove", "uninstall":
		if len(args) != 1 {
			return fmt.Errorf("name one mod, for example: dlmod %s 2", command)
		}
		entries, err := store.entries()
		if err != nil {
			return err
		}
		entry, err := resolve(entries, args[0])
		if err != nil {
			return err
		}
		return change(ui, store, entry, command)
	case "patch", "repair":
		changed, err := game.patch()
		if err == nil && changed {
			ui.say("Updated gameinfo.gi so the game loads mods from game/citadel/addons " +
				"(previous copy: game/dlmod/gameinfo.gi.bak).")
		} else if err == nil {
			ui.say("The game already loads mods from game/citadel/addons.")
		}
		return err
	case "unpatch":
		changed, err := game.unpatch()
		if err == nil {
			if changed {
				ui.say("gameinfo.gi is back to normal; the game no longer loads mods.")
			} else {
				ui.say("gameinfo.gi has no dlmod changes.")
			}
		}
		return err
	case "play":
		if err := ensurePatched(ui, game); err != nil {
			return err
		}
		ui.say("Starting Deadlock through Steam...")
		return launchGame()
	case "update":
		if err := store.updateHUD(ui); err != nil {
			return err
		}
		return ensurePatched(ui, game)
	case "profile", "profiles":
		return profileCommand(ui, store, args)
	case "search":
		mods, _, err := loadCatalog(game, false)
		if err != nil {
			return err
		}
		found := search(mods, filter{text: strings.Join(args, " "), hero: matchName(mods, opts.hero, true),
			category: matchName(mods, opts.category, false)})
		for i, mod := range found[:min(len(found), 25)] {
			ui.say("%3d. %-48s %-12s %-14s %7s  %s", i+1, truncate(mod.Name, 48), truncate(mod.Hero, 12),
				truncate(mod.Category, 14), formatCount(mod.Downloads), mod.ID)
		}
		ui.say("%s. Install one with: dlmod get <id>", plural(len(found), "%d mod found", "%d mods found"))
		return nil
	case "model":
		return modelCommand(ui, store, game, args, opts)
	case "select":
		return selectCommand(ui, game, args)
	case "icons":
		return iconsCommand(ui, store, game, args)
	case "meshy":
		return meshyCommand(ui, game, args)
	case "apply":
		if len(args) == 1 && strings.HasSuffix(strings.ToLower(args[0]), packExt) {
			return applyPack(ui, store, game, args[0], opts)
		}
		return applyCommand(ui, store, game, args, opts)
	case "pack":
		return packCommand(ui, game, args)
	case "unpack":
		return unpackCommand(ui, game, args)
	case "text":
		return textCommand(ui, game, args)
	case "image":
		return imageCommand(ui, store, game, args)
	case "prop":
		return propCommand(ui, store, game, args, opts)
	case "effects":
		return effectsCommand(ui, store, game, args)
	case "paint":
		return paintCommand(ui, store, game, args, opts)
	case "studio":
		return runStudio(ui, game, args...)
	case "extract":
		if len(args) < 1 || len(args) > 2 {
			return errors.New("name what to extract, for example: dlmod extract models/heroes_staging/haze")
		}
		return extractGameFiles(ui, game, args)
	case "get":
		if len(args) != 1 {
			return errors.New("name one mod, for example: dlmod get 742960 or its GameBanana address")
		}
		id, ok := parseModID(args[0])
		if !ok {
			return fmt.Errorf("%s is not a GameBanana mod number or address", args[0])
		}
		mod := RemoteMod{ID: id, Name: "GameBanana mod " + id}
		if mods, _, err := loadCatalog(game, false); err == nil {
			for _, candidate := range mods {
				if candidate.ID == id {
					mod = candidate
				}
			}
		}
		if err := store.installRemote(ui, mod); err != nil {
			return err
		}
		return ensurePatched(ui, game)
	}
	return fmt.Errorf("unknown command %s", command)
}

func profileCommand(ui *UI, store *Store, args []string) error {
	if len(args) == 0 || args[0] == "list" {
		names := store.profileNames()
		if len(names) == 0 {
			ui.say("No profiles yet. Switch on the mods you want together, then run: dlmod profile save <name>")
			return nil
		}
		entries, err := store.entries()
		if err != nil {
			return err
		}
		active := store.activeProfile(entries)
		for _, name := range names {
			marker := "  "
			if name == active {
				marker = "> "
			}
			ui.say("%s%s (%s)", marker, name, plural(len(store.manifest.Profiles[name]), "%d mod", "%d mods"))
		}
		return nil
	}
	if len(args) < 2 {
		return fmt.Errorf("name the profile, for example: dlmod profile %s Ranked", args[0])
	}
	name := strings.Join(args[1:], " ")
	switch args[0] {
	case "save":
		count, err := store.saveProfile(name)
		if err == nil {
			ui.say("Saved profile '%s' with %s.", strings.TrimSpace(name), plural(count, "%d mod", "%d mods"))
		}
		return err
	case "use", "load":
		return store.useProfile(ui, name)
	case "delete", "remove":
		if err := store.deleteProfile(name); err != nil {
			return err
		}
		ui.say("Deleted profile '%s'.", name)
		return nil
	}
	return fmt.Errorf("unknown profile command %s; use save, use, delete or list", args[0])
}

func change(p prompter, store *Store, entry *Entry, command string) error {
	name := entry.Name()
	switch command {
	case "on", "enable":
		if err := store.enable(entry); err != nil {
			return err
		}
		p.say("'%s' is on (pak%02d).", name, entry.Mod.Slot)
		return ensurePatched(p, store.game)
	case "off", "disable":
		if err := store.disable(entry); err != nil {
			return err
		}
		p.say("'%s' is off. It stays in dlmod's folder until you switch it on or remove it.", name)
	default:
		if err := store.remove(entry); err != nil {
			return err
		}
		p.say("Removed '%s'.", name)
	}
	return nil
}

func ensurePatched(p prompter, game Game) error {
	changed, err := game.patch()
	if err != nil {
		return err
	}
	if changed {
		p.say("Updated gameinfo.gi so the game loads mods from game/citadel/addons " +
			"(previous copy: game/dlmod/gameinfo.gi.bak).")
	}
	return nil
}

// outdated reports whether an interface mod was installed on an earlier game build.
func outdated(entry *Entry, current int) bool {
	mod := entry.Mod
	return mod != nil && entry.Enabled() && mod.GameVersion > 0 && current > 0 &&
		mod.GameVersion != current && changesInterface(entry.Paths)
}

func changesInterface(paths []string) bool {
	for _, path := range paths {
		if strings.HasPrefix(path, "panorama/") {
			return true
		}
	}
	return false
}

func formatSize(bytes int64) string {
	switch {
	case bytes >= 1<<30:
		return fmt.Sprintf("%.1f GB", float64(bytes)/(1<<30))
	case bytes >= 10<<20:
		return fmt.Sprintf("%d MB", bytes>>20)
	case bytes >= 1<<20:
		return fmt.Sprintf("%.1f MB", float64(bytes)/(1<<20))
	case bytes >= 1<<10:
		return fmt.Sprintf("%d KB", bytes>>10)
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}

func showStatus(ui *UI, game Game, store *Store) error {
	p := ui.p
	build := ""
	if v := game.Version(); v > 0 {
		build = fmt.Sprintf(" (build %d)", v)
	}
	ui.say("Deadlock: %s%s", game.Root, build)
	if _, state, err := game.readGameinfo(); err != nil {
		ui.say("%s %v", p.paint(p.warnBadge, " MODS ? "), err)
	} else if !state.Loaded {
		ui.say("%s gameinfo.gi does not include citadel/addons; game updates and \"Verify integrity\" reset it. "+
			"Run: dlmod patch", p.paint(p.offBadge, " MODS OFF "))
	} else if len(state.Missing) > 0 {
		ui.say("%s but gameinfo.gi lacks %s, so settings may be saved in addons. Run: dlmod patch",
			p.paint(p.warnBadge, " MODS ON "), strings.Join(state.Missing, ", "))
	} else {
		ui.say("%s loaded from game/citadel/addons", p.paint(p.okBadge, " MODS ON "))
	}

	entries, err := store.entries()
	if err != nil {
		return err
	}
	if len(entries) == 0 {
		ui.say("\nNo mods installed.")
		return nil
	}
	ui.say("\n  #  slot   mod")
	current := game.Version()
	var warnings []string
	for i, entry := range entries {
		slot := p.paint(p.green, fmt.Sprintf("pak%02d", entry.Slot))
		if !entry.Enabled() {
			slot = p.paint(p.dim, "off  ")
		}
		detail := ""
		switch {
		case entry.Err != nil:
			detail = p.paint(p.red, "  (unreadable: "+entry.Err.Error()+")")
		case len(entry.Paths) > 0:
			detail = p.paint(p.dim, "  changes "+describePaths(entry.Paths))
		}
		ui.say("%3d  %s  %s%s", i+1, slot, entry.Name(), detail)
		if outdated(entry, current) {
			warnings = append(warnings, fmt.Sprintf("Warning: '%s' was installed on build %d and the game is now on %d. "+
				"Interface mods can stop the game from starting after an update; if that happens, switch it off "+
				"and look for a newer version.", entry.Name(), entry.Mod.GameVersion, current))
		}
	}
	for _, clash := range conflicts(entries) {
		ui.say("\nConflict: pak%02d '%s' and pak%02d '%s' both change %s. Only one of them can take effect.",
			clash.A.Slot, clash.A.Name(), clash.B.Slot, clash.B.Name(), describePaths(clash.Paths))
	}
	for _, warning := range warnings {
		ui.say("\n%s", warning)
	}
	return nil
}

// menu is the numbered fallback for consoles that cannot show the full-screen interface.
func menu(ui *UI, game Game) int {
	for {
		store, err := openStore(game)
		if err != nil {
			ui.say("Error: %v", err)
			return 1
		}
		ui.say("")
		if err := showStatus(ui, game, store); err != nil {
			ui.say("Error: %v", err)
		}
		ui.say("\n  1  Install a mod (.zip or .vpk)\n  2  Switch a mod on or off\n  3  Remove a mod\n" +
			"  4  Repair after a game update (patch gameinfo.gi)\n  5  Start Deadlock\n" +
			"  6  Download the Counter Intelligence HUD for this game build\n  0  Exit")
		choice := ui.ask("\nChoose: ")
		if ui.eof {
			return 0
		}
		var failure error
		switch choice {
		case "1":
			if file := ui.ask("Drag the mod file into this window, then press Enter: "); file != "" {
				failure = execute(ui, game, "install", []string{file}, options{})
			}
		case "2", "3":
			entries, err := store.entries()
			if err != nil {
				failure = err
				break
			}
			if len(entries) == 0 {
				ui.say("No mods installed.")
				break
			}
			answer := ui.ask("Mod number: ")
			if answer == "" {
				break
			}
			entry, err := resolve(entries, answer)
			if err != nil {
				failure = err
				break
			}
			command := "remove"
			if choice == "2" {
				command = "off"
				if !entry.Enabled() {
					command = "on"
				}
			} else if !ui.confirm(fmt.Sprintf("Delete '%s'?", entry.Name()), false) {
				break
			}
			if gameRunning() {
				failure = errRunning
				break
			}
			failure = change(ui, store, entry, command)
		case "4":
			failure = execute(ui, game, "patch", nil, options{})
		case "5":
			failure = execute(ui, game, "play", nil, options{})
		case "6":
			failure = execute(ui, game, "update", nil, options{})
		case "0", "q", "exit":
			return 0
		}
		if failure != nil {
			ui.say("%s", ui.p.paint(ui.p.red, "Error: "+failure.Error()))
		}
	}
}

// matchName finds the hero or category a user typed, ignoring case; empty when nothing matches.
func matchName(mods []RemoteMod, text string, hero bool) string {
	if text == "" {
		return ""
	}
	categories, heroes := facets(mods)
	names := categories
	if hero {
		names = heroes
	}
	for _, name := range names {
		if strings.EqualFold(name, text) {
			return name
		}
	}
	for _, name := range names {
		if strings.HasPrefix(strings.ToLower(name), strings.ToLower(text)) {
			return name
		}
	}
	return text
}

// extractGameFiles copies the game files whose paths start with a prefix into a folder
// (dlmod-extract next to dlmod by default), keeping their paths.
func extractGameFiles(ui *UI, game Game, args []string) error {
	prefix := strings.Trim(strings.ReplaceAll(args[0], "\\", "/"), "/")
	folder := "dlmod-extract"
	if len(args) == 2 {
		folder = args[1]
	}
	index, err := source2.OpenVPKFiltered(filepath.Join(game.Citadel(), "pak01_dir.vpk"), func(name string) bool {
		return name == prefix || strings.HasPrefix(name, prefix+"/")
	})
	if err != nil {
		return err
	}
	if len(index.Files) == 0 {
		return fmt.Errorf("no game file starts with %s", prefix)
	}
	names := make([]string, 0, len(index.Files))
	for name := range index.Files {
		names = append(names, name)
	}
	sort.Strings(names)
	var total int64
	for _, name := range names {
		data, err := index.Read(name)
		if err != nil {
			return err
		}
		target := filepath.Join(folder, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(target, data, 0o644); err != nil {
			return err
		}
		total += int64(len(data))
	}
	abs, _ := filepath.Abs(folder)
	ui.say("Extracted %s (%s) into %s.", plural(len(names), "%d file", "%d files"), formatSize(total), abs)
	return nil
}
