package main

import (
	"bytes"
	"errors"
	"fmt"
	"image/png"
	"net/http"
	"path"
	"sort"
	"strconv"
	"strings"
	"unicode"

	"github.com/MrHakan/deadlock-modkit/tools/dlmod/source2"
)

// The studio's Abilities tab shows a hero's four abilities the way the game will show them with
// the mods that are switched on, without starting the game: each ability's name, text and
// values, its icon, the colours of its effects and the models it throws or places, from the
// game's own files and from the mod that replaces each of them, side by side.

// workspaceIndex indexes the files the Create and Abilities tabs read, once.
func (s *studio) workspaceIndex() (*source2.VPKIndex, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.workIndex == nil {
		index, err := openWorkspaceIndex(s.game)
		if err != nil {
			return nil, fmt.Errorf("the game's files could not be read: %v", err)
		}
		s.workIndex = index
	}
	return s.workIndex, nil
}

// installedFile is a game file that a switched-on mod replaces.
type installedFile struct {
	vpk string // the mod's pakNN_dir.vpk
	mod string
}

// installedFiles maps each game file the switched-on mods change to the mod the game takes it
// from. When several change one file, the lowest slot wins: the game reads pak01 first.
func installedFiles(game Game) (map[string]installedFile, error) {
	store, err := openStore(game)
	if err != nil {
		return nil, err
	}
	entries, err := store.entries()
	if err != nil {
		return nil, err
	}
	sort.Slice(entries, func(a, b int) bool { return entries[a].Slot < entries[b].Slot })
	out := map[string]installedFile{}
	for _, e := range entries {
		if !e.Enabled() || e.Err != nil || len(e.Files) == 0 || !strings.HasPrefix(e.Files[0], game.Addons()) {
			continue
		}
		for _, p := range e.Paths {
			if _, taken := out[p]; !taken {
				out[p] = installedFile{e.Files[0], e.Name()}
			}
		}
	}
	return out, nil
}

// modReader reads game files from the mods that replace them, opening each mod once.
type modReader struct {
	files  map[string]installedFile
	opened map[string]*source2.VPKIndex
}

func newModReader(game Game) (*modReader, error) {
	files, err := installedFiles(game)
	if err != nil {
		return nil, err
	}
	return &modReader{files: files, opened: map[string]*source2.VPKIndex{}}, nil
}

// read gives a mod's version of a game file, or nil when no switched-on mod replaces it.
func (m *modReader) read(name string) ([]byte, string, error) {
	file, ok := m.files[name]
	if !ok {
		return nil, "", nil
	}
	index := m.opened[file.vpk]
	if index == nil {
		var err error
		if index, err = source2.OpenVPK(file.vpk); err != nil {
			return nil, file.mod, err
		}
		m.opened[file.vpk] = index
	}
	data, err := index.Read(name)
	return data, file.mod, err
}

type abilityColor struct {
	Color [3]uint8 `json:"color"`
	Uses  int      `json:"uses"`
}

type abilityModel struct {
	Path string `json:"path"`
	Mod  string `json:"mod,omitempty"` // the mod that replaces it, if one does
}

type abilityView struct {
	Slot        int                 `json:"slot"`
	Key         string              `json:"key"`
	Name        string              `json:"name"`
	Quip        string              `json:"quip,omitempty"`
	Description string              `json:"description,omitempty"`
	Values      []map[string]string `json:"values"`
	Icon        string              `json:"icon,omitempty"`
	IconMod     string              `json:"iconMod,omitempty"`
	Particles   int                 `json:"particles"`
	Recoloured  int                 `json:"recoloured"` // effects a mod replaces
	EffectMods  []string            `json:"effectMods,omitempty"`
	Before      []abilityColor      `json:"before"`
	After       []abilityColor      `json:"after"`
	Models      []abilityModel      `json:"models"`
	Sounds      []string            `json:"sounds,omitempty"`
}

