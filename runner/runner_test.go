package runner_test

import (
	"os"
	"path/filepath"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/mandelsoft/ttyframes/runner"
)

var _ = Describe("Runner", func() {
	It("quotes for the shell", func() {
		Expect(runner.ShellQuote("/tmp/a b")).To(Equal(`'/tmp/a b'`))
		Expect(runner.ShellQuote("it's")).To(Equal(`'it'\''s'`))
	})

	It("provides the command line", func() {
		o := runner.Options{Columns: 80, Rows: 24}
		Expect(o.Command("/tmp/x")).To(Equal([]string{"script", "-qec", "stty cols 80 rows 24; exec '/tmp/x'", "/dev/null"}))
	})

	It("fixes random values", func() {
		o := runner.Options{Env: []string{"A=b"}}
		Expect(o.Environment()).To(ContainElements("GODEBUG=randautoseed=0", "A=b"))
	})

	It("lists main packages", func() {
		pkgs, err := runner.List("..", "./...")
		Expect(err).To(Succeed())
		Expect(pkgs).To(HaveLen(1))
		Expect(pkgs[0].ImportPath).To(Equal("github.com/mandelsoft/ttyframes/cmd/ttyframes"))
		abs, _ := filepath.Abs("../cmd/ttyframes")
		Expect(pkgs[0].Dir).To(Equal(abs))
	})

	It("reports invalid patterns", func() {
		_, err := runner.List("..", "./unknown")
		Expect(err).To(HaveOccurred())
	})

	It("builds and runs a program", func() {
		if _, err := os.Stat("/usr/bin/script"); err != nil {
			Skip("script not available")
		}
		dir := GinkgoT().TempDir()
		Expect(os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/tty\n\ngo 1.23\n"), 0o644)).To(Succeed())
		Expect(os.WriteFile(filepath.Join(dir, "main.go"), []byte(`package main

import (
	"fmt"
	"os"
	"strings"
)

func main() {
	fi, _ := os.Stdout.Stat()
	fmt.Println("tty", fi.Mode()&os.ModeCharDevice != 0, strings.Contains(os.Getenv("GODEBUG"), "randautoseed=0"))
}
`), 0o644)).To(Succeed())
		pkgs, err := runner.List(dir, ".")
		Expect(err).To(Succeed())
		Expect(pkgs).To(HaveLen(1))
		bin, err := runner.Build(pkgs[0], GinkgoT().TempDir())
		Expect(err).To(Succeed())
		o := runner.DefaultOptions
		o.Timeout = time.Minute
		out, err := runner.Run(bin, dir, o)
		Expect(err).To(Succeed())
		Expect(string(out)).To(ContainSubstring("tty true true"))
	})

	It("reports a timeout", func() {
		if _, err := os.Stat("/usr/bin/script"); err != nil {
			Skip("script not available")
		}
		bin := filepath.Join(GinkgoT().TempDir(), "wait")
		Expect(os.WriteFile(bin, []byte("#!/bin/sh\nsleep 10\n"), 0o755)).To(Succeed())
		_, err := runner.Run(bin, ".", runner.Options{Columns: 80, Rows: 24, Timeout: 200 * time.Millisecond})
		Expect(err).To(MatchError(ContainSubstring("timeout")))
	})
})
