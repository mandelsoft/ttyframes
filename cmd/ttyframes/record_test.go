package main

import (
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/mandelsoft/ttyframes/behaviour"
)

var _ = Describe("Record", func() {
	BeforeEach(func() {
		if !hasScript() {
			Skip("script not available")
		}
	})

	It("records the behaviour of all main packages", func() {
		dir := tempModule(map[string]string{"anim/main.go": program, "other/main.go": program})
		inDir(dir)
		Expect(run([]string{"record", "-runs", "2", "-file", "b.json", "./..."})).To(Equal(0))
		for _, p := range []string{"anim", "other"} {
			f, err := os.Open(filepath.Join(dir, p, "b.json"))
			Expect(err).To(Succeed())
			b, err := behaviour.Read(f)
			f.Close()
			Expect(err).To(Succeed())
			Expect(b.Frames).To(HaveLen(2))
			Expect(b.StateKeys()).To(ContainElement(`"step 4" 0:-`))
		}
	})

	It("reports packages, which cannot be built", func() {
		dir := tempModule(map[string]string{"anim/main.go": program, "bad/main.go": "package main\n\nfunc main() { undefined() }\n"})
		inDir(dir)
		Expect(run([]string{"record", "-runs", "1", "./..."})).To(Equal(1))
		Expect(filepath.Join(dir, "anim", DefaultFile)).To(BeAnExistingFile())
		Expect(filepath.Join(dir, "bad", DefaultFile)).NotTo(BeAnExistingFile())
	})
})
