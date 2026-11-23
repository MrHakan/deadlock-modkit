package main

import (
	"archive/zip"
	"bytes"
	"crypto/rand"
	_ "embed"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"image/png"
	"io"
	"math"
	"net"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/MrHakan/deadlock-modkit/tools/dlmod/source2"
)

// The model studio is a small web page dlmod serves on this computer only: pick a hero, drop a
// model, see it on the hero, check the rig, install. The page draws with three.js; everything
// else (reading the game, fitting, rigging, compiling) happens here.

//go:embed studio.html
var studioPage string

type studio struct {
	mu    sync.Mutex
	game  Game
	index *source2.VPKIndex
	token string

	hero      string
	modelPath string
	raw       []byte
	model     *source2.Model

	files    map[string][]byte // what the user dropped, by lower-case path
	fileName string            // the model file among them
	mesh     *source2.ImportedMesh

	rigKey string
	rigged *source2.Rigged

	sounds    *source2.VPKIndex // the game's sounds, read on the Sounds tab's first use
	allSounds []string
	images    *source2.VPKIndex // the game's interface images, read on the Images tab's first use
	particles *source2.VPKIndex // the game's particle systems, read on the Effects tab's first use

	// The Create tab: the files dlmod meshy reads, and each hero's creation.
	workIndex *source2.VPKIndex
	sessions  map[string]*creatorSession
}

// studioOptions are the page's settings, as ImportOptions takes them.
type studioOptions struct {
	Keep   map[string]bool `json:"keep"`
	Scale  float64         `json:"scale"`
	Yaw    float64         `json:"yaw"`
	ZUp    bool            `json:"zup"`
	Offset [3]float64      `json:"offset"`
	Pose   *source2.Pose   `json:"pose"` // nil to match the model's pose automatically

	// The advanced rig tools' edits.
	JointMap map[string]string     `json:"jointMap"`
	Pins     map[int]string        `json:"pins"`
	Paint    []source2.PaintStroke `json:"paint"`
}

func (o studioOptions) importOptions() source2.ImportOptions {
	keep := map[string]bool{}
	for k, v := range o.Keep {
		keep[k] = v
	}
	return source2.ImportOptions{Keep: keep, Scale: o.Scale, Yaw: o.Yaw, ZUp: o.ZUp, Offset: o.Offset,
		JointMap: o.JointMap, Pins: o.Pins, Paint: o.Paint}
}

// runStudio serves the studio until dlmod is closed. A model file given (dropped on dlmod) is
// loaded before the page opens, with the files next to it that it names.
func runStudio(ui *UI, game Game, preload ...string) error {
	url, serve, loaded, err := startStudio(game, preload...)
	if err != nil {
		return err
	}
	if loaded != "" {
		ui.say("%s", loaded)
	}
	ui.say("The model studio is at %s", url)
	ui.say("It opens in your browser; leave this window open while you use it, and press Ctrl+C to stop.")
	if os.Getenv("DLMOD_NO_BROWSER") != "" { // headless use and tests
		return serve()
	}
	if err := openInPlayer(url); err != nil {
		ui.say("Open the address above in your browser.")
	}
	return serve()
}

// startStudio gets the studio ready on a port of this computer and returns its address and the
// function that serves it (until dlmod quits), so the full-screen view can run it in the
// background. loaded describes a preloaded model.
func startStudio(game Game, preload ...string) (url string, serve func() error, loaded string, err error) {
	index, err := openHeroModels(game)
	if err != nil {
		return "", nil, "", err
	}
	secret := make([]byte, 16)
	if _, err := rand.Read(secret); err != nil {
		return "", nil, "", err
	}
	s := &studio{game: game, index: index, token: hex.EncodeToString(secret)}
	if len(preload) > 0 {
		mesh, err := source2.LoadMeshFile(preload[0])
		if err != nil {
			return "", nil, "", err
		}
		s.mesh, s.fileName = mesh, filepath.Base(preload[0])
		loaded = fmt.Sprintf("Loaded %s: %d triangles.", s.fileName, mesh.TriangleCount())
	}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", nil, "", err
	}
	server := &http.Server{Handler: s.routes(), ReadHeaderTimeout: 10 * time.Second}
	return fmt.Sprintf("http://%s/?t=%s", listener.Addr(), s.token), func() error { return server.Serve(listener) }, loaded, nil
}

