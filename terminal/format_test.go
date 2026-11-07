package terminal_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/mandelsoft/ttycolors/ansi"

	"github.com/mandelsoft/ttyframes/terminal"
)

var _ = Describe("Format", func() {
	format := func(seqs ...string) terminal.Format {
		t := terminal.New()
		for _, s := range seqs {
			t.Write([]byte(s))
		}
		t.Write([]byte("x"))
		return t.Lines()[0][0].Format
	}

	DescribeTable("describes the output format",
		func(expected terminal.Format, seqs ...string) {
			Expect(format(seqs...)).To(Equal(expected))
		},
		Entry("default", terminal.Format("")),
		Entry("in area order", terminal.Format("hl=1,rev=7,fg=32"), ansi.Mode(ansi.COLOR_FG_GREEN, ansi.MODE_REVERSE, ansi.MODE_BOLD)),
		Entry("independent of the encoding", terminal.Format("hl=1,fg=32"), ansi.Mode(ansi.MODE_BOLD), ansi.Mode(ansi.COLOR_FG_GREEN)),
		Entry("last code of an area", terminal.Format("fg=34"), ansi.Mode(ansi.COLOR_FG_RED, ansi.COLOR_FG_BLUE)),
		Entry("reset of an area", terminal.Format("it=3"), ansi.Mode(ansi.MODE_BOLD, ansi.MODE_ITALIC), ansi.Mode(ansi.MODE_RESET_BOLD)),
		Entry("reset of all modes", terminal.Format("bg=44"), ansi.Mode(ansi.MODE_BOLD), ansi.Mode(ansi.MODE_RESET_ALL, ansi.COLOR_BG_BLUE)),
		Entry("extended colors", terminal.Format("fg=38;5;208,ulc=58:2::1:2:3"), ansi.Mode(ansi.Code_FG_ID(208)), ansi.Mode(ansi.Code_UL_RGB(1, 2, 3))),
		Entry("sub parameters", terminal.Format("ul=4:3"), ansi.Mode(ansi.UNDERLINE_CURLY)),
	)

	It("provides the codes of the areas", func() {
		Expect(terminal.Format("hl=1,ul=4:3,fg=38;5;208").Codes()).To(Equal(map[string]string{"hl": "1", "ul": "4:3", "fg": "38;5;208"}))
		Expect(terminal.Format("").Codes()).To(BeNil())
	})

	It("creates formats from codes in area order", func() {
		f, err := terminal.NewFormat(map[string]string{"fg": "32", "rev": "7"})
		Expect(err).To(Succeed())
		Expect(f).To(Equal(terminal.Format("rev=7,fg=32")))
		f, err = terminal.NewFormat(nil)
		Expect(err).To(Succeed())
		Expect(f).To(Equal(terminal.Format("")))
	})

	It("rejects invalid codes", func() {
		_, err := terminal.NewFormat(map[string]string{"color": "32"})
		Expect(err).To(MatchError(ContainSubstring("unknown output format area")))
		_, err = terminal.NewFormat(map[string]string{"fg": ""})
		Expect(err).To(HaveOccurred())
		_, err = terminal.NewFormat(map[string]string{"fg": "3,2"})
		Expect(err).To(HaveOccurred())
	})
})
