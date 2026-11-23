package main

import (
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"unicode"

	"github.com/MrHakan/deadlock-modkit/tools/dlmod/source2"
)

// What a hero is made of in the game, gathered for a Meshy workspace (meshy_command.go): the
// files a replacement character has to provide, with their sizes, and the hero's own words
// (abilities, role, weapon), which make each generation prompt specific to the hero.

type assetFile struct {
	Path   string `json:"path"`
	Label  string `json:"label,omitempty"`
	Width  int    `json:"width,omitempty"`
	Height int    `json:"height,omitempty"`
}

type abilityInfo struct {
	Key   string `json:"key"`
	Name  string `json:"name"`
	Quip  string `json:"quip,omitempty"`
	Desc  string `json:"description,omitempty"`
	Icon  string `json:"icon,omitempty"`  // its icon file, paired by name (check in game)
	Slug  string `json:"slug"`            // short name for asset files
	Props string `json:"props,omitempty"` // ability models that look like this ability's
}

type heroAssets struct {
	ID        string        `json:"id"`
	Name      string        `json:"name"`
	Role      string        `json:"role,omitempty"`
	Playstyle string        `json:"playstyle,omitempty"`
	Weapon    string        `json:"weapon,omitempty"`
	Tags      []string      `json:"tags,omitempty"`
	Abilities []abilityInfo `json:"abilities"`

	Model       string   `json:"model"`
	ModelFolder string   `json:"modelFolder"`
	Height      float64  `json:"height,omitempty"` // units, from the model's bounds
	Skeleton    string   `json:"skeleton,omitempty"`
	ClipFolder  string   `json:"clipFolder,omitempty"`
	Clips       int      `json:"clips,omitempty"`
	KeyClips    []string `json:"keyClips,omitempty"`
	Masks       []string `json:"boneMasks,omitempty"`

	Images     []assetFile     `json:"images"`
	Icons      []assetFile     `json:"icons"`
	Select     heroSelectFiles `json:"select"`
	Props      []string        `json:"abilityModels,omitempty"`
	Particles  []string        `json:"particleFolders,omitempty"`
	Effects    int             `json:"effects"`
	NameKeys   []string        `json:"nameKeys,omitempty"`
	TagKeys    []string        `json:"tagKeys,omitempty"`
	TextSource string          `json:"textSection,omitempty"`
}

// openWorkspaceIndex indexes every file a workspace looks at, in one pass over the archive.
func openWorkspaceIndex(game Game) (*source2.VPKIndex, error) {
	return source2.OpenVPKFiltered(filepath.Join(game.Citadel(), "pak01_dir.vpk"), func(n string) bool {
		switch {
		case strings.HasPrefix(n, "models/"):
			return strings.HasSuffix(n, ".vmdl_c") || strings.HasSuffix(n, ".vnmskel_c") || strings.HasSuffix(n, ".vnmclip_c")
		case n == heroesVdata || n == abilitiesVdata:
			return true
		case strings.HasPrefix(n, "particles/"):
			return strings.HasSuffix(n, ".vpcf_c")
		case strings.HasPrefix(n, "panorama/images/"):
			return strings.HasSuffix(n, ".vtex_c") || strings.HasSuffix(n, ".vsvg_c")
		}
		return false
	})
}

var (
	htmlTag     = regexp.MustCompile(`<[^>]*>`)
	placeholder = regexp.MustCompile(`\{[a-z]:[^}]*\}`)
	spaces      = regexp.MustCompile(`\s+`)
)

func plainText(s string) string {
	s = strings.ReplaceAll(s, "<br>", " ")
	s = placeholder.ReplaceAllString(htmlTag.ReplaceAllString(s, ""), "X")
	return strings.TrimSpace(spaces.ReplaceAllString(s, " "))
}

func words(s string) []string {
	return strings.FieldsFunc(strings.ToLower(s), func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsDigit(r) })
}