func (s *studio) routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" || r.URL.Query().Get("t") != s.token {
			http.Error(w, "Open the address dlmod printed.", http.StatusForbidden)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		io.WriteString(w, strings.ReplaceAll(studioPage, "{{TOKEN}}", s.token))
	})
	api := func(name string, handler func(r *http.Request) (any, error)) {
		mux.HandleFunc("/api/"+name, func(w http.ResponseWriter, r *http.Request) {
			// Only this page may use the studio: other sites the browser has open cannot read the
			// token, and the server listens on this computer alone.
			if r.Header.Get("X-Studio") != s.token && r.URL.Query().Get("t") != s.token {
				http.Error(w, "forbidden", http.StatusForbidden)
				return
			}
			result, err := handler(r)
			if err != nil {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
				return
			}
			if image, ok := result.([]byte); ok {
				w.Header().Set("Content-Type", "image/png")
				w.Write(image)
				return
			}
			if file, ok := result.(download); ok {
				w.Header().Set("Content-Type", file.contentType)
				w.Write(file.data)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(result)
		})
	}
	api("heroes", s.heroes)
	api("hero", s.loadHero)
	api("upload", s.upload)
	api("current", s.current)
	api("texture", s.texture)
	api("rig", s.rig)
	api("bend", s.bend)
	api("install", s.install)
	api("sounds", s.soundList)
	api("sound", s.soundAudio)
	api("sounds/replace", s.soundReplace)
	api("sounds/restore", s.soundRestore)
	api("images", s.imageList)
	api("image", s.imageFile)
	api("images/replace", s.imageReplace)
	api("images/restore", s.imageRestore)
	api("text", s.textList)
	api("text/save", s.textSave)
	api("effects", s.effectList)
	api("effects/apply", s.effectApply)
	api("effects/restore", s.effectRestore)
	s.creatorRoutes(api) // the Create tab, when dlmod is built with Meshy (creator.go)
	return mux
}

func readJSON(r *http.Request, into any) error {
	// Painted weights of a big model take some megabytes.
	if err := json.NewDecoder(io.LimitReader(r.Body, 64<<20)).Decode(into); err != nil {
		return fmt.Errorf("bad request: %v", err)
	}
	return nil
}

func floats(values [][3]float32) string {
	out := make([]byte, 0, len(values)*12)
	for _, v := range values {
		for _, f := range v {
			out = binary.LittleEndian.AppendUint32(out, math.Float32bits(f))
		}
	}
	return base64.StdEncoding.EncodeToString(out)
}

func uvFloats(values [][2]float32) string {
	out := make([]byte, 0, len(values)*8)
	for _, v := range values {
		out = binary.LittleEndian.AppendUint32(out, math.Float32bits(v[0]))
		out = binary.LittleEndian.AppendUint32(out, math.Float32bits(v[1]))
	}
	return base64.StdEncoding.EncodeToString(out)
}

func indexBytes(values []uint32) string {
	out := make([]byte, 0, len(values)*4)
	for _, v := range values {
		out = binary.LittleEndian.AppendUint32(out, v)
	}
	return base64.StdEncoding.EncodeToString(out)
}

func (s *studio) heroes(*http.Request) (any, error) {
	type hero struct {
		ID        string `json:"id"`
		Name      string `json:"name"`
		Available bool   `json:"available"`
	}
	var out []hero
	for _, id := range heroIDs() {
		_, err := heroModelPath(s.index, id)
		out = append(out, hero{id, heroDisplayName(id), err == nil})
	}
	return out, nil
}

