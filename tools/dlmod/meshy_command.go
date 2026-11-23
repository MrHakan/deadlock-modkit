package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
)

// dlmod meshy <hero|all> [folder] [concept=...] writes a Meshy workspace: for each hero a folder
// with the Meshy prompts that make a full replacement character (3D model, hero select art,
// cards, ability icons, ability props), written for that hero's role, weapon and abilities, the
// game files each result replaces with their sizes, and workspace.json, which dlmod apply reads to
// install whatever has been generated into assets/. The prompts follow what has worked in practice:
// a design sheet first, every later picture made from it, glyph icons drawn white on
// black, card pictures on flat green so they can be keyed out.

const conceptMark = "{{CONCEPT}}"

type workspaceStep struct {
	ID           string         `json:"id"`
	Title        string         `json:"title"`
	Tool         string         `json:"tool"`
	AIModel      string         `json:"aiModel,omitempty"`
	TextureModel string         `json:"textureModel,omitempty"`
	Credits      int            `json:"credits"`
	Aspect       string         `json:"aspectRatio,omitempty"`
	Reference    string         `json:"reference,omitempty"`
	Options      map[string]any `json:"options,omitempty"`
	Prompt       string         `json:"prompt,omitempty"`
	Output       string         `json:"output"`
	Kind         string         `json:"kind,omitempty"`   // how dlmod apply installs it: model, image, icon, prop, select
	Target       string         `json:"target,omitempty"` // the game file it replaces
	Size         string         `json:"size,omitempty"`
	Optional     bool           `json:"optional,omitempty"`
	Note         string         `json:"note,omitempty"`
}

type workspaceSelect struct {
	Name      string   `json:"name"`
	Tags      []string `json:"tags"`
	Colors    []string `json:"colors"`
	HideScene bool     `json:"hideScene"`
	Scene     string   `json:"scene,omitempty"` // "frame": the installed 3D model in front of the picture, instead of hiding the scene
}

type workspace struct {
	Hero    string          `json:"hero"`
	Name    string          `json:"name"`
	Concept string          `json:"concept"`
	Select  workspaceSelect `json:"select"`
	Steps   []workspaceStep `json:"steps"`
	Game    *heroAssets     `json:"game"`
}

func (w *workspace) credits() (sum, required int) {
	for _, s := range w.Steps {
		sum += s.Credits
		if !s.Optional {
			required += s.Credits
		}
	}
	return sum, required
}

// closestAspect picks the aspect ratio Meshy's image models offer that is nearest a size.
func closestAspect(width, height int) string {
	if width == 0 || height == 0 {
		return "1:1"
	}
	best, bestD := "1:1", math.Inf(1)
	for _, a := range []struct {
		name string
		r    float64
	}{{"1:1", 1}, {"16:9", 16.0 / 9}, {"9:16", 9.0 / 16}, {"4:3", 4.0 / 3}, {"3:4", 3.0 / 4}} {
		if d := math.Abs(math.Log(float64(width)/float64(height)) - math.Log(a.r)); d < bestD {
			best, bestD = a.name, d
		}
	}
	return best
}

func clip(s string, n int) string {
	if len(s) <= n {
		return s
	}
	cut := strings.LastIndex(s[:n], " ")
	if cut < n/2 {
		cut = n
	}
	return strings.TrimRight(s[:cut], " ,.;:") + "…"
}

// heroBrief is the part of every prompt that says who the character replaces and what it does.
func heroBrief(a *heroAssets) string {
	var b strings.Builder
	fmt.Fprintf(&b, "It replaces %s in Deadlock, a stylized third-person hero shooter", a.Name)
	if a.Role != "" {
		fmt.Fprintf(&b, " (%s: %s)", a.Name, strings.TrimSuffix(a.Role, "."))
	}
	b.WriteString(".")
	if len(a.Abilities) > 0 {
		var names []string
		for _, ab := range a.Abilities {
			names = append(names, ab.Name)
		}
		fmt.Fprintf(&b, " Its abilities are %s.", strings.Join(names, ", "))
	}
	return b.String()
}