// iconFor finds an ability icon by its base name: among the hero's icons, or anywhere under the
// ability icons (some heroes keep theirs in a folder of another name), and adds it to the list.
func iconFor(index *source2.VPKIndex, a *heroAssets, icon string) string {
	if icon == "" {
		return ""
	}
	// icon is the compiled file (from the game's data) or the base of its name (from the table).
	match := func(name string) bool {
		if strings.Contains(icon, "/") {
			return name == icon
		}
		return strings.HasPrefix(name, "panorama/images/hud/abilities/") && path.Base(name) == icon+"_psd.vtex_c"
	}
	for _, f := range a.Icons {
		if match(f.Path) {
			return f.Path
		}
	}
	for name := range index.Files {
		if match(name) {
			f := assetFile{Path: name, Label: "ability icon"}
			if head, err := index.ReadHead(name, 16<<10); err == nil {
				if info, _, err := source2.TextureHeader(head); err == nil {
					f.Width, f.Height = info.Width, info.Height
				}
			}
			a.Icons = append(a.Icons, f)
			return name
		}
	}
	return ""
}

// heroText reads a hero's abilities, role, playstyle, weapon and tags from the English hero text.
// The abilities come from the game's data or heroAbilities (slots and icons) when they know the
// hero, else they are the lines with a _quip line beside them in the text's section for the hero.
func heroText(game Game, index *source2.VPKIndex, hero string, a *heroAssets) error {
	data, err := os.ReadFile(filepath.Join(game.Citadel(), filepath.FromSlash(localizationFile("citadel_heroes", "english"))))
	if errors.Is(err, os.ErrNotExist) {
		return nil // no hero text: the prompts go without the hero's role and abilities
	}
	if err != nil {
		return err
	}
	tokens := parseTokens(string(data))
	values := map[string]string{}
	for _, t := range tokens {
		values[strings.ToLower(t.Key)] = t.Value
	}
	names := map[string]bool{}
	for _, n := range selectFileNames(hero) {
		names[strings.ToLower(n)] = true
		for _, w := range words(n) {
			names[w] = true
		}
	}
	inSection := func(section string) bool {
		for _, w := range words(section) {
			if names[w] {
				return true
			}
		}
		return false
	}
	for _, n := range selectFileNames(hero) {
		n = strings.ToLower(n)
		for key, field := range map[string]*string{"hero_" + n + "_role": &a.Role, "hero_" + n + "_playstyle": &a.Playstyle, "citadel_weapon_" + n + "_set": &a.Weapon} {
			if v := values[key]; v != "" && *field == "" {
				*field = plainText(v)
			}
		}
	}
	slugOf := func(key string) string {
		slug := strings.TrimPrefix(strings.TrimPrefix(strings.ToLower(key), "citadel_ability_"), "ability_")
		for _, n := range selectFileNames(hero) {
			slug = strings.TrimPrefix(slug, strings.ToLower(n)+"_")
		}
		return slug
	}
	if keys, icons, fallback, source := heroAbilityKeys(index, hero); source != "" {
		// The game's data (or the table) knows the slots and icons; the text adds what each
		// ability does.
		for i, key := range keys {
			lower := strings.ToLower(key)
			name := plainText(values[lower])
			if name == "" {
				name = fallback[i]
			}
			if name == "" {
				name = key
			}
			a.Abilities = append(a.Abilities, abilityInfo{
				Key: key, Name: name, Quip: plainText(values[lower+"_quip"]), Desc: plainText(values[lower+"_desc"]),
				Slug: slugOf(key), Icon: iconFor(index, a, icons[i]),
			})
		}
		a.TextSource = source
	}
	seen := map[string]bool{}
	for _, t := range tokens {
		key := strings.ToLower(t.Key)
		if len(a.Abilities) > 0 || !inSection(t.Section) || strings.Contains(key, "/") || seen[key] || values[key+"_quip"] == "" {
			continue
		}
		seen[key] = true
		if a.TextSource == "" {
			a.TextSource = t.Section
		}
		a.Abilities = append(a.Abilities, abilityInfo{
			Key: t.Key, Name: plainText(t.Value), Quip: plainText(values[key+"_quip"]), Desc: plainText(values[key+"_desc"]), Slug: slugOf(t.Key),
		})
	}
	nameKeys, tagKeys, err := heroTextKeys(game, hero)
	if err != nil {
		return err
	}
	a.NameKeys = nameKeys
	for _, k := range tagKeys {
		if k != "" {
			a.TagKeys = append(a.TagKeys, k)
			a.Tags = append(a.Tags, values[strings.ToLower(k)])
		}
	}
	return nil
}

