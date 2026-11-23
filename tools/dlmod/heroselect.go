package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode"

	"github.com/MrHakan/deadlock-modkit/tools/dlmod/source2"
)

// The hero select screen, made for a replacement character: dlmod select <hero> sets the
// screen's background picture, the big name logo, the three tags and their colours, and can empty
// the hero's 3D scene so the picture shows on its own. Everything but the wording installs as one
// mod, "Hero select", rebuilt from game/dlmod/select/settings.json for every hero at once (the tag
// colours of all heroes live in one stylesheet). The name and tags are lines of the hero text, so
// they go into the "Custom text" mod.

const heroSelectName = "Hero select"

type selectSettings struct {
	Background string   `json:"background,omitempty"` // a copy of the picture in the select folder
	Name       string   `json:"name,omitempty"`       // the name shown, drawn as the logo unless Logo is set
	Logo       string   `json:"logo,omitempty"`       // a copy of an SVG logo in the select folder
	Tags       []string `json:"tags,omitempty"`
	Colors     []string `json:"colors,omitempty"` // tag backgrounds, #rrggbb, repeated over the tags
	HideScene  bool     `json:"hideScene,omitempty"`
}

func (s selectSettings) empty() bool {
	return s.Background == "" && s.Name == "" && s.Logo == "" && len(s.Tags) == 0 && len(s.Colors) == 0 && !s.HideScene
}

func (g Game) selectDir() string { return filepath.Join(g.StateDir(), "select") }

func loadSelectSettings(game Game) (map[string]selectSettings, error) {
	all := map[string]selectSettings{}
	data, err := os.ReadFile(filepath.Join(game.selectDir(), "settings.json"))
	if errors.Is(err, os.ErrNotExist) {
		return all, nil
	}
	if err == nil {
		err = json.Unmarshal(data, &all)
	}
	return all, err
}

func saveSelectSettings(game Game, all map[string]selectSettings) error {
	if err := os.MkdirAll(game.selectDir(), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(all, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(game.selectDir(), "settings.json"), data, 0o644)
}

// prefabAliases are hero select scene names no other file uses (maps/ui/hero_prefabs, build 6864).
var prefabAliases = map[string][]string{"dynamo": {"prof_dynamo"}}

// selectFileNames are the names a hero's select-screen files may go by: heroFileNames, the
// name players know written with underscores (grey_talon, mo_krill) and scene aliases.
func selectFileNames(hero string) []string {
	names := heroFileNames(hero)
	words := strings.FieldsFunc(strings.ToLower(heroDisplayName(hero)), func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsDigit(r) })
	return append(append(names, strings.Join(words, "_")), prefabAliases[hero]...)
}

// heroSelectFiles are where a hero's select-screen pieces live in the game.
type heroSelectFiles struct {
	Background string   // panorama/images/heroes/backgrounds/<name>_bg_psd.vtex_c
	Logos      []string // panorama/images/heroes/hero_names/<name>(_localized).vsvg_c
	Scene      string   // maps/ui/hero_prefabs/<name>.vpk, a file of its own next to the archive
}

const tagStyles = "panorama/styles/citadel_hero_tags.vcss_c"

func findSelectFiles(game Game, index *source2.VPKIndex, hero string) heroSelectFiles {
	var f heroSelectFiles
	for _, name := range selectFileNames(hero) {
		if bg := "panorama/images/heroes/backgrounds/" + name + "_bg_psd.vtex_c"; f.Background == "" && has(index, bg) {
			f.Background = bg
		}
		if len(f.Logos) == 0 {
			for _, logo := range []string{name + ".vsvg_c", name + "_localized.vsvg_c"} {
				if file := "panorama/images/heroes/hero_names/" + logo; has(index, file) {
					f.Logos = append(f.Logos, file)
				}
			}
		}
		if scene := "maps/ui/hero_prefabs/" + name + ".vpk"; f.Scene == "" {
			if _, err := os.Stat(filepath.Join(game.Citadel(), filepath.FromSlash(scene))); err == nil {
				f.Scene = scene
			}
		}
	}
	return f
}

