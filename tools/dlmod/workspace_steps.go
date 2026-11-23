package main

import (
	"fmt"
	"path"
	"strings"
)

// The steps of a hero's workspace. Prompts name the character two ways: {{CONCEPT}} is its full
// description, used where Meshy has nothing else to go on (the design sheet, 3D props), and
// {{CHARACTER}} its short name, used where the design sheet is the reference picture. Every
// picture after the sheet is made from the sheet, so the character stays the same everywhere.

const characterMark = "{{CHARACTER}}"

const sheetFile = "assets/design_sheet.png"

// greenBackground ends the prompts of pictures with a background that dlmod keys out.
const greenBackground = " On a flat pure green (#00FF00) background, no green on the character, no text."

// fileSlug names an asset after an ability: "Entangling Thorns" -> entangling_thorns.
func fileSlug(name string) string { return strings.Join(words(name), "_") }

// buildWorkspace writes the steps for one hero.
func buildWorkspace(a *heroAssets) *workspace {
	w := &workspace{Hero: a.ID, Name: a.Name, Game: a}
	w.Select = workspaceSelect{Tags: append([]string{}, a.Tags...), HideScene: true}
	add := func(s workspaceStep) { w.Steps = append(w.Steps, s) }

	add(workspaceStep{
		ID: "design_sheet", Title: "Character design sheet", Tool: "meshy_text_to_image", AIModel: "nano-banana-pro", Credits: 9, Aspect: "16:9",
		Prompt: fmt.Sprintf("Character design sheet for a 3D game character: %s. %s The character must read at a glance from behind in third person, so give it a strong silhouette and one or two bold colours. It carries %s. Show a large hero render, a front / three-quarter / side / back turnaround in a relaxed A-pose, close-ups of the head, hands and weapon, a colour palette with hex codes, and its two or three iconic symbols. Stylized PBR, clean studio light, dark background, labels only.", conceptMark, heroBrief(a), weaponPhrase(a)),
		Output: sheetFile, Note: "Every later picture uses this sheet as its reference, so redo it until the character is right.",
	})
	add(workspaceStep{
		ID: "turnaround", Title: "Front, side and back views for 3D", Tool: "meshy_image_to_image", AIModel: "nano-banana-pro", Credits: 9, Reference: sheetFile,
		Options: map[string]any{"generate_multi_view": true},
		Prompt:  fmt.Sprintf("%s from the reference sheet alone, full body, standing in a relaxed A-pose (arms 45 degrees down, legs slightly apart), empty hands, neutral expression, even light, plain light grey background, no shadows, no text.", characterMark),
		Output:  "assets/views.png", Note: "Empty hands: a held weapon is better as its own model (weapon step), and dlmod can keep the hero's gun instead (-keep gun).",
	})
	fit := ""
	if a.Height > 0 {
		fit = fmt.Sprintf(" dlmod scales it to %s's height (%.0f units) and rigs it to %s's skeleton.", a.Name, a.Height, a.Name)
	}
	add(workspaceStep{
		ID: "model", Title: "3D model", Tool: "meshy_multi_image_to_3d", AIModel: "meshy-7.1", TextureModel: "meshy-7", Credits: 30, Reference: "assets/views.png",
		Options: map[string]any{"texture_resolution": "2k", "enable_pbr": false},
		Output:  "assets/model.glb", Kind: "model", Target: a.Model,
		Note: "Detailed geometry reads best close up in third person, and Meshy models of about 200k triangles install fine." + fit + " For a lighter model use meshy_image_to_3d with meshy-t2.5 + meshy-n1 (15 credits, target_polycount 60000) on the front view.",
	})
	add(workspaceStep{
		ID: "rig", Title: "Rig (humanoid characters)", Tool: "meshy_rig", Credits: 5, Reference: "assets/model.glb", Output: "assets/model.glb", Optional: true,
		Note: "Only for characters with arms and legs: dlmod maps Meshy's joints to the hero's bones by their structure. Skip it for blob-like characters: dlmod rigs them on the hero's own bones (see Animations in README.md). Meshy's animation clips are not used: the game plays the hero's own clips.",
	})
	add(workspaceStep{
		ID: "weapon", Title: "Signature weapon or held prop", Tool: "meshy_text_to_3d", AIModel: "meshy-t2.5", Credits: 15, Optional: true,
		Options: map[string]any{"target_polycount": 15000, "refine": true},
		Prompt:  fmt.Sprintf("A single game-ready prop: the signature weapon of %s. It is %s. Stylized PBR in the character's colours, centered, no hands, no base.", conceptMark, weaponPhrase(a)),
		Output:  "assets/weapon.glb", Note: "Join it to model.glb in the right hand before installing, or keep the hero's own gun with -keep gun.",
	})
	if bg := a.Select.Background; bg != "" {
		size := ""
		for _, f := range a.Images {
			if f.Path == bg {
				size = describeSize(f)
			}
		}
		mood := "a setting that shows off its personality"
		if a.Role != "" {
			mood += fmt.Sprintf(" and its part in a team (%s)", strings.ToLower(strings.TrimSuffix(a.Role, ".")))
		}
		add(workspaceStep{
			ID: "select_art", Title: "Hero select screen art", Tool: "meshy_image_to_image", AIModel: "nano-banana-pro", Credits: 9, Aspect: "16:9", Reference: sheetFile,
			Prompt: fmt.Sprintf("Splash art for a hero select screen: %s from the reference sheet in %s. Put the character in the RIGHT half of the frame, its centre at about 70%% of the width and its head in the top third; keep the LEFT 45%% calm and dark, it sits under the hero grid; keep the bottom-right corner simple, the name, tags and ability icons go there. Cinematic light, rich detail, no text, no logos.", characterMark, mood),
			Output: "assets/select.png", Kind: "select", Target: bg, Size: size,
			Note: "dlmod select hides the hero's 3D scene so the picture shows on its own (scene=hide) and fades the left side to black under the hero grid.",
		})
	}
	cards := []struct{ suffix, id, title, prompt string }{
		{"_card_psd", "card", "Hero card", "Hero card portrait of %s from the reference sheet: chest-up, three-quarter view, facing the viewer with confidence, its weapon in view, dramatic rim light, stylized 3D render, centered with some headroom."},
		{"_card_critical_psd", "card_critical", "Card at low health", "The same hero card portrait of %s from the reference sheet, badly hurt: scuffed and cracked, cringing but defiant, red danger light, same framing."},
		{"_card_gloat_psd", "card_gloat", "Card after a kill", "The same hero card portrait of %s from the reference sheet, triumphant: a taunting gesture after a kill, warm golden light, same framing."},
		{"_vertical_psd", "vertical", "Tall portrait", "Full-body portrait of %s from the reference sheet standing tall in its signature pose, the whole character in frame."},
		{"_sm_psd", "sm", "Small portrait", "Head-and-shoulders portrait of %s from the reference sheet, front-facing, big simple shapes that read when small."},
		{"_mm_psd", "mm", "Minimap icon", "Minimap icon of %s from the reference sheet: the head only, front view, flat colours, thick dark outline, readable at 32 pixels."},
		{"_gun_psd", "gun", "Weapon picture", "Side view of the weapon or held prop of %s from the reference sheet, alone, horizontal, stylized 3D render."},
	}
	for _, c := range cards {
		f, ok := a.image(c.suffix)
		if !ok {
			continue
		}
		add(workspaceStep{
			ID: c.id, Title: c.title, Tool: "meshy_image_to_image", AIModel: "nano-banana-pro", Credits: 9, Aspect: closestAspect(f.Width, f.Height), Reference: sheetFile,
			Prompt: fmt.Sprintf(c.prompt, characterMark) + greenBackground,
			Output: "assets/" + c.id + ".png", Kind: "image", Target: f.Path, Size: describeSize(f),
			Note: "dlmod apply keys out the green background.",
		})
	}
	for _, ab := range a.Abilities {
		if ab.Icon == "" {
			continue
		}
		size := ""
		for _, f := range a.Icons {
			if f.Path == ab.Icon {
				size = describeSize(f)
			}
		}
		what := strings.TrimSuffix(ab.Quip, ".")
		if ab.Desc != "" {
			if what != "" {
				what += ". "
			}
			what += clip(ab.Desc, 220)
		}
		what = strings.TrimRight(what, ".…")
		add(workspaceStep{
			ID: "icon_" + fileSlug(ab.Name), Title: "Ability icon: " + ab.Name, Tool: "meshy_image_to_image", AIModel: "nano-banana-pro", Credits: 9, Aspect: "1:1", Reference: sheetFile,
			Prompt: fmt.Sprintf("A single game ability icon glyph for %q, an ability of %s from the reference sheet: %s. Draw the idea with the character's own shapes and props, not its whole body. Flat solid pure white silhouette with a few clean cut-out interior lines, bold readable shapes, centered, filling about 80%% of the frame, on a pure flat black background. No gradients, no shading, no colour, no text, no border, no circle frame.", ab.Name, characterMark, what),
			Output: "assets/icon_" + fileSlug(ab.Name) + ".png", Kind: "icon", Target: ab.Icon, Size: size,
			Note: "dlmod turns it into a white glyph on transparency.",
		})
	}
	for _, p := range a.Props {
		base := strings.TrimSuffix(path.Base(p), ".vmdl")
		ability := ""
		for _, ab := range a.Abilities {
			if strings.Contains(ab.Props, p) {
				ability = fmt.Sprintf(" It is used by %q (%s).", ab.Name, ab.summary())
			}
		}
		add(workspaceStep{
			ID: "prop_" + base, Title: "Ability model: " + base, Tool: "meshy_text_to_3d", AIModel: "meshy-t2.5", Credits: 15, Optional: true,
			Options: map[string]any{"target_polycount": 10000, "refine": true},
			Prompt:  fmt.Sprintf("A game-ready 3D prop for %s: it replaces %s's %q ability model.%s Single object, stylized PBR in the character's colours, centered, no base, no characters.", conceptMark, a.Name, strings.Join(words(base), " "), ability),
			Output:  "assets/prop_" + base + ".glb", Kind: "prop", Target: p,
			Note: "Thrown and placed ability props (bombs, statues, traps) are replaced this way; dlmod prop pins it to the prop's root bone.",
		})
	}
	return w
}

// characterName is the short name prompts use: the name chosen for the hero select screen, or
// the concept up to its first comma.
func characterName(w *workspace) string {
	if w.Select.Name != "" {
		return w.Select.Name
	}
	if w.Concept != "" {
		name, _, _ := strings.Cut(w.Concept, ",")
		return strings.TrimSpace(name)
	}
	return ""
}

// filled puts the concept and the character's name into a prompt, leaving the marks when unset.
func filled(text string, w *workspace) string {
	if w.Concept != "" {
		text = strings.ReplaceAll(text, conceptMark, w.Concept)
	}
	if name := characterName(w); name != "" {
		text = strings.ReplaceAll(text, characterMark, name)
	}
	return text
}