// pairIcons gives each ability the icon whose file name shares a word with it, then the rest in
// order. The game's icon names do not always follow the ability names (Ivy's Entangling Thorns
// is tengu_storm_flask), so the pairing is a guess the workspace asks to check.
func pairIcons(abilities []abilityInfo, icons []assetFile, hero string) {
	base := func(icon string) string {
		b := strings.TrimSuffix(path.Base(icon), "_psd.vtex_c")
		for _, n := range selectFileNames(hero) {
			b = strings.TrimPrefix(b, strings.ToLower(n)+"_")
		}
		return b
	}
	used := map[string]bool{}
	for i := range abilities {
		for _, icon := range icons {
			b := base(icon.Path)
			if used[icon.Path] {
				continue
			}
			match := strings.Contains(abilities[i].Slug, b) || strings.Contains(b, abilities[i].Slug)
			for _, w := range words(b) {
				if len(w) > 3 && (strings.Contains(abilities[i].Slug, w) || strings.Contains(strings.ToLower(abilities[i].Name), w)) {
					match = true
				}
			}
			if match {
				abilities[i].Icon = icon.Path
				used[icon.Path] = true
				break
			}
		}
	}
	for i := range abilities {
		if abilities[i].Icon != "" {
			continue
		}
		for _, icon := range icons {
			if !used[icon.Path] {
				abilities[i].Icon = icon.Path
				used[icon.Path] = true
				break
			}
		}
	}
}

// keyClipNames are the clips worth knowing about when animating a replacement character.
var keyClipNames = []string{"weapon_idle", "out_of_combat_idle", "weapon_run_n", "weapon_crouch_idle", "melee_quick_1", "melee_hit", "reload_stand_idle", "weapon_shoot", "jump", "slide"}

// gatherHero collects everything a workspace needs about one hero.
func gatherHero(game Game, index *source2.VPKIndex, hero string) (*heroAssets, error) {
	a := &heroAssets{ID: hero, Name: heroDisplayName(hero)}
	model, err := heroModelPath(index, hero)
	if err != nil {
		return nil, err
	}
	a.Model, a.ModelFolder = model, path.Dir(model)+"/"
	if raw, err := index.Read(model + "_c"); err == nil {
		if m, err := source2.LoadModel(raw); err == nil {
			if lo, hi, err := m.Bounds(source2.ImportOptions{}); err == nil {
				a.Height = hi[2] - lo[2]
			}
			if refs := m.Data.Get("m_vecNmSkeletonRefs"); refs != nil && len(refs.Items) > 0 {
				a.Skeleton = refs.Items[0].S
			}
		}
	}
	if a.Skeleton != "" {
		if i := strings.Index(a.Skeleton, "dmx/"); i > 0 {
			a.ClipFolder = a.Skeleton[:i] + "clips/"
		}
		if raw, err := index.Read(a.Skeleton + "_c"); err == nil {
			a.Masks = maskIDs(raw)
		}
	}
	if a.ClipFolder != "" {
		have := map[string]bool{}
		for name := range index.Files {
			if strings.HasPrefix(name, a.ClipFolder) && strings.HasSuffix(name, ".vnmclip_c") {
				a.Clips++
				have[strings.TrimSuffix(strings.TrimPrefix(name, a.ClipFolder), ".vnmclip_c")] = true
			}
		}
		for _, c := range keyClipNames {
			if have[c] {
				a.KeyClips = append(a.KeyClips, c)
			}
		}
	}
	s := &studio{game: game, images: index, particles: index}
	images, err := s.heroImages(hero)
	if err != nil {
		return nil, err
	}
	for _, img := range images {
		f := assetFile{Path: img.Path, Label: img.Label}
		if head, err := index.ReadHead(img.Path, 16<<10); err == nil {
			if info, _, err := source2.TextureHeader(head); err == nil {
				f.Width, f.Height = info.Width, info.Height
			}
		}
		if img.Group == "Abilities" {
			a.Icons = append(a.Icons, f)
		} else {
			a.Images = append(a.Images, f)
		}
	}
	a.Select = findSelectFiles(game, index, hero)
	abilities, weapon, err := s.heroParticles(hero)
	if err != nil {
		return nil, err
	}
	a.Effects = len(abilities) + len(weapon)
	folders := map[string]bool{}
	for _, p := range append(abilities, weapon...) {
		folders[path.Dir(p)+"/"] = true
	}
	for f := range folders {
		a.Particles = append(a.Particles, f)
	}
	sort.Strings(a.Particles)
	names := selectFileNames(hero)
	for name := range index.Files {
		if !strings.HasPrefix(name, "models/abilities/") || !strings.HasSuffix(name, ".vmdl_c") {
			continue
		}
		b := path.Base(name)
		for _, n := range names {
			if strings.HasPrefix(b, strings.ToLower(n)+"_") {
				a.Props = append(a.Props, strings.TrimSuffix(name, "_c"))
				break
			}
		}
	}
	sort.Strings(a.Props)
	if err := heroText(game, index, hero, a); err != nil {
		return nil, err
	}
	if a.TextSource != abilitiesFromGame && a.TextSource != abilitiesFromTable {
		pairIcons(a.Abilities, a.Icons, hero)
	}
	var data *gameData
	if a.TextSource == abilitiesFromGame {
		data, _ = loadGameData(index)
	}
	for i := range a.Abilities {
		var props []string
		var named map[string]bool // the ability models the game's data names for this ability
		if data != nil && data.abilities[a.Abilities[i].Key] != nil {
			named = map[string]bool{}
			for _, m := range data.abilities[a.Abilities[i].Key].Models {
				named[m] = true
			}
		}
		for _, p := range a.Props {
			if named[p] || propOfAbility(p, a.Abilities[i], hero) {
				props = append(props, p)
			}
		}
		a.Abilities[i].Props = strings.Join(props, ", ")
	}
	return a, nil
}

