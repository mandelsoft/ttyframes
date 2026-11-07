// Package terminal provides a line based emulation of a terminal.
// It keeps all lines written to the terminal (there is no screen size
// and no scrolling) together with the output format of every character.
// It supports the escape sequences used to update trailing lines:
// cursor movements within lines and between lines, erasing lines and
// output format sequences (Select Graphic Rendition). Other escape
// sequences are ignored.
package terminal

import (
	"strconv"
	"unicode/utf8"

	"github.com/mandelsoft/ttycolors/ansi"
)

// Cell is a character on the terminal together with its output format.
type Cell struct {
	Rune   rune
	Format Format
}

// Line is a line of cells.
type Line []Cell

// Text provides the characters of the line.
func (l Line) Text() string {
	r := make([]rune, len(l))
	for i, c := range l {
		r[i] = c.Rune
	}
	return string(r)
}

// Terminal is a line based terminal emulation. It is an io.Writer,
// the written data may be split at arbitrary positions.
type Terminal struct {
	lines    []Line
	row, col int
	format   formatState
	// pending is an incomplete escape sequence or character
	// at the end of the data written last.
	pending []byte
}

// New provides a new empty terminal.
func New() *Terminal {
	return &Terminal{format: newFormatState()}
}

// Write processes output written to the terminal.
func (t *Terminal) Write(p []byte) (int, error) {
	data := append(t.pending, p...)
	t.pending = nil
	for i := 0; i < len(data); {
		switch c := data[i]; {
		case c == '\x1b':
			l, e := ansi.ScanSequenceEffect(data[i:])
			if l == 0 {
				if incompleteSequence(data[i:]) {
					t.pending = append([]byte(nil), data[i:]...)
					return len(p), nil
				}
				// no valid escape sequence, the escape character is ignored
				i++
				continue
			}
			if e.Set&ansi.S_CONTROL == 0 {
				t.format.apply(string(data[i : i+l]))
			} else {
				t.control(data[i : i+l])
			}
			i += l
		case c == '\r':
			t.col = 0
			i++
		case c == '\n':
			t.row++
			t.col = 0
			i++
		case c == '\b':
			t.col = max(0, t.col-1)
			i++
		case c == '\t':
			t.col = (t.col/8 + 1) * 8
			i++
		case c < ' ' || c == 0x7f:
			// other control characters are ignored
			i++
		default:
			if !utf8.FullRune(data[i:]) {
				t.pending = append([]byte(nil), data[i:]...)
				return len(p), nil
			}
			r, l := utf8.DecodeRune(data[i:])
			t.put(r)
			i += l
		}
	}
	return len(p), nil
}

// incompleteSequence checks whether data may be the beginning of
// an escape sequence, which is continued by further data.
func incompleteSequence(data []byte) bool {
	if len(data) < 2 {
		return true
	}
	switch c := data[1]; {
	case c == '[':
		for _, b := range data[2:] {
			if b < ' ' || b > '?' {
				return false
			}
		}
		return true
	case c == ']' || c == 'P' || c == 'X' || c == '^' || c == '_':
		for i, b := range data[2:] {
			if b == '\a' || (b == '\x1b' && i+3 < len(data)) {
				return false
			}
		}
		return true
	case c >= ' ' && c <= '/':
		for _, b := range data[2:] {
			if b < ' ' || b > '/' {
				return false
			}
		}
		return true
	}
	return false
}

// control processes a control sequence (other than an output format sequence).
func (t *Terminal) control(seq []byte) {
	if len(seq) < 3 || seq[1] != '[' {
		return
	}
	params := string(seq[2 : len(seq)-1])
	if params != "" && (params[0] < '0' || params[0] > ';') {
		// private sequences, for example hiding the cursor
		return
	}
	n, err := strconv.Atoi(params)
	if err != nil || n < 1 {
		n = 1
	}
	switch seq[len(seq)-1] {
	case 'A':
		t.row = max(0, t.row-n)
	case 'B':
		t.row += n
	case 'C':
		t.col += n
	case 'D':
		t.col = max(0, t.col-n)
	case 'E':
		t.row += n
		t.col = 0
	case 'F':
		t.row = max(0, t.row-n)
		t.col = 0
	case 'G':
		t.col = n - 1
	case 'K':
		t.eraseLine(params)
	case 'J':
		t.eraseDisplay(params)
	}
}

// eraseLine erases (a part of) the current line.
func (t *Terminal) eraseLine(params string) {
	if t.row >= len(t.lines) {
		return
	}
	l := t.lines[t.row]
	switch params {
	case "", "0":
		if t.col < len(l) {
			t.lines[t.row] = l[:t.col]
		}
	case "1":
		for i := 0; i <= t.col && i < len(l); i++ {
			l[i] = Cell{Rune: ' '}
		}
	case "2":
		t.lines[t.row] = nil
	}
}

// eraseDisplay erases (a part of) the display.
func (t *Terminal) eraseDisplay(params string) {
	switch params {
	case "", "0":
		t.eraseLine("0")
		if t.row+1 < len(t.lines) {
			t.lines = t.lines[:t.row+1]
		}
	case "1":
		for i := 0; i < t.row && i < len(t.lines); i++ {
			t.lines[i] = nil
		}
		t.eraseLine("1")
	case "2", "3":
		t.lines = nil
	}
}

// put writes a character at the cursor position.
func (t *Terminal) put(r rune) {
	for len(t.lines) <= t.row {
		t.lines = append(t.lines, nil)
	}
	l := t.lines[t.row]
	for len(l) < t.col {
		l = append(l, Cell{Rune: ' '})
	}
	c := Cell{Rune: r, Format: t.format.Format()}
	if t.col < len(l) {
		l[t.col] = c
	} else {
		l = append(l, c)
	}
	t.lines[t.row] = l
	t.col++
}

// Lines provides a copy of the lines of the terminal. Trailing spaces
// without output format and trailing empty lines are omitted.
func (t *Terminal) Lines() []Line {
	var r []Line
	for _, l := range t.lines {
		for len(l) > 0 && l[len(l)-1] == (Cell{Rune: ' '}) {
			l = l[:len(l)-1]
		}
		r = append(r, append(Line(nil), l...))
	}
	for len(r) > 0 && len(r[len(r)-1]) == 0 {
		r = r[:len(r)-1]
	}
	return r
}