func (s *studio) loadHero(r *http.Request) (any, error) {
	var request struct {
		Hero string `json:"hero"`
	}
	if err := readJSON(r, &request); err != nil {
		return nil, err
	}
	modelPath, err := heroModelPath(s.index, request.Hero)
	if err != nil {
		return nil, err
	}
	raw, err := s.index.Read(modelPath + "_c")
	if err != nil {
		return nil, err
	}
	model, err := source2.LoadModel(raw)
	if err != nil {
		return nil, fmt.Errorf("%s's model cannot be read: %v", heroDisplayName(request.Hero), err)
	}
	geometry, err := model.Geometry()
	if err != nil {
		return nil, err
	}
	s.mu.Lock()
	s.hero, s.modelPath, s.raw, s.model = request.Hero, modelPath, raw, model
	s.rigKey, s.rigged = "", nil
	s.mu.Unlock()
	type part struct {
		Name      string `json:"name"`
		Vertices  int    `json:"vertices"`
		Keep      bool   `json:"keep"`
		Positions string `json:"positions"`
		Indices   string `json:"indices"`
	}
	var parts []part
	for _, g := range geometry {
		parts = append(parts, part{g.Name, g.Vertices, g.Keep, floats(g.Positions), indexBytes(g.Indices)})
	}
	var pelvis []float64 // where the page centres models, as Fit does
	if skeleton, err := model.Skeleton(); err == nil {
		if p, ok := skeleton.Position("pelvis"); ok {
			pelvis = []float64{p[0], p[1]}
		}
	}
	return map[string]any{"name": heroDisplayName(request.Hero), "path": modelPath, "parts": parts, "pelvis": pelvis}, nil
}

// upload takes the dropped files: a model with its textures and .mtl or .bin, or a .zip of them.
func (s *studio) upload(r *http.Request) (any, error) {
	if err := r.ParseMultipartForm(64 << 20); err != nil {
		return nil, fmt.Errorf("the upload failed: %v", err)
	}
	files := map[string][]byte{}
	for _, header := range r.MultipartForm.File["file"] {
		f, err := header.Open()
		if err != nil {
			return nil, err
		}
		data, err := io.ReadAll(io.LimitReader(f, 1<<30))
		f.Close()
		if err != nil {
			return nil, err
		}
		name := strings.ToLower(strings.ReplaceAll(header.Filename, "\\", "/"))
		if strings.HasSuffix(name, ".zip") {
			archive, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
			if err != nil {
				return nil, fmt.Errorf("%s: %v", header.Filename, err)
			}
			for _, entry := range archive.File {
				if entry.FileInfo().IsDir() || entry.UncompressedSize64 > 1<<30 {
					continue
				}
				rc, err := entry.Open()
				if err != nil {
					continue
				}
				content, err := io.ReadAll(rc)
				rc.Close()
				if err == nil {
					files[strings.ToLower(strings.ReplaceAll(entry.Name, "\\", "/"))] = content
				}
			}
			continue
		}
		files[name] = data
	}
	var main string
	for _, extension := range []string{".glb", ".gltf", ".obj", ".fbx"} {
		for name := range files {
			if strings.HasSuffix(name, extension) && (main == "" || len(name) < len(main)) {
				main = name
			}
		}
		if main != "" {
			break
		}
	}
	if main == "" {
		return nil, errors.New("drop a .glb, .gltf or .obj file (with its textures), or a .zip holding one")
	}
	open := func(name string) ([]byte, error) {
		name = strings.ToLower(strings.ReplaceAll(name, "\\", "/"))
		if data, ok := files[path.Clean(path.Join(path.Dir(main), name))]; ok {
			return data, nil
		}
		base := path.Base(name)
		for candidate, data := range files { // the files may have lost their folders on the way
			if path.Base(candidate) == base {
				return data, nil
			}
		}
		return nil, fmt.Errorf("%s was not dropped with the model", name)
	}
	mesh, err := source2.LoadMesh(path.Base(main), files[main], open)
	if err != nil {
		return nil, err
	}
	s.mu.Lock()
	s.files, s.fileName, s.mesh = files, path.Base(main), mesh
	s.rigKey, s.rigged = "", nil
	s.mu.Unlock()
	return modelPayload(path.Base(main), mesh), nil
}

