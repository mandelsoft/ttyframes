package main

import (
	"path/filepath"
	"sort"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/mandelsoft/ttyframes/runner"
)

var _ = Describe("Process", func() {
	It("names packages by their directory relative to the working directory", func() {
		dir := tempModule(map[string]string{"a/b/main.go": "package main\n\nfunc main() {}\n"})
		inDir(dir)
		Expect(name(runner.Package{ImportPath: "example.com/anim/a/b", Dir: filepath.Join(dir, "a", "b")})).To(Equal(filepath.Join("a", "b")))
	})

	It("provides the path of the expected behaviour", func() {
		cfg := &config{file: "x.json"}
		Expect(expectedFile(cfg, runner.Package{Dir: "/p"})).To(Equal("/p/x.json"))
	})

	It("processes all main packages in their order", func() {
		dir := tempModule(map[string]string{
			"b/main.go":   "package main\n\nfunc main() {}\n",
			"a/main.go":   "package main\n\nfunc main() {}\n",
			"lib/lib.go":  "package lib\n",
			"c/x/main.go": "package main\n\nfunc main() {}\n",
		})
		inDir(dir)
		cfg := &config{parallel: 2, patterns: []string{"./..."}}
		pkgs, results, err := forPackages(cfg, func(pkg runner.Package, bindir string) string {
			return pkg.ImportPath
		})
		Expect(err).To(Succeed())
		Expect(pkgs).To(HaveLen(3))
		Expect(results).To(Equal([]string{"example.com/anim/a", "example.com/anim/b", "example.com/anim/c/x"}))
		Expect(sort.StringsAreSorted(results)).To(BeTrue())
	})

	It("reports patterns without main packages", func() {
		dir := tempModule(map[string]string{"lib/lib.go": "package lib\n"})
		inDir(dir)
		_, _, err := forPackages(&config{parallel: 1, patterns: []string{"./..."}}, func(runner.Package, string) bool { return true })
		Expect(err).To(MatchError(ContainSubstring("no main packages")))
	})

	It("reports build errors when capturing", func() {
		dir := tempModule(map[string]string{"bad/main.go": "package main\n\nfunc main() { undefined() }\n"})
		inDir(dir)
		pkgs, err := runner.List(dir, "./...")
		Expect(err).To(Succeed())
		_, err = capture(&config{runs: 1, options: runner.DefaultOptions}, pkgs[0], GinkgoT().TempDir())
		Expect(err).To(MatchError(ContainSubstring("build")))
	})
})