func weaponPhrase(a *heroAssets) string {
	if a.Weapon == "" {
		return "a signature weapon"
	}
	return fmt.Sprintf("its own take on %s's weapon, %s", a.Name, a.Weapon)
}

// writeWorkspace writes a hero's folder: README.md, prompts.md, workspace.json and assets/. A
// workspace.json already there keeps its concept and select settings, which the user fills in.
func writeWorkspace(dir string, w *workspace) error {
	if err := os.MkdirAll(filepath.Join(dir, "assets"), 0o755); err != nil {
		return err
	}
	if data, err := os.ReadFile(filepath.Join(dir, "workspace.json")); err == nil {
		var old workspace
		if json.Unmarshal(data, &old) == nil {
			if old.Concept != "" && w.Concept == "" {
				w.Concept = old.Concept
			}
			if old.Select.Name != "" || len(old.Select.Colors) > 0 {
				w.Select = old.Select
			}
		}
	}
	data, err := json.MarshalIndent(w, "", "  ")
	if err != nil {
		return err
	}
	files := map[string]string{
		"workspace.json":   string(data) + "\n",
		"README.md":        workspaceReadme(w),
		"prompts.md":       workspacePrompts(w),
		"assets/README.md": "Put each generated file here under the name its step gives (model.glb, select.png, card.png, icon_<ability>.png, prop_<model>.glb), and logo.svg for a name logo of your own (white, filled paths), then run `dlmod apply` on the hero's folder.\n",
	}
	for name, text := range files {
		if err := os.WriteFile(filepath.Join(dir, filepath.FromSlash(name)), []byte(text), 0o644); err != nil {
			return err
		}
	}
	return nil
}

