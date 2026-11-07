package frames_test

import (
	"regexp"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/mandelsoft/ttycolors/ansi"

	"github.com/mandelsoft/ttyframes/frames"
	"github.com/mandelsoft/ttyframes/terminal"
)

// line provides a terminal line for the given output.
func line(s string) terminal.Line {
	t := terminal.New()
	t.Write([]byte(s))
	return t.Lines()[0]
}

var _ = Describe("Frames", func() {
	rev := ansi.Mode(ansi.MODE_REVERSE)
	norev := ansi.Mode(ansi.MODE_RESET_REVERSE)

	Context("states", func() {
		It("describes a line by its segments", func() {
			s := frames.NewState(line("ab"+rev+"cd"+norev+"e"), nil)
			Expect(s).To(Equal(frames.State{{"ab", ""}, {"cd", "rev=7"}, {"e", ""}}))
			Expect(s.Text()).To(Equal("abcde"))
			Expect(s.Runs()).To(Equal("0:- 2:rev=7 4:-"))
			Expect(s.Key()).To(Equal(`"abcde" 0:- 2:rev=7 4:-`))
		})

		It("counts characters", func() {
			Expect(frames.NewState(line("äö"+rev+"ü"), nil).Runs()).To(Equal("0:- 2:rev=7"))
		})

		It("describes an empty line", func() {
			Expect(frames.NewState(nil, nil).Key()).To(Equal(`""`))
		})

		It("masks values without shifting following formats", func() {
			s := frames.NewState(line("time 12s, "+rev+"5%"+norev+" done"), frames.DefaultMasks)
			Expect(s.Key()).To(Equal(`"time Ns, N% done" 0:- 9:rev=7 11:-`))
		})

		It("masks values spanning format changes", func() {
			s := frames.NewState(line(rev+"done 6"+norev+"0%"), frames.DefaultMasks)
			Expect(s.Key()).To(Equal(`"done N%" 0:rev=7`))
		})

		It("supports replacements with groups", func() {
			m, err := frames.ParseMask(`(\d+) KB=N $1`)
			Expect(err).To(Succeed())
			Expect(m.String()).To(Equal(`(\d+) KB=N $1`))
			Expect(frames.NewState(line("12 KB"), frames.Masks{m}).Text()).To(Equal("N 12"))
		})

		It("rejects invalid masks", func() {
			_, err := frames.ParseMask("abc")
			Expect(err).To(HaveOccurred())
			_, err = frames.ParseMask("(=x")
			Expect(err).To(HaveOccurred())
		})

		It("applies masks to texts", func() {
			Expect(frames.Masks{{regexp.MustCompile("a+"), "x"}}.Apply("baab")).To(Equal("bxb"))
		})
	})

	Context("splitting", func() {
		up := "\x1b[1A\x1b[2K"

		It("detects redraws", func() {
			data := "a\nb\n" + up + up + "c\nd\n" + up + up + "e\nf\n"
			Expect(frames.Starts([]byte(data))).To(Equal([]int{4, 24}))
		})

		It("ignores cursor movements after other movements", func() {
			data := "a\n\x1b[2K\x1b[1A\x1b[1Ab"
			Expect(frames.Starts([]byte(data))).To(BeEmpty())
		})

		It("detects movements to previous lines", func() {
			Expect(frames.Starts([]byte("a\n\x1b[1Fb"))).To(Equal([]int{2}))
		})

		It("provides the frames", func() {
			data := "a 1s\nb\n" + up + up + "a 2s\n" + rev + "c" + norev + "\n"
			fs := frames.Split([]byte(data), frames.DefaultMasks)
			Expect(fs).To(HaveLen(2))
			Expect(fs[0]).To(Equal(frames.Frame{{{"a Ns", ""}}, {{"b", ""}}}))
			Expect(fs[1]).To(Equal(frames.Frame{{{"a Ns", ""}}, {{"c", "rev=7"}}}))
		})
	})
})
