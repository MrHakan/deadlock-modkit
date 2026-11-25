//go:build !nomeshy

package main

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"math"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/MrHakan/deadlock-modkit/tools/dlmod/source2"
)

// triangleGLB is the smallest model Meshy could send back: one textureless triangle.
func triangleGLB() []byte {
	var bin bytes.Buffer
	for _, v := range []float32{0, 0, 0, 1, 0, 0, 0, 1, 0} {
		binary.Write(&bin, binary.LittleEndian, math.Float32bits(v))
	}
	binary.Write(&bin, binary.LittleEndian, []uint16{0, 1, 2, 0}) // indices, padded to 4 bytes
	doc := map[string]any{
		"asset": map[string]any{"version": "2.0"}, "scene": 0,
		"scenes": []any{map[string]any{"nodes": []int{0}}}, "nodes": []any{map[string]any{"mesh": 0}},
		"meshes": []any{map[string]any{"primitives": []any{map[string]any{"attributes": map[string]int{"POSITION": 0}, "indices": 1}}}},
		"accessors": []any{
			map[string]any{"bufferView": 0, "componentType": 5126, "count": 3, "type": "VEC3", "min": []float32{0, 0, 0}, "max": []float32{1, 1, 0}},
			map[string]any{"bufferView": 1, "componentType": 5123, "count": 3, "type": "SCALAR"},
		},
		"bufferViews": []any{map[string]any{"buffer": 0, "byteLength": 36}, map[string]any{"buffer": 0, "byteOffset": 36, "byteLength": 6}},
		"buffers":     []any{map[string]any{"byteLength": bin.Len()}},
	}
	js, _ := json.Marshal(doc)
	for len(js)%4 != 0 {
		js = append(js, ' ')
	}
	var out bytes.Buffer
	out.WriteString("glTF")
	binary.Write(&out, binary.LittleEndian, uint32(2))
	binary.Write(&out, binary.LittleEndian, uint32(12+8+len(js)+8+bin.Len()))
	binary.Write(&out, binary.LittleEndian, uint32(len(js)))
	out.WriteString("JSON")
	out.Write(js)
	binary.Write(&out, binary.LittleEndian, uint32(bin.Len()))
	out.WriteString("BIN\x00")
	out.Write(bin.Bytes())
	return out.Bytes()
}

// fakeMeshy answers like Meshy's API: tasks run for one look, then finish.
type fakeMeshy struct {
	mu     sync.Mutex
	url    string
	bodies map[string]map[string]any // task ID -> request body
	kinds  map[string]string         // task ID -> task type
	looks  map[string]int
	auth   []string
}

func (f *fakeMeshy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.auth = append(f.auth, r.Header.Get("Authorization"))
	switch {
	case r.URL.Path == "/picture.png":
		img := image.NewNRGBA(image.Rect(0, 0, 8, 8))
		for i := range img.Pix {
			img.Pix[i] = 200
		}
		img.Set(0, 0, color.NRGBA{0, 255, 0, 255})
		png.Encode(w, img)
	case r.URL.Path == "/model.glb":
		w.Write(triangleGLB())
	case r.URL.Path == "/openapi/v1/balance":
		json.NewEncoder(w).Encode(map[string]int{"balance": 100})
	case r.Method == "POST":
		kind := strings.TrimPrefix(r.URL.Path, "/openapi/v1/")
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		id := "task-" + kind + "-" + string(rune('a'+len(f.bodies)))
		f.bodies[id], f.kinds[id] = body, kind
		json.NewEncoder(w).Encode(map[string]string{"result": id})
	default:
		id := r.URL.Path[strings.LastIndex(r.URL.Path, "/")+1:]
		kind, ok := f.kinds[id]
		if !ok {
			http.Error(w, `{"message":"Task not found"}`, http.StatusNotFound)
			return
		}
		f.looks[id]++
		task := map[string]any{"id": id, "status": "IN_PROGRESS", "progress": 40, "consumed_credits": 9}
		if f.looks[id] > 1 {
			task["status"], task["progress"] = "SUCCEEDED", 100
			switch kind {
			case "text-to-image", "image-to-image":
				urls := []string{f.url + "/picture.png"}
				if f.bodies[id]["generate_multi_view"] == true {
					urls = append(urls, f.url+"/picture.png", f.url+"/picture.png")
				}
				task["image_urls"] = urls
			default:
				task["model_urls"] = map[string]string{"glb": f.url + "/model.glb"}
				task["thumbnail_url"] = f.url + "/picture.png"
			}
		}
		json.NewEncoder(w).Encode(task)
	}
}