// valueLabel turns a property name such as AbilityCastRange into "Cast range".
func valueLabel(name string) string {
	name = strings.TrimPrefix(name, "Ability")
	var words []string
	start := 0
	runes := []rune(name)
	for i := 1; i <= len(runes); i++ {
		if i == len(runes) || (unicode.IsUpper(runes[i]) && !unicode.IsUpper(runes[i-1])) {
			words = append(words, string(runes[start:i]))
			start = i
		}
	}
	for i := range words {
		if i > 0 && !isAllUpper(words[i]) {
			words[i] = strings.ToLower(words[i])
		}
	}
	return strings.Join(words, " ")
}

func isAllUpper(s string) bool { return strings.ToUpper(s) == s }

// valueText writes a value with a short unit when the value does not carry one.
func valueText(v abilityValue) string {
	units := map[string]string{"Seconds": "s", "Meters": "m", "MetersPerSecond": "m/s", "Percent": "%", "Degrees": "°"}[v.Units]
	last := v.Value[len(v.Value)-1]
	if units == "" || (last < '0' || last > '9') {
		return v.Value
	}
	return v.Value + units
}

// topColors orders colours by how often the effects use them and keeps the most used.
func topColors(counts map[[3]uint8]int, keep int) []abilityColor {
	out := []abilityColor{}
	for c, n := range counts {
		out = append(out, abilityColor{c, n})
	}
	sort.Slice(out, func(a, b int) bool {
		if out[a].Uses != out[b].Uses {
			return out[a].Uses > out[b].Uses
		}
		return fmt.Sprint(out[a].Color) < fmt.Sprint(out[b].Color)
	})
	return out[:min(len(out), keep)]
}

func (s *studio) abilityList(r *http.Request) (any, error) {
	var request struct {
		Hero string `json:"hero"`
	}
	if err := readJSON(r, &request); err != nil {
		return nil, err
	}
	index, err := s.workspaceIndex()
	if err != nil {
		return nil, err
	}
	hero, err := gatherHero(s.game, index, request.Hero)
	if err != nil {
		return nil, err
	}
	data, dataErr := loadGameData(index)
	mods, err := newModReader(s.game)
	if err != nil {
		return nil, err
	}
	views := []abilityView{}
	for i, ab := range hero.Abilities {
		view := abilityView{Slot: i + 1, Key: ab.Key, Name: ab.Name, Quip: ab.Quip, Description: ab.Desc, Icon: ab.Icon,
			Values: []map[string]string{}, Models: []abilityModel{}}
		if ab.Icon != "" {
			view.IconMod = mods.files[ab.Icon].mod
		}
		var game *gameAbility
		if data != nil {
			game = data.abilities[ab.Key]
		}
		models := map[string]bool{}
		for _, m := range strings.Split(ab.Props, ", ") {
			if m != "" {
				models[m] = true
			}
		}
		if game != nil {
			for _, v := range game.Values {
				view.Values = append(view.Values, map[string]string{"label": valueLabel(v.Name), "value": valueText(v)})
			}
			view.Sounds = game.Sounds
			before, after := map[[3]uint8]int{}, map[[3]uint8]int{}
			effectMods := map[string]bool{}
			for _, particle := range game.Particles {
				original, err := index.Read(particle + "_c")
				if err != nil {
					continue
				}
				view.Particles++
				colors, _ := source2.ParticleColors(original)
				for _, c := range colors {
					before[c]++
				}
				if replaced, mod, err := mods.read(particle + "_c"); err == nil && replaced != nil {
					view.Recoloured++
					effectMods[mod] = true
					colors, _ = source2.ParticleColors(replaced)
				}
				for _, c := range colors {
					after[c]++
				}
			}
			view.Before, view.After = topColors(before, 12), topColors(after, 12)
			for mod := range effectMods {
				view.EffectMods = append(view.EffectMods, mod)
			}
			sort.Strings(view.EffectMods)
			for _, m := range game.Models {
				if _, ok := index.Files[m+"_c"]; ok {
					models[m] = true
				}
			}
		}
		var names []string
		for m := range models {
			names = append(names, m)
		}
		sort.Strings(names)
		for _, m := range names {
			view.Models = append(view.Models, abilityModel{Path: m, Mod: mods.files[m+"_c"].mod})
		}
		if view.Before == nil {
			view.Before, view.After = []abilityColor{}, []abilityColor{}
		}
		views = append(views, view)
	}
	note := ""
	if dataErr != nil {
		note = "The game's ability data could not be read (" + dataErr.Error() + "), so values, effects and sounds are missing."
	}
	return map[string]any{"hero": hero.Name, "abilities": views, "note": note}, nil
}

