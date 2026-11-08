package main

import (
	"bytes"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/mandelsoft/ttyframes/behaviour"
	"github.com/mandelsoft/ttyframes/compare"
	"github.com/mandelsoft/ttyframes/frames"
	"github.com/mandelsoft/ttyframes/terminal"
)

// program is an animated program redrawing two lines.
const program = `package main

import (
	"fmt"
	"os"
	"time"
)

func main() {
	format := "\x1b[32m"
	if len(os.Getenv("CHANGED")) > 0 {
		format = "\x1b[31m"
	}
	for i := 0; i <= 4; i++ {
		if i > 0 {
			fmt.Print("\x1b[1A\x1b[2K\x1b[1A\x1b[2K")
		}
		fmt.Printf("step %d\n%s%d%%\x1b[0m done\n", i, format, i*25)
		time.Sleep(10 * time.Millisecond)
	}
}
`

var _ = Describe("Check", func() {
	Context("report", func() {
		st := func(text, format string) frames.State {
			return frames.State{{Text: text, Format: terminal.Format(format)}}
		}
		expected := behaviour.Analyze([]frames.Frame{{st("a", "")}, {st("b", "")}})

		It("reports failures with details", func() {
			actual := behaviour.Analyze([]frames.Frame{{st("a", "hl=1")}, {st("b", "")}})
			var buf bytes.Buffer
			counts := map[string]int{}
			cfg := &config{file: DefaultFile, details: 10}
			Expect(report(&buf, cfg, "ex", checked{expected: expected, actual: actual, result: compare.Compare(expected, actual)}, counts)).To(BeTrue())
			Expect(buf.String()).To(Equal(`FORMAT    ex: frames 2/2, states 2/2 (only expected 1, only actual 1), transitions 1/1 (only expected 1, only actual 1)
          format of "a"
            expected 0:-
            actual   0:hl=1
`))
			Expect(counts).To(Equal(map[string]int{"FORMAT": 1}))
		})

		It("reports successes without details", func() {
			var buf bytes.Buffer
			counts := map[string]int{}
			cfg := &config{file: DefaultFile, details: 10}
			Expect(report(&buf, cfg, "ex", checked{expected: expected, actual: expected, result: compare.Compare(expected, expected)}, counts)).To(BeFalse())
			Expect(buf.String()).To(HavePrefix("SAME      ex: "))
			Expect(buf.String()).NotTo(ContainSubstring("\n "))
		})

		It("reports other values with details, but not as failure", func() {
			actual := behaviour.Analyze([]frames.Frame{{st("a 1", "")}, {st("b", "")}})
			expected := behaviour.Analyze([]frames.Frame{{st("a 2", "")}, {st("b", "")}})
			c := checked{expected: expected, actual: actual, result: compare.Compare(expected, actual)}
			var buf bytes.Buffer
			cfg := &config{file: DefaultFile, details: 10}
			Expect(report(&buf, cfg, "ex", c, map[string]int{})).To(BeFalse())
			Expect(buf.String()).To(HavePrefix("VALUES    ex: "))
			Expect(buf.String()).To(ContainSubstring(`only expected: "a 2" 0:-`))
			Expect(buf.String()).To(ContainSubstring(`only actual:   "a 1" 0:-`))

			cfg.strict = true
			Expect(report(&buf, cfg, "ex", c, map[string]int{})).To(BeTrue())
		})

		It("reports transient states with details, but not as failure", func() {
			merged := func(fs ...[]frames.Frame) *behaviour.Behaviour {
				b := behaviour.Analyze(fs[0])
				for _, f := range fs[1:] {
					b.Merge(behaviour.Analyze(f))
				}
				return b
			}
			expected := merged([]frames.Frame{{st("a 1", "")}, {st("a 1", "")}}, []frames.Frame{{st("a 1", "")}, {st("a 1", "")}})
			actual := merged([]frames.Frame{{st("setup", "")}, {st("a 2", "")}, {st("a 2", "")}}, []frames.Frame{{st("a 2", "")}, {st("a 2", "")}})
			c := checked{expected: expected, actual: actual, result: compare.Compare(expected, actual)}
			var buf bytes.Buffer
			cfg := &config{file: DefaultFile, details: 10}
			Expect(report(&buf, cfg, "ex", c, map[string]int{})).To(BeFalse())
			Expect(buf.String()).To(HavePrefix("TRANSIENT ex: "))
			Expect(buf.String()).To(ContainSubstring("without transient states: VALUES\n"))
			Expect(buf.String()).To(ContainSubstring(`transient:     "setup" 0:-`))

			// other values besides the transient states are failures with -strict
			cfg.strict = true
			Expect(report(&buf, cfg, "ex", c, map[string]int{})).To(BeTrue())
		})

		It("reports missing files and errors", func() {
			var buf bytes.Buffer
			counts := map[string]int{}
			cfg := &config{file: DefaultFile}
			Expect(report(&buf, cfg, "ex", checked{missing: true, err: os.ErrNotExist}, counts)).To(BeTrue())
			Expect(report(&buf, cfg, "ex", checked{err: os.ErrInvalid}, counts)).To(BeTrue())
			Expect(buf.String()).To(Equal("MISSING   ex: no expected.ttyframes.json, record it with ttyframes record\nERROR     ex: invalid argument\n"))
			buf.Reset()
			summary(&buf, counts)
			Expect(buf.String()).To(Equal("\n1 MISSING, 1 ERROR\n"))
		})
	})

	Context("end to end", Ordered, func() {
		var wd, dir string

		BeforeAll(func() {
			if _, err := os.Stat("/usr/bin/script"); err != nil {
				Skip("script not available")
			}
			var err error
			wd, err = os.Getwd()
			Expect(err).To(Succeed())
			dir = tempModule(map[string]string{"anim/main.go": program})
			Expect(os.Chdir(dir)).To(Succeed())
			DeferCleanup(func() { os.Chdir(wd) })
		})

		It("reports missing behaviour files", func() {
			Expect(run([]string{"check", "-runs", "1", "./..."})).To(Equal(1))
		})

		It("records the behaviour", func() {
			Expect(run([]string{"record", "-runs", "1", "./..."})).To(Equal(0))
			data, err := os.ReadFile(filepath.Join(dir, "anim", DefaultFile))
			Expect(err).To(Succeed())
			Expect(string(data)).To(ContainSubstring(`{"text":"N% done","runs":[{"start":0,"format":{"fg":"32"}},{"start":2}],"frames":[5]}`))
			Expect(string(data)).To(ContainSubstring(`{"text":"step 4","runs":[{"start":0}],"frames":[1]}`))
		})

		It("checks the behaviour", func() {
			Expect(run([]string{"check", "-runs", "1", "./..."})).To(Equal(0))
		})

		It("detects changed behaviour", func() {
			Expect(run([]string{"check", "-runs", "1", "-env", "CHANGED=1", "./..."})).To(Equal(1))
		})
	})
})
