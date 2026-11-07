// Package runner builds the main packages of a Go module and runs them
// in a pseudo terminal, so that they generate the output for a terminal.
// The pseudo terminal is provided by the command script (util-linux).
package runner

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// Package is a main package.
type Package struct {
	ImportPath string
	// Dir is the directory of the package.
	Dir string
}

// List provides the main packages matching the given package patterns
// (see go help packages), evaluated in the given directory.
func List(dir string, patterns ...string) ([]Package, error) {
	args := append([]string{"list", "-f", "{{if eq .Name \"main\"}}{{.ImportPath}}\t{{.Dir}}{{end}}"}, patterns...)
	cmd := exec.Command("go", args...)
	cmd.Dir = dir
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("go list: %w: %s", err, strings.TrimSpace(stderr.String()))
	}
	var r []Package
	for _, l := range strings.Split(string(out), "\n") {
		if f := strings.SplitN(l, "\t", 2); len(f) == 2 {
			r = append(r, Package{ImportPath: f[0], Dir: f[1]})
		}
	}
	return r, nil
}

// Build builds a main package into the given directory
// and provides the path of the binary.
func Build(pkg Package, dir string) (string, error) {
	bin := filepath.Join(dir, strings.ReplaceAll(pkg.ImportPath, "/", "_"))
	cmd := exec.Command("go", "build", "-o", bin, pkg.ImportPath)
	cmd.Dir = pkg.Dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("build %s: %w: %s", pkg.ImportPath, err, strings.TrimSpace(string(out)))
	}
	return bin, nil
}

// Options are the options for running a program.
type Options struct {
	// Columns and Rows define the size of the pseudo terminal.
	Columns, Rows int
	// Timeout is the maximum duration of a run.
	Timeout time.Duration
	// Env are additional environment variables (NAME=value).
	Env []string
}

// DefaultOptions are the default options for running programs.
var DefaultOptions = Options{
	Columns: 160,
	Rows:    50,
	Timeout: 3 * time.Minute,
}

// Environment provides the environment variables set for a run:
// the random number generator of package math/rand is used with
// a fixed seed (GODEBUG=randautoseed=0), so that the programs use
// the same random values for every run. The generators of package
// math/rand/v2 cannot be fixed this way.
func (o Options) Environment() []string {
	env := []string{"GODEBUG=randautoseed=0"}
	if os.Getenv("TERM") == "" {
		env = append(env, "TERM=xterm-256color")
	}
	return append(env, o.Env...)
}

// Command provides the command line running a binary in a pseudo
// terminal of the given size.
func (o Options) Command(binary string) []string {
	return []string{"script", "-qec",
		fmt.Sprintf("stty cols %d rows %d; exec %s", o.Columns, o.Rows, ShellQuote(binary)),
		"/dev/null"}
}

// Run runs a binary in a pseudo terminal with the given working
// directory and provides its output.
func Run(binary, dir string, o Options) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), o.Timeout)
	defer cancel()
	args := o.Command(binary)
	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), o.Environment()...)
	out, err := cmd.Output()
	if ctx.Err() != nil {
		return out, fmt.Errorf("run %s: timeout after %s", binary, o.Timeout)
	}
	if err != nil {
		var exit *exec.ExitError
		if errors.As(err, &exit) {
			return out, fmt.Errorf("run %s: %w: %s", binary, err, strings.TrimSpace(string(exit.Stderr)))
		}
		return out, fmt.Errorf("run %s: %w", binary, err)
	}
	return out, nil
}

// ShellQuote quotes a string for a POSIX shell.
func ShellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}
