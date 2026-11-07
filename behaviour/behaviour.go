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
}

// New provides an empty behaviour.
func New() *Behaviour {
	return &Behaviour{
		States:      map[string]frames.State{},
		Transitions: map[string]Transition{},
	}
}

// Analyze provides the behaviour of a single run described by its frames.
// All lines of all frames are states. Transitions are only recorded for
// frames with the same number of lines, they relate the lines at the
// same position.
func Analyze(fs []frames.Frame) *Behaviour {
	b := New()
	b.Frames = []int{len(fs)}
	for i, f := range fs {
		for _, s := range f {
			b.States[s.Key()] = s
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
	return b
}

// Merge adds the behaviour of other runs.
func (b *Behaviour) Merge(o *Behaviour) {
	b.Frames = append(b.Frames, o.Frames...)
	maps.Copy(b.States, o.States)
	maps.Copy(b.Transitions, o.Transitions)
}

// StateKeys provides the sorted keys of the states.
func (b *Behaviour) StateKeys() []string {
	return slices.Sorted(maps.Keys(b.States))
}

// TransitionKeys provides the sorted keys of the transitions.
func (b *Behaviour) TransitionKeys() []string {
	return slices.Sorted(maps.Keys(b.Transitions))
}
