package compare_test

import (
	"regexp"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/mandelsoft/ttyframes/compare"
)

var _ = Describe("Layouts", func() {
	t1 := time.Date(2026, time.November, 7, 9, 5, 3, 123456789, time.FixedZone("CET", 3600))
	t2 := time.Date(2025, time.January, 23, 17, 45, 59, 0, time.FixedZone("PST", -8*3600))

	mask := func(m *compare.ValueMask, text string) string {
		return m.Apply(text)
	}

	It("masks all standard layouts", func() {
		m, err := compare.NewValueMask()
		Expect(err).To(Succeed())
		for _, l := range compare.StandardLayouts {
			for _, t := range []time.Time{t1, t2} {
				Expect(mask(m, "at "+t.Format(l)+" done")).To(Equal("at N done"), "layout %q: %q", l, t.Format(l))
			}
		}
	})

	It("masks durations", func() {
		m, _ := compare.NewValueMask()
		for _, d := range []time.Duration{time.Hour + 2*time.Minute + 3500*time.Millisecond, 250 * time.Millisecond, 1500 * time.Nanosecond, -3 * time.Second} {
			Expect(mask(m, "took "+d.String()+".")).To(Equal("took N."), d.String())
		}
	})

	It("masks names only as part of dates", func() {
		m, _ := compare.NewValueMask()
		Expect(mask(m, "Monday meeting in Jan at PST")).To(Equal("Monday meeting in Jan at PST"))
		Expect(mask(m, "2s ago, 3msec")).To(Equal("N ago, Nmsec"))
	})

	It("masks additional layouts", func() {
		m, err := compare.NewValueMask("_2 January 2006")
		Expect(err).To(Succeed())
		// the padding space is kept as text, white space is normalized by the comparison
		Expect(mask(m, "on "+t1.Format("_2 January 2006"))).To(Equal("on  N"))
		Expect(mask(m, "on "+t2.Format("_2 January 2006"))).To(Equal("on N"))
	})

	It("rejects layouts without date or time elements", func() {
		_, err := compare.NewValueMask("abc")
		Expect(err).To(MatchError(ContainSubstring("no date or time element")))
	})

	DescribeTable("converts layouts to regular expressions",
		func(layout, expected string) {
			p, err := compare.LayoutPattern(layout)
			Expect(err).To(Succeed())
			Expect(p).To(Equal(expected))
			Expect(regexp.Compile(p)).NotTo(BeNil())
		},
		Entry("numbers", "2006-01-02", `\b(?:\d{4})-(?:\d{2})-(?:\d{2})\b`),
		Entry("fixed fraction", "05.000", `\b(?:\d{2})[.,]\d{3}\b`),
		Entry("optional fraction", "05,999", `\b(?:\d{2})(?:[.,]\d{1,3})?\b`),
		Entry("space padded day at the beginning", "_2 Jan", `\b(?:\d{1,2}) (?:Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Sep|Oct|Nov|Dec)\b`),
		Entry("space padded day", "Jan _2", `\b(?:Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Sep|Oct|Nov|Dec) (?: ?\d{1,2})\b`),
		Entry("names", "Mon Jan", `\b(?:Mon|Tue|Wed|Thu|Fri|Sat|Sun) (?:Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Sep|Oct|Nov|Dec)\b`),
		Entry("zone", "15:04Z07:00", `\b(?:\d{2}):(?:\d{2})(?:Z|[+-]\d{2}:\d{2})\b`),
	)
})
