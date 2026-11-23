package source2

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"
)

// Heroes animate through AnimGraph2: an animation skeleton (.vnmskel) and clips (.vnmclip) of
// compressed bone poses. dlmod reads them to see how a hero really moves, so a model can be rigged
// to that motion and previewed with it. The layout follows Source 2 Viewer (ValveResourceFormat).

// AnimSkeleton is a .vnmskel: the bones the clips animate, in track order.
type AnimSkeleton struct {
	Bones  []string
	Parent []int
	Rest   []BonePose                    // in the parent's space
	Masks  map[string]map[string]float64 // bone masks by ID: each bone's weight
}

// BonePose is one bone's place in its parent's space.
type BonePose struct {
	Rotation [4]float64 // x, y, z, w
	Position [3]float64
	Scale    float64
}

func (b BonePose) matrix() mat34 { return fromRotation(b.Rotation, b.Position, b.Scale) }

// Clip is a .vnmclip: frames of bone poses, one track per animation skeleton bone.
type Clip struct {
	Skeleton string // the .vnmskel it animates
	Frames   int
	Duration float64 // seconds
	Additive bool    // added over another pose rather than a pose of its own

	tracks  []trackSettings
	data    []uint16
	offsets []int
}

type trackSettings struct {
	start, length                        [3]float64 // translation ranges
	scaleStart, scaleLength              float64
	rotation                             [4]float64 // when it does not change
	staticRot, staticPos, staticScaleOff bool
}

func numberOf(v *Value) float64 {
	if v == nil {
		return 0
	}
	switch v.Kind {
	case Float32, Float64:
		return v.F
	case Int16, Int32, Int64:
		return float64(v.I)
	case Uint16, Uint32, Uint64:
		return float64(v.U)
	}
	return 0
}

func resourceData(raw []byte) (*Value, error) {
	res, err := DecodeResource(raw)
	if err != nil {
		return nil, err
	}
	block := res.Block("DATA")
	if block == nil {
		return nil, errors.New("no DATA block")
	}
	v, _, err := ReadKV3(block)
	return v, err
}

// readBonePose reads a pose stored as 8 numbers: position, scale, then the rotation.
func readBonePose(v *Value) (BonePose, error) {
	if v == nil || len(v.Items) != 8 {
		return BonePose{}, errors.New("a bone pose is not 8 numbers")
	}
	n := func(i int) float64 { return numberOf(v.Items[i]) }
	return BonePose{Position: [3]float64{n(0), n(1), n(2)}, Scale: n(3), Rotation: [4]float64{n(4), n(5), n(6), n(7)}}, nil
}

// LoadAnimSkeleton reads a compiled .vnmskel.
func LoadAnimSkeleton(raw []byte) (*AnimSkeleton, error) {
	v, err := resourceData(raw)
	if err != nil {
		return nil, err
	}
	ids, parents, rest := v.Get("m_boneIDs"), v.Get("m_parentIndices"), v.Get("m_parentSpaceReferencePose")
	if ids == nil || parents == nil || rest == nil || len(parents.Items) != len(ids.Items) || len(rest.Items) != len(ids.Items) {
		return nil, errors.New("not an animation skeleton")
	}
	s := &AnimSkeleton{}
	for i, id := range ids.Items {
		pose, err := readBonePose(rest.Items[i])
		if err != nil {
			return nil, err
		}
		parent := int(numberOf(parents.Items[i]))
		if parent >= i {
			return nil, fmt.Errorf("bone %s comes before its parent", id.S)
		}
		s.Bones = append(s.Bones, id.S)
		s.Parent = append(s.Parent, parent)
		s.Rest = append(s.Rest, pose)
	}
	s.Masks = readMasks(v)
	return s, nil
}

