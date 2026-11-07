// Package frames splits the output of a program updating trailing lines
// of a terminal into frames and describes the frames by their lines.
//
// A frame is the screen content before a redraw. A redraw starts with a
// cursor movement up (or to a previous line) following other output, for
// example the moves to the first line to update. The lines of a frame are
// described as States: sequences of Segments, which are texts with an
// output format. Timing dependent values, like elapsed times or
// percentages, are masked by configurable Masks.
package frames

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/mandelsoft/ttycolors/ansi"

	"github.com/mandelsoft/ttyframes/terminal"
)

// Segment is a text with an output format.
type Segment struct {
	Text   string
	Format terminal.Format
}

// State describes a line of a frame by its segments. Adjacent
// segments have different output formats.
type State []Segment

// Text provides the text of the line.
func (s State) Text() string {
	var b strings.Builder
	for _, seg := range s {
		b.WriteString(seg.Text)
	}
	return b.String()
}

// Runs provides the output formats of the line as space separated
// list of the character positions where an output format starts,
// followed by a colon and the output format ("-" for the default),
// for example "0:fg=32 5:-".
func (s State) Runs() string {
	var runs []string
	pos := 0
	for _, seg := range s {
		f := string(seg.Format)
		if f == "" {
			f = "-"
		}
		runs = append(runs, fmt.Sprintf("%d:%s", pos, f))
		pos += len([]rune(seg.Text))
	}
	return strings.Join(runs, " ")
}

// Key provides a unique description of the state: the quoted text
// followed by its runs (if not empty).
func (s State) Key() string {
	if len(s) == 0 {
		return strconv.Quote("")
	}
	return strconv.Quote(s.Text()) + " " + s.Runs()
}

// String provides the Key.
func (s State) String() string {
	return s.Key()
}

// Frame is the screen content described by its lines.
type Frame []State

// Mask replaces timing dependent parts of texts.
type Mask struct {
	Pattern     *regexp.Regexp
	Replacement string
}

// ParseMask parses a mask given as regular expression and
// replacement (see regexp.Regexp.ReplaceAllString) separated by =.
func ParseMask(s string) (Mask, error) {
	i := strings.LastIndex(s, "=")
	if i < 0 {
		return Mask{}, fmt.Errorf("mask %q: <regexp>=<replacement> required", s)
	}
	re, err := regexp.Compile(s[:i])
	if err != nil {
		return Mask{}, fmt.Errorf("mask %q: %w", s, err)
	}
	return Mask{re, s[i+1:]}, nil
}

// String provides the mask in the format accepted by ParseMask.
func (m Mask) String() string {
	return m.Pattern.String() + "=" + m.Replacement
}

// Masks is a list of masks applied in order.
type Masks []Mask

// DefaultMasks masks elapsed times in seconds and percentages.
var DefaultMasks = Masks{
	{regexp.MustCompile(`\b\d+s\b`), "Ns"},
	{regexp.MustCompile(`\b\d+%`), "N%"},
}

// Apply applies the masks to a text.
func (m Masks) Apply(text string) string {
	for _, e := range m {
		text = e.Pattern.ReplaceAllString(text, e.Replacement)
	}
	return text
}

// apply applies the masks to the characters of a line given with
// their output formats. A replacement gets the output format of the
// first replaced character, so that masked values of different lengths
// do not shift the positions of following output formats, even if
// an output format changes within a masked value.
func (m Masks) apply(text []rune, formats []terminal.Format) ([]rune, []terminal.Format) {
	for _, e := range m {
		s := string(text)
		matches := e.Pattern.FindAllStringSubmatchIndex(s, -1)
		if len(matches) == 0 {
			continue
		}
		var rt []rune
		var rf []terminal.Format
		last := 0 // byte index in s
		pos := 0  // rune index of last
		for _, match := range matches {
			n := utf8.RuneCountInString(s[last:match[0]])
			rt = append(rt, text[pos:pos+n]...)
			rf = append(rf, formats[pos:pos+n]...)
			pos += n
			f := terminal.Format("")
			if pos < len(formats) {
				f = formats[pos]
			}
			for _, r := range string(e.Pattern.ExpandString(nil, e.Replacement, s, match)) {
				rt = append(rt, r)
				rf = append(rf, f)
			}
			pos += utf8.RuneCountInString(s[match[0]:match[1]])
			last = match[1]
		}
		text = append(rt, text[pos:]...)
		formats = append(rf, formats[pos:]...)
	}
	return text, formats
}

// NewState provides the state for a line of the terminal
// with the masks applied to its text.
func NewState(l terminal.Line, masks Masks) State {
	text := make([]rune, len(l))
	formats := make([]terminal.Format, len(l))
	for i, c := range l {
		text[i], formats[i] = c.Rune, c.Format
	}
	text, formats = masks.apply(text, formats)
	var s State
	start := 0
	for i := 1; i <= len(text); i++ {
		if i == len(text) || formats[i] != formats[start] {
			s = append(s, Segment{string(text[start:i]), formats[start]})
			start = i
		}
	}
	return s
}

// Starts provides the byte positions of all redraws in the output:
// cursor movements up (or to a previous line) following other output
// than cursor movements or erasing.
func Starts(data []byte) []int {
	var starts []int
	moving := false
	for i := 0; i < len(data); {
		l, e := ansi.ScanSequenceEffect(data[i:])
		if l == 0 {
			moving = false
			i++
			continue
		}
		if e.Set&ansi.S_CONTROL != 0 && data[i+1] == '[' {
			switch data[i+l-1] {
			case 'A', 'F':
				if !moving {
					starts = append(starts, i)
				}
				moving = true
			case 'B', 'C', 'D', 'E', 'G', 'H', 'J', 'K':
				moving = true
			default:
				moving = false
			}
		} else {
			moving = false
		}
		i += l
	}
	return starts
}

// Split provides the frames of an output: the screen content before
// every redraw and at the end of the output.
func Split(data []byte, masks Masks) []Frame {
	var frames []Frame
	t := terminal.New()
	last := 0
	for _, s := range append(Starts(data), len(data)) {
		t.Write(data[last:s])
		last = s
		var f Frame
		for _, l := range t.Lines() {
			f = append(f, NewState(l, masks))
		}
		frames = append(frames, f)
	}
	return frames
}
