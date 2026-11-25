package main

import "strings"

// The creator's saved state, which character packs (pack.go) carry with or without Meshy in dlmod.

type creatorAnswers struct {
	Name    string   `json:"name"`
	What    string   `json:"what"`  // human, creature, robot, animal, alien, monster…
	Body    string   `json:"body"`  // humanoid, round, quadruped
	Style   string   `json:"style"` // stylized cartoon, painterly, realistic, toy…
	Colors  []string `json:"colors"`
	Item    string   `json:"item"`
	Details string   `json:"details"`
	Tags    []string `json:"tags"`
}

type creatorGen struct {
	ID       string   `json:"id"`
	Step     string   `json:"step"`
	Kind     string   `json:"kind"` // Meshy task type
	Model    string   `json:"model"`
	TaskID   string   `json:"taskId"`
	Status   string   `json:"status"` // PENDING, IN_PROGRESS, SUCCEEDED (downloaded), FAILED, CANCELED
	Progress int      `json:"progress"`
	Queue    int      `json:"queue"`
	Files    []string `json:"files"` // under gen/, once downloaded
	Thumb    string   `json:"thumb,omitempty"`
	Error    string   `json:"error,omitempty"`
	Credits  int      `json:"credits"`
	Prompt   string   `json:"prompt"`
	Created  int64    `json:"created"`
	EditOf   string   `json:"editOf,omitempty"` // the result this one changes
}

func (g *creatorGen) running() bool {
	return g.Status != "SUCCEEDED" && g.Status != "FAILED" && g.Status != "CANCELED"
}

type creatorState struct {
	Hero       string            `json:"hero"`
	Answers    creatorAnswers    `json:"answers"`
	Concept    string            `json:"concept"`
	Gens       []*creatorGen     `json:"generations"`
	Chosen     map[string]string `json:"chosen"`               // step -> generation kept
	RefsForAll bool              `json:"refsForAll,omitempty"` // every picture gets the reference pictures, not only the design sheet
}

// creatorSteps are the workspace steps the creator runs, in the order the page shows them. An
// ability model is made from a picture, as the character's model is from its views: first the
// object drawn alone from the design sheet (propimg_<name>), then a 3D model of that picture with
// smart topology (prop_<name>).
func creatorSteps(w *workspace) []workspaceStep {
	var out []workspaceStep
	for _, s := range w.Steps {
		switch {
		case s.ID == "weapon":
			continue
		case strings.HasPrefix(s.ID, "prop_"):
			base := strings.TrimPrefix(s.ID, "prop_")
			picture := s
			picture.ID, picture.Title, picture.Kind = "propimg_"+base, "Ability model picture: "+base, "propimg"
			picture.Tool, picture.AIModel, picture.Credits, picture.Aspect = "meshy_image_to_image", "nano-banana-pro", 9, "1:1"
			picture.Reference, picture.Output, picture.Options = sheetFile, "assets/prop_"+base+".png", nil
			picture.Prompt = "Draw only this object, in the style and colours of the character design sheet: alone, centered, " +
				"in a three-quarter view, on a plain white background, with no characters and no text. " + s.Prompt
			picture.Note = "The picture the ability model is made from."
			model := s
			model.Tool, model.AIModel, model.Credits, model.Reference = "meshy_image_to_3d", "meshy-t2.5", 15, picture.Output
			model.Note = "Made from the picture with smart topology, then put in place of the game's model; dlmod pins it to the prop's root bone."
			out = append(out, picture, model)
		case s.Kind == "image":
			// Cards and portraits stand on transparency: Meshy removes the background itself, so
			// the prompt need not ask for a green one to key out (and the character may be green).
			options := map[string]any{"remove_background": true}
			for k, v := range s.Options {
				options[k] = v
			}
			s.Options = options
			s.Prompt = strings.Replace(s.Prompt, greenBackground, " On a plain light background, no text.", 1)
			s.Note = "Meshy removes the background."
			out = append(out, s)
		default:
			out = append(out, s)
		}
	}
	return out
}
