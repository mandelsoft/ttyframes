// Package behaviour describes the behaviour of a program updating
// trailing lines of a terminal by the set of states (lines) shown
// in its frames and the transitions between them. Because the frames
// depend on the timing, the description of several runs is merged.
// A behaviour is stored as JSON (see File, Write and Read).
package behaviour

import (
	"maps"
	"slices"

	"github.com/mandelsoft/ttyframes/frames"
)

// Transition is the change of a line between two consecutive frames.
type Transition struct {
	From, To frames.State
}

// Key provides a unique description of the transition.
func (t Transition) Key() string {
	return t.From.Key() + " -> " + t.To.Key()
}

// Behaviour describes the behaviour of a program by the states and the
// transitions of its frames, collected from one or more runs.
type Behaviour struct {
	// Frames are the numbers of frames of all runs.
	Frames      []int
	States      map[string]frames.State
	Transitions map[string]Transition
	// StateFrames are the numbers of frames showing a state in each run
	// (in the order of Frames). Missing numbers are unknown (behaviour
	// files written before the numbers were recorded).
	StateFrames map[string][]int
}

// New provides an empty behaviour.
func New() *Behaviour {
	return &Behaviour{
		States:      map[string]frames.State{},
		Transitions: map[string]Transition{},
		StateFrames: map[string][]int{},
	}
}

// Analyze provides the behaviour of a single run described by its frames.
// All lines of all frames are states, the frames showing a state are
// counted. Transitions are only recorded for frames with the same number
// of lines, they relate the lines at the same position.
func Analyze(fs []frames.Frame) *Behaviour {
	b := New()
	b.Frames = []int{len(fs)}
	counts := map[string]int{}
	for i, f := range fs {
		shown := map[string]bool{}
		for _, s := range f {
			k := s.Key()
			b.States[k] = s
			if !shown[k] {
				shown[k] = true
				counts[k]++
			}
		}
		if i == 0 || len(fs[i-1]) != len(f) {
			continue
		}
		for j, s := range f {
			if p := fs[i-1][j]; p.Key() != s.Key() {
				t := Transition{p, s}
				b.Transitions[t.Key()] = t
			}
		}
	}
	for k, n := range counts {
		b.StateFrames[k] = []int{n}
	}
	return b
}

// Merge adds the behaviour of other runs. The numbers of frames of
// the states are appended, a state not shown by a run is shown in 0
// frames. If they are unknown for one of the behaviours, they are
// unknown for the result.
func (b *Behaviour) Merge(o *Behaviour) {
	counts := map[string][]int{}
	for _, k := range slices.Concat(slices.Collect(maps.Keys(b.States)), slices.Collect(maps.Keys(o.States))) {
		bc, bok := b.frameCounts(k)
		oc, ook := o.frameCounts(k)
		if bok && ook {
			counts[k] = slices.Concat(bc, oc)
		}
	}
	b.Frames = append(b.Frames, o.Frames...)
	maps.Copy(b.States, o.States)
	maps.Copy(b.Transitions, o.Transitions)
	b.StateFrames = counts
}

// frameCounts provides the numbers of frames showing a state in each run
// and whether they are known.
func (b *Behaviour) frameCounts(key string) ([]int, bool) {
	if c, ok := b.StateFrames[key]; ok {
		return c, true
	}
	if _, ok := b.States[key]; ok {
		return nil, false
	}
	return make([]int, len(b.Frames)), true
}

// Transient checks whether a state is transient: a snapshot shown by
// chance. It is shown in at most one frame of a run, and not in all runs.
// States with unknown numbers of frames are not transient.
func (b *Behaviour) Transient(key string) bool {
	c, ok := b.StateFrames[key]
	if !ok {
		return false
	}
	return slices.Max(c) <= 1 && slices.Contains(c, 0)
}

// Without provides the behaviour without the given states and without
// the transitions from or to these states.
func (b *Behaviour) Without(keys ...string) *Behaviour {
	removed := map[string]bool{}
	for _, k := range keys {
		removed[k] = true
	}
	r := New()
	r.Frames = b.Frames
	for k, s := range b.States {
		if !removed[k] {
			r.States[k] = s
		}
	}
	for k, c := range b.StateFrames {
		if !removed[k] {
			r.StateFrames[k] = c
		}
	}
	for k, t := range b.Transitions {
		if !removed[t.From.Key()] && !removed[t.To.Key()] {
			r.Transitions[k] = t
		}
	}
	return r
}

// StateKeys provides the sorted keys of the states.
func (b *Behaviour) StateKeys() []string {
	return slices.Sorted(maps.Keys(b.States))
}

// TransitionKeys provides the sorted keys of the transitions.
func (b *Behaviour) TransitionKeys() []string {
	return slices.Sorted(maps.Keys(b.Transitions))
}