// abilityFile reads one of an ability's files, the game's own or the installed mod's.
func (s *studio) abilityFile(r *http.Request, name string) ([]byte, error) {
	if strings.Contains(name, "..") {
		return nil, errors.New("bad file name")
	}
	if r.URL.Query().Get("which") == "yours" {
		mods, err := newModReader(s.game)
		if err != nil {
			return nil, err
		}
		data, _, err := mods.read(name)
		if err == nil && data == nil {
			err = fmt.Errorf("no switched-on mod replaces %s", name)
		}
		return data, err
	}
	index, err := s.workspaceIndex()
	if err != nil {
		return nil, err
	}
	if _, ok := index.Files[name]; !ok {
		return nil, fmt.Errorf("the game has no %s", name)
	}
	return index.Read(name)
}

// abilityIcon sends an ability icon as PNG.
func (s *studio) abilityIcon(r *http.Request) (any, error) {
	name := r.URL.Query().Get("path")
	if !strings.HasPrefix(name, "panorama/images/") || !strings.HasSuffix(name, ".vtex_c") {
		return nil, errors.New("not an image")
	}
	compiled, err := s.abilityFile(r, name)
	if err != nil {
		return nil, err
	}
	img, _, err := source2.DecodeTexture(compiled)
	if err != nil {
		return nil, err
	}
	var out bytes.Buffer
	if err := png.Encode(&out, source2.Thumbnail(img, 256)); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

// modelTriangles collects a compiled model's geometry, in either model layout.
func modelTriangles(raw []byte) ([]source2.Triangles, error) {
	if model, err := source2.LoadModel(raw); err == nil {
		var sets []source2.Triangles
		for _, mesh := range model.Meshes {
			if tris, err := mesh.Triangles(); err == nil && len(tris.Indices) > 0 {
				sets = append(sets, tris)
			}
		}
		if len(sets) > 0 {
			return sets, nil
		}
	}
	prop, err := source2.LoadLegacyProp(raw)
	if err != nil {
		return nil, fmt.Errorf("the model cannot be read: %v", err)
	}
	tris, err := prop.Triangles()
	if err != nil {
		return nil, err
	}
	return []source2.Triangles{tris}, nil
}

// abilityModelPicture draws an ability model as PNG, from the side the page asks for.
func (s *studio) abilityModelPicture(r *http.Request) (any, error) {
	name := r.URL.Query().Get("path")
	if !strings.HasPrefix(name, "models/") || path.Ext(name) != ".vmdl" {
		return nil, errors.New("not a model")
	}
	raw, err := s.abilityFile(r, name+"_c")
	if err != nil {
		return nil, err
	}
	sets, err := modelTriangles(raw)
	if err != nil {
		return nil, err
	}
	yaw, _ := strconv.ParseFloat(r.URL.Query().Get("yaw"), 64)
	var out bytes.Buffer
	if err := png.Encode(&out, source2.RenderPreview(sets, 240, 240, yaw)); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

func (s *studio) abilityRoutes(api func(string, func(*http.Request) (any, error))) {
	api("abilities", s.abilityList)
	api("abilities/icon", s.abilityIcon)
	api("abilities/model", s.abilityModelPicture)
}