// current is the model already loaded, such as one dropped on dlmod, or nothing.
func (s *studio) current(*http.Request) (any, error) {
	s.mu.Lock()
	name, mesh := s.fileName, s.mesh
	s.mu.Unlock()
	if mesh == nil {
		return map[string]any{}, nil
	}
	return modelPayload(name, mesh), nil
}

// modelPayload describes a loaded model to the page: its geometry and its materials.
func modelPayload(name string, mesh *source2.ImportedMesh) map[string]any {
	type part struct {
		Name    string     `json:"name"`
		Indices string     `json:"indices"`
		Color   [4]float32 `json:"color"`
		Texture bool       `json:"texture"`
		Note    string     `json:"note,omitempty"`
	}
	var parts []part
	for _, p := range mesh.Parts {
		entry := part{Name: p.Name, Indices: indexBytes(p.Indices), Color: p.Color}
		if p.Texture != nil {
			if _, err := source2.DecodeImage(p.Texture); err != nil {
				entry.Note = err.Error()
			} else {
				entry.Texture = true
			}
		}
		parts = append(parts, entry)
	}
	return map[string]any{
		"name": name, "vertices": len(mesh.Positions), "triangles": mesh.TriangleCount(),
		"rigged": len(mesh.Joints) == len(mesh.Positions), "positions": floats(mesh.Positions), "uvs": uvFloats(mesh.UVs),
		"parts": parts,
	}
}