// LoadClip reads a compiled .vnmclip.
func LoadClip(raw []byte) (*Clip, error) {
	v, err := resourceData(raw)
	if err != nil {
		return nil, err
	}
	data, settings, offsets := v.Get("m_compressedPoseData"), v.Get("m_trackCompressionSettings"), v.Get("m_compressedPoseOffsets")
	if data == nil || settings == nil || offsets == nil {
		return nil, errors.New("not an animation clip")
	}
	c := &Clip{Frames: int(numberOf(v.Get("m_nNumFrames"))), Duration: numberOf(v.Get("m_flDuration"))}
	if s := v.Get("m_skeleton"); s != nil {
		c.Skeleton = s.S
	}
	if a := v.Get("m_bIsAdditive"); a != nil {
		c.Additive = a.B
	}
	raw16 := data.Blob
	if data.Kind == Array {
		raw16 = make([]byte, len(data.Items))
		for i, b := range data.Items {
			raw16[i] = byte(numberOf(b))
		}
	}
	c.data = make([]uint16, len(raw16)/2)
	for i := range c.data {
		c.data[i] = binary.LittleEndian.Uint16(raw16[2*i:])
	}
	for _, o := range offsets.Items {
		c.offsets = append(c.offsets, int(numberOf(o)))
	}
	rangeOf := func(r *Value) (float64, float64) {
		return numberOf(r.Get("m_flRangeStart")), numberOf(r.Get("m_flRangeLength"))
	}
	flag := func(s *Value, key string) bool { f := s.Get(key); return f != nil && f.B }
	for _, s := range settings.Items {
		var t trackSettings
		for k, key := range []string{"m_translationRangeX", "m_translationRangeY", "m_translationRangeZ"} {
			t.start[k], t.length[k] = rangeOf(s.Get(key))
		}
		t.scaleStart, t.scaleLength = rangeOf(s.Get("m_scaleRange"))
		if q := s.Get("m_constantRotation"); q != nil && len(q.Items) == 4 {
			for k := 0; k < 4; k++ {
				t.rotation[k] = numberOf(q.Items[k])
			}
		}
		t.staticRot, t.staticPos, t.staticScaleOff = flag(s, "m_bIsRotationStatic"), flag(s, "m_bIsTranslationStatic"), flag(s, "m_bIsScaleStatic")
		c.tracks = append(c.tracks, t)
	}
	if c.Frames <= 0 || len(c.offsets) < c.Frames {
		return nil, errors.New("the clip has no frames")
	}
	return c, nil
}

// Tracks is how many bones the clip animates.
func (c *Clip) Tracks() int { return len(c.tracks) }

// Frame decodes one frame: a pose per track, in the parent's space.
func (c *Clip) Frame(frame int) ([]BonePose, error) {
	if frame < 0 || frame >= c.Frames {
		return nil, fmt.Errorf("frame %d of %d", frame, c.Frames)
	}
	at := c.offsets[frame]
	next := func(n int) ([]uint16, error) {
		if at+n > len(c.data) {
			return nil, errors.New("the clip's data ends early")
		}
		out := c.data[at : at+n]
		at += n
		return out, nil
	}
	out := make([]BonePose, len(c.tracks))
	for i, t := range c.tracks {
		b := BonePose{Rotation: t.rotation, Position: t.start, Scale: t.scaleStart}
		if !t.staticRot {
			d, err := next(3)
			if err != nil {
				return nil, err
			}
			b.Rotation = decodeQuaternion(d)
		}
		if !t.staticPos {
			d, err := next(3)
			if err != nil {
				return nil, err
			}
			for k := 0; k < 3; k++ {
				b.Position[k] = t.start[k] + float64(d[k])/65535*t.length[k]
			}
		}
		if !t.staticScaleOff {
			d, err := next(1)
			if err != nil {
				return nil, err
			}
			b.Scale = t.scaleStart + float64(d[0])/65535*t.scaleLength
		}
		out[i] = b
	}
	return out, nil
}

// decodeQuaternion reads 48 bits: three components of 15 bits in ±1/√2, the largest one left
// out and rebuilt, its place in the top bits of the first two words.
func decodeQuaternion(d []uint16) [4]float64 {
	lo, length := -1/math.Sqrt2, 2/math.Sqrt2
	a := float64(d[0]&0x7fff)/0x7fff*length + lo
	b := float64(d[1]&0x7fff)/0x7fff*length + lo
	c := float64(d[2])/0x7fff*length + lo
	w := math.Sqrt(math.Max(0, 1-a*a-b*b-c*c))
	switch int(d[0]>>14&2) | int(d[1]>>15) {
	case 0:
		return [4]float64{w, a, b, c}
	case 1:
		return [4]float64{a, w, b, c}
	case 2:
		return [4]float64{a, b, w, c}
	}
	return [4]float64{a, b, c, w}
}

// Skinning gives, for each bone of a pose, the matrix that carries a vertex bound to it at rest
// to where the pose puts it (row-major 3x4). The root is left out: the game does not pose it
// from its rest (see rootUnturn).
func (s *AnimSkeleton) Skinning(pose []BonePose) map[string][12]float64 {
	rest := s.ModelSpace(nil)
	now := s.ModelSpace(pose)
	out := map[string][12]float64{}
	for i, name := range s.Bones {
		if s.Parent[i] < 0 {
			continue
		}
		inv, ok := rest[i].inverse()
		if !ok {
			continue
		}
		out[name] = [12]float64(now[i].mul(inv))
	}
	return out
}

// ModelSpace places every bone of a pose (one per skeleton bone, in the parent's space) in the
// model's space.
func (s *AnimSkeleton) ModelSpace(pose []BonePose) []mat34 {
	out := make([]mat34, len(s.Bones))
	for i := range s.Bones {
		local := s.Rest[i]
		if i < len(pose) {
			local = pose[i]
		}
		out[i] = local.matrix()
		if p := s.Parent[i]; p >= 0 {
			out[i] = out[p].mul(out[i])
		}
	}
	return out
}
