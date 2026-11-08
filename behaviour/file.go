package behaviour

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"slices"
	"strings"

	"github.com/mandelsoft/ttyframes/frames"
	"github.com/mandelsoft/ttyframes/terminal"
)

// FormatVersion identifies the file format and its version.
const FormatVersion = "ttyframes/v1"

// File is the JSON representation of a behaviour:
//
//	{
//	  "format": "ttyframes/v1",
//	  "frames": [520,521,519],
//	  "states": [
//	    {"text":"Downloading... Ns","runs":[{"start":0,"format":{"rev":"7"}},{"start":8}],"frames":[12,11,12]}
//	  ],
//	  "transitions": [
//	    {"from":{"text":...,"runs":...},"to":{"text":...,"runs":...}}
//	  ]
//	}
//
// The states and transitions are sorted.
type File struct {
	// Format is the FormatVersion.
	Format string `json:"format"`
	// Frames are the numbers of frames of the runs (informational only).
	Frames []int `json:"frames"`
	// States are the states (lines) of all frames.
	States []State `json:"states"`
	// Transitions are the transitions between the states.
	Transitions []TransitionDesc `json:"transitions"`
}

// State is the JSON representation of a state.
type State struct {
	Text string `json:"text"`
	// Runs are the output formats of the text.
	// They are omitted for an empty text.
	Runs []Run `json:"runs,omitempty"`
	// Frames are the numbers of frames showing the state in each run
	// (in the order of the frames of the behaviour). They are only given
	// for the states of a behaviour, not for the states of transitions,
	// and they are omitted if unknown.
	Frames []int `json:"frames,omitempty"`
}

// Run is the output format starting at a character position of a text.
type Run struct {
	Start int `json:"start"`
	// Format are the codes of the output format areas indexed
	// by their names, for example {"fg":"32"}. It is omitted
	// for the terminal default.
	Format map[string]string `json:"format,omitempty"`
}

// TransitionDesc is the JSON representation of a transition.
type TransitionDesc struct {
	From State `json:"from"`
	To   State `json:"to"`
}

// NewState provides the JSON representation of a state.
func NewState(s frames.State) State {
	r := State{Text: s.Text()}
	pos := 0
	for _, seg := range s {
		r.Runs = append(r.Runs, Run{Start: pos, Format: seg.Format.Codes()})
		pos += len([]rune(seg.Text))
	}
	return r
}

// State provides the state described by the JSON representation.
func (s State) State() (frames.State, error) {
	runes := []rune(s.Text)
	if len(runes) == 0 {
		if len(s.Runs) > 0 {
			return nil, fmt.Errorf("runs for an empty text")
		}
		return nil, nil
	}
	if len(s.Runs) == 0 || s.Runs[0].Start != 0 {
		return nil, fmt.Errorf("runs must start at position 0 (text %q)", s.Text)
	}
	var r frames.State
	for i, run := range s.Runs {
		end := len(runes)
		if i+1 < len(s.Runs) {
			end = s.Runs[i+1].Start
		}
		if end <= run.Start || end > len(runes) {
			return nil, fmt.Errorf("invalid run position %d (text %q)", run.Start, s.Text)
		}
		f, err := terminal.NewFormat(run.Format)
		if err != nil {
			return nil, fmt.Errorf("%w (text %q)", err, s.Text)
		}
		r = append(r, frames.Segment{Text: string(runes[run.Start:end]), Format: f})
	}
	return r, nil
}

// NewFile provides the JSON representation of a behaviour.
func NewFile(b *Behaviour) *File {
	f := &File{
		Format:      FormatVersion,
		Frames:      append([]int{}, b.Frames...),
		States:      []State{},
		Transitions: []TransitionDesc{},
	}
	for _, k := range b.StateKeys() {
		s := NewState(b.States[k])
		s.Frames = append([]int(nil), b.StateFrames[k]...)
		f.States = append(f.States, s)
	}
	for _, k := range b.TransitionKeys() {
		t := b.Transitions[k]
		f.Transitions = append(f.Transitions, TransitionDesc{NewState(t.From), NewState(t.To)})
	}
	return f
}

