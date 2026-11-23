package main

import (
	"errors"
	"fmt"
	"image"
	"image/color"
	"math"
	"strings"
)

// ColorPalette maps organic/plant colours onto a hero's custom multi-tone theme rather than a
// single flat tint: greens and yellow-greens map to Primary (dark greens to a deep shade),
// browns/bark/thorns/warm accents map to Accent, and pale/yellow highlights lift towards White/Glow,
// while preserving lightness, shading detail, neutral greys, and alpha.
type ColorPalette struct {
	Name    string
	Primary [3]float64 // main colour (e.g. #2467FF blue)
	Accent  [3]float64 // bark/thorns/warm accent (e.g. #FF3B3B red)
	White   [3]float64 // highlight white (e.g. #F5F5F5)
	Glow    [3]float64 // bright accent glow (e.g. #7AB6FF)
}

// CobaltPalette is the cobalt palette (blue #2467FF, red #FF3B3B, white #F5F5F5,
// accent glow #7AB6FF).
var CobaltPalette = ColorPalette{
	Name:    "cobalt",
	Primary: [3]float64{0x24, 0x67, 0xFF}, // #2467FF
	Accent:  [3]float64{0xFF, 0x3B, 0x3B}, // #FF3B3B
	White:   [3]float64{0xF5, 0xF5, 0xF5}, // #F5F5F5
	Glow:    [3]float64{0x7A, 0xB6, 0xFF}, // #7AB6FF
}

// LookupPalette finds a built-in palette by name (defaulting to "cobalt" when empty), or makes
// one from colours such as "#2467FF,#FF3B3B,#7AB6FF".
func LookupPalette(name string) (ColorPalette, error) {
	if strings.Contains(name, "#") {
		return paletteFromColors(name)
	}
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "", "cobalt":
		return CobaltPalette, nil
	default:
		return ColorPalette{}, fmt.Errorf("unknown palette %q (use cobalt, or one to three colours such as #2467FF,#FF3B3B,#7AB6FF)", name)
	}
}

// paletteFromColors makes a palette from one to three colours: the main colour, an accent and a
// glow. A missing accent repeats the main colour; a missing glow is the main colour lightened.
func paletteFromColors(text string) (ColorPalette, error) {
	var colors [][3]float64
	for _, part := range strings.Split(text, ",") {
		if part = strings.TrimSpace(part); part == "" {
			continue
		}
		c, ok := parseHex(part)
		if !ok {
			return ColorPalette{}, fmt.Errorf("%q is not a colour like #2467FF", part)
		}
		colors = append(colors, c)
	}
	if len(colors) == 0 || len(colors) > 3 {
		return ColorPalette{}, errors.New("give one to three colours, such as #2467FF,#FF3B3B,#7AB6FF")
	}
	p := ColorPalette{Name: strings.ReplaceAll(strings.TrimSpace(text), " ", ""), Primary: colors[0], Accent: colors[0],
		White: [3]float64{0xF5, 0xF5, 0xF5}, Glow: lerpRGB(colors[0], [3]float64{255, 255, 255}, 0.45)}
	if len(colors) > 1 {
		p.Accent = colors[1]
	}
	if len(colors) > 2 {
		p.Glow = colors[2]
	}
	return p, nil
}

func smoothstep(edge0, edge1, x float64) float64 {
	if edge0 == edge1 {
		if x < edge0 {
			return 0
		}
		return 1
	}
	t := math.Max(0, math.Min(1, (x-edge0)/(edge1-edge0)))
	return t * t * (3 - 2*t)
}

func clamp01(x float64) float64 {
	return math.Max(0, math.Min(1, x))
}

func lerpRGB(a, b [3]float64, t float64) [3]float64 {
	return [3]float64{
		a[0] + (b[0]-a[0])*t,
		a[1] + (b[1]-a[1])*t,
		a[2] + (b[2]-a[2])*t,
	}
}

