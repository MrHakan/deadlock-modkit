package main

import (
	"fmt"
	"math"
	"net/http"
	"sort"
	"strings"

	"github.com/MrHakan/deadlock-modkit/tools/dlmod/source2"
)

// The studio's rig check skins the rigged model through the hero's own clips, the way the game
// plays them (masks and IK included), and measures where it suffers: surface stretched or
// squashed, parts sinking into the ground, parts swinging behind the hero into the camera. It
// then suggests the rig's tuning options (source2.RigTuning) that address what it found.

// checkClip is one clip the rig is checked against, laid over another through a mask or not.
type checkClip struct {
	Clip, Over, Mask string
}

// rigCheckClips picks the clips that show a rig's problems: moving, standing, melee, reloading,
// casting and jumping, each the way the game lays it over the movement.
func rigCheckClips(clips []string, masks []string) []checkClip {
	prefixed := func(prefixes ...string) string {
		for _, p := range prefixes {
			for _, c := range clips {
				if strings.HasPrefix(c, p) {
					return c
				}
			}
		}
		return ""
	}
	either := func(name, fallback string) string {
		if name != "" {
			return name
		}
		return fallback
	}
	run := firstOf(clips, "weapon_run_n", "run_n", "weapon_walk_n")
	idle := firstOf(clips, "weapon_idle", "out_of_combat_idle", "idle")
	var out []checkClip
	add := func(clip, over string) {
		if clip == "" {
			return
		}
		c := checkClip{Clip: clip}
		if mask := suggestedMask(clip, masks); mask != "" && over != "" {
			c.Over, c.Mask = over, mask
		}
		out = append(out, c)
	}
	add(run, "")
	add(idle, "")
	add(either(firstOf(clips, "melee_quick_1", "melee_hit"), prefixed("melee_")), either(run, idle))
	add(prefixed("reload_stand", "reload_"), either(idle, run))
	add(prefixed("ability_"), either(idle, run))
	add(either(firstOf(clips, "jump", "jump_start", "weapon_jump"), prefixed("jump")), "")
	return out
}

// framesOf poses a clip's frames (at most limit of them) for the check, by bone name.
func (h *heroAnimation) framesOf(c checkClip, limit int) ([]map[string][12]float64, error) {
	clip, err := h.clip(c.Clip)
	if err != nil {
		return nil, err
	}
	var base *source2.Clip
	if c.Over != "" {
		if base, err = h.clip(c.Over); err != nil {
			return nil, err
		}
	}
	step := max(1, (clip.Frames+limit-1)/limit)
	var out []map[string][12]float64
	for f := 0; f < clip.Frames; f += step {
		pose, err := clip.Frame(f)
		if err != nil {
			return nil, err
		}
		if base != nil {
			under, err := base.Frame(f % base.Frames)
			if err != nil {
				return nil, err
			}
			pose = h.skeleton.BlendPoses(under, pose, c.Mask)
		}
		out = append(out, h.skeleton.ByName(h.skeleton.PlayPose(pose, true)))
	}
	return out, nil
}

// usualPose is the hero's average pose over its movement and standing clips.
func (h *heroAnimation) usualPose() (map[string][12]float64, error) {
	clips := heroClips(h.index, h.folder)
	var poses [][]source2.BonePose
	for _, name := range []string{firstOf(clips, "weapon_run_n", "run_n"), firstOf(clips, "weapon_idle", "out_of_combat_idle", "idle")} {
		if name == "" {
			continue
		}
		clip, err := h.clip(name)
		if err != nil {
			return nil, err
		}
		for f := 0; f < clip.Frames; f++ {
			pose, err := clip.Frame(f)
			if err != nil {
				return nil, err
			}
			poses = append(poses, pose)
		}
	}
	if len(poses) == 0 {
		return nil, fmt.Errorf("the hero has no movement or standing clip to take a usual pose from")
	}
	return h.skeleton.ByName(h.skeleton.PlayPose(h.skeleton.AveragePose(poses), false)), nil
}

// leanDegrees is how far a pose tips the line from the pelvis to the neck away from the bind pose.
func leanDegrees(skeleton *source2.Skeleton, pose map[string][12]float64) float64 {
	pelvis, okP := skeleton.Position("pelvis")
	neck, okN := skeleton.Position("neck_0")
	p, okp := pose["pelvis"]
	n, okn := pose["neck_0"]
	if !okP || !okN || !okp || !okn {
		return 0
	}
	a := [3]float64{neck[0] - pelvis[0], neck[1] - pelvis[1], neck[2] - pelvis[2]}
	b := [3]float64{n[3] - p[3], n[7] - p[7], n[11] - p[11]}
	la, lb := math.Sqrt(a[0]*a[0]+a[1]*a[1]+a[2]*a[2]), math.Sqrt(b[0]*b[0]+b[1]*b[1]+b[2]*b[2])
	if la == 0 || lb == 0 {
		return 0
	}
	cos := (a[0]*b[0] + a[1]*b[1] + a[2]*b[2]) / (la * lb)
	return math.Acos(math.Max(-1, math.Min(1, cos))) * 180 / math.Pi
}

type rigCheckRow struct {
	checkClip
	source2.RigStrain
	Hero     source2.RigStrain `json:"hero"` // the hero's own body in the same clip
	TopBones []string          `json:"topBones"`
}

type rigSuggestion struct {
	Option string `json:"option"` // steadyBody, noArmIK, usualPose, or "" for advice without an option
	Text   string `json:"text"`
}