func workspacePrompts(w *workspace) string {
	var b strings.Builder
	sum, required := w.credits()
	fmt.Fprintf(&b, "# %s: Meshy prompts\n\n", w.Name)
	if w.Concept == "" {
		fmt.Fprintf(&b, "Replace `%s` with a description of your character, for example *Rook, a hulking stone golem with glowing blue runes and a lantern for a heart*, and `%s` with its short name (*Rook*). Or set `concept` and `select.name` in `workspace.json` and run `dlmod meshy %s` again to fill them in.\n\n", conceptMark, characterMark, w.Hero)
	} else {
		fmt.Fprintf(&b, "Character: %s\n\n", w.Concept)
	}
	fmt.Fprintf(&b, "Everything: about %d credits (%d without the optional steps). Prices are Meshy's as of 2026-10; confirm before generating.\n\n", sum, required)
	for i, s := range w.Steps {
		fmt.Fprintf(&b, "## %d. %s", i+1, s.Title)
		if s.Optional {
			b.WriteString(" (optional)")
		}
		b.WriteString("\n\n")
		fmt.Fprintf(&b, "- Tool: `%s`", s.Tool)
		if s.AIModel != "" {
			fmt.Fprintf(&b, ", model `%s`", s.AIModel)
		}
		if s.TextureModel != "" {
			fmt.Fprintf(&b, ", texture `%s`", s.TextureModel)
		}
		fmt.Fprintf(&b, ", about %d credits\n", s.Credits)
		if s.Aspect != "" {
			fmt.Fprintf(&b, "- Aspect ratio: %s\n", s.Aspect)
		}
		if s.Reference != "" {
			fmt.Fprintf(&b, "- Reference: `%s`\n", s.Reference)
		}
		if len(s.Options) > 0 {
			keys := make([]string, 0, len(s.Options))
			for k := range s.Options {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			var opts []string
			for _, k := range keys {
				opts = append(opts, fmt.Sprintf("`%s: %v`", k, s.Options[k]))
			}
			fmt.Fprintf(&b, "- Options: %s\n", strings.Join(opts, ", "))
		}
		fmt.Fprintf(&b, "- Save as: `%s`", s.Output)
		if s.Target != "" {
			fmt.Fprintf(&b, ", replaces `%s`", s.Target)
			if s.Size != "" {
				fmt.Fprintf(&b, " (%s)", s.Size)
			}
		}
		b.WriteString("\n")
		if s.Note != "" {
			fmt.Fprintf(&b, "- %s\n", s.Note)
		}
		if s.Prompt != "" {
			fmt.Fprintf(&b, "\n```text\n%s\n```\n", filled(s.Prompt, w))
		}
		b.WriteString("\n")
	}
	return b.String()
}

func workspaceReadme(w *workspace) string {
	a := w.Game
	var b strings.Builder
	fmt.Fprintf(&b, "# %s (`%s`)\n\n", a.Name, a.ID)
	if a.Role != "" {
		fmt.Fprintf(&b, "%s.", strings.TrimSuffix(a.Role, "."))
		if a.Playstyle != "" {
			fmt.Fprintf(&b, " %s", a.Playstyle)
		}
		b.WriteString("\n\n")
	}
	if a.Weapon != "" {
		fmt.Fprintf(&b, "Weapon: %s. ", a.Weapon)
	}
	if len(a.Tags) > 0 {
		fmt.Fprintf(&b, "Tags: %s.", strings.Join(a.Tags, ", "))
	}
	b.WriteString("\n\nMake a replacement character with [prompts.md](prompts.md), save each result in `assets/`, then install everything that is there:\n\n")
	fmt.Fprintf(&b, "```powershell\ndlmod apply %s\n```\n\n", "meshy/"+slugify(a.Name))
	b.WriteString("Set the character's description (`concept`), the name and tags shown on the hero select screen and the tag colours in `workspace.json` first.\n\n")

	b.WriteString("## Abilities\n\n| Ability | What it does | Icon | Ability models |\n| --- | --- | --- | --- |\n")
	for _, ab := range a.Abilities {
		fmt.Fprintf(&b, "| %s | %s | `%s` | %s |\n", ab.Name, ab.summary(), path.Base(ab.Icon), codeList(ab.Props))
	}
	b.WriteString("\n## Game files\n\n")
	fmt.Fprintf(&b, "| What | File | Size |\n| --- | --- | --- |\n")
	fmt.Fprintf(&b, "| Model | `%s` | %s |\n", a.Model, heightText(a.Height))
	if a.Skeleton != "" {
		fmt.Fprintf(&b, "| Animation skeleton | `%s` | |\n", a.Skeleton)
	}
	if a.ClipFolder != "" {
		fmt.Fprintf(&b, "| Animation clips | `%s` | %d clips |\n", a.ClipFolder, a.Clips)
	}
	for _, f := range a.Images {
		fmt.Fprintf(&b, "| %s | `%s` | %s |\n", f.Label, f.Path, describeSize(f))
	}
	for _, f := range a.Icons {
		fmt.Fprintf(&b, "| Ability icon | `%s` | %s |\n", f.Path, describeSize(f))
	}
	for _, l := range a.Select.Logos {
		fmt.Fprintf(&b, "| Name logo (SVG) | `%s` | |\n", l)
	}
	if a.Select.Scene != "" {
		fmt.Fprintf(&b, "| Hero select 3D scene | `game/citadel/%s` | |\n", a.Select.Scene)
	}
	for _, p := range a.Props {
		fmt.Fprintf(&b, "| Ability model | `%s` | |\n", p)
	}
	for _, p := range a.Particles {
		fmt.Fprintf(&b, "| Effects | `%s` | |\n", p)
	}
	if len(a.NameKeys)+len(a.TagKeys) > 0 {
		fmt.Fprintf(&b, "| Name and tag text | %s | `citadel_heroes`, `citadel_gc_hero_names` |\n", codeList(strings.Join(append(append([]string{}, a.NameKeys...), a.TagKeys...), ", ")))
	}

	b.WriteString("\n## Animations\n\n")
	fmt.Fprintf(&b, "The game plays %s's own %d clips on whatever model is installed, so a replacement is rigged to %s's bones rather than given new animations. What the game does to those clips matters:\n\n", a.Name, a.Clips, a.Name)
	if len(a.KeyClips) > 0 {
		fmt.Fprintf(&b, "- Clips worth checking a rig against: %s.\n", codeList(strings.Join(a.KeyClips, ", ")))
	}
	if len(a.Masks) > 0 {
		fmt.Fprintf(&b, "- Melee, reloads and casts are laid over the movement through bone masks (%s): only the upper body takes the clip, so a fit to the whole clip is wrong in game.\n", codeList(strings.Join(a.Masks, ", ")))
	}
	b.WriteString("- The hero graph solves both arms by IK towards the gun (`weaponHand_L`, `weaponHand_R`) and the legs towards `leg_*_IKTARGET`, so the arm bones in game are not what the clips say. Body parts that should follow the hero safely use `head`, `neck_0`, the spine, the clavicles, `pelvis` and the leg IK targets.\n")
	b.WriteString("- `attachWorld` never moves: weight on it keeps a part steady. `root_motion` is the character's place; its rest pose in the file is turned, which dlmod corrects.\n")
	b.WriteString("- The third-person camera looks from behind and close: a wide character hides what happens in front of it, and a body moved backwards puts the camera inside it.\n\n")

	b.WriteString("## Install by hand\n\n```powershell\n")
	fmt.Fprintf(&b, "dlmod model %s assets/model.glb\n", a.ID)
	if a.Select.Background != "" {
		fmt.Fprintf(&b, "dlmod select %s bg=assets/select.png name=NAME tags=A,B,C colors=#7A3CFF,#FFB020 scene=hide\n", a.ID)
	}
	if len(a.Icons) > 0 {
		fmt.Fprintf(&b, "dlmod icons %s 1=assets/icon_....png\n", a.ID)
	}
	if f, ok := a.image("_card_psd"); ok {
		fmt.Fprintf(&b, "dlmod image %s assets/card.png\n", f.Path)
	}
	if len(a.Props) > 0 {
		fmt.Fprintf(&b, "dlmod prop %s assets/prop_%s.glb\n", a.Props[0], strings.TrimSuffix(path.Base(a.Props[0]), ".vmdl"))
	}
	fmt.Fprintf(&b, "dlmod paint %s\n```\n", a.ID)
	return b.String()
}

func codeList(s string) string {
	if s == "" {
		return ""
	}
	parts := strings.Split(s, ", ")
	for i, p := range parts {
		parts[i] = "`" + p + "`"
	}
	return strings.Join(parts, ", ")
}

func heightText(h float64) string {
	if h == 0 {
		return ""
	}
	return fmt.Sprintf("%.0f units tall", h)
}

// workspaceIndex is the top README listing every hero's workspace.
func workspaceIndex(all []*workspace) string {
	var b strings.Builder
	b.WriteString("# Meshy workspace for Deadlock heroes\n\n")
	b.WriteString("One folder per hero with the [Meshy](https://www.meshy.ai) prompts that make a full replacement character, written for that hero's role, weapon and abilities, and the game files each result replaces. Generated by `dlmod meshy all meshy` from the installed game; run it again after a game update.\n\n")
	b.WriteString("## How to make a character\n\n")
	b.WriteString("1. Open the hero's folder and write the character in `workspace.json` (`concept`), with the name, tags and tag colours for the hero select screen (`select`). Run `dlmod meshy <hero> meshy` to put the concept into the prompts.\n")
	b.WriteString("2. Generate the steps of `prompts.md` in order with the Meshy MCP tools or the Meshy site: the design sheet first, then everything else from it. Save each result in the hero's `assets/` under the name its step gives.\n")
	b.WriteString("3. Close the game and run `dlmod apply meshy/<hero>`. It installs the model, the hero select screen (art, name logo, tags, colours, with the 3D scene hidden), the cards with their green keyed out, the ability icons as white glyphs and the ability models, each step whose file is in `assets/`.\n")
	b.WriteString("4. `dlmod paint <hero>` recolours the ability effects to a palette.\n\n")
	b.WriteString("## Heroes\n\n| Hero | Folder | Abilities | Icons | Pictures | Ability models | Credits |\n| --- | --- | --- | --- | --- | --- | --- |\n")
	for _, w := range all {
		a := w.Game
		_, required := w.credits()
		fmt.Fprintf(&b, "| %s | [%s](%s/) | %d | %d | %d | %d | %d |\n", a.Name, slugify(a.Name), slugify(a.Name), len(a.Abilities), len(a.Icons), len(a.Images), len(a.Props), required)
	}
	b.WriteString("\n## Where it lives in the repository\n\nThe code is in [tools/dlmod](../tools/dlmod). Each hero's README lists that hero's game files: model, skeleton, clips, pictures, icons, name logo, hero select scene, ability models and effects.\n\n")
	b.WriteString("| File | What it does |\n| --- | --- |\n")
	for _, row := range [][2]string{
		{"tools/dlmod/meshy_command.go", "Writes these folders: the steps, prompts and README of each hero."},
		{"tools/dlmod/workspace.go", "Reads a hero from the game: model, skeleton, clips, bone masks, pictures, icons, ability models, effects, text."},
		{"tools/dlmod/apply_command.go", "`dlmod apply`: installs the generated files of a hero's folder."},
		{"tools/dlmod/models.go", "Hero model paths and `dlmod model`."},
		{"tools/dlmod/source2/replace.go", "Rewrites a hero's `.vmdl_c` around a new mesh, fitted and rigged to the hero's bones."},
		{"tools/dlmod/source2/anim.go", "Reads the hero's animation skeleton and clips."},
		{"tools/dlmod/heroselect.go", "`dlmod select`: hero select background, name logo, tags, tag colours, hidden 3D scene."},
		{"tools/dlmod/source2/heroscene.go", "Panorama text files (SVG logos, stylesheets) and the hero select scene."},
		{"tools/dlmod/logo.go", "The stroke font that draws name logos."},
		{"tools/dlmod/icons_command.go tools/dlmod/glyph.go", "`dlmod icons`: ability icons as white glyphs."},
		{"tools/dlmod/image_command.go tools/dlmod/chroma.go", "`dlmod image`: cards and portraits; green-screen keying."},
		{"tools/dlmod/prop_command.go tools/dlmod/source2/prop.go tools/dlmod/source2/legacy.go", "`dlmod prop`: ability models."},
		{"tools/dlmod/paint_command.go tools/dlmod/palette.go", "`dlmod paint`: ability effects, materials and textures recoloured."},
	} {
		var links []string
		for _, file := range strings.Fields(row[0]) {
			links = append(links, fmt.Sprintf("[%s](../%s)", strings.TrimPrefix(file, "tools/dlmod/"), file))
		}
		fmt.Fprintf(&b, "| %s | %s |\n", strings.Join(links, ", "), row[1])
	}
	return b.String()
}

// meshyCommand: dlmod meshy <hero|all> [folder] [concept=...].
func meshyCommand(ui *UI, game Game, args []string) error {
	if len(args) == 0 {
		return errors.New("name a hero or all, for example: dlmod meshy ivy meshy concept=\"Rook, a hulking stone golem with glowing blue runes\"")
	}
	dir, concept := "meshy", ""
	for _, arg := range args[1:] {
		if v, ok := strings.CutPrefix(arg, "concept="); ok {
			concept = strings.TrimSpace(v)
		} else {
			dir = arg
		}
	}
	heroes := heroIDs()
	if args[0] != "all" {
		hero, err := findHero(args[0])
		if err != nil {
			return err
		}
		heroes = []string{hero}
	}
	index, err := openWorkspaceIndex(game)
	if err != nil {
		return fmt.Errorf("the game's files could not be read: %v", err)
	}
	var all []*workspace
	for _, hero := range heroes {
		a, err := gatherHero(game, index, hero)
		if err != nil {
			ui.say("%s: %v", heroDisplayName(hero), err)
			continue
		}
		w := buildWorkspace(a)
		w.Concept = concept
		folder := filepath.Join(dir, slugify(a.Name))
		if err := writeWorkspace(folder, w); err != nil {
			return err
		}
		all = append(all, w)
		_, required := w.credits()
		ui.say("%-14s %d abilities, %d icons, %d pictures, %d ability models, about %d credits -> %s", a.Name, len(a.Abilities), len(a.Icons), len(a.Images), len(a.Props), required, folder)
	}
	if args[0] == "all" {
		sort.Slice(all, func(i, j int) bool { return all[i].Name < all[j].Name })
		if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte(workspaceIndex(all)), 0o644); err != nil {
			return err
		}
	}
	return nil
}
