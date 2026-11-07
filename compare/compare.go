// Package compare compares the behaviour of a program with an expected
// behaviour. Because the frames of a program depend on the timing, the
// differences are classified (see Category): differences, which can be
// explained by the timing, are distinguished from differences indicating
// a changed behaviour.
package compare

import (
	"maps"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/mandelsoft/ttyframes/behaviour"
	"github.com/mandelsoft/ttyframes/frames"
)

// Category classifies the result of a comparison.
type Category struct {
	// Name is the name used in reports.
	Name string
	// Failure indicates a difference, which cannot be explained by the timing.
	Failure bool
	// Details indicates a difference, whose differing states should be reported.
	Details bool
	// Description describes the category.
	Description string
}

func (c *Category) String() string {
	return c.Name
}

// The categories, in the order they are checked. The first matching
// category is the result of a comparison.
var (
	Same = &Category{"SAME", false, false,
		"identical states and transitions"}
	Timing = &Category{"TIMING", false, false,
		"identical after normalizing white space, for example columns shifted by values of other lengths"}
	Sampling = &Category{"SAMPLING", false, false,
		"the same line segments, only in other combinations or with other transitions"}
	Values = &Category{"VALUES", false, true,
		"the same line segments up to values (numbers, dates, times and durations), for example random values"}
	FormatChange = &Category{"FORMAT", true, true,
		"identical texts (up to the values of numbers) with other output formats"}
	TextChange = &Category{"TEXT", true, true,
		"missing or additional texts"}
)

// Categories lists all categories in the order they are checked.
var Categories = []*Category{Same, Timing, Sampling, Values, FormatChange, TextChange}

// Change is a pair of states with identical (normalized) text, but
// different output formats.
type Change struct {
	Expected, Actual frames.State
}

// Result is the result of a comparison.
type Result struct {
	Category *Category
	// OnlyExpected and OnlyActual are the states only found in the
	// expected or the actual behaviour.
	OnlyExpected, OnlyActual []frames.State
	// TransitionsOnlyExpected and TransitionsOnlyActual are the numbers of
	// transitions only found in the expected or the actual behaviour.
	TransitionsOnlyExpected, TransitionsOnlyActual int
	// Changes are the states with changed output formats
	// (for FormatChange).
	Changes []Change
}

// Comparer compares behaviours. It recognizes values according
// to a ValueMask.
type Comparer struct {
	mask *ValueMask
}

// New provides a comparer recognizing dates and times formatted with
// the standard layouts and the given additional layouts of package time.
func New(layouts ...string) (*Comparer, error) {
	m, err := NewValueMask(layouts...)
	if err != nil {
		return nil, err
	}
	return &Comparer{m}, nil
}

// defaultComparer recognizes the standard layouts.
var defaultComparer = func() *Comparer {
	c, err := New()
	if err != nil {
		panic(err)
	}
	return c
}()

// Compare compares an actual behaviour with the expected one
// using a comparer for the standard layouts.
func Compare(expected, actual *behaviour.Behaviour) *Result {
	return defaultComparer.Compare(expected, actual)
}

// Compare compares an actual behaviour with the expected one.
func (c *Comparer) Compare(expected, actual *behaviour.Behaviour) *Result {
	r := &Result{
		OnlyExpected:            only(expected.States, actual.States),
		OnlyActual:              only(actual.States, expected.States),
		TransitionsOnlyExpected: len(onlyKeys(expected.Transitions, actual.Transitions)),
		TransitionsOnlyActual:   len(onlyKeys(actual.Transitions, expected.Transitions)),
	}
	switch {
	case len(r.OnlyExpected) == 0 && len(r.OnlyActual) == 0 &&
		r.TransitionsOnlyExpected == 0 && r.TransitionsOnlyActual == 0:
		r.Category = Same
	case equalKeys(normalizedStates(expected), normalizedStates(actual)) &&
		equalKeys(normalizedTransitions(expected), normalizedTransitions(actual)):
		r.Category = Timing
	case equalKeys(segments(expected, normalize), segments(actual, normalize)):
		r.Category = Sampling
	case equalKeys(segments(expected, c.maskValues), segments(actual, c.maskValues)):
		r.Category = Values
	default:
		r.Changes = c.changes(r.OnlyExpected, r.OnlyActual)
		if len(r.Changes) > 0 {
			r.Category = FormatChange
		} else {
			r.Category = TextChange
		}
	}
	return r
}