// maskIDs lists a skeleton's bone masks (the layers the game plays melee, reloads and casts on).
func maskIDs(raw []byte) []string {
	r, err := source2.DecodeResource(raw)
	if err != nil {
		return nil
	}
	root, _, err := source2.ReadKV3(r.Block("DATA"))
	if err != nil {
		return nil
	}
	var ids []string
	if defs := root.Get("m_maskDefinitions"); defs != nil {
		for _, d := range defs.Items {
			if id := d.Get("m_ID"); id != nil {
				ids = append(ids, id.S)
			}
		}
	}
	return ids
}

func slugify(s string) string { return strings.Join(words(s), "-") }

func (a *heroAssets) image(suffix string) (assetFile, bool) {
	for _, f := range a.Images {
		if strings.HasSuffix(f.Path, suffix+".vtex_c") {
			return f, true
		}
	}
	return assetFile{}, false
}

func describeSize(f assetFile) string {
	if f.Width == 0 {
		return ""
	}
	return fmt.Sprintf("%dx%d", f.Width, f.Height)
}

// summary is what an ability does in a line: its quip, or the start of its description.
func (ab abilityInfo) summary() string {
	if ab.Quip != "" {
		return ab.Quip
	}
	return clip(ab.Desc, 120)
}

// propOfAbility guesses whether an ability model belongs to an ability: a word of the model's
// name (not the hero's or "model") starts like a word of the ability's name, key or text, as
// ivy_thornbomb_model and "a patch of choking thorns" do.
func propOfAbility(model string, ab abilityInfo, hero string) bool {
	skip := map[string]bool{"model": true, "models": true, "fx": true, "ability": true, "abilities": true}
	for _, n := range selectFileNames(hero) {
		for _, w := range words(n) {
			skip[w] = true
		}
	}
	text := words(ab.Key + " " + ab.Name + " " + ab.Quip + " " + ab.Desc)
	for _, w := range words(strings.TrimSuffix(path.Base(model), ".vmdl")) {
		if skip[w] || len(w) < 4 || strings.HasPrefix(w, "abilit") {
			continue
		}
		for _, t := range text {
			n := 0
			for n < len(w) && n < len(t) && w[n] == t[n] {
				n++
			}
			if n >= 5 || (n >= 4 && n == len(w) && n == len(t)) {
				return true
			}
		}
	}
	return false
}
