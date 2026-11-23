//go:build !nomeshy

package main

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"

	"net/http"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/MrHakan/deadlock-modkit/tools/dlmod/source2"
)

// The studio's character creator, its Create tab: a few questions describe a character, and
// Meshy makes it step by step from a hero's workspace (workspace_steps.go): a design sheet, views
// from the front, side and back, the 3D model, an optional Meshy rig, then the hero select art,
// cards and ability icons. Every generation runs in the background while the page shows its
// progress; each result can be kept or made again, and the kept one moves on to the next step.
// The 3D model goes straight to the Model tab to be rigged on the hero; pictures install with
// installPictures. Each hero's creation lives in game/dlmod/characters/<hero>, with the workspace
// files of dlmod meshy, so dlmod apply can install it too.

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

// concept is the description every prompt is built on, written from the answers.
func (a creatorAnswers) concept() string {
	name := strings.TrimSpace(a.Name)
	if name == "" {
		name = "The character"
	}
	thing := strings.TrimSpace(strings.ToLower(strings.TrimSpace(a.Style) + " " + strings.TrimSpace(a.What)))
	if thing == "" {
		thing = "character"
	}
	article := "a"
	if strings.ContainsRune("aeiou", rune(thing[0])) {
		article = "an"
	}
	var b strings.Builder
	fmt.Fprintf(&b, "%s, %s %s", name, article, thing)
	switch a.Body {
	case "humanoid":
		b.WriteString(" with two arms and two legs")
	case "round":
		b.WriteString(" with a round body and no legs")
	case "quadruped":
		b.WriteString(" that walks on four legs")
	}
	var colors []string
	for _, c := range a.Colors {
		if c = strings.TrimSpace(c); c != "" {
			colors = append(colors, c)
		}
	}
	if len(colors) > 0 {
		fmt.Fprintf(&b, ", coloured %s", joinWords(colors))
	}
	if item := strings.TrimSpace(a.Item); item != "" {
		fmt.Fprintf(&b, ", holding %s", item)
	}
	text := b.String()
	if details := strings.TrimSpace(a.Details); details != "" {
		text += ". " + strings.TrimRight(details, ".")
	}
	return text
}

func joinWords(list []string) string {
	if len(list) < 2 {
		return strings.Join(list, "")
	}
	return strings.Join(list[:len(list)-1], ", ") + " and " + list[len(list)-1]
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
	Hero    string            `json:"hero"`
	Answers creatorAnswers    `json:"answers"`
	Concept string            `json:"concept"`
	Gens    []*creatorGen     `json:"generations"`
	Chosen  map[string]string `json:"chosen"` // step -> generation kept
}