// texture sends a part's texture as PNG, for the preview.
func (s *studio) texture(r *http.Request) (any, error) {
	s.mu.Lock()
	mesh := s.mesh
	s.mu.Unlock()
	var index int
	fmt.Sscan(r.URL.Query().Get("part"), &index)
	if mesh == nil || index < 0 || index >= len(mesh.Parts) || mesh.Parts[index].Texture == nil {
		return nil, errors.New("no such texture")
	}
	img, err := source2.DecodeImage(mesh.Parts[index].Texture)
	if err != nil {
		return nil, err
	}
	var out bytes.Buffer
	if err := png.Encode(&out, img); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

// rigged fits and rigs the model for the given options, reusing the last result when they match.
func (s *studio) rigFor(o studioOptions) (*source2.Rigged, error) {
	key, _ := json.Marshal(o)
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.model == nil {
		return nil, errors.New("pick a hero first")
	}
	if s.mesh == nil {
		return nil, errors.New("drop a model first")
	}
	if s.rigged != nil && s.rigKey == string(key) {
		return s.rigged, nil
	}
	opts := o.importOptions()
	fitted, err := s.model.Fit(s.mesh, opts)
	if err != nil {
		return nil, err
	}
	rigged, err := s.model.AutoRig(fitted, opts, o.Pose, 4)
	if err != nil {
		return nil, err
	}
	s.rigKey, s.rigged = string(key), rigged
	return rigged, nil
}

func (s *studio) rig(r *http.Request) (any, error) {
	var o studioOptions
	if err := readJSON(r, &o); err != nil {
		return nil, err
	}
	start := time.Now()
	rigged, err := s.rigFor(o)
	if err != nil {
		return nil, err
	}
	out := map[string]any{"pose": rigged.Pose, "positions": floats(rigged.Mesh.Positions), "seconds": time.Since(start).Seconds()}
	s.mu.Lock()
	model, mesh := s.model, s.mesh
	s.mu.Unlock()
	// For the advanced rig tools: the hero's skeleton and each vertex's bones, to show weights and
	// bend the model in the page; the model's pieces, to pin; a rigged model's joints, to remap.
	skeleton, err := model.Skeleton()
	if err != nil {
		return out, nil
	}
	index := map[string]int{}
	var core []bool
	var world []float64
	for i, name := range skeleton.Names {
		index[name] = i
		core = append(core, source2.CoreBone(name))
		world = append(world, skeleton.World[i][:]...)
	}
	bones := make([]byte, 0, len(rigged.Skin)*8)
	weights := make([]byte, 0, len(rigged.Skin)*16)
	for _, influences := range rigged.Skin {
		for k := 0; k < 4; k++ {
			bone, weight := 0, float32(0)
			if k < len(influences) {
				bone, weight = index[influences[k].Bone], influences[k].Weight
			}
			bones = binary.LittleEndian.AppendUint16(bones, uint16(bone))
			weights = binary.LittleEndian.AppendUint32(weights, math.Float32bits(weight))
		}
	}
	pieces := source2.Pieces(mesh)
	var pieceIDs []uint32
	count := 0
	for _, p := range pieces {
		pieceIDs = append(pieceIDs, uint32(p))
		count = max(count, p+1)
	}
	joints := map[string]int{} // vertices each joint holds most of
	if len(mesh.Joints) == len(mesh.Positions) {
		for _, influences := range mesh.Joints {
			best := -1
			for k, j := range influences {
				if best < 0 || j.Weight > influences[best].Weight {
					best = k
				}
			}
			if best >= 0 && influences[best].Joint < len(mesh.JointNames) {
				joints[mesh.JointNames[influences[best].Joint]]++
			}
		}
	}
	out["skeleton"] = map[string]any{"names": skeleton.Names, "parents": skeleton.Parent, "core": core, "world": world}
	out["bones"] = base64.StdEncoding.EncodeToString(bones)
	out["weights"] = base64.StdEncoding.EncodeToString(weights)
	out["pieces"] = indexBytes(pieceIDs)
	out["pieceCount"] = count
	out["jointMap"] = rigged.JointMap
	out["jointVertices"] = joints
	return out, nil
}

func (s *studio) bend(r *http.Request) (any, error) {
	var pose source2.Pose
	if err := readJSON(r, &pose); err != nil {
		return nil, err
	}
	s.mu.Lock()
	model, rigged := s.model, s.rigged
	s.mu.Unlock()
	if model == nil || rigged == nil {
		return nil, errors.New("rig the model first")
	}
	return map[string]any{"positions": floats(model.Bend(rigged, pose))}, nil
}

// webPrompter collects what installing says, and takes the default answer to any question
// (such as replacing the hero's previous custom model).
type webPrompter struct{ messages []string }

func (w *webPrompter) say(format string, args ...any) {
	w.messages = append(w.messages, fmt.Sprintf(format, args...))
}
func (w *webPrompter) confirm(question string, fallback bool) bool {
	answer := "no"
	if fallback {
		answer = "yes"
	}
	w.say("%s %s", question, answer)
	return fallback
}
func (w *webPrompter) choose(question string, options []string) (int, bool, bool) {
	return 0, false, len(options) > 0
}
func (w *webPrompter) progress(string, int64, int64) {}

func (s *studio) install(r *http.Request) (any, error) {
	var o studioOptions
	if err := readJSON(r, &o); err != nil {
		return nil, err
	}
	if gameRunning() {
		return nil, errRunning
	}
	rigged, err := s.rigFor(o)
	if err != nil {
		return nil, err
	}
	s.mu.Lock()
	hero, modelPath, raw, mesh := s.hero, s.modelPath, s.raw, s.mesh
	s.mu.Unlock()
	pose := rigged.Pose
	result, err := source2.ImportModel(raw, modelPath, mesh, o.importOptions(), &pose)
	if err != nil {
		return nil, err
	}
	// Read the mod list again: dlmod's full-screen view may have changed it since the studio
	// opened, and installing from a stale list would undo that.
	store, err := openStore(s.game)
	if err != nil {
		return nil, err
	}
	p := &webPrompter{}
	p.messages = append(p.messages, result.Notes...)
	if err := installModel(p, store, hero, result); err != nil {
		return nil, err
	}
	if err := ensurePatched(p, s.game); err != nil {
		return nil, err
	}
	var size int
	for _, data := range result.Files {
		size += len(data)
	}
	p.say("%d files, %s. Start the game and pick %s.", len(result.Files), formatSize(int64(size)), heroDisplayName(hero))
	return map[string]any{"messages": p.messages}, nil
}
