package terminal

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/mandelsoft/ttycolors/ansi"
)

// Format describes the output format of a cell by the codes set for
// the output format areas, for example "fg=32,rev=7". The areas are
// always listed in the same order, so that identical output formats
// have identical descriptions, independently of the escape sequences
// used to establish them. The empty Format is the terminal default.
type Format string

// areas lists the output format areas with their names and reset
// codes in the order used to describe formats.
var areas = []struct {
	area  ansi.Covered
	name  string
	reset ansi.Param
}{
	{ansi.S_HIGHLIGHT, "hl", ansi.MODE_RESET_BOLD},
	{ansi.S_ITALIC, "it", ansi.MODE_RESET_ITALIC},
	{ansi.S_UNDERLINE, "ul", ansi.MODE_RESET_UNDERLINE},
	{ansi.S_BLINK, "bl", ansi.MODE_RESET_BLINK},
	{ansi.S_REVERSE, "rev", ansi.MODE_RESET_REVERSE},
	{ansi.S_INVIS, "inv", ansi.MODE_RESET_INVIS},
	{ansi.S_STRIKE, "st", ansi.MODE_RESET_STRIKE},
	{ansi.S_OVERLINE, "ol", ansi.MODE_RESET_OVERLINE},
	{ansi.S_FG_COLOR, "fg", ansi.COLOR_FG_DEFAULT},
	{ansi.S_BG_COLOR, "bg", ansi.COLOR_BG_DEFAULT},
	{ansi.S_UL_COLOR, "ulc", ansi.COLOR_UL_DEFAULT},
}

// Codes provides the codes of the output format areas set by the
// format, indexed by the names of the areas (for example "fg").
// It provides nil for the terminal default.
func (f Format) Codes() map[string]string {
	if f == "" {
		return nil
	}
	r := map[string]string{}
	for _, e := range strings.Split(string(f), ",") {
		name, code, _ := strings.Cut(e, "=")
		r[name] = code
	}
	return r
}

// NewFormat provides the Format for the codes of output format areas
// indexed by the names of the areas (see Format.Codes).
func NewFormat(codes map[string]string) (Format, error) {
	f := newFormatState()
	found := 0
	for i, a := range areas {
		if c, ok := codes[a.name]; ok {
			if c == "" || strings.ContainsAny(c, ",=") {
				return "", fmt.Errorf("invalid code %q for output format area %q", c, a.name)
			}
			f[i] = c
			found++
		}
	}
	if found != len(codes) {
		for name := range codes {
			if areaIndex(name) < 0 {
				return "", fmt.Errorf("unknown output format area %q", name)
			}
		}
	}
	return f.Format(), nil
}

// areaIndex provides the index of an area given by its name, or -1.
func areaIndex(name string) int {
	for i, a := range areas {
		if a.name == name {
			return i
		}
	}
	return -1
}

// formatState is the output format state of the terminal:
// the code for every area (in the order of areas), the empty
// string means the terminal default.
type formatState []string

func newFormatState() formatState {
	return make(formatState, len(areas))
}

// apply applies the codes of an output format sequence.
func (f formatState) apply(seq string) {
	for area, code := range ansi.ScanCodes(seq) {
		if area == ansi.S_ALL {
			clear(f)
			continue
		}
		for i, a := range areas {
			if a.area != area {
				continue
			}
			c := string(code.Bytes())
			if c == strconv.Itoa(int(a.reset)) {
				c = ""
			}
			f[i] = c
		}
	}
}

// Format provides the description of the state.
func (f formatState) Format() Format {
	var parts []string
	for i, c := range f {
		if c != "" {
			parts = append(parts, areas[i].name+"="+c)
		}
	}
	return Format(strings.Join(parts, ","))
}