func (s *studio) rigCheck(r *http.Request) (any, error) {
	var o studioOptions
	if err := readJSON(r, &o); err != nil {
		return nil, err
	}
	rigged, err := s.rigFor(o)
	if err != nil {
		return nil, err
	}
	s.mu.Lock()
	model, mesh, hero := s.model, s.mesh, s.hero
	s.mu.Unlock()
	skeleton, err := model.Skeleton()
	if err != nil {
		return nil, err
	}
	own, _ := model.OwnRig(o.importOptions()) // the yardstick: the hero's own body strains too
	h, err := s.heroAnimation(hero)
	if err != nil {
		return nil, err
	}
	masks := []string{}
	for id := range h.skeleton.Masks {
		masks = append(masks, id)
	}
	type played struct {
		clip   checkClip
		frames []map[string][12]float64
		hero   source2.RigStrain
	}
	var clips []played
	for _, c := range rigCheckClips(heroClips(h.index, h.folder), masks) {
		frames, err := h.framesOf(c, 24)
		if err != nil || len(frames) == 0 {
			continue
		}
		p := played{clip: c, frames: frames}
		if own != nil {
			p.hero = own.Strain(skeleton, frames)
			p.hero.Bones = nil
		}
		clips = append(clips, p)
	}
	if len(clips) == 0 {
		return nil, fmt.Errorf("none of %s's clips could be played", heroDisplayName(hero))
	}
	// excess is the most a rig strains beyond the hero's own body, over the clips.
	excess := func(rows []rigCheckRow) (stretch, floor, behind float64) {
		for _, row := range rows {
			stretch = math.Max(stretch, row.Stretched-row.Hero.Stretched)
			floor = math.Max(floor, row.BelowFloor-row.Hero.BelowFloor)
			behind = math.Max(behind, row.Behind-row.Hero.Behind)
		}
		return
	}
	measure := func(rig *source2.Rigged) []rigCheckRow {
		var rows []rigCheckRow
		for _, p := range clips {
			row := rigCheckRow{checkClip: p.clip, RigStrain: rig.Strain(skeleton, p.frames), Hero: p.hero}
			type count struct {
				bone string
				n    int
			}
			var counts []count
			for b, n := range row.Bones {
				counts = append(counts, count{b, n})
			}
			sort.Slice(counts, func(a, b int) bool {
				return counts[a].n > counts[b].n || (counts[a].n == counts[b].n && counts[a].bone < counts[b].bone)
			})
			for _, c := range counts[:min(3, len(counts))] {
				row.TopBones = append(row.TopBones, c.bone)
			}
			row.Bones = nil
			rows = append(rows, row)
		}
		return rows
	}
	rows := measure(rigged)
	stretch, floor, behind := excess(rows)

	// Each tuning option the rig does not use yet is tried, and suggested only if it helps.
	var suggestions []rigSuggestion
	if stretch > 0.02 {
		for _, option := range []struct{ key, text string }{
			{"noArmIK", "Arms without IK: the arms follow the shoulders instead of being pulled to the gun"},
			{"steadyBody", "A steady body: the body follows the hips, not the spine's lean and twist"},
		} {
			try := o
			switch {
			case option.key == "noArmIK" && !o.NoArmIK:
				try.NoArmIK = true
			case option.key == "steadyBody" && !o.SteadyBody:
				try.SteadyBody = true
			default:
				continue
			}
			var usual map[string][12]float64
			if o.UsualPose {
				usual, _ = h.usualPose()
			}
			tuned, err := rigWith(model, mesh, try, usual)
			if err != nil {
				continue
			}
			if after, _, _ := excess(measure(tuned)); after < stretch-0.01 {
				suggestions = append(suggestions, rigSuggestion{option.key, fmt.Sprintf("%s. Tried here: the worst stretching beyond %s's own body drops from %.0f%% to %.0f%%.",
					option.text, heroDisplayName(hero), stretch*100, math.Max(after, 0)*100)})
			}
		}
	}
	if usual, err := h.usualPose(); err == nil && !o.UsualPose {
		if lean := leanDegrees(skeleton, usual); lean > 12 {
			suggestions = append(suggestions, rigSuggestion{"usualPose", fmt.Sprintf("%s leans %.0f° from the bind pose while moving or standing. "+
				"If your model was made standing upright, binding its body in that usual pose keeps it upright in play.", heroDisplayName(hero), lean)})
		}
	}
	if floor > 0.005 {
		suggestions = append(suggestions, rigSuggestion{"", fmt.Sprintf("%.1f%% of the surface sinks into the ground: raise the model with Up, or make it smaller.", floor*100)})
	}
	if behind > 0.02 {
		suggestions = append(suggestions, rigSuggestion{"", fmt.Sprintf("%.0f%% more of the surface than %s's swings well behind the hero, where the third-person camera sits: "+
			"move the model forward or make what sticks out backwards smaller.", behind*100, heroDisplayName(hero))})
	}
	verdict := fmt.Sprintf("The rig holds up in these clips about as well as %s's own body.", heroDisplayName(hero))
	if stretch > 0.02 || floor > 0.005 || behind > 0.02 {
		verdict = fmt.Sprintf("The rig strains more than %s's own body in some clips.", heroDisplayName(hero))
	}
	return map[string]any{"rows": rows, "suggestions": suggestions, "verdict": verdict}, nil
}