func TestCreatorMakesACharacterWithMeshy(t *testing.T) {
	game := newFakeGame(t)
	card := source2.SampleInterfaceImage(28, 38, 3)
	game.write("game/citadel/pak01_dir.vpk", source2.WriteVPK(map[string][]byte{
		heroModels["wraith"] + "_c":                     source2.SampleHero(),
		"models/abilities/wraith_card.vmdl_c":           source2.SampleHero(), // an ability model, to make one for
		"panorama/images/heroes/wraith_card_psd.vtex_c": card,                 // a card, to make a picture for
		tagStyles: (&source2.Resource{Blocks: []source2.Block{{Type: "DATA", Data: append(make([]byte, 6), ".heroTag {}"...)}}}).Encode(),
	}))
	fake := &fakeMeshy{bodies: map[string]map[string]any{}, kinds: map[string]string{}, looks: map[string]int{}}
	meshy := httptest.NewServer(fake)
	defer meshy.Close()
	fake.url = meshy.URL
	oldBase, oldEvery := meshyBase, creatorPollEvery
	meshyBase, creatorPollEvery = meshy.URL, 10*time.Millisecond
	defer func() { meshyBase, creatorPollEvery = oldBase, oldEvery }()
	t.Setenv("MESHY_API_KEY", "")

	s := &studio{game: game.Game, token: "secret"}
	server := httptest.NewServer(s.routes())
	defer server.Close()
	call := func(name string, body any, into any) (int, string) {
		data, _ := json.Marshal(body)
		request, _ := http.NewRequest("POST", server.URL+"/api/"+name, bytes.NewReader(data))
		request.Header.Set("X-Studio", "secret")
		response, err := http.DefaultClient.Do(request)
		if err != nil {
			t.Fatal(err)
		}
		defer response.Body.Close()
		var raw bytes.Buffer
		raw.ReadFrom(response.Body)
		if into != nil {
			json.Unmarshal(raw.Bytes(), into)
		}
		return response.StatusCode, raw.String()
	}
	type view struct {
		Concept string
		HasKey  bool
		Steps   []struct {
			ID     string
			Ready  bool
			Filled string
			Models map[string]int
		}
		Generations []creatorGen
		Chosen      map[string]string
	}

	var v view
	if code, body := call("creator/state", map[string]string{"hero": "wraith"}, &v); code != 200 || v.HasKey || len(v.Steps) == 0 {
		t.Fatalf("state: %d %s", code, body)
	}
	if code, body := call("creator/key", map[string]string{"key": "msy_test"}, nil); code != 200 || !strings.Contains(body, `"balance":100`) {
		t.Fatalf("key: %d %s", code, body)
	}
	if loadMeshyKey(game.Game) != "msy_test" {
		t.Fatal("the key was not kept")
	}
	if code, _ := call("creator/run", map[string]any{"hero": "wraith", "step": "design_sheet", "confirm": true}, nil); code == 200 {
		t.Fatal("a design sheet ran before the character was described")
	}
	answers := creatorAnswers{Name: "Rook", What: "Creature", Body: "humanoid", Style: "Stylized cartoon", Colors: []string{"#112233"}, Item: "a hammer"}
	call("creator/answers", map[string]any{"hero": "wraith", "answers": answers}, &v)
	if want := "Rook, a stylized cartoon creature with two arms and two legs, coloured #112233, holding a hammer"; v.Concept != want {
		t.Fatalf("concept %q", v.Concept)
	}
	if !strings.Contains(v.Steps[0].Filled, "Rook, a stylized cartoon creature") || v.Steps[0].Models["nano-banana-pro"] != 9 {
		t.Fatalf("design sheet step %+v", v.Steps[0])
	}
	if code, body := call("creator/run", map[string]any{"hero": "wraith", "step": "design_sheet"}, nil); code == 200 || !strings.Contains(body, "confirm") {
		t.Fatalf("a paid step ran without confirming: %d %s", code, body)
	}

	var wait func(step string) creatorGen
	run := func(step, model string) creatorGen {
		t.Helper()
		if code, body := call("creator/run", map[string]any{"hero": "wraith", "step": step, "model": model, "confirm": true}, nil); code != 200 {
			t.Fatalf("run %s: %d %s", step, code, body)
		}
		return wait(step)
	}
	wait = func(step string) creatorGen {
		t.Helper()
		deadline := time.Now().Add(10 * time.Second)
		for time.Now().Before(deadline) {
			var p struct {
				Generations []creatorGen
				Chosen      map[string]string
			}
			call("creator/poll", map[string]string{"hero": "wraith"}, &p)
			last := p.Generations[len(p.Generations)-1]
			if last.Step == step && !last.running() {
				if last.Status != "SUCCEEDED" || p.Chosen[step] != last.ID {
					t.Fatalf("%s ended %+v, kept %q", step, last, p.Chosen[step])
				}
				return last
			}
			time.Sleep(20 * time.Millisecond)
		}
		t.Fatalf("%s did not finish", step)
		return creatorGen{}
	}
	dir := filepath.Join(game.Game.StateDir(), "characters", "wraith")
	// Two reference pictures of the character: the design sheet is drawn from them.
	var form bytes.Buffer
	parts := multipart.NewWriter(&form)
	parts.WriteField("hero", "wraith")
	for _, name := range []string{"sketch.png", "figure.png"} {
		f, _ := parts.CreateFormFile("file", name)
		f.Write(pngFile(t, 3000, 1500, color.NRGBA{200, 40, 40, 255}))
	}
	parts.Close()
	request, _ := http.NewRequest("POST", server.URL+"/api/creator/refs/add", &form)
	request.Header.Set("Content-Type", parts.FormDataContentType())
	request.Header.Set("X-Studio", "secret")
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	var withRefs struct{ Refs []string }
	json.NewDecoder(response.Body).Decode(&withRefs)
	response.Body.Close()
	if response.StatusCode != 200 || len(withRefs.Refs) != 2 {
		t.Fatalf("references: %d %+v", response.StatusCode, withRefs)
	}
	if kept, err := os.ReadFile(filepath.Join(dir, "refs", withRefs.Refs[0])); err != nil {
		t.Fatal(err)
	} else if img, err := png.Decode(bytes.NewReader(kept)); err != nil || img.Bounds().Dx() != refPixel {
		t.Fatalf("a reference is kept at most %d across: %v %v", refPixel, img.Bounds(), err)
	}
	sheet := run("design_sheet", "")
	if len(sheet.Files) != 1 || !fileExists(filepath.Join(dir, "assets", "design_sheet.png")) {
		t.Fatalf("design sheet %+v", sheet)
	}
	if body := fake.bodies[sheet.TaskID]; fake.kinds[sheet.TaskID] != "image-to-image" || len(body["reference_image_urls"].([]any)) != 2 ||
		!strings.HasPrefix(body["prompt"].(string), "Draw the character in the reference pictures") || body["aspect_ratio"] != "16:9" {
		t.Fatalf("a design sheet from references: %s %v", fake.kinds[sheet.TaskID], body)
	}
	views := run("turnaround", "nano-banana")
	if len(views.Files) != 3 || !fileExists(filepath.Join(dir, "assets", "views_3.png")) {
		t.Fatalf("views %+v", views)
	}
	body := fake.bodies[views.TaskID]
	refs, _ := body["reference_image_urls"].([]any)
	if body["generate_multi_view"] != true || len(refs) != 1 || !strings.HasPrefix(refs[0].(string), "data:image/png;base64,") || body["ai_model"] != "nano-banana" {
		t.Fatalf("views request %v", body)
	}
	model := run("model", "meshy-7.1")
	body = fake.bodies[model.TaskID]
	if fake.kinds[model.TaskID] != "multi-image-to-3d" || len(body["image_urls"].([]any)) != 3 || body["texture_model"] != "meshy-7" {
		t.Fatalf("model request %s %v", fake.kinds[model.TaskID], body)
	}
	if model.Thumb == "" || !fileExists(filepath.Join(dir, "assets", "model.glb")) {
		t.Fatalf("model %+v", model)
	}
	for _, a := range fake.auth {
		if a != "Bearer msy_test" && a != "" { // downloads of results carry no key
			t.Fatalf("Meshy got the header %q", a)
		}
	}

	var payload struct {
		Name      string
		Triangles int
	}
	if code, body := call("creator/import", map[string]string{"hero": "wraith"}, &payload); code != 200 || payload.Name != "rook.glb" || payload.Triangles != 1 {
		t.Fatalf("import: %d %s", code, body)
	}
	if s.mesh == nil || s.fileName != "rook.glb" {
		t.Fatal("the model is not loaded in the studio")
	}
	// An ability model: a picture of it from the design sheet, then a small 3D model of the picture.
	picture := run("propimg_wraith_card", "")
	body = fake.bodies[picture.TaskID]
	if fake.kinds[picture.TaskID] != "image-to-image" || !strings.Contains(body["prompt"].(string), "Draw only this object") ||
		!fileExists(filepath.Join(dir, "assets", "prop_wraith_card.png")) {
		t.Fatalf("prop picture %s %v", fake.kinds[picture.TaskID], body)
	}
	prop := run("prop_wraith_card", "")
	body = fake.bodies[prop.TaskID]
	if fake.kinds[prop.TaskID] != "image-to-3d" || body["target_polycount"] != float64(10000) || body["ai_model"] != "meshy-t2.5" ||
		!strings.HasPrefix(body["image_url"].(string), "data:image/png;base64,") || !fileExists(filepath.Join(dir, "assets", "prop_wraith_card.glb")) {
		t.Fatalf("prop model %s %v", fake.kinds[prop.TaskID], body)
	}
	// Several pictures at once, after one confirmation of their total price.
	// With "every picture", the card gets the references after the design sheet.
	if code, body := call("creator/refs/options", map[string]any{"hero": "wraith", "forAll": true}, nil); code != 200 {
		t.Fatalf("options: %d %s", code, body)
	}
	batch := map[string]any{"hero": "wraith", "steps": []map[string]string{{"step": "card"}}}
	if code, body := call("creator/batch", batch, nil); code == 200 || !strings.Contains(body, "cost 9 credits together") {
		t.Fatalf("a batch ran without confirming: %d %s", code, body)
	}
	batch["confirm"], batch["total"] = true, 8
	if code, body := call("creator/batch", batch, nil); code == 200 || !strings.Contains(body, "confirm that price") {
		t.Fatalf("a batch ran at a price the page did not show: %d %s", code, body)
	}
	batch["total"] = 9
	var started struct{ Started []string }
	if code, body := call("creator/batch", batch, &started); code != 200 || len(started.Started) != 1 || started.Started[0] != "card" {
		t.Fatalf("batch: %d %s", code, body)
	}
	cardGen := wait("card")
	body = fake.bodies[cardGen.TaskID]
	if body["remove_background"] != true || strings.Contains(body["prompt"].(string), "green") || len(body["reference_image_urls"].([]any)) != 3 ||
		!strings.Contains(body["prompt"].(string), "reference pictures after the first") {
		t.Fatalf("a card has Meshy remove its background: %v", body)
	}
	// An edit changes that picture and takes its place.
	if code, body := call("creator/run", map[string]any{"hero": "wraith", "step": "card", "edit": cardGen.ID, "prompt": "make the cape red", "confirm": true}, nil); code != 200 {
		t.Fatalf("edit: %d %s", code, body)
	}
	edited := wait("card")
	body = fake.bodies[edited.TaskID]
	if edited.EditOf != cardGen.ID || !strings.Contains(body["prompt"].(string), "make the cape red") || fake.kinds[edited.TaskID] != "image-to-image" {
		t.Fatalf("edit %+v %v", edited, body)
	}
	if code, body := call("creator/run", map[string]any{"hero": "wraith", "step": "turnaround", "edit": views.ID, "prompt": "x", "confirm": true}, nil); code == 200 {
		t.Fatalf("a set of views cannot be edited: %s", body)
	}

	if code, body := call("creator/install", map[string]any{"hero": "wraith", "effects": true}, nil); code != 200 {
		t.Fatalf("install: %d %s", code, body)
	}
	if !game.hasMod(t, "Custom prop: wraith_card.vmdl") {
		t.Fatal("the ability model was not installed")
	}

	// The character is saved as a pack beside others for the hero, downloads as one, and opens again.
	var packs struct {
		Packs []savedPack
		Saved string
	}
	if code, body := call("creator/pack/save", map[string]string{"hero": "wraith"}, &packs); code != 200 || len(packs.Packs) != 1 || packs.Saved != "rook.dlmodchar" {
		t.Fatalf("save: %d %s", code, body)
	}
	response, err = http.Get(server.URL + "/api/creator/pack/download?hero=wraith&t=secret")
	if err != nil {
		t.Fatal(err)
	}
	var pack bytes.Buffer
	pack.ReadFrom(response.Body)
	response.Body.Close()
	if manifest, err := packManifestOf(pack.Bytes()); err != nil || manifest.Name != "Rook" || manifest.Hero != "wraith" {
		t.Fatalf("download: %+v %v", manifest, err)
	}
	if z, _, err := openPack(pack.Bytes()); err != nil || !packHas(z, "refs/"+withRefs.Refs[0]) {
		t.Fatal("the pack leaves the reference pictures out")
	}
	var opened struct{ Opened string }
	if code, body := call("creator/pack/open", map[string]string{"hero": "wraith", "pack": "rook.dlmodchar"}, &opened); code != 200 || opened.Opened != "Rook" {
		t.Fatalf("open: %d %s", code, body)
	}

	// The creation is kept on disk, with the workspace dlmod apply reads.
	if w, err := readWorkspace(dir); err != nil || w.Concept != v.Concept || w.Select.Name != "Rook" {
		t.Fatalf("workspace %+v %v", w, err)
	}
	if _, err := os.Stat(filepath.Join(dir, "creator.json")); err != nil {
		t.Fatal(err)
	}
}