// only provides the states of a, which are not in b, sorted by their keys.
func only(a, b map[string]frames.State) []frames.State {
	var r []frames.State
	for _, k := range onlyKeys(a, b) {
		r = append(r, a[k])
	}
	return r
}

// onlyKeys provides the sorted keys of a, which are not in b.
func onlyKeys[T any](a, b map[string]T) []string {
	var r []string
	for _, k := range slices.Sorted(maps.Keys(a)) {
		if _, ok := b[k]; !ok {
			r = append(r, k)
		}
	}
	return r
}

func equalKeys(a, b map[string]bool) bool {
	return maps.Equal(a, b)
}

var whitespace = regexp.MustCompile(`\s+`)

// normalize provides a state with normalized white space: sequences of
// white space are replaced by a single space, leading and trailing white
// space of the line is removed.
func normalize(s frames.State) frames.State {
	var r frames.State
	for _, seg := range s {
		text := whitespace.ReplaceAllString(seg.Text, " ")
		if len(r) > 0 {
			last := &r[len(r)-1]
			if strings.HasSuffix(last.Text, " ") {
				text = strings.TrimPrefix(text, " ")
			}
			if last.Format == seg.Format {
				last.Text += text
				continue
			}
		} else {
			text = strings.TrimLeft(text, " ")
		}
		if text != "" {
			r = append(r, frames.Segment{Text: text, Format: seg.Format})
		}
	}
	if len(r) > 0 {
		last := &r[len(r)-1]
		last.Text = strings.TrimRight(last.Text, " ")
		if last.Text == "" {
			r = r[:len(r)-1]
		}
	}
	return r
}

func normalizedStates(b *behaviour.Behaviour) map[string]bool {
	r := map[string]bool{}
	for _, s := range b.States {
		r[normalize(s).Key()] = true
	}
	return r
}

func normalizedTransitions(b *behaviour.Behaviour) map[string]bool {
	r := map[string]bool{}
	for _, t := range b.Transitions {
		f, to := normalize(t.From), normalize(t.To)
		if f.Key() != to.Key() {
			r[f.Key()+" -> "+to.Key()] = true
		}
	}
	return r
}

// maskValues provides a normalized state with all values replaced by N.
func (c *Comparer) maskValues(s frames.State) frames.State {
	var r frames.State
	for _, seg := range normalize(s) {
		r = append(r, frames.Segment{Text: c.mask.Apply(seg.Text), Format: seg.Format})
	}
	return r
}

// segments provides the segments of all states
// prepared by the given function.
func segments(b *behaviour.Behaviour, prepare func(frames.State) frames.State) map[string]bool {
	r := map[string]bool{}
	for _, s := range b.States {
		for _, seg := range prepare(s) {
			r[strconv.Quote(seg.Text)+" "+string(seg.Format)] = true
		}
	}
	return r
}

// changes provides the pairs of states with identical normalized text
// (up to values), but different output formats.
func (c *Comparer) changes(expected, actual []frames.State) []Change {
	byText := map[string][]frames.State{}
	for _, s := range expected {
		t := c.maskValues(s).Text()
		byText[t] = append(byText[t], s)
	}
	var r []Change
	for _, a := range actual {
		for _, e := range byText[c.maskValues(a).Text()] {
			if c.maskValues(e).Runs() != c.maskValues(a).Runs() {
				r = append(r, Change{e, a})
			}
		}
	}
	return r
}
