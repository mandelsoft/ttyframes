package compare

import (
	"fmt"
	"regexp"
	"slices"
	"strings"
	"time"
)

// StandardLayouts are the layouts of package time recognized
// as values by default.
var StandardLayouts = []string{
	time.Layout, time.ANSIC, time.UnixDate, time.RubyDate,
	time.RFC822, time.RFC822Z, time.RFC850, time.RFC1123, time.RFC1123Z,
	time.RFC3339, time.RFC3339Nano, time.Kitchen,
	time.Stamp, time.StampMilli, time.StampMicro, time.StampNano,
	time.DateTime, time.DateOnly, time.TimeOnly,
}

var (
	months       = "January|February|March|April|May|June|July|August|September|October|November|December"
	shortMonths  = "Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Sep|Oct|Nov|Dec"
	weekdays     = "Monday|Tuesday|Wednesday|Thursday|Friday|Saturday|Sunday"
	shortWeekday = "Mon|Tue|Wed|Thu|Fri|Sat|Sun"
)

// layoutElements are the elements of the layouts of package time
// with the regular expressions for their values. Elements with a
// common prefix are ordered from the longest to the shortest one.
var layoutElements = []struct {
	element string
	pattern string
}{
	{"January", months},
	{"Jan", shortMonths},
	{"Monday", weekdays},
	{"Mon", shortWeekday},
	{"MST", `[A-Z]{3,5}|[+-]\d{2,4}`},
	{"2006", `\d{4}`},
	{"__2", ` {0,2}\d{1,3}`},
	{"_2", ` ?\d{1,2}`},
	{"002", `\d{3}`},
	{"01", `\d{2}`},
	{"02", `\d{2}`},
	{"03", `\d{2}`},
	{"04", `\d{2}`},
	{"05", `\d{2}`},
	{"06", `\d{2}`},
	{"15", `\d{2}`},
	{"Z07:00:00", `Z|[+-]\d{2}:\d{2}:\d{2}`},
	{"Z070000", `Z|[+-]\d{6}`},
	{"Z07:00", `Z|[+-]\d{2}:\d{2}`},
	{"Z0700", `Z|[+-]\d{4}`},
	{"Z07", `Z|[+-]\d{2}`},
	{"-07:00:00", `[+-]\d{2}:\d{2}:\d{2}`},
	{"-070000", `[+-]\d{6}`},
	{"-07:00", `[+-]\d{2}:\d{2}`},
	{"-0700", `[+-]\d{4}`},
	{"-07", `[+-]\d{2}`},
	{"PM", `AM|PM`},
	{"pm", `am|pm`},
	{"1", `\d{1,2}`},
	{"2", `\d{1,2}`},
	{"3", `\d{1,2}`},
	{"4", `\d{1,2}`},
	{"5", `\d{1,2}`},
}

// LayoutPattern provides a regular expression matching the values
// formatted with a layout of package time (see time.Layout). It fails
// for layouts without any date or time element.
func LayoutPattern(layout string) (string, error) {
	var b strings.Builder
	found := false
	for i := 0; i < len(layout); {
		if p, l := fraction(layout[i:]); l > 0 {
			b.WriteString(p)
			i += l
			found = true
			continue
		}
		matched := false
		for _, e := range layoutElements {
			if strings.HasPrefix(layout[i:], e.element) {
				p := e.pattern
				if i == 0 && e.element[0] == '_' {
					// a value at the beginning starts with its
					// digits, padding spaces are kept as text.
					p = strings.TrimLeft(p, " ?{0,2}")
				}
				b.WriteString("(?:" + p + ")")
				i += len(e.element)
				matched, found = true, true
				break
			}
		}
		if !matched {
			b.WriteString(regexp.QuoteMeta(layout[i : i+1]))
			i++
		}
	}
	if !found {
		return "", fmt.Errorf("layout %q contains no date or time element", layout)
	}
	p := b.String()
	if isWordChar(layout[0]) {
		p = `\b` + p
	}
	if isWordChar(layout[len(layout)-1]) {
		p += `\b`
	}
	return p, nil
}

// fraction provides the regular expression for fractional seconds
// (.000 or .999 with a dot or comma) at the beginning of a layout
// and the length of the element in the layout.
func fraction(layout string) (string, int) {
	if len(layout) < 2 || (layout[0] != '.' && layout[0] != ',') || (layout[1] != '0' && layout[1] != '9') {
		return "", 0
	}
	n := 1
	for n+1 < len(layout) && layout[n+1] == layout[1] {
		n++
	}
	if n+1 < len(layout) && layout[n+1] >= '0' && layout[n+1] <= '9' {
		return "", 0
	}
	if layout[1] == '0' {
		return fmt.Sprintf(`[.,]\d{%d}`, n), n + 1
	}
	// trailing zeros (and the separator for zero) are omitted
	return fmt.Sprintf(`(?:[.,]\d{1,%d})?`, n), n + 1
}

func isWordChar(c byte) bool {
	return c == '_' || (c >= '0' && c <= '9') || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}

// durations matches durations in the format of time.Duration,
// for example 1h2m3.5s or 250ms.
var durations = regexp.MustCompile(`-?(?:\d+(?:\.\d+)?(?:ns|us|µs|μs|ms|s|m|h))+\b`)

// values matches values: an optionally signed number followed by further
// numbers separated by '.', ',', ':', '/' or '-'. It covers integers,
// floating point numbers (1.5), dates (2026-11-07, 07.11.2026, 11/07/2026),
// times and durations (12:05, 1:02:03.5). A value is replaced as a whole, so
// that values with a different number of parts (1 and 1.5, 1:05 and 1:02:03)
// are equal after masking.
var values = regexp.MustCompile(`-?\d+(?:[.,:/-]\d+)*`)

// ValueMask replaces values in texts by N: dates and times formatted
// according to layouts of package time, durations in the format of
// time.Duration and other values (see values), in this order. Names of
// months, weekdays and time zones are only replaced as part of a date.
type ValueMask struct {
	dates *regexp.Regexp
}

// NewValueMask provides a value mask for the standard layouts
// and the given additional layouts (see LayoutPattern).
func NewValueMask(layouts ...string) (*ValueMask, error) {
	all := append(slices.Clone(StandardLayouts), layouts...)
	// the longest layouts are preferred at the same position
	slices.SortStableFunc(all, func(a, b string) int { return len(b) - len(a) })
	var patterns []string
	for _, l := range all {
		p, err := LayoutPattern(l)
		if err != nil {
			return nil, err
		}
		patterns = append(patterns, "(?:"+p+")")
	}
	return &ValueMask{regexp.MustCompile(strings.Join(patterns, "|"))}, nil
}

// Apply replaces the values of a text by N. Dates and durations are
// replaced by a digit first, so that the values pattern joins them with
// adjacent parts of values (for example a fraction not covered by the
// layout) to a single value.
func (m *ValueMask) Apply(text string) string {
	text = m.dates.ReplaceAllString(text, "0")
	text = durations.ReplaceAllString(text, "0")
	return values.ReplaceAllString(text, "N")
}