func TestCreatorConcept(t *testing.T) {
	for _, c := range []struct {
		a    creatorAnswers
		want string
	}{
		{creatorAnswers{}, "The character, a character"},
		{creatorAnswers{Name: "Ola", What: "Animal", Body: "quadruped", Details: "Wears a red scarf."}, "Ola, an animal that walks on four legs. Wears a red scarf"},
		{creatorAnswers{Name: "Blob", Style: "Claymation", What: "Alien", Body: "round", Colors: []string{"#FFFFFF", "#0000FF"}}, "Blob, a claymation alien with a round body and no legs, coloured #FFFFFF and #0000FF"},
	} {
		if got := c.a.concept(); got != c.want {
			t.Errorf("%+v: %q, want %q", c.a, got, c.want)
		}
	}
}

func TestMeshyPrice(t *testing.T) {
	for _, c := range []struct {
		kind, model string
		want        int
	}{
		{"text-to-image", "nano-banana-pro", 9}, {"image-to-image", "nano-banana", 3}, {"image-to-image", "gpt-image-2", 12},
		{"multi-image-to-3d", "meshy-7.1", 30}, {"image-to-3d", "meshy-t2.5", 15}, {"rigging", "meshyrig", 5},
	} {
		if got := meshyPrice(c.kind, c.model); got != c.want {
			t.Errorf("%s %s: %d, want %d", c.kind, c.model, got, c.want)
		}
	}
}