// Recolor maps one RGB colour (0..255) into the palette while preserving lightness and shading detail.
func (p ColorPalette) Recolor(c [3]float64) [3]float64 {
	h, s, v := rgbToHSV(c)
	// Preserve pure blacks, neutral greys, and whites untouched so masks, outlines, and sparks stay clean.
	if s < 0.08 || v < 0.03 {
		return c
	}
	strength := smoothstep(0.08, 0.16, s)

	blueH, blueS, _ := rgbToHSV(p.Primary)
	redH, redS, _ := rgbToHSV(p.Accent)
	glowH, glowS, _ := rgbToHSV(p.Glow)

	// Map to primary (blue): dark greens -> deep blue, mid/bright greens -> cobalt blue,
	// pale/yellow highlights -> white / white-blue glow.
	mapToBlue := func() [3]float64 {
		darkWeight := 1 - smoothstep(0.28, 0.55, v)
		targetH := blueH + 4*darkWeight
		// Bright greens lean slightly toward the ice-blue glow hue.
		brightWeight := smoothstep(0.70, 0.95, v)
		targetH += (glowH - targetH) * (0.45 * brightWeight)

		minSat := blueS * (0.85 + 0.12*darkWeight)
		targetS := clamp01(math.Max(s*1.08, minSat*smoothstep(0.12, 0.42, s)))

		// Green has higher perceived luminance than blue; lift dark/mid values slightly so shaded
		// vines keep their readability without crushing to black.
		targetV := clamp01(v * (1.0 + 0.10*(1-brightWeight)))

		out := hsvToRGB(targetH, targetS, targetV)

		// Pale / yellow-green highlights lift smoothly toward crisp white (#F5F5F5) / accent glow.
		paleHi := smoothstep(0.72, 0.90, v) * smoothstep(0.55, 0.25, s)
		yellowHi := 0.0
		if h >= 50 && h <= 82 {
			yellowHi = smoothstep(0.84, 0.95, v) * (1 - math.Abs(h-66)/18)
			if yellowHi < 0 {
				yellowHi = 0
			}
		}
		hiW := math.Max(paleHi, 0.85*yellowHi)
		if hiW > 0 {
			whiteTarget := hsvToRGB(glowH, glowS*0.18*(1-hiW), clamp01(math.Max(v, 0.95)))
			out = lerpRGB(out, whiteTarget, clamp01(hiW))
		}
		return out
	}

	// Map to accent (red #FF3B3B): brown/olive bark, thorns, warm petals, and magentas.
	mapToRed := func() [3]float64 {
		targetH := redH
		targetS := clamp01(math.Max(s*1.05, redS*smoothstep(0.14, 0.40, s)))
		targetV := clamp01(v * 1.06)
		out := hsvToRGB(targetH, targetS, targetV)
		// Bright pale yellow/amber highlights (e.g. #fecf78) lift cleanly toward white (#F5F5F5).
		if h >= 30 && h <= 56 {
			hiW := smoothstep(0.80, 0.95, v) * smoothstep(0.68, 0.35, s)
			if hiW > 0 {
				whiteTarget := [3]float64{p.White[0] * v, p.White[1] * v, p.White[2] * v}
				out = lerpRGB(out, whiteTarget, clamp01(hiW))
			}
		}
		return out
	}

	// Choose palette role by hue. Keep warm olive-brown (<= 52°) cleanly red and olive-green
	// (>= 56°) cleanly blue; across the narrow 52°..56° boundary pass briefly through a cool
	// neutral value instead of mixing red+blue into purple.
	var mapped [3]float64
	switch {
	case h <= 52 || h >= 270:
		// Red, crimson, brown, warm olive, orange-brown, magenta/purple -> accent red (#FF3B3B).
		mapped = mapToRed()
	case h < 56:
		// Narrow olive boundary (52°..56°): red -> light neutral -> blue, avoiding purple mud.
		t := smoothstep(52, 56, h)
		mid := [3]float64{p.White[0] * v, p.White[1] * v, p.White[2] * v}
		if t < 0.5 {
			mapped = lerpRGB(mapToRed(), mid, t*2)
		} else {
			mapped = lerpRGB(mid, mapToBlue(), (t-0.5)*2)
		}
	case h <= 175:
		// Olive-green, yellow-green, green, emerald -> cobalt blue / deep blue / white highlights.
		mapped = mapToBlue()
	default:
		// Existing cyan/blue/indigo (175°..270°) -> harmonize smoothly toward cobalt blue.
		targetH := blueH + (h-blueH)*0.20
		targetS := clamp01(s*0.6 + blueS*0.4*smoothstep(0.12, 0.40, s))
		mapped = hsvToRGB(targetH, targetS, v)
	}

	if strength < 1 {
		return lerpRGB(c, mapped, strength)
	}
	return mapped
}

// RepaintImage applies recolor to every pixel's RGB channels of img (including pixels with A == 0,
// because Source 2 PBR colour textures pack metalness in alpha) while preserving alpha verbatim.
// It returns the repainted image and how many pixels changed noticeably.
func RepaintImage(img image.Image, recolor func([3]float64) [3]float64) (*image.NRGBA, int) {
	b := img.Bounds()
	out := image.NewNRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	changed := 0
	for y := 0; y < b.Dy(); y++ {
		for x := 0; x < b.Dx(); x++ {
			src := color.NRGBAModel.Convert(img.At(b.Min.X+x, b.Min.Y+y)).(color.NRGBA)
			res := recolor([3]float64{float64(src.R), float64(src.G), float64(src.B)})
			nr := uint8(math.Round(math.Max(0, math.Min(255, res[0]))))
			ng := uint8(math.Round(math.Max(0, math.Min(255, res[1]))))
			nb := uint8(math.Round(math.Max(0, math.Min(255, res[2]))))
			if absInt(int(nr)-int(src.R))+absInt(int(ng)-int(src.G))+absInt(int(nb)-int(src.B)) >= 6 {
				changed++
			}
			out.SetNRGBA(x, y, color.NRGBA{nr, ng, nb, src.A})
		}
	}
	return out, changed
}

func absInt(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
