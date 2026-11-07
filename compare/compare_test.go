package compare_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/mandelsoft/ttyframes/behaviour"
	"github.com/mandelsoft/ttyframes/compare"
	"github.com/mandelsoft/ttyframes/frames"
	"github.com/mandelsoft/ttyframes/terminal"
)

// seg provides a segment.
func seg(text string, format string) frames.Segment {
	return frames.Segment{Text: text, Format: terminal.Format(format)}
}

// run provides the behaviour of a run given by frames of single lines.
func run(lines ...frames.State) *behaviour.Behaviour {
	var fs []frames.Frame
	for _, l := range lines {
		fs = append(fs, frames.Frame{l})
	}
	return behaviour.Analyze(fs)
}

var _ = Describe("Compare", func() {
	a := frames.State{seg("load ", ""), seg("x", "rev=7"), seg(" 5s", "")}
	b := frames.State{seg("load ", ""), seg("y", "rev=7"), seg(" 5s", "")}
	c := frames.State{seg("load ", ""), seg("z", "rev=7"), seg(" 5s", "")}

	It("detects identical behaviour", func() {
		r := compare.Compare(run(a, b, c), run(a, b, c))
		Expect(r.Category).To(BeIdenticalTo(compare.Same))
		Expect(r.Category.Failure).To(BeFalse())
	})

	It("detects shifted columns", func() {
		shifted := func(s frames.State) frames.State {
			return frames.State{s[0], s[1], seg("  5s", "")}
		}
		r := compare.Compare(run(a, b, c), run(shifted(a), shifted(b), shifted(c)))
		Expect(r.Category).To(BeIdenticalTo(compare.Timing))
		Expect(r.OnlyActual).To(HaveLen(3))
	})

	It("detects other combinations of segments", func() {
		// the same segments, but another sequence of states
		r := compare.Compare(run(a, b, c), run(a, c, b))
		Expect(r.Category).To(BeIdenticalTo(compare.Sampling))
		Expect(r.TransitionsOnlyExpected).To(Equal(2))

		// a missing state with segments found in other states
		x := frames.State{seg("load ", ""), seg("x", "rev=7"), seg(" 6s", "")}
		y := frames.State{seg("load ", ""), seg("y", "rev=7"), seg(" 6s", "")}
		r = compare.Compare(run(a, y), run(x, b))
		Expect(r.Category).To(BeIdenticalTo(compare.Sampling))
	})

	It("detects other values of numbers", func() {
		step := func(n string, f string) frames.State {
			return frames.State{seg("doing step "+n+" of task 1 [", ""), seg(n+"°", f), seg("]", "")}
		}
		r := compare.Compare(run(step("1", "fg=31"), step("2", "fg=31")), run(step("1", "fg=31"), step("12", "fg=31")))
		Expect(r.Category).To(BeIdenticalTo(compare.Values))
		Expect(r.Category.Failure).To(BeFalse())
		Expect(r.Category.Details).To(BeTrue())
		Expect(r.OnlyExpected).To(Equal([]frames.State{step("2", "fg=31")}))
		Expect(r.OnlyActual).To(Equal([]frames.State{step("12", "fg=31")}))
	})

	DescribeTable("detects other values of different kinds",
		func(expected, actual string) {
			line := func(v string) frames.State {
				return frames.State{seg("value ", ""), seg(v, "fg=32"), seg(" done", "")}
			}
			r := compare.Compare(run(line(expected)), run(line(actual)))
			Expect(r.Category).To(BeIdenticalTo(compare.Values))
		},
		Entry("integers", "1", "123"),
		Entry("negative numbers", "-3", "7"),
		Entry("floating point numbers", "1.5", "12"),
		Entry("decimal comma", "1,5", "2"),
		Entry("dates", "2026-11-07", "2025-01-03"),
		Entry("dates with dots", "07.11.2026", "3.1.2025"),
		Entry("dates with slashes", "11/07/2026", "1/3/2025"),
		Entry("times", "12:05", "9:30"),
		Entry("durations with different numbers of parts", "1:05", "12:05:30"),
		Entry("durations with fractions", "00:01:23.456", "1:02"),
		Entry("dates and times", "2026-11-07 12:00", "2025-01-03 9:30"),
		Entry("dates with names", "Sat, 07 Nov 2026 09:05:03 CET", "Thu, 23 Jan 2025 17:45:59 PST"),
		Entry("RFC3339 dates", "2026-11-07T09:05:03.123+01:00", "2025-01-23T17:45:59Z"),
		Entry("go durations", "1h2m3.5s", "250ms"),
		Entry("kitchen times", "9:05AM", "5:45PM"),
	)

	It("detects other values of additional layouts", func() {
		line := func(v string) frames.State {
			return frames.State{seg("on "+v, "")}
		}
		r := compare.Compare(run(line("7 November 2026")), run(line("23 January 2025")))
		Expect(r.Category).To(BeIdenticalTo(compare.TextChange))
		c, err := compare.New("_2 January 2006")
		Expect(err).To(Succeed())
		r = c.Compare(run(line("7 November 2026")), run(line("23 January 2025")))
		Expect(r.Category).To(BeIdenticalTo(compare.Values))
	})

	It("does not accept other texts around values", func() {
		line := func(t string) frames.State {
			return frames.State{seg(t, "")}
		}
		r := compare.Compare(run(line("done 1:05")), run(line("failed 1:05")))
		Expect(r.Category).To(BeIdenticalTo(compare.TextChange))
	})

	It("detects changed output formats of texts with other numbers", func() {
		step := func(n string, f string) frames.State {
			return frames.State{seg("step "+n+" ", ""), seg("done", f)}
		}
		r := compare.Compare(run(step("1", "fg=32"), step("2", "fg=32")), run(step("1", "fg=32"), step("3", "fg=31")))
		Expect(r.Category).To(BeIdenticalTo(compare.FormatChange))
		Expect(r.Changes).To(ContainElement(compare.Change{Expected: step("2", "fg=32"), Actual: step("3", "fg=31")}))
	})

	It("detects changed output formats", func() {
		changed := frames.State{seg("load ", ""), seg("x", "rev=7,fg=32"), seg(" 5s", "")}
		r := compare.Compare(run(a, b), run(changed, b))
		Expect(r.Category).To(BeIdenticalTo(compare.FormatChange))
		Expect(r.Category.Failure).To(BeTrue())
		Expect(r.Changes).To(Equal([]compare.Change{{Expected: a, Actual: changed}}))
	})

	It("detects changed texts", func() {
		other := frames.State{seg("loading ", ""), seg("x", "rev=7"), seg(" 5s", "")}
		r := compare.Compare(run(a, b), run(other, b))
		Expect(r.Category).To(BeIdenticalTo(compare.TextChange))
		Expect(r.Category.Failure).To(BeTrue())
		Expect(r.Category.Details).To(BeTrue())
		Expect(r.OnlyExpected).To(Equal([]frames.State{a}))
		Expect(r.OnlyActual).To(Equal([]frames.State{other}))
	})

	It("lists the categories in check order", func() {
		Expect(compare.Categories).To(Equal([]*compare.Category{
			compare.Same, compare.Timing, compare.Sampling, compare.Values, compare.FormatChange, compare.TextChange,
		}))
		Expect(compare.Timing.String()).To(Equal("TIMING"))
	})
})
