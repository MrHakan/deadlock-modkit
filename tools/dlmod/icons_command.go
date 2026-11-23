package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/MrHakan/deadlock-modkit/tools/dlmod/source2"
)

// iconsCommand replaces a hero's ability icons with pictures: dlmod icons <hero> lists them,
// dlmod icons <hero> <icon>=<picture> ... replaces them. An icon is named by its number in the
// list or part of its file name (stone_form). Pictures become white glyphs on transparency, as
// the game's icons are (glyphImage): draw them white on black, or on transparency. They go into
// the "Custom images" mod with the hero's other pictures.
func iconsCommand(ui prompter, store *Store, game Game, args []string) error {
	if len(args) == 0 {
		return errors.New("name a hero, for example: dlmod icons ivy or dlmod icons ivy stone_form=stone.png")
	}
	hero, err := findHero(args[0])
	if err != nil {
		return err
	}
	s := &studio{game: game}
	images, err := s.heroImages(hero)
	if err != nil {
		return err
	}
	var icons []string
	for _, img := range images {
		if img.Group == "Abilities" {
			icons = append(icons, img.Path)
		}
	}
	if len(icons) == 0 {
		return fmt.Errorf("%s has no ability icons in this game build", heroDisplayName(hero))
	}
	if len(args) == 1 {
		ui.say("%s's ability icons:", heroDisplayName(hero))
		for i, icon := range icons {
			ui.say("  %d  %s", i+1, icon)
		}
		ui.say("Replace one with: dlmod icons %s %d=picture.png (white on black, or on transparency)", hero, 1)
		return nil
	}
	if gameRunning() {
		return errRunning
	}
	index, err := s.imageIndex()
	if err != nil {
		return err
	}
	project, err := loadImageProject(game)
	if err != nil {
		return err
	}
	for _, arg := range args[1:] {
		which, file, ok := strings.Cut(arg, "=")
		if !ok {
			return fmt.Errorf("%s is not ICON=PICTURE", arg)
		}
		target, err := pickIcon(icons, which)
		if err != nil {
			return err
		}
		data, err := os.ReadFile(file)
		if err != nil {
			return err
		}
		picture, err := source2.DecodeImage(data)
		if err != nil {
			return fmt.Errorf("%s: %v (use a PNG or JPEG)", file, err)
		}
		original, err := index.Read(target)
		if err != nil {
			return err
		}
		_, info, err := source2.DecodeTexture(original)
		if err != nil {
			return err
		}
		compiled, err := source2.ReplaceTextureImage(original, glyphImage(picture, info.Width, 0.78))
		if err != nil {
			return err
		}
		if err := project.add(target, filepath.Base(file), compiled); err != nil {
			return err
		}
		ui.say("%s now shows %s as a %dx%d glyph.", filepath.Base(target), filepath.Base(file), info.Width, info.Height)
	}
	if err := store.installReplacements(ui, project); err != nil {
		return err
	}
	return ensurePatched(ui, game)
}

// pickIcon finds an icon by its number in the list or by part of its file name.
func pickIcon(icons []string, which string) (string, error) {
	if n, err := strconv.Atoi(which); err == nil {
		if n < 1 || n > len(icons) {
			return "", fmt.Errorf("there is no icon %d (1 to %d)", n, len(icons))
		}
		return icons[n-1], nil
	}
	var found []string
	for _, icon := range icons {
		if strings.Contains(filepath.Base(icon), strings.ToLower(which)) {
			found = append(found, icon)
		}
	}
	switch len(found) {
	case 1:
		return found[0], nil
	case 0:
		return "", fmt.Errorf("no icon is called %q", which)
	}
	return "", fmt.Errorf("%q could be %s", which, strings.Join(found, ", "))
}
