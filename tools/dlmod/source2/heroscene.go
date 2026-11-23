package source2

import (
	"encoding/binary"
	"errors"
	"fmt"
	"hash/crc32"
	"math"
	"strings"
)

// Panorama text resources (.vsvg_c, .vcss_c) keep their text in the DATA block after a CRC32
// (IEEE) of it and two zero bytes. The hero name logos on the hero select screen are such SVGs
// (panorama/images/heroes/hero_names/ivy.vsvg_c), and so are the interface's stylesheets.

// PanoramaText returns the text of a panorama text resource.
func PanoramaText(compiled []byte) (string, error) {
	r, err := DecodeResource(compiled)
	if err != nil {
		return "", err
	}
	data := r.Block("DATA")
	if len(data) < 6 {
		return "", errors.New("the file has no panorama text")
	}
	return string(data[6:]), nil
}

// ReplacePanoramaText writes a copy of a panorama text resource with new text, keeping its other
// blocks.
func ReplacePanoramaText(original []byte, text string) ([]byte, error) {
	r, err := DecodeResource(original)
	if err != nil {
		return nil, err
	}
	data := binary.LittleEndian.AppendUint32(nil, crc32.ChecksumIEEE([]byte(text)))
	data = append(data, 0, 0)
	data = append(data, text...)
	found := false
	for i := range r.Blocks {
		if r.Blocks[i].Type == "DATA" {
			r.Blocks[i].Data = data
			found = true
		}
	}
	if !found {
		return nil, errors.New("the file has no DATA block")
	}
	return r.Encode(), nil
}

// HideHeroScene empties a hero's hero-select scene (maps/ui/hero_prefabs/<hero>.vpk, given as its
// files): the hero model and the other props (Ivy's leaves) and the world meshes (her ink
// backdrop, wind wisp and shadow disc) shrink to nothing far below the camera. Lights, the camera
// and the sky stay, so the screen shows only the hero's background picture
// (panorama/images/heroes/backgrounds/<hero>_bg_psd), for example a full illustration of a
// replacement character. It returns how many entities and world objects it hid.
func HideHeroScene(files map[string][]byte) (props, objects int, err error) {
	for name, data := range files {
		switch {
		case strings.HasSuffix(name, ".vents_c"):
			out, n, err := editKV3Data(data, hideProps)
			if err != nil {
				return 0, 0, fmt.Errorf("%s: %v", name, err)
			}
			files[name] = out
			props += n
		case strings.HasSuffix(name, ".vwnod_c"):
			out, n, err := editKV3Data(data, hideSceneObjects)
			if err != nil {
				return 0, 0, fmt.Errorf("%s: %v", name, err)
			}
			files[name] = out
			objects += n
		}
	}
	if props == 0 && objects == 0 {
		return 0, 0, errors.New("the scene has no props or world meshes to hide")
	}
	return props, objects, nil
}

const hiddenDrop = 5000 // units below the scene

func editKV3Data(compiled []byte, edit func(*Value) int) ([]byte, int, error) {
	r, err := DecodeResource(compiled)
	if err != nil {
		return nil, 0, err
	}
	for i, b := range r.Blocks {
		if b.Type != "DATA" {
			continue
		}
		root, format, err := ReadKV3(b.Data)
		if err != nil {
			return nil, 0, err
		}
		n := edit(root)
		data, err := WriteKV3(root, format)
		if err != nil {
			return nil, 0, err
		}
		r.Blocks[i].Data = data
		return r.Encode(), n, nil
	}
	return nil, 0, errors.New("no DATA block")
}

// hideProps shrinks every prop_dynamic of an entity lump and drops it below the scene.
func hideProps(root *Value) int {
	n := 0
	for _, ent := range root.Get("m_entityKeyValues").items() {
		values := ent.Get("keyValues3Data").Get("values")
		if values == nil {
			continue
		}
		if class := values.Get("classname"); class == nil || class.S != "prop_dynamic" {
			continue
		}
		hideProp(values)
		n++
	}
	return n
}

// hideSceneObjects shrinks every scene object of a world node (m_vTransform, a 3x4 matrix) and
// drops it below the scene.
func hideSceneObjects(root *Value) int {
	n := 0
	for _, obj := range root.Get("m_sceneObjects").items() {
		rows := obj.Get("m_vTransform").items()
		if len(rows) != 3 {
			continue
		}
		for r, row := range rows {
			for c, cell := range row.items() {
				v := 0.0
				if r == c {
					v = 0.001
				}
				if c == 3 && r == 2 {
					v = -hiddenDrop
				}
				setFloat(cell, v)
			}
		}
		n++
	}
	return n
}