type creatorSession struct {
	mu    sync.Mutex
	dir   string
	state creatorState
	work  *workspace
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

// creatorKind is the Meshy task type a step runs with a model: the 3D step makes one model from
// several views, or from the front view with smart topology.
func creatorKind(step workspaceStep, model string) string {
	switch {
	case step.ID == "model" && (model == "meshy-t2.5" || model == "meshy-t2"):
		return "image-to-3d"
	case step.ID == "model":
		return "multi-image-to-3d"
	case strings.HasPrefix(step.ID, "prop_"):
		return "image-to-3d"
	case step.ID == "rig":
		return "rigging"
	case step.ID == "design_sheet":
		return "text-to-image"
	}
	return "image-to-image"
}

// creatorModels are the choices the page offers for a step, best first.
func creatorModels(step workspaceStep) []string {
	switch step.ID {
	case "model":
		return []string{"meshy-7.1", "meshy-t2.5"}
	case "rig":
		return []string{"meshyrig"}
	}
	if strings.HasPrefix(step.ID, "prop_") {
		return []string{"meshy-t2.5", "meshy-7.1"}
	}
	return []string{"nano-banana-pro", "nano-banana-2", "nano-banana", "gpt-image-2"}
}

func (s *studio) charactersDir(hero string) string {
	return filepath.Join(s.game.StateDir(), "characters", hero)
}

// session loads (or starts) a hero's creation, and picks up generations still running.
func (s *studio) session(hero string) (*creatorSession, error) {
	hero, err := findHero(hero)
	if err != nil {
		return nil, err
	}
	s.mu.Lock()
	if s.sessions == nil {
		s.sessions = map[string]*creatorSession{}
	}
	if c := s.sessions[hero]; c != nil {
		s.mu.Unlock()
		return c, nil
	}
	s.mu.Unlock()
	c := &creatorSession{dir: s.charactersDir(hero), state: creatorState{Hero: hero, Chosen: map[string]string{}}}
	if data, err := os.ReadFile(filepath.Join(c.dir, "creator.json")); err == nil {
		if err := json.Unmarshal(data, &c.state); err != nil {
			return nil, fmt.Errorf("creator.json: %v", err)
		}
		if c.state.Chosen == nil {
			c.state.Chosen = map[string]string{}
		}
	}
	if err := s.rebuildWorkspace(c); err != nil {
		return nil, err
	}
	s.mu.Lock()
	if existing := s.sessions[hero]; existing != nil { // another request got there first
		s.mu.Unlock()
		return existing, nil
	}
	s.sessions[hero] = c
	s.mu.Unlock()
	for _, g := range c.state.Gens {
		if g.running() && g.TaskID != "" {
			go s.follow(c, g)
		}
	}
	return c, nil
}

// rebuildWorkspace writes the hero's workspace with the current concept and name.
func (s *studio) rebuildWorkspace(c *creatorSession) error {
	index, err := s.workspaceIndex()
	if err != nil {
		return err
	}
	a, err := gatherHero(s.game, index, c.state.Hero)
	if err != nil {
		return err
	}
	w := buildWorkspace(a)
	w.Concept = c.state.Concept
	w.Select.Name = strings.TrimSpace(c.state.Answers.Name)
	if len(c.state.Answers.Tags) > 0 {
		w.Select.Tags = c.state.Answers.Tags
	}
	for _, color := range c.state.Answers.Colors {
		if _, ok := parseHex(color); ok {
			w.Select.Colors = append(w.Select.Colors, strings.ToUpper(color))
		}
	}
	if err := writeWorkspace(c.dir, w); err != nil {
		return err
	}
	c.work = w
	return nil
}

func (c *creatorSession) save() error {
	data, err := json.MarshalIndent(c.state, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(c.dir, 0o755); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(c.dir, "creator.json"), data, 0o644)
}

func (c *creatorSession) step(id string) (workspaceStep, bool) {
	for _, s := range creatorSteps(c.work) {
		if s.ID == id {
			return s, true
		}
	}
	return workspaceStep{}, false
}

func (c *creatorSession) gen(id string) *creatorGen {
	for _, g := range c.state.Gens {
		if g.ID == id {
			return g
		}
	}
	return nil
}

// kept is the generation kept for a step, if it finished.
func (c *creatorSession) kept(step string) *creatorGen {
	g := c.gen(c.state.Chosen[step])
	if g == nil || g.Status != "SUCCEEDED" {
		return nil
	}
	return g
}

// needs is the step a step builds on, which must have a result kept first.
func creatorNeeds(step string) string {
	switch step {
	case "design_sheet":
		return ""
	case "model":
		return "turnaround"
	case "rig":
		return "model"
	}
	if base, ok := strings.CutPrefix(step, "prop_"); ok {
		return "propimg_" + base
	}
	return "design_sheet"
}

type creatorStepView struct {
	workspaceStep
	Task    string         `json:"task"`   // the Meshy task type of the default model
	Models  map[string]int `json:"models"` // choice -> credits
	Default string         `json:"default"`
	Group   string         `json:"group"`
	Needs   string         `json:"needs"`
	Ready   bool           `json:"ready"`
	Filled  string         `json:"filled"` // the prompt with the character put in
}

// view is everything the Create tab shows.
func (c *creatorSession) view(s *studio) map[string]any {
	var steps []creatorStepView
	for _, st := range creatorSteps(c.work) {
		models := map[string]int{}
		for _, m := range creatorModels(st) {
			models[m] = meshyPrice(creatorKind(st, m), m)
		}
		group := "pictures"
		switch st.ID {
		case "design_sheet", "turnaround":
			group = "character"
		case "model", "rig":
			group = "model"
		}
		if strings.HasPrefix(st.ID, "prop_") || strings.HasPrefix(st.ID, "propimg_") {
			group = "props"
		}
		needs := creatorNeeds(st.ID)
		steps = append(steps, creatorStepView{
			workspaceStep: st, Task: creatorKind(st, creatorModels(st)[0]), Models: models, Default: creatorModels(st)[0], Group: group, Needs: needs,
			Ready: needs == "" || c.kept(needs) != nil, Filled: filled(st.Prompt, c.work),
		})
	}
	gens := c.state.Gens
	if gens == nil {
		gens = []*creatorGen{}
	}
	return map[string]any{
		"hero": c.state.Hero, "name": heroDisplayName(c.state.Hero), "answers": c.state.Answers, "concept": c.state.Concept,
		"steps": steps, "generations": gens, "chosen": c.state.Chosen, "hasKey": loadMeshyKey(s.game) != "",
		"folder": c.dir,
	}
}

type creatorRequest struct {
	Hero    string         `json:"hero"`
	Step    string         `json:"step"`
	Gen     string         `json:"gen"`
	Prompt  string         `json:"prompt"`
	Model   string         `json:"model"`
	Confirm bool           `json:"confirm"`
	Answers creatorAnswers `json:"answers"`
	Concept string         `json:"concept"`
	Key     string         `json:"key"`
	Edit    string         `json:"edit"`    // run: change this earlier result instead
	Effects bool           `json:"effects"` // install: recolour the hero's effects with the character's colours
}

func (s *studio) creatorRequest(r *http.Request) (creatorRequest, *creatorSession, error) {
	var req creatorRequest
	if err := readJSON(r, &req); err != nil {
		return req, nil, err
	}
	c, err := s.session(req.Hero)
	return req, c, err
}

func (s *studio) creatorState(r *http.Request) (any, error) {
	_, c, err := s.creatorRequest(r)
	if err != nil {
		return nil, err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.view(s), nil
}

// creatorKey checks a Meshy API key against the account's balance and keeps it.
func (s *studio) creatorKey(r *http.Request) (any, error) {
	var req creatorRequest
	if err := readJSON(r, &req); err != nil {
		return nil, err
	}
	key := strings.TrimSpace(req.Key)
	if key == "" {
		return nil, errors.New("paste your Meshy API key (meshy.ai → Settings → API)")
	}
	balance, err := newMeshyClient(s.game, key).balance()
	if err != nil {
		return nil, err
	}
	if err := saveMeshyKey(s.game, key); err != nil {
		return nil, err
	}
	return map[string]any{"balance": balance}, nil
}

func (s *studio) creatorBalance(*http.Request) (any, error) {
	balance, err := newMeshyClient(s.game, loadMeshyKey(s.game)).balance()
	if err != nil {
		return nil, err
	}
	return map[string]any{"balance": balance}, nil
}

// creatorAnswer keeps the answers and writes the prompts from them (or from a concept the user
// wrote themselves).
func (s *studio) creatorAnswer(r *http.Request) (any, error) {
	req, c, err := s.creatorRequest(r)
	if err != nil {
		return nil, err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.state.Answers = req.Answers
	c.state.Concept = strings.TrimSpace(req.Concept)
	if c.state.Concept == "" {
		c.state.Concept = req.Answers.concept()
	}
	if err := s.rebuildWorkspace(c); err != nil {
		return nil, err
	}
	if err := c.save(); err != nil {
		return nil, err
	}
	return c.view(s), nil
}

// dataURI reads a picture as a data: address, which Meshy takes in place of a public one.
func dataURI(file string) (string, error) {
	data, err := os.ReadFile(file)
	if err != nil {
		return "", err
	}
	mime := "image/png"
	if ext := strings.ToLower(filepath.Ext(file)); ext == ".jpg" || ext == ".jpeg" {
		mime = "image/jpeg"
	}
	return "data:" + mime + ";base64," + base64.StdEncoding.EncodeToString(data), nil
}

// editableSteps take an edit: steps with one picture as their result.
func editable(step workspaceStep) bool {
	return step.ID != "turnaround" && step.ID != "model" && step.ID != "rig" && !strings.HasPrefix(step.ID, "prop_")
}

// creatorRun starts one step, or an edit of one of its pictures, after the price is confirmed.
func (s *studio) creatorRun(r *http.Request) (any, error) {
	req, c, err := s.creatorRequest(r)
	if err != nil {
		return nil, err
	}
	price, err := c.price(req.Step, req.Model, req.Edit)
	if err != nil {
		return nil, err
	}
	if !req.Confirm {
		return nil, fmt.Errorf("this costs %d credits; confirm it first", price)
	}
	if _, err := s.startStep(c, req.Step, req.Model, req.Prompt, req.Edit); err != nil {
		return nil, err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.view(s), nil
}

// creatorBatch starts several steps at once, after their total price is confirmed; Meshy works
// on them side by side.
func (s *studio) creatorBatch(r *http.Request) (any, error) {
	var req struct {
		Hero    string `json:"hero"`
		Confirm bool   `json:"confirm"`
		Steps   []struct {
			Step  string `json:"step"`
			Model string `json:"model"`
		} `json:"steps"`
	}
	if err := readJSON(r, &req); err != nil {
		return nil, err
	}
	c, err := s.session(req.Hero)
	if err != nil {
		return nil, err
	}
	if len(req.Steps) == 0 {
		return nil, errors.New("there is no step to make")
	}
	total := 0
	for _, item := range req.Steps {
		price, err := c.price(item.Step, item.Model, "")
		if err != nil {
			return nil, err
		}
		total += price
	}
	if !req.Confirm {
		return nil, fmt.Errorf("these %d steps cost %d credits together; confirm it first", len(req.Steps), total)
	}
	var problems []string
	started := 0
	for _, item := range req.Steps {
		if _, err := s.startStep(c, item.Step, item.Model, "", ""); err != nil {
			problems = append(problems, item.Step+": "+err.Error())
			continue
		}
		started++
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	view := c.view(s)
	view["started"], view["problems"] = started, problems
	return view, nil
}

// price is what a step (or an edit of one of its pictures) costs with a model.
func (c *creatorSession) price(stepID, model, edit string) (int, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	step, ok := c.step(stepID)
	if !ok {
		return 0, fmt.Errorf("there is no step %q", stepID)
	}
	if model == "" {
		model = creatorModels(step)[0]
	}
	kind := creatorKind(step, model)
	if edit != "" {
		kind = "image-to-image"
	}
	return meshyPrice(kind, model), nil
}

// startStep sends one step to Meshy and follows it. With edit, the picture of that earlier result
// is changed as the prompt says instead, and the result counts as the step's.
func (s *studio) startStep(c *creatorSession, stepID, model, prompt, edit string) (*creatorGen, error) {
	c.mu.Lock()
	step, ok := c.step(stepID)
	if !ok {
		c.mu.Unlock()
		return nil, fmt.Errorf("there is no step %q", stepID)
	}
	if model == "" {
		model = creatorModels(step)[0]
	}
	kind := creatorKind(step, model)
	var base *creatorGen
	prompt = strings.TrimSpace(prompt)
	if edit != "" {
		if base = c.gen(edit); base == nil || base.Status != "SUCCEEDED" || base.Step != stepID || len(base.Files) == 0 {
			c.mu.Unlock()
			return nil, errors.New("only a finished picture of this step can be edited")
		}
		if !editable(step) {
			c.mu.Unlock()
			return nil, fmt.Errorf("the %s step cannot be edited; make another instead", stepID)
		}
		if prompt == "" {
			c.mu.Unlock()
			return nil, errors.New("say what to change")
		}
		kind = "image-to-image"
		prompt = "Change this picture: " + prompt + ". Keep everything else exactly as it is: the character, the style, the framing and the background."
	} else {
		if prompt == "" {
			prompt = filled(step.Prompt, c.work)
		}
		if needs := creatorNeeds(step.ID); needs != "" {
			if base = c.kept(needs); base == nil {
				c.mu.Unlock()
				return nil, fmt.Errorf("keep a result of the %s step first", needs)
			}
		}
	}
	if strings.Contains(prompt, conceptMark) || strings.Contains(prompt, characterMark) {
		c.mu.Unlock()
		return nil, errors.New("describe your character first: the prompt still has a blank in it")
	}
	price := meshyPrice(kind, model)
	var files []string
	if base != nil {
		for _, f := range base.Files {
			files = append(files, filepath.Join(c.dir, "gen", f))
		}
	}
	c.mu.Unlock()

	body := map[string]any{}
	switch kind {
	case "text-to-image":
		body = map[string]any{"ai_model": model, "prompt": prompt, "aspect_ratio": orDefault(step.Aspect, "16:9")}
	case "image-to-image":
		ref, err := dataURI(files[0])
		if err != nil {
			return nil, err
		}
		body = map[string]any{"ai_model": model, "prompt": prompt, "reference_image_urls": []string{ref}}
		if step.ID == "turnaround" && edit == "" {
			body["generate_multi_view"] = true
		} else {
			body["aspect_ratio"] = orDefault(step.Aspect, "1:1")
		}
		if step.Options["remove_background"] == true {
			body["remove_background"] = true
		}
	case "multi-image-to-3d":
		var refs []string
		for i, f := range files {
			if i == 4 {
				break
			}
			ref, err := dataURI(f)
			if err != nil {
				return nil, err
			}
			refs = append(refs, ref)
		}
		body = map[string]any{"image_urls": refs, "ai_model": model, "texture_model": "meshy-7", "should_texture": true,
			"texture_resolution": "2k", "target_formats": []string{"glb"}}
	case "image-to-3d":
		ref, err := dataURI(files[0])
		if err != nil {
			return nil, err
		}
		polycount := 30000
		switch n := step.Options["target_polycount"].(type) { // ability models are small and many
		case int:
			polycount = n
		case float64:
			polycount = int(n)
		}
		body = map[string]any{"image_url": ref, "ai_model": model, "texture_model": "meshy-n1", "should_texture": true,
			"target_polycount": polycount}
	case "rigging":
		body = map[string]any{"input_task_id": base.TaskID, "height_meters": 1.7}
	}
	client := newMeshyClient(s.game, loadMeshyKey(s.game))
	taskID, err := client.create(kind, body)
	if err != nil {
		return nil, err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	g := &creatorGen{
		ID: fmt.Sprintf("%s-%d", step.ID, time.Now().UnixNano()), Step: step.ID, Kind: kind, Model: model, TaskID: taskID,
		Status: "PENDING", Credits: price, Prompt: prompt, Created: time.Now().Unix(), EditOf: edit,
	}
	if kind == "rigging" || kind == "multi-image-to-3d" || kind == "image-to-3d" {
		g.Prompt = "" // the 3D steps work from pictures
	}
	c.state.Gens = append(c.state.Gens, g)
	if err := c.save(); err != nil {
		return nil, err
	}
	go s.follow(c, g)
	return g, nil
}

func orDefault(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}

// creatorPoll is the progress of every generation, which the page asks for while one runs.
func (s *studio) creatorPoll(r *http.Request) (any, error) {
	_, c, err := s.creatorRequest(r)
	if err != nil {
		return nil, err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	steps := map[string]bool{}
	for _, st := range creatorSteps(c.work) {
		needs := creatorNeeds(st.ID)
		steps[st.ID] = needs == "" || c.kept(needs) != nil
	}
	gens := c.state.Gens
	if gens == nil {
		gens = []*creatorGen{}
	}
	return map[string]any{"generations": gens, "chosen": c.state.Chosen, "ready": steps}, nil
}

var creatorPollEvery = 2 * time.Second // tests make it shorter

// follow watches a generation on Meshy until it ends, then downloads what it made.
func (s *studio) follow(c *creatorSession, g *creatorGen) {
	client := newMeshyClient(s.game, loadMeshyKey(s.game))
	failures := 0
	for {
		t, err := client.task(g.Kind, g.TaskID)
		if err != nil {
			failures++
			if failures < 5 { // a dropped connection is not a failed task
				time.Sleep(creatorPollEvery * 2)
				continue
			}
			c.mu.Lock()
			g.Status, g.Error = "FAILED", err.Error()
			c.save()
			c.mu.Unlock()
			return
		}
		failures = 0
		c.mu.Lock()
		g.Progress, g.Queue = t.Progress, t.Preceding
		if t.Credits > 0 {
			g.Credits = t.Credits
		}
		if !t.finished() {
			g.Status = t.Status
			c.mu.Unlock()
			time.Sleep(creatorPollEvery)
			continue
		}
		c.mu.Unlock()
		if t.Status != "SUCCEEDED" {
			c.mu.Lock()
			g.Status, g.Error = t.Status, t.failure()
			c.save()
			c.mu.Unlock()
			return
		}
		files, thumb, err := s.downloadResult(c, g, t)
		c.mu.Lock()
		if err != nil {
			g.Status, g.Error = "FAILED", "the result could not be downloaded: "+err.Error()
		} else {
			g.Status, g.Progress, g.Files, g.Thumb = "SUCCEEDED", 100, files, thumb
			// The first result of a step is kept until another is picked, and an edit replaces what it edits.
			if kept := c.kept(g.Step); kept == nil || (g.EditOf != "" && kept.ID == g.EditOf) {
				c.choose(g)
			}
		}
		c.save()
		c.mu.Unlock()
		return
	}
}

// downloadResult saves a finished task's pictures or model under gen/.
func (s *studio) downloadResult(c *creatorSession, g *creatorGen, t meshyTask) (files []string, thumb string, err error) {
	dir := filepath.Join(c.dir, "gen")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, "", err
	}
	get := func(address, name string) error {
		file, err := os.Create(filepath.Join(dir, name))
		if err != nil {
			return err
		}
		defer file.Close()
		return fetch(address, file)
	}
	switch g.Kind {
	case "text-to-image", "image-to-image":
		for i, address := range t.ImageURLs {
			name := fmt.Sprintf("%s_%d.png", g.ID, i+1)
			if err := get(address, name); err != nil {
				return nil, "", err
			}
			files = append(files, name)
		}
		if len(files) == 0 {
			return nil, "", errors.New("Meshy sent no picture")
		}
	default:
		address := t.ModelURLs["glb"]
		if t.Result != nil && t.Result.RiggedGLB != "" {
			address = t.Result.RiggedGLB
		}
		if address == "" {
			return nil, "", errors.New("Meshy sent no .glb model")
		}
		name := g.ID + ".glb"
		if err := get(address, name); err != nil {
			return nil, "", err
		}
		files = []string{name}
		if t.ThumbnailURL != "" && get(t.ThumbnailURL, g.ID+"_thumb.png") == nil {
			thumb = g.ID + "_thumb.png"
		}
	}
	return files, thumb, nil
}

// choose keeps a generation for its step and copies it to the workspace's assets, where dlmod
// apply finds it. The caller holds c.mu.
func (c *creatorSession) choose(g *creatorGen) error {
	step, ok := c.step(g.Step)
	if !ok {
		return fmt.Errorf("there is no step %q", g.Step)
	}
	c.state.Chosen[g.Step] = g.ID
	copyTo := func(from, to string) error {
		data, err := os.ReadFile(filepath.Join(c.dir, "gen", from))
		if err != nil {
			return err
		}
		to = filepath.Join(c.dir, filepath.FromSlash(to))
		if err := os.MkdirAll(filepath.Dir(to), 0o755); err != nil {
			return err
		}
		return os.WriteFile(to, data, 0o644)
	}
	switch g.Step {
	case "turnaround":
		for i, f := range g.Files {
			if err := copyTo(f, fmt.Sprintf("assets/views_%d.png", i+1)); err != nil {
				return err
			}
		}
		return copyTo(g.Files[0], "assets/views.png")
	case "rig":
		return copyTo(g.Files[0], "assets/model.glb") // the rigged model takes the plain one's place
	}
	return copyTo(g.Files[0], step.Output)
}

func (s *studio) creatorChoose(r *http.Request) (any, error) {
	req, c, err := s.creatorRequest(r)
	if err != nil {
		return nil, err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	g := c.gen(req.Gen)
	if g == nil || g.Status != "SUCCEEDED" {
		return nil, errors.New("that result is not ready")
	}
	if err := c.choose(g); err != nil {
		return nil, err
	}
	if err := c.save(); err != nil {
		return nil, err
	}
	return c.view(s), nil
}

// creatorFile sends a generated picture to the page (GET with the token in the address).
func (s *studio) creatorFile(r *http.Request) (any, error) {
	c, err := s.session(r.URL.Query().Get("hero"))
	if err != nil {
		return nil, err
	}
	name := path.Base(r.URL.Query().Get("name"))
	if name == "." || name == "/" || strings.ContainsFunc(name, func(r rune) bool { return r == '\\' || unicode.IsControl(r) }) {
		return nil, errors.New("bad file name")
	}
	data, err := os.ReadFile(filepath.Join(c.dir, "gen", name))
	if err != nil {
		return nil, err
	}
	contentType := "image/png"
	if strings.HasSuffix(name, ".glb") {
		contentType = "model/gltf-binary"
	}
	return download{data: data, contentType: contentType}, nil
}

// creatorImport loads the kept 3D model into the studio, as if it had been dropped on the Model
// tab, so it is fitted and rigged on the hero there.
func (s *studio) creatorImport(r *http.Request) (any, error) {
	_, c, err := s.creatorRequest(r)
	if err != nil {
		return nil, err
	}
	file := filepath.Join(c.dir, "assets", "model.glb")
	data, err := os.ReadFile(file)
	if err != nil {
		return nil, errors.New("keep a 3D model first")
	}
	mesh, err := source2.LoadMesh("model.glb", data, func(name string) ([]byte, error) {
		return nil, fmt.Errorf("%s is not inside the .glb", name)
	})
	if err != nil {
		return nil, err
	}
	name := strings.TrimSpace(c.state.Answers.Name)
	if name == "" {
		name = "model"
	}
	name = strings.ToLower(strings.Join(words(name), "_")) + ".glb"
	s.mu.Lock()
	s.files, s.fileName, s.mesh = map[string][]byte{name: data}, name, mesh
	s.rigKey, s.rigged = "", nil
	s.mu.Unlock()
	return modelPayload(name, mesh), nil
}

// creatorInstall installs the kept hero select art, cards and icons, with the name, tags and
// colours from the answers.
func (s *studio) creatorInstall(r *http.Request) (any, error) {
	req, c, err := s.creatorRequest(r)
	if err != nil {
		return nil, err
	}
	if gameRunning() {
		return nil, errRunning
	}
	store, err := openStore(s.game)
	if err != nil {
		return nil, err
	}
	c.mu.Lock()
	w, dir, hero, colors := c.work, c.dir, c.state.Hero, append([]string{}, c.state.Answers.Colors...)
	c.mu.Unlock()
	p := &webPrompter{}
	done, err := installPictures(p, store, s.game, w, dir)
	if err != nil {
		return nil, err
	}
	// The ability models kept so far, each in place of the game's.
	for _, st := range creatorSteps(w) {
		file := filepath.Join(dir, filepath.FromSlash(st.Output))
		if !strings.HasPrefix(st.ID, "prop_") || !fileExists(file) {
			continue
		}
		if err := propCommand(p, store, s.game, []string{st.Target, file}, options{}); err != nil {
			return nil, err
		}
		done++
	}
	// The hero's ability effects in the character's colours.
	var palette []string
	for _, c := range colors {
		if _, ok := parseHex(c); ok {
			palette = append(palette, c)
		}
	}
	if req.Effects && len(palette) > 0 {
		// The rest is installed already; effects that cannot be recoloured do not undo it.
		if err := paintCommand(p, store, s.game, []string{hero, strings.Join(palette[:min(3, len(palette))], ",")}, options{}); err != nil {
			p.say("The effects were not recoloured: %v", err)
		} else {
			done++
		}
	}
	if err := ensurePatched(p, s.game); err != nil {
		return nil, err
	}
	p.say("%d pieces installed. Start the game and pick %s.", done, heroDisplayName(c.state.Hero))
	return map[string]any{"messages": p.messages}, nil
}

// creatorList says whether a Meshy key is set and lists the heroes with a creation started,
// newest first.
func (s *studio) creatorList(*http.Request) (any, error) {
	hasKey := loadMeshyKey(s.game) != ""
	entries, _ := os.ReadDir(filepath.Join(s.game.StateDir(), "characters"))
	type item struct {
		Hero, Name, Character string
		Changed               int64
	}
	var out []item
	for _, e := range entries {
		data, err := os.ReadFile(filepath.Join(s.game.StateDir(), "characters", e.Name(), "creator.json"))
		if err != nil {
			continue
		}
		var st creatorState
		if json.Unmarshal(data, &st) != nil {
			continue
		}
		info, _ := e.Info()
		out = append(out, item{st.Hero, heroDisplayName(st.Hero), st.Answers.Name, info.ModTime().Unix()})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Changed > out[j].Changed })
	return map[string]any{"hasKey": hasKey, "characters": out}, nil
}

// creatorRoutes adds the Create tab's API. dlmod built with -tags nomeshy leaves Meshy out
// (creator_off.go), and the page hides the tab; the creator is meant to become a separate
// download later.
func (s *studio) creatorRoutes(api func(string, func(*http.Request) (any, error))) {
	api("creator/state", s.creatorState)
	api("creator/key", s.creatorKey)
	api("creator/balance", s.creatorBalance)
	api("creator/answers", s.creatorAnswer)
	api("creator/run", s.creatorRun)
	api("creator/batch", s.creatorBatch)
	api("creator/poll", s.creatorPoll)
	api("creator/choose", s.creatorChoose)
	api("creator/file", s.creatorFile)
	api("creator/import", s.creatorImport)
	api("creator/install", s.creatorInstall)
	api("creator/list", s.creatorList)
}
