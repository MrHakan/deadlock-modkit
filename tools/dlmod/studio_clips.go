package main

import (
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"net/http"
	"path"
	"sort"
	"strings"

	"github.com/MrHakan/deadlock-modkit/tools/dlmod/source2"
)

// The model studio's pose preview plays the hero's own animation clips on a rigged model: a clip
// on its own, or laid over a movement clip through one of the skeleton's bone masks, with the
// arms and legs pulled to their IK targets as the hero graph does.

// heroSkeleton finds the animation skeleton the hero's model names.
func heroSkeleton(index *source2.VPKIndex, hero string) (string, error) {
	model, err := heroModelPath(index, hero)
	if err != nil {
		return "", err
	}
	raw, err := index.Read(model + "_c")
	if err != nil {
		return "", err
	}
	m, err := source2.LoadModel(raw)
	if err != nil {
		return "", err
	}
	if refs := m.Data.Get("m_vecNmSkeletonRefs"); refs != nil && len(refs.Items) > 0 && refs.Items[0].S != "" {
		return refs.Items[0].S, nil
	}
	return "", fmt.Errorf("%s's model names no animation skeleton", heroDisplayName(hero))
}

// heroClipFolder finds the folder of a hero's clips: of the folders of clips under the
// skeleton's own (above its dmx/ folder, if it has one), the one with the most. Heroes keep them
// in clips/, clip/ or dmx/animation/ag2/.
func heroClipFolder(index *source2.VPKIndex, skeleton string) string {
	base := path.Dir(skeleton) + "/"
	if i := strings.Index(skeleton, "dmx/"); i > 0 {
		base = skeleton[:i]
	}
	counts := map[string]int{}
	for name := range index.Files {
		if strings.HasPrefix(name, base) && strings.HasSuffix(name, ".vnmclip_c") {
			counts[path.Dir(name)+"/"]++
		}
	}
	best := ""
	for folder, n := range counts {
		if n > counts[best] || (n == counts[best] && folder < best) {
			best = folder
		}
	}
	return best
}

// clipName is a clip's name in its folder; derived non-additive copies are left out.
func clipName(file, folder string) (string, bool) {
	name := strings.TrimSuffix(strings.TrimPrefix(file, folder), ".vnmclip_c")
	if strings.Contains(name, "/") || strings.Contains(name, ".vnmclip+") {
		return "", false
	}
	return name, true
}

// heroClips lists the hero's clips, the clips worth knowing first.
func heroClips(index *source2.VPKIndex, folder string) []string {
	var names []string
	have := map[string]bool{}
	for file := range index.Files {
		if strings.HasPrefix(file, folder) && strings.HasSuffix(file, ".vnmclip_c") {
			if name, ok := clipName(file, folder); ok {
				names = append(names, name)
				have[name] = true
			}
		}
	}
	sort.Strings(names)
	var key []string
	for _, k := range keyClipNames {
		if have[k] {
			key = append(key, k)
		}
	}
	rest := names[:0:0]
	for _, n := range names {
		if !containsString(key, n) {
			rest = append(rest, n)
		}
	}
	return append(key, rest...)
}

func containsString(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}

// firstOf picks the first name that is in the list.
func firstOf(list []string, names ...string) string {
	for _, n := range names {
		if containsString(list, n) {
			return n
		}
	}
	return ""
}

// suggestedMask is the mask the hero graph most likely plays a clip through, by its name.
func suggestedMask(clip string, masks []string) string {
	for _, rule := range [][2]string{{"melee", "melee_UpperBody"}, {"parry", "Parry_UpperBody"}, {"reload", "Reload_Idle"},
		{"throw", "Throw_UpperBody"}, {"ability", "Cast_UpperBody"}, {"cast", "Cast_UpperBody"}, {"shoot", "shoot_R"}} {
		if strings.Contains(clip, rule[0]) && containsString(masks, rule[1]) {
			return rule[1]
		}
	}
	return ""
}

type heroAnimation struct {
	skeleton *source2.AnimSkeleton
	folder   string
	index    *source2.VPKIndex
}

func (s *studio) heroAnimation(hero string) (*heroAnimation, error) {
	index, err := s.workspaceIndex()
	if err != nil {
		return nil, err
	}
	skeletonPath, err := heroSkeleton(index, hero)
	if err != nil {
		return nil, err
	}
	raw, err := index.Read(skeletonPath + "_c")
	if err != nil {
		return nil, err
	}
	skeleton, err := source2.LoadAnimSkeleton(raw)
	if err != nil {
		return nil, err
	}
	folder := heroClipFolder(index, skeletonPath)
	if folder == "" {
		return nil, fmt.Errorf("no animation clips were found for %s", heroDisplayName(hero))
	}
	return &heroAnimation{skeleton, folder, index}, nil
}