func (v *Value) items() []*Value {
	if v == nil {
		return nil
	}
	return v.Items
}

func floatOf(v *Value) float64 {
	switch v.Kind {
	case Float32, Float64:
		return v.F
	case Int16, Int32, Int64:
		return float64(v.I)
	case Uint16, Uint32, Uint64:
		return float64(v.U)
	}
	return 0
}

// setFloat stores a number in a value, as a float when it was an integer (KV3 readers accept
// either for entity keys and transforms).
func setFloat(v *Value, f float64) {
	if v.Kind != Float32 && v.Kind != Float64 {
		v.Kind = Float32
	}
	v.F = f
}

// FrameHeroScene fits another model into the hero's place in a hero select scene (given as its
// files): the prop showing model is scaled by scale about its origin and moved by offset, given in
// the model's own space and turned with the prop, so a replacement of another size or shape stands
// where the camera expects the hero. The scene's other props and world meshes (the hero's own
// decor) are hidden as HideHeroScene does; lights, camera and sky stay. It returns how many props
// show the model.
func FrameHeroScene(files map[string][]byte, modelPath string, scale float64, offset [3]float64) (int, error) {
	model := strings.ToLower(modelPath)
	framed := 0
	for name, data := range files {
		switch {
		case strings.HasSuffix(name, ".vents_c"):
			out, n, err := editKV3Data(data, func(root *Value) int {
				n := 0
				for _, ent := range root.Get("m_entityKeyValues").items() {
					values := ent.Get("keyValues3Data").Get("values")
					if values == nil {
						continue
					}
					if class := values.Get("classname"); class == nil || class.S != "prop_dynamic" {
						continue
					}
					// The hero's prop is the one named hero_root or showing the model. A few scenes
					// show an older copy of the hero (Yamato, Infernus): it shows the model instead.
					m := values.Get("model")
					name := values.Get("targetname")
					isHero := (m != nil && strings.ToLower(m.S) == model) || (name != nil && strings.Contains(name.S, "hero_root"))
					if !isHero {
						hideProp(values)
						continue
					}
					values.Set("model", NewString(modelPath))
					for _, s := range values.Get("scales").items() {
						setFloat(s, floatOf(s)*scale)
					}
					var angles [3]float64
					for i, a := range values.Get("angles").items() {
						if i < 3 {
							angles[i] = floatOf(a)
						}
					}
					moved := turnByAngles(offset, angles)
					if o := values.Get("origin").items(); len(o) == 3 {
						for k := 0; k < 3; k++ {
							setFloat(o[k], floatOf(o[k])+moved[k])
						}
					}
					n++
				}
				return n
			})
			if err != nil {
				return 0, fmt.Errorf("%s: %v", name, err)
			}
			files[name] = out
			framed += n
		case strings.HasSuffix(name, ".vwnod_c"):
			out, _, err := editKV3Data(data, hideSceneObjects)
			if err != nil {
				return 0, fmt.Errorf("%s: %v", name, err)
			}
			files[name] = out
		}
	}
	if framed == 0 {
		return 0, fmt.Errorf("the scene has no prop showing %s", model)
	}
	return framed, nil
}

// turnByAngles turns a vector by entity angles (pitch, yaw, roll in degrees, as Source orders
// them: roll about X, then pitch about Y, then yaw about Z).
func turnByAngles(v [3]float64, angles [3]float64) [3]float64 {
	pitch, yaw, roll := angles[0]*math.Pi/180, angles[1]*math.Pi/180, angles[2]*math.Pi/180
	sp, cp := math.Sincos(pitch)
	sy, cy := math.Sincos(yaw)
	sr, cr := math.Sincos(roll)
	x, y, z := v[0], v[1], v[2]
	y, z = y*cr-z*sr, y*sr+z*cr  // roll about X
	x, z = x*cp+z*sp, -x*sp+z*cp // pitch about Y
	x, y = x*cy-y*sy, x*sy+y*cy  // yaw about Z
	return [3]float64{x, y, z}
}

// hideProp shrinks a prop_dynamic's values and drops it below the scene.
func hideProp(values *Value) {
	for _, s := range values.Get("scales").items() {
		setFloat(s, 0.001)
	}
	if o := values.Get("origin").items(); len(o) == 3 {
		setFloat(o[2], floatOf(o[2])-hiddenDrop)
	}
}