func openSelectIndex(game Game) (*source2.VPKIndex, error) {
	index, err := source2.OpenVPKFiltered(filepath.Join(game.Citadel(), "pak01_dir.vpk"), func(name string) bool {
		return strings.HasPrefix(name, "panorama/images/heroes/backgrounds/") || strings.HasPrefix(name, "panorama/images/heroes/hero_names/") || name == tagStyles
	})
	if err != nil {
		return nil, fmt.Errorf("the game's interface files could not be read: %v", err)
	}
	return index, nil
}

// buildHeroSelect builds the Hero select mod's files for every hero with settings.
func buildHeroSelect(p prompter, game Game, all map[string]selectSettings) (map[string][]byte, error) {
	index, err := openSelectIndex(game)
	if err != nil {
		return nil, err
	}
	files := map[string][]byte{}
	var css strings.Builder
	heroes := make([]string, 0, len(all))
	for hero := range all {
		heroes = append(heroes, hero)
	}
	sort.Strings(heroes)
	for _, hero := range heroes {
		s := all[hero]
		where := findSelectFiles(game, index, hero)
		if s.Background != "" {
			if where.Background == "" {
				p.say("%s has no select-screen background in this game build; the picture is left out.", heroDisplayName(hero))
			} else {
				data, err := os.ReadFile(s.Background)
				if err != nil {
					return nil, err
				}
				picture, err := source2.DecodeImage(data)
				if err != nil {
					return nil, fmt.Errorf("%s: %v", s.Background, err)
				}
				picture = fadeLeft(picture, 0.38, 0.52) // the hero grid covers the left side, as on the game's own backgrounds
				original, err := index.Read(where.Background)
				if err != nil {
					return nil, err
				}
				if files[where.Background], err = source2.ReplaceTextureImage(original, picture); err != nil {
					return nil, err
				}
			}
		}
		if s.Logo != "" || s.Name != "" {
			svg, err := heroLogo(s)
			if err != nil {
				return nil, err
			}
			if len(where.Logos) == 0 {
				p.say("%s has no name logo in this game build; the logo is left out.", heroDisplayName(hero))
			}
			for _, file := range where.Logos {
				original, err := index.Read(file)
				if err != nil {
					return nil, err
				}
				if files[file], err = source2.ReplacePanoramaText(original, svg); err != nil {
					return nil, err
				}
			}
		}
		css.WriteString(tagColorRules(hero, s.Colors))
		if s.HideScene {
			if where.Scene == "" {
				p.say("%s has no 3D hero select scene in this game build; it is left as it is.", heroDisplayName(hero))
			} else {
				scene, err := hiddenScene(filepath.Join(game.Citadel(), filepath.FromSlash(where.Scene)))
				if err != nil {
					return nil, fmt.Errorf("%s: %v", where.Scene, err)
				}
				files[where.Scene] = scene
			}
		}
	}
	if css.Len() > 0 {
		original, err := index.Read(tagStyles)
		if err != nil {
			return nil, err
		}
		text, err := source2.PanoramaText(original)
		if err != nil {
			return nil, err
		}
		if files[tagStyles], err = source2.ReplacePanoramaText(original, text+css.String()); err != nil {
			return nil, err
		}
	}
	return files, nil
}

func heroLogo(s selectSettings) (string, error) {
	if s.Logo != "" {
		data, err := os.ReadFile(s.Logo)
		if err != nil {
			return "", err
		}
		if !strings.Contains(string(data), "<svg") {
			return "", fmt.Errorf("%s is not an SVG", s.Logo)
		}
		return string(data), nil
	}
	return textLogoSVG(s.Name)
}

// tagColorRules are stylesheet rules giving a hero's three tags their own backgrounds, with
// white or dark lettering, whichever reads better on each.
func tagColorRules(hero string, colors []string) string {
	if len(colors) == 0 {
		return ""
	}
	var sb strings.Builder
	for i := 1; i <= 3; i++ {
		c := colors[(i-1)%len(colors)]
		rgb, _ := parseHex(c)
		label := "#F5F5F5"
		if (0.299*rgb[0]+0.587*rgb[1]+0.114*rgb[2])/255 > 0.62 {
			label = "#1A1A1A"
		}
		fmt.Fprintf(&sb, ".hero_%s CitadelHeroTags .heroTag.Tag%d .tagBacker{background-color: %s;box-shadow: 0px 0px 6px %s;}", hero, i, c, c)
		fmt.Fprintf(&sb, ".hero_%s CitadelHeroTags .heroTag.Tag%d Label{color: %s;}", hero, i, label)
	}
	return sb.String()
}