// clip loads one of the hero's clips; for an additive clip, its non-additive copy.
func (h *heroAnimation) clip(name string) (*source2.Clip, error) {
	if strings.ContainsAny(name, `/\`) || name == "" {
		return nil, errors.New("bad clip name")
	}
	raw, err := h.index.Read(h.folder + name + ".vnmclip_c")
	if err != nil {
		return nil, fmt.Errorf("there is no clip %s", name)
	}
	clip, err := source2.LoadClip(raw)
	if err != nil {
		return nil, err
	}
	if clip.Additive {
		whole, err := h.index.Read(h.folder + name + ".vnmclip+non_additive.vnmclip_c")
		if err != nil {
			return nil, fmt.Errorf("%s is only added over other clips, and has no copy of its own to play", name)
		}
		if clip, err = source2.LoadClip(whole); err != nil {
			return nil, err
		}
	}
	if clip.Tracks() != len(h.skeleton.Bones) {
		return nil, fmt.Errorf("%s animates %d bones, the skeleton has %d", name, clip.Tracks(), len(h.skeleton.Bones))
	}
	return clip, nil
}

func (s *studio) clipList(r *http.Request) (any, error) {
	var request struct {
		Hero string `json:"hero"`
	}
	if err := readJSON(r, &request); err != nil {
		return nil, err
	}
	h, err := s.heroAnimation(request.Hero)
	if err != nil {
		return nil, err
	}
	clips := heroClips(h.index, h.folder)
	masks := []string{}
	for id := range h.skeleton.Masks {
		masks = append(masks, id)
	}
	sort.Strings(masks)
	suggest := map[string]string{}
	for _, c := range clips {
		if m := suggestedMask(c, masks); m != "" {
			suggest[c] = m
		}
	}
	return map[string]any{
		"clips":   clips,
		"masks":   masks,
		"suggest": suggest,
		"play":    firstOf(clips, "weapon_run_n", "run_n", "weapon_walk_n", "weapon_idle", "out_of_combat_idle"),
		"base":    firstOf(clips, "weapon_idle", "out_of_combat_idle", "idle", "weapon_run_n"),
		"folder":  h.folder,
	}, nil
}

// maxClipFrames bounds what one clip sends to the page.
const maxClipFrames = 120

func (s *studio) clipFrames(r *http.Request) (any, error) {
	var request struct {
		Hero string `json:"hero"`
		Clip string `json:"clip"`
		Over string `json:"over"` // a clip to lay this one over, or ""
		Mask string `json:"mask"` // the bone mask to lay it through, or "" for the whole body
		IK   bool   `json:"ik"`
	}
	if err := readJSON(r, &request); err != nil {
		return nil, err
	}
	h, err := s.heroAnimation(request.Hero)
	if err != nil {
		return nil, err
	}
	clip, err := h.clip(request.Clip)
	if err != nil {
		return nil, err
	}
	var base *source2.Clip
	if request.Over != "" && request.Over != request.Clip {
		if base, err = h.clip(request.Over); err != nil {
			return nil, err
		}
	}
	if request.Mask != "" && h.skeleton.Masks[request.Mask] == nil {
		return nil, fmt.Errorf("the skeleton has no mask %s", request.Mask)
	}
	count, step := clip.Frames, 1
	for count > maxClipFrames {
		step++
		count = (clip.Frames + step - 1) / step
	}
	duration := clip.Duration
	if duration <= 0 {
		duration = float64(clip.Frames) / 30
	}
	out := make([]byte, 0, count*len(h.skeleton.Bones)*48)
	for f := 0; f < clip.Frames; f += step {
		pose, err := clip.Frame(f)
		if err != nil {
			return nil, err
		}
		if base != nil {
			// The base runs at its own speed and loops under the clip.
			t := float64(f) / float64(max(clip.Frames-1, 1)) * duration
			baseFrame := 0
			if base.Duration > 0 && base.Frames > 1 {
				baseFrame = int(math.Mod(t/base.Duration*float64(base.Frames-1), float64(base.Frames-1)))
			}
			under, err := base.Frame(baseFrame)
			if err != nil {
				return nil, err
			}
			pose = h.skeleton.BlendPoses(under, pose, request.Mask)
		} else if request.Mask != "" {
			pose = h.skeleton.BlendPoses(h.skeleton.Rest, pose, request.Mask)
		}
		for _, m := range h.skeleton.PlayPose(pose, request.IK) {
			for _, v := range m {
				out = binary.LittleEndian.AppendUint32(out, math.Float32bits(float32(v)))
			}
		}
	}
	return map[string]any{
		"bones":    h.skeleton.Bones,
		"parents":  h.skeleton.Parent,
		"frames":   count,
		"duration": duration,
		"matrices": base64.StdEncoding.EncodeToString(out),
	}, nil
}

func (s *studio) clipRoutes(api func(string, func(*http.Request) (any, error))) {
	api("clips", s.clipList)
	api("clip", s.clipFrames)
}