// Behaviour provides the behaviour described by the JSON representation.
func (f *File) Behaviour() (*Behaviour, error) {
	if f.Format != FormatVersion {
		return nil, unsupportedFormat(f.Format)
	}
	b := New()
	b.Frames = f.Frames
	for i, d := range f.States {
		s, err := d.State()
		if err != nil {
			return nil, fmt.Errorf("state %d: %w", i, err)
		}
		b.States[s.Key()] = s
		if d.Frames != nil {
			if len(d.Frames) != len(f.Frames) {
				return nil, fmt.Errorf("state %d: %d numbers of frames for %d runs", i, len(d.Frames), len(f.Frames))
			}
			if slices.ContainsFunc(d.Frames, func(n int) bool { return n < 0 }) {
				return nil, fmt.Errorf("state %d: negative number of frames", i)
			}
			b.StateFrames[s.Key()] = d.Frames
		}
	}
	for i, d := range f.Transitions {
		from, err := d.From.State()
		if err != nil {
			return nil, fmt.Errorf("transition %d: %w", i, err)
		}
		to, err := d.To.State()
		if err != nil {
			return nil, fmt.Errorf("transition %d: %w", i, err)
		}
		t := Transition{from, to}
		b.Transitions[t.Key()] = t
	}
	return b, nil
}

// Write writes a behaviour in its JSON representation (see File). Every
// state and transition is written in a line of its own, so that changes
// of a behaviour provide readable differences.
func Write(w io.Writer, b *Behaviour) error {
	f := NewFile(b)
	bw := bufio.NewWriter(w)
	fmt.Fprintf(bw, "{\n  \"format\": %s,\n  \"frames\": %s,\n", marshal(f.Format), marshal(f.Frames))
	writeList(bw, "states", f.States, ",")
	writeList(bw, "transitions", f.Transitions, "")
	fmt.Fprintln(bw, "}")
	return bw.Flush()
}

// writeList writes a JSON list attribute with an element per line.
func writeList[T any](w io.Writer, name string, elems []T, sep string) {
	if len(elems) == 0 {
		fmt.Fprintf(w, "  %q: []%s\n", name, sep)
		return
	}
	fmt.Fprintf(w, "  %q: [\n", name)
	for i, e := range elems {
		comma := ","
		if i == len(elems)-1 {
			comma = ""
		}
		fmt.Fprintf(w, "    %s%s\n", marshal(e), comma)
	}
	fmt.Fprintf(w, "  ]%s\n", sep)
}

// marshal provides the compact JSON representation of a value
// without escaping HTML characters.
func marshal(v any) string {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		panic(err)
	}
	return strings.TrimSuffix(buf.String(), "\n")
}

// Read reads a behaviour in its JSON representation. Any valid JSON
// representation is accepted, not only the layout written by Write.
// The format version is checked before the content, so that files of
// other versions are reported as unsupported, instead of failing on
// attributes unknown to this version.
func Read(r io.Reader) (*Behaviour, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("cannot read behaviour file: %w", err)
	}
	var v struct {
		Format string `json:"format"`
	}
	if err := json.Unmarshal(data, &v); err != nil {
		return nil, fmt.Errorf("no behaviour file: %w", err)
	}
	if v.Format != FormatVersion {
		return nil, unsupportedFormat(v.Format)
	}
	var f File
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&f); err != nil {
		return nil, fmt.Errorf("no behaviour file: %w", err)
	}
	return f.Behaviour()
}

// unsupportedFormat provides the error for an unsupported format version.
func unsupportedFormat(format string) error {
	return fmt.Errorf("unsupported format %q (%q expected)", format, FormatVersion)
}
