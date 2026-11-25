package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/MrHakan/deadlock-modkit/tools/dlmod/source2"
)

// imageCommand puts pictures over game images from the command line, as the studio's Images tab
// does: dlmod image <game image> <picture> [<game image> <picture> ...]. A game image is named by
// its path in the game (panorama/images/heroes/tengu_card_psd.vtex_c, with or without _c).
func imageCommand(ui prompter, store *Store, game Game, args []string) error {
	if len(args) == 0 || len(args)%2 != 0 {
		return errors.New("name pairs of a game image and a picture, for example: dlmod image panorama/images/heroes/tengu_card_psd.vtex_c card.png")
	}
	if gameRunning() {
		return errRunning
	}
	index, err := source2.OpenVPKFiltered(filepath.Join(game.Citadel(), "pak01_dir.vpk"), func(name string) bool {
		return strings.HasPrefix(name, "panorama/images/") && strings.HasSuffix(name, ".vtex_c")
	})
	if err != nil {
		return fmt.Errorf("the game's images could not be read: %v", err)
	}
	project, err := loadImageProject(game)
	if err != nil {
		return err
	}
	for i := 0; i < len(args); i += 2 {
		target := strings.TrimPrefix(filepath.ToSlash(args[i]), "/")
		if !strings.HasSuffix(target, "_c") {
			target += "_c"
		}
		if _, ok := index.Files[target]; !ok {
			return fmt.Errorf("the game has no image %s", target)
		}
		data, err := os.ReadFile(args[i+1])
		if err != nil {
			return err
		}
		picture, err := source2.DecodeImage(data)
		if err != nil {
			return fmt.Errorf("%s: %v (use a PNG or JPEG)", args[i+1], err)
		}
		original, err := index.Read(target)
		if err != nil {
			return err
		}
		compiled, err := source2.ReplaceTextureImage(original, picture)
		if err != nil {
			return err
		}
		if err := project.add(target, filepath.Base(args[i+1]), compiled); err != nil {
			return err
		}
		_, info, _ := source2.DecodeTexture(compiled)
		b := picture.Bounds()
		ui.say("%s now shows %s (%dx%d, fitted to %dx%d).", filepath.Base(target), filepath.Base(args[i+1]), b.Dx(), b.Dy(), info.Width, info.Height)
	}
	if err := store.installReplacements(ui, project); err != nil {
		return err
	}
	return ensurePatched(ui, game)
}
