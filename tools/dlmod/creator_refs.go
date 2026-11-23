//go:build !nomeshy

package main

import (
	"bytes"
	"errors"
	"fmt"
	"image/png"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/MrHakan/deadlock-modkit/tools/dlmod/source2"
)

// Reference pictures: the user's own pictures of the character (a sketch, concept art, a photo of
// a figure), kept in refs/ beside the creation. The design sheet is drawn from them; with
// RefsForAll, every other picture gets them too, after the picture it is made from.

const (
	maxRefs  = 4    // Meshy takes five references: the picture a step builds on, and these
	refPixel = 2048 // a reference is kept at most this wide and tall, to keep requests small
)

func (c *creatorSession) refsDir() string { return filepath.Join(c.dir, "refs") }

// refs lists the reference pictures, oldest first.
func (c *creatorSession) refs() []string {
	entries, _ := os.ReadDir(c.refsDir())
	var out []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".png") {
			out = append(out, filepath.Join(c.refsDir(), e.Name()))
		}
	}
	sort.Strings(out)
	return out
}

func refNames(files []string) []string {
	out := []string{}
	for _, f := range files {
		out = append(out, filepath.Base(f))
	}
	return out
}

// kind is the Meshy task type a step runs with: the design sheet is drawn from the reference
// pictures when there are any.
func (c *creatorSession) kind(step workspaceStep, model string) string {
	if step.ID == "design_sheet" && len(c.refs()) > 0 {
		return "image-to-image"
	}
	return creatorKind(step, model)
}

// creatorRefsAdd keeps pictures dropped on the page as references: decoded (PNG, JPEG or GIF),
// made at most refPixel across, and saved as PNG.
func (s *studio) creatorRefsAdd(r *http.Request) (any, error) {
	if err := r.ParseMultipartForm(64 << 20); err != nil {
		return nil, fmt.Errorf("the upload failed: %v", err)
	}
	c, err := s.session(r.FormValue("hero"))
	if err != nil {
		return nil, err
	}
	files := r.MultipartForm.File["file"]
	if len(files) == 0 {
		return nil, errors.New("drop one or more pictures")
	}
	if have := len(c.refs()); have+len(files) > maxRefs {
		return nil, fmt.Errorf("keep at most %d reference pictures (%d kept already)", maxRefs, have)
	}
	if err := os.MkdirAll(c.refsDir(), 0o755); err != nil {
		return nil, err
	}
	for i, header := range files {
		f, err := header.Open()
		if err != nil {
			return nil, err
		}
		data, err := io.ReadAll(io.LimitReader(f, 40<<20))
		f.Close()
		if err != nil {
			return nil, err
		}
		img, err := source2.DecodeImage(data)
		if err != nil {
			return nil, fmt.Errorf("%s is not a picture dlmod reads (PNG, JPEG or GIF): %v", header.Filename, err)
		}
		var out bytes.Buffer
		if err := png.Encode(&out, source2.Thumbnail(img, refPixel)); err != nil {
			return nil, err
		}
		name := fmt.Sprintf("ref_%d_%d.png", time.Now().UnixMilli(), i)
		if err := os.WriteFile(filepath.Join(c.refsDir(), name), out.Bytes(), 0o644); err != nil {
			return nil, err
		}
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.view(s), nil
}

// refFile finds a reference picture by name.
func (c *creatorSession) refFile(name string) (string, error) {
	if name == "" || path.Base(name) != name || !localPath(name) || !strings.HasSuffix(name, ".png") {
		return "", errors.New("bad picture name")
	}
	file := filepath.Join(c.refsDir(), name)
	if !fileExists(file) {
		return "", fmt.Errorf("there is no reference picture %s", name)
	}
	return file, nil
}

func (s *studio) creatorRefsRemove(r *http.Request) (any, error) {
	req, c, err := s.creatorRequest(r)
	if err != nil {
		return nil, err
	}
	file, err := c.refFile(req.Ref)
	if err != nil {
		return nil, err
	}
	if err := os.Remove(file); err != nil {
		return nil, err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.view(s), nil
}

// creatorRefsOptions says whether every picture gets the references, or the design sheet only.
func (s *studio) creatorRefsOptions(r *http.Request) (any, error) {
	req, c, err := s.creatorRequest(r)
	if err != nil {
		return nil, err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.state.RefsForAll = req.ForAll
	if err := c.save(); err != nil {
		return nil, err
	}
	return c.view(s), nil
}

// creatorRef sends one reference picture.
func (s *studio) creatorRef(r *http.Request) (any, error) {
	c, err := s.session(r.URL.Query().Get("hero"))
	if err != nil {
		return nil, err
	}
	file, err := c.refFile(r.URL.Query().Get("name"))
	if err != nil {
		return nil, err
	}
	return os.ReadFile(file)
}

func (s *studio) creatorRefRoutes(api func(string, func(*http.Request) (any, error))) {
	api("creator/refs/add", s.creatorRefsAdd)
	api("creator/refs/remove", s.creatorRefsRemove)
	api("creator/refs/options", s.creatorRefsOptions)
	api("creator/ref", s.creatorRef)
}