// hiddenScene reads a hero select scene and returns it rebuilt with the hero and props hidden.
func hiddenScene(path string) ([]byte, error) {
	index, err := source2.OpenVPKFiltered(path, func(string) bool { return true })
	if err != nil {
		return nil, err
	}
	files := map[string][]byte{}
	for name := range index.Files {
		if files[name], err = index.Read(name); err != nil {
			return nil, err
		}
	}
	if _, _, err := source2.HideHeroScene(files); err != nil {
		return nil, err
	}
	return source2.WriteVPK(files), nil
}

// heroTextKeys finds a hero's name and tag lines in the English hero text: name keys hold the
// name players see (hero_tengu:n), tag keys the three tags (Citadel_Tengu_HeroTag_1..3).
func heroTextKeys(game Game, hero string) (nameKeys []string, tagKeys [3]string, err error) {
	names := map[string]bool{}
	for _, n := range selectFileNames(hero) {
		names[strings.ToLower(n)] = true
	}
	for _, file := range textFiles {
		data, err := os.ReadFile(filepath.Join(game.Citadel(), filepath.FromSlash(localizationFile(file, "english"))))
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return nil, tagKeys, err
		}
		for _, t := range parseTokens(string(data)) {
			key := strings.ToLower(t.Key)
			for _, form := range []string{"hero_%s:n", "hero_%s_search:n", "hero_%s_sort:n", "steam_rp_hero_%s"} {
				for n := range names {
					if key == fmt.Sprintf(form, n) {
						nameKeys = append(nameKeys, t.Key)
					}
				}
			}
			for n := range names {
				for i := 1; i <= 3; i++ {
					if key == fmt.Sprintf("citadel_%s_herotag_%d", n, i) {
						tagKeys[i-1] = t.Key
					}
				}
			}
		}
	}
	sort.Strings(nameKeys)
	return nameKeys, tagKeys, nil
}

// selectTextEdits sets (or, with empty values, clears) a hero's name and tag lines.
func selectTextEdits(game Game, hero string, name string, tags []string) (*textProject, error) {
	project, err := loadTextProject(game)
	if err != nil {
		return nil, err
	}
	nameKeys, tagKeys, err := heroTextKeys(game, hero)
	if err != nil {
		return nil, err
	}
	set := func(key, value string) {
		for _, file := range textFiles {
			path := localizationFile(file, "english")
			data, err := os.ReadFile(filepath.Join(game.Citadel(), filepath.FromSlash(path)))
			if err != nil || !strings.Contains(string(data), `"`+key+`"`) {
				continue
			}
			if value == "" {
				delete(project.Edits[path], key)
				if len(project.Edits[path]) == 0 {
					delete(project.Edits, path)
				}
				continue
			}
			if project.Edits[path] == nil {
				project.Edits[path] = map[string]string{}
			}
			project.Edits[path][key] = value
		}
	}
	for _, key := range nameKeys {
		set(key, name)
	}
	for i, key := range tagKeys {
		if key == "" {
			continue
		}
		value := ""
		if i < len(tags) {
			value = tags[i]
		}
		set(key, value)
	}
	return project, nil
}

