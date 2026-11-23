//go:build !nomeshy

package main

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// The Create tab keeps several characters per hero: the one being made, and others saved as
// character packs (pack.go) to switch back to, download or share.

type savedPack struct {
	File    string `json:"file"`
	Name    string `json:"name"`
	Created string `json:"created"`
	Size    int64  `json:"size"`
}

func (s *studio) savedPacks(hero string) []savedPack {
	out := []savedPack{}
	entries, _ := os.ReadDir(s.game.savedDir(hero))
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), packExt) {
			continue
		}
		data, err := os.ReadFile(filepath.Join(s.game.savedDir(hero), e.Name()))
		if err != nil {
			continue
		}
		manifest, err := packManifestOf(data)
		if err != nil {
			continue
		}
		out = append(out, savedPack{File: e.Name(), Name: manifest.Name, Created: manifest.Created, Size: int64(len(data))})
	}
	sort.Slice(out, func(a, b int) bool { return out[a].Created > out[b].Created })
	return out
}

// dropSession forgets a hero's creation, to be read again from its folder.
func (s *studio) dropSession(hero string) {
	s.mu.Lock()
	delete(s.sessions, hero)
	s.mu.Unlock()
}

// busy reports whether a creation has generations still running, which a switch would cut off.
func (c *creatorSession) busy() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, g := range c.state.Gens {
		if g.running() {
			return true
		}
	}
	return false
}

func (s *studio) creatorPacks(r *http.Request) (any, error) {
	req, _, err := s.creatorRequest(r)
	if err != nil {
		return nil, err
	}
	hero, err := findHero(req.Hero)
	if err != nil {
		return nil, err
	}
	return map[string]any{"packs": s.savedPacks(hero)}, nil
}

func (s *studio) creatorPackSave(r *http.Request) (any, error) {
	req, c, err := s.creatorRequest(r)
	if err != nil {
		return nil, err
	}
	if !fileExists(filepath.Join(c.dir, "workspace.json")) {
		return nil, errors.New("there is nothing to save yet")
	}
	file, manifest, err := savePack(s.game, c.dir)
	if err != nil {
		return nil, err
	}
	hero, _ := findHero(req.Hero)
	return map[string]any{"packs": s.savedPacks(hero), "saved": filepath.Base(file), "name": manifest.Name}, nil
}

// switchTo makes a pack the hero's character, saving the current one first.
func (s *studio) switchTo(hero string, data []byte) (any, error) {
	s.mu.Lock()
	c := s.sessions[hero] // only a creation already open can be busy
	s.mu.Unlock()
	if c != nil && c.busy() {
		return nil, errors.New("wait for the steps still being made to finish first")
	}
	_, manifest, saved, err := unpackCharacter(s.game, data)
	if err != nil {
		return nil, err
	}
	s.dropSession(manifest.Hero)
	c, err = s.session(manifest.Hero)
	if err != nil {
		return nil, err
	}
	c.mu.Lock()
	view := c.view(s)
	c.mu.Unlock()
	view["packs"], view["opened"], view["savedFirst"] = s.savedPacks(manifest.Hero), manifest.Name, filepath.Base(saved)
	return view, nil
}

func (s *studio) creatorPackOpen(r *http.Request) (any, error) {
	req, _, err := s.creatorRequest(r)
	if err != nil {
		return nil, err
	}
	hero, err := findHero(req.Hero)
	if err != nil {
		return nil, err
	}
	name := filepath.Base(req.Pack)
	if name != req.Pack || !strings.HasSuffix(name, packExt) {
		return nil, errors.New("bad pack name")
	}
	data, err := os.ReadFile(filepath.Join(s.game.savedDir(hero), name))
	if err != nil {
		return nil, err
	}
	if manifest, err := packManifestOf(data); err != nil || manifest.Hero != hero {
		return nil, fmt.Errorf("%s is not a character for %s", name, heroDisplayName(hero))
	}
	return s.switchTo(hero, data)
}

// creatorPackUpload opens a pack file dropped on the page: it becomes its hero's character.
func (s *studio) creatorPackUpload(r *http.Request) (any, error) {
	if err := r.ParseMultipartForm(64 << 20); err != nil {
		return nil, fmt.Errorf("the upload failed: %v", err)
	}
	files := r.MultipartForm.File["file"]
	if len(files) != 1 {
		return nil, errors.New("drop one character pack")
	}
	f, err := files[0].Open()
	if err != nil {
		return nil, err
	}
	data, err := io.ReadAll(io.LimitReader(f, packMaxBytes))
	f.Close()
	if err != nil {
		return nil, err
	}
	manifest, err := packManifestOf(data)
	if err != nil {
		return nil, err
	}
	view, err := s.switchTo(manifest.Hero, data)
	if err != nil {
		return nil, err
	}
	view.(map[string]any)["hero"] = manifest.Hero
	return view, nil
}

// creatorPackDownload sends the hero's current character as a pack.
func (s *studio) creatorPackDownload(r *http.Request) (any, error) {
	c, err := s.session(r.URL.Query().Get("hero"))
	if err != nil {
		return nil, err
	}
	data, _, err := exportPack(c.dir)
	if err != nil {
		return nil, err
	}
	return download{data: data, contentType: "application/zip"}, nil
}

// creatorPackName answers the page with the file name a download should get.
func (s *studio) creatorPackName(r *http.Request) (any, error) {
	c, err := s.session(r.URL.Query().Get("hero"))
	if err != nil {
		return nil, err
	}
	w, err := readWorkspace(c.dir)
	if err != nil {
		return nil, err
	}
	return map[string]any{"file": slugify(packName(w)) + "-" + w.Hero + packExt}, nil
}

func (s *studio) creatorPackRoutes(api func(string, func(*http.Request) (any, error))) {
	api("creator/packs", s.creatorPacks)
	api("creator/pack/save", s.creatorPackSave)
	api("creator/pack/open", s.creatorPackOpen)
	api("creator/pack/upload", s.creatorPackUpload)
	api("creator/pack/download", s.creatorPackDownload)
	api("creator/pack/name", s.creatorPackName)
}
