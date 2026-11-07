package main

import (
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestConfig(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Command Test Suite")
}

// tempModule creates a temporary module example.com/anim with the
// given files (indexed by their paths relative to the module).
func tempModule(files map[string]string) string {
	dir := GinkgoT().TempDir()
	Expect(os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/anim\n\ngo 1.23\n"), 0o644)).To(Succeed())
	for name, content := range files {
		path := filepath.Join(dir, name)
		Expect(os.MkdirAll(filepath.Dir(path), 0o755)).To(Succeed())
		Expect(os.WriteFile(path, []byte(content), 0o644)).To(Succeed())
	}
	return dir
}

// inDir runs the test with the given working directory.
func inDir(dir string) {
	wd, err := os.Getwd()
	Expect(err).To(Succeed())
	Expect(os.Chdir(dir)).To(Succeed())
	DeferCleanup(func() { os.Chdir(wd) })
}

// hasScript checks whether the command script is available.
func hasScript() bool {
	_, err := os.Stat("/usr/bin/script")
	return err == nil
}