// selectCommand: dlmod select <hero> [bg=picture] [name=TEXT] [logo=file.svg] [tags=A,B,C]
// [colors=#rrggbb,...] [scene=hide|show], or dlmod select <hero> restore.
func selectCommand(ui prompter, game Game, args []string) error {
	if len(args) == 0 {
		return errors.New("name a hero, for example: dlmod select ivy bg=select.png name=SHADE tags=Sneaky,Swift,Deadly colors=#7A3CFF,#FFB020 scene=hide")
	}
	if gameRunning() {
		return errRunning
	}
	hero, err := findHero(args[0])
	if err != nil {
		return err
	}
	all, err := loadSelectSettings(game)
	if err != nil {
		return err
	}
	s := all[hero]
	if len(args) == 1 {
		index, err := openSelectIndex(game)
		if err != nil {
			return err
		}
		where := findSelectFiles(game, index, hero)
		ui.say("%s's hero select screen:", heroDisplayName(hero))
		ui.say("  background  %s", orNone(where.Background))
		ui.say("  name logo   %s", orNone(strings.Join(where.Logos, ", ")))
		ui.say("  3D scene    %s", orNone(where.Scene))
		ui.say("Change it with: dlmod select %s bg=picture.png name=TEXT tags=A,B,C colors=#rrggbb,#rrggbb scene=hide", hero)
		return nil
	}
	textChanged := false
	if len(args) == 2 && args[1] == "restore" {
		textChanged = s.Name != "" || len(s.Tags) > 0
		delete(all, hero)
		s = selectSettings{}
	} else {
		for _, arg := range args[1:] {
			key, value, ok := strings.Cut(arg, "=")
			if !ok {
				return fmt.Errorf("%s is not KEY=VALUE (bg, name, logo, tags, colors, scene)", arg)
			}
			switch strings.ToLower(key) {
			case "bg", "background":
				if s.Background, err = keepCopy(game, value, hero+"_bg"); err != nil {
					return err
				}
			case "logo":
				if s.Logo, err = keepCopy(game, value, hero+"_logo"); err != nil {
					return err
				}
			case "name":
				s.Name = strings.TrimSpace(value)
				if s.Logo == "" && s.Name != "" {
					if _, err := textLogoSVG(s.Name); err != nil {
						return err
					}
				}
				textChanged = true
			case "tags":
				s.Tags = nil
				for _, t := range strings.Split(value, ",") {
					if t = strings.TrimSpace(t); t != "" {
						s.Tags = append(s.Tags, t)
					}
				}
				if len(s.Tags) > 3 {
					return errors.New("a hero has three tags")
				}
				textChanged = true
			case "colors", "colours":
				s.Colors = nil
				for _, c := range strings.Split(value, ",") {
					c = strings.TrimSpace(c)
					if _, ok := parseHex(c); !ok {
						return fmt.Errorf("%s is not a colour like #2467FF", c)
					}
					s.Colors = append(s.Colors, strings.ToUpper(c))
				}
			case "scene":
				switch value {
				case "hide":
					s.HideScene = true
				case "show":
					s.HideScene = false
				default:
					return errors.New("scene is hide or show")
				}
			default:
				return fmt.Errorf("unknown setting %s (bg, name, logo, tags, colors, scene)", key)
			}
		}
		all[hero] = s
		if s.empty() {
			delete(all, hero)
		}
	}
	files, err := buildHeroSelect(ui, game, all)
	if err != nil {
		return err
	}
	if err := saveSelectSettings(game, all); err != nil {
		return err
	}
	if err := installGeneratedMod(ui, game, heroSelectName, game.selectDir(), files); err != nil {
		return err
	}
	if err := dropFromImages(ui, game, files); err != nil {
		return err
	}
	if textChanged {
		project, err := selectTextEdits(game, hero, s.Name, s.Tags)
		if err != nil {
			return err
		}
		if err := project.save(game); err != nil {
			return err
		}
		if err := buildTextMod(ui, game, project); err != nil {
			return err
		}
	}
	ui.say("%s's hero select screen is updated.", heroDisplayName(hero))
	return nil
}

// keepCopy copies a file the user named into the select folder, so the mod can be rebuilt later.
func keepCopy(game Game, file, base string) (string, error) {
	data, err := os.ReadFile(file)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(game.selectDir(), 0o755); err != nil {
		return "", err
	}
	dest := filepath.Join(game.selectDir(), base+strings.ToLower(filepath.Ext(file)))
	return dest, os.WriteFile(dest, data, 0o644)
}

func orNone(s string) string {
	if s == "" {
		return "(none in this game build)"
	}
	return s
}

func has(index *source2.VPKIndex, file string) bool {
	_, ok := index.Files[file]
	return ok
}

// dropFromImages takes pictures the Hero select mod now holds (a background) out of the Custom
// images mod, so the two mods do not change the same file.
func dropFromImages(ui prompter, game Game, files map[string][]byte) error {
	project, err := loadImageProject(game)
	if err != nil {
		return err
	}
	var paths []string
	for name := range files {
		if _, ok := project.Replaced[name]; ok {
			paths = append(paths, name)
		}
	}
	if len(paths) == 0 {
		return nil
	}
	if _, err := project.restore(paths); err != nil {
		return err
	}
	store, err := openStore(game)
	if err != nil {
		return err
	}
	ui.say("%s moved from %s to %s.", strings.Join(paths, ", "), customImagesName, heroSelectName)
	return store.installReplacements(ui, project)
}
