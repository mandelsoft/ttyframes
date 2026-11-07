package terminal_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/mandelsoft/ttycolors/ansi"

	"github.com/mandelsoft/ttyframes/terminal"
)

// texts writes the data and provides the texts of the lines.
func texts(data ...string) []string {
	t := terminal.New()
	for _, d := range data {
		t.Write([]byte(d))
	}
	var r []string
	for _, l := range t.Lines() {
		r = append(r, l.Text())
	}
	return r
}

var _ = Describe("Terminal", func() {
	DescribeTable("processes output",
		func(data string, expected ...string) {
			if len(expected) == 1 && expected[0] == "" {
				expected = nil
			}
			if len(expected) == 0 {
				Expect(texts(data)).To(BeEmpty())
			} else {
				Expect(texts(data)).To(Equal(expected))
			}
		},
		Entry("empty", ""),
		Entry("lines", "a\nb\r\nc", "a", "b", "c"),
		Entry("multi byte characters", "äöü\nß", "äöü", "ß"),
		Entry("carriage return", "abc\rx", "xbc"),
		Entry("backspace", "abc\bx", "abx"),
		Entry("tab", "a\tb", "a       b"),
		Entry("cursor up and erase line", "a\nb\n\x1b[1A\x1b[2K\x1b[1A\x1b[2Kc\nd", "c", "d"),
		Entry("cursor up keeps the column", "a\nbc\x1b[1Ad", "a d", "bc"),
		Entry("erase to end of line", "abcd\r\x1b[2C\x1b[K", "ab"),
		Entry("erase to beginning of line", "abcd\r\x1b[2C\x1b[1K", "   d"),
		Entry("erase display below", "a\nb\nc\x1b[1A\r\x1b[J", "a"),
		Entry("erase display", "a\nb\x1b[2J", ""),
		Entry("cursor down", "a\x1b[2Bb", "a", "", " b"),
		Entry("cursor backward and forward", "abc\x1b[2Dx\x1b[1Cy", "axcy"),
		Entry("column", "abc\x1b[2Gx", "axc"),
		Entry("next and previous line", "a\x1b[Eb\x1b[Fc", "c", "b"),
		Entry("ignores private and other sequences", "a\x1b[?25l\x1b]0;title\a\x1b7b", "ab"),
		Entry("ignores control characters", "a\x07b", "ab"),
		Entry("ignores invalid escape sequences", "a\x1b\x01b", "ab"),
		Entry("omits trailing spaces and lines", "a  \n\n", "a"),
		Entry("keeps formatted spaces", "a"+ansi.Mode(ansi.MODE_REVERSE)+" ", "a "),
	)

	It("handles data split within escape sequences and characters", func() {
		data := []byte("a\x1b[1;31mä\x1b]0;t\x1b\\b\x1b[0m")
		t := terminal.New()
		for _, b := range data {
			t.Write([]byte{b})
		}
		l := t.Lines()
		Expect(l).To(HaveLen(1))
		Expect(l[0].Text()).To(Equal("aäb"))
		Expect(l[0][1].Format).To(Equal(terminal.Format("hl=1,fg=31")))
	})

	It("keeps the output format of the cells", func() {
		t := terminal.New()
		t.Write([]byte("a" + ansi.Mode(ansi.MODE_BOLD) + "b" + ansi.Mode(ansi.MODE_RESET_BOLD) + "c"))
		Expect(t.Lines()[0]).To(Equal(terminal.Line{
			{'a', ""}, {'b', "hl=1"}, {'c', ""},
		}))
	})

	It("provides copies of the lines", func() {
		t := terminal.New()
		t.Write([]byte("ab"))
		l := t.Lines()
		l[0][0].Rune = 'x'
		Expect(t.Lines()[0].Text()).To(Equal("ab"))
	})
})
