package main

import (
	"fmt"
	"path"
	"sort"
	"strings"
	"sync"

	"github.com/MrHakan/deadlock-modkit/tools/dlmod/source2"
)

// The game describes heroes and abilities in scripts/heroes.vdata and scripts/abilities.vdata:
// which abilities sit in a hero's four slots, each ability's icon, the particle systems, models
// and sounds it uses, and its values (cooldown, duration, range...). Both are KV3 compressed with
// zstd. When they cannot be read, hero_abilities.go (from deadlock-api) still gives the slots and
// icons.

const (
	heroesVdata    = "scripts/heroes.vdata_c"
	abilitiesVdata = "scripts/abilities.vdata_c"

	// Where a workspace's abilities came from.
	abilitiesFromGame  = "scripts/heroes.vdata"
	abilitiesFromTable = "hero_abilities.go"
)

// gameAbility is one ability as scripts/abilities.vdata describes it.
type gameAbility struct {
	Key       string         `json:"key"`
	Icon      string         `json:"icon,omitempty"` // compiled icon in the game's archive
	Particles []string       `json:"particles,omitempty"`
	Models    []string       `json:"models,omitempty"`
	Sounds    []string       `json:"sounds,omitempty"`
	Values    []abilityValue `json:"values,omitempty"`
}

// abilityValue is one of an ability's numbers as the game shows it, such as AbilityCooldown 4.
type abilityValue struct {
	Name  string `json:"name"`
	Value string `json:"value"`
	Units string `json:"units,omitempty"`
}

type gameData struct {
	heroes    map[string][4]string // hero → its four abilities' keys, in slot order
	abilities map[string]*gameAbility
}

var (
	gameDataMu    sync.Mutex
	gameDataCache = map[*source2.VPKIndex]*gameData{}
)

// loadGameData reads the hero and ability data from the game's archive, once per index.
func loadGameData(index *source2.VPKIndex) (*gameData, error) {
	gameDataMu.Lock()
	defer gameDataMu.Unlock()
	if data, ok := gameDataCache[index]; ok {
		return data, nil
	}
	heroes, err := readVdata(index, heroesVdata)
	if err != nil {
		return nil, err
	}
	abilities, err := readVdata(index, abilitiesVdata)
	if err != nil {
		return nil, err
	}
	data := &gameData{heroes: map[string][4]string{}, abilities: map[string]*gameAbility{}}
	for i, key := range heroes.Keys {
		bound := heroes.Fields[i].Get("m_mapBoundAbilities")
		if !strings.HasPrefix(key, "hero_") || bound == nil {
			continue
		}
		var slots [4]string
		complete := true
		for slot := range slots {
			v := bound.Get(fmt.Sprintf("ESlot_Signature_%d", slot+1))
			if v == nil || v.S == "" {
				complete = false
				break
			}
			slots[slot] = v.S
		}
		if complete {
			data.heroes[strings.TrimPrefix(key, "hero_")] = slots
		}
	}
	for i, key := range abilities.Keys {
		if fields := abilities.Fields[i]; fields.Kind == source2.Object {
			data.abilities[key] = readGameAbility(index, key, fields)
		}
	}
	gameDataCache[index] = data
	return data, nil
}

func readVdata(index *source2.VPKIndex, name string) (*source2.Value, error) {
	raw, err := index.Read(name)
	if err != nil {
		return nil, err
	}
	resource, err := source2.DecodeResource(raw)
	if err != nil {
		return nil, fmt.Errorf("%s: %v", name, err)
	}
	root, _, err := source2.ReadKV3(resource.Block("DATA"))
	if err != nil {
		return nil, fmt.Errorf("%s: %v", name, err)
	}
	return root, nil
}

// String flags of resource references in KV3.
const (
	kv3Resource  = 2 // a compiled resource: particles, models
	kv3Panorama  = 3 // a panorama file: images
	kv3SoundName = 4 // a sound event
)

func readGameAbility(index *source2.VPKIndex, key string, fields *source2.Value) *gameAbility {
	ability := &gameAbility{Key: key}
	seen := map[string]bool{}
	var walk func(v *source2.Value)
	walk = func(v *source2.Value) {
		if v.Kind == source2.String && v.S != "" && !seen[v.S] {
			seen[v.S] = true
			switch {
			case v.Flag == kv3Resource && strings.HasSuffix(v.S, ".vpcf"):
				ability.Particles = append(ability.Particles, v.S)
			case v.Flag == kv3Resource && strings.HasSuffix(v.S, ".vmdl"):
				ability.Models = append(ability.Models, v.S)
			case v.Flag == kv3SoundName:
				ability.Sounds = append(ability.Sounds, v.S)
			}
		}
		for _, f := range v.Fields {
			walk(f)
		}
		for _, item := range v.Items {
			walk(item)
		}
	}
	walk(fields)
	sort.Strings(ability.Particles)
	sort.Strings(ability.Models)
	sort.Strings(ability.Sounds)
	if image := fields.Get("m_strAbilityImage"); image != nil {
		ability.Icon = compiledPanoramaImage(image.S)
		if index != nil {
			if _, ok := index.Files[ability.Icon]; !ok {
				ability.Icon = ""
			}
		}
	}
	if properties := fields.Get("m_mapAbilityProperties"); properties != nil {
		for i, name := range properties.Keys {
			property := properties.Fields[i]
			value := property.Get("m_strValue")
			if value == nil || value.S == "" || value.S == "0" {
				continue
			}
			entry := abilityValue{Name: name, Value: value.S}
			if units := property.Get("m_eDisplayUnits"); units != nil {
				entry.Units = strings.TrimPrefix(units.S, "EDisplayUnit_")
			}
			ability.Values = append(ability.Values, entry)
		}
	}
	return ability
}

// compiledPanoramaImage turns an image as panorama names it, "file://{images}/hud/x.psd", into
// its compiled file in the archive, "panorama/images/hud/x_psd.vtex_c".
func compiledPanoramaImage(name string) string {
	rest, ok := strings.CutPrefix(name, "file://{images}/")
	if !ok || path.Ext(rest) == "" {
		return ""
	}
	ext := path.Ext(rest)
	return "panorama/images/" + strings.TrimSuffix(rest, ext) + "_" + strings.TrimPrefix(ext, ".") + ".vtex_c"
}

// heroAbilityKeys gives a hero's four ability keys and icons, from the game's data when it can
// be read, else from hero_abilities.go. source names where they came from.
func heroAbilityKeys(index *source2.VPKIndex, hero string) (keys [4]string, icons [4]string, names [4]string, source string) {
	if index != nil {
		if data, err := loadGameData(index); err == nil {
			if slots, ok := data.heroes[hero]; ok {
				for i, key := range slots {
					keys[i] = key
					if ability := data.abilities[key]; ability != nil {
						icons[i] = ability.Icon
					}
				}
				if table, ok := heroAbilities[hero]; ok {
					for i, row := range table {
						if row[0] == keys[i] {
							names[i] = row[1]
						}
					}
				}
				return keys, icons, names, abilitiesFromGame
			}
		}
	}
	if table, ok := heroAbilities[hero]; ok {
		for i, row := range table {
			keys[i], names[i], icons[i] = row[0], row[1], row[2] // the base of the icon name
		}
		return keys, icons, names, abilitiesFromTable
	}
	return keys, icons, names, ""
}
