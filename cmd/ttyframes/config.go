package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/mandelsoft/ttyframes/frames"
	"github.com/mandelsoft/ttyframes/runner"
)

// DefaultFile is the default name of the file
// containing the expected behaviour.
const DefaultFile = "expected.ttyframes.json"

// config is the configuration of a command.
type config struct {
	runs     int
	file     string
	parallel int
	details  int
	verbose  bool
	strict   bool
	masks    frames.Masks
	options  runner.Options
	patterns []string
}

// masksFlag collects additional masks.
type masksFlag struct {
	masks *frames.Masks
}

func (m masksFlag) String() string {
	if m.masks == nil {
		return ""
	}
	var r []string
	for _, e := range *m.masks {
		r = append(r, e.String())
	}
	return strings.Join(r, " ")
}

func (m masksFlag) Set(s string) error {
	e, err := frames.ParseMask(s)
	if err != nil {
		return err
	}
	*m.masks = append(*m.masks, e)
	return nil
}

// envFlag collects environment variables.
type envFlag struct {
	env *[]string
}

func (e envFlag) String() string {
	if e.env == nil {
		return ""
	}
	return strings.Join(*e.env, " ")
}

func (e envFlag) Set(s string) error {
	if !strings.Contains(s, "=") {
		return fmt.Errorf("NAME=value expected")
	}
	*e.env = append(*e.env, s)
	return nil
}

// parseConfig parses the options and package patterns of a command.
func parseConfig(name string, args []string) (*config, error) {
	cfg := &config{
		masks:   append(frames.Masks(nil), frames.DefaultMasks...),
		options: runner.DefaultOptions,
	}
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.IntVar(&cfg.runs, "runs", 3, "number of runs per package, the behaviour of all runs is merged")
	fs.StringVar(&cfg.file, "file", DefaultFile, "name of the file containing the expected behaviour in the package directory")
	fs.IntVar(&cfg.parallel, "parallel", 4, "number of packages processed in parallel")
	fs.IntVar(&cfg.details, "details", 10, "maximum number of differing states reported per package")
	fs.BoolVar(&cfg.verbose, "v", false, "report the differing states for all categories")
	fs.BoolVar(&cfg.strict, "strict", false, "differences of the values of numbers (category VALUES) are failures")
	fs.Var(masksFlag{&cfg.masks}, "mask", "additional mask <regexp>=<replacement> for timing dependent values (repeatable)")
	fs.Var(envFlag{&cfg.options.Env}, "env", "additional environment variable NAME=value for the runs (repeatable)")
	fs.IntVar(&cfg.options.Columns, "cols", cfg.options.Columns, "number of columns of the pseudo terminal")
	fs.IntVar(&cfg.options.Rows, "rows", cfg.options.Rows, "number of rows of the pseudo terminal")
	fs.DurationVar(&cfg.options.Timeout, "timeout", cfg.options.Timeout, "maximum duration of a run")
	fs.Usage = func() {
		fmt.Fprintf(fs.Output(), "usage: ttyframes %s [options] <packages>\n\noptions:\n", name)
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		return nil, err
	}
	cfg.patterns = fs.Args()
	switch {
	case len(cfg.patterns) == 0:
		err := fmt.Errorf("package patterns required")
		fmt.Fprintln(os.Stderr, err)
		fs.Usage()
		return nil, err
	case cfg.runs < 1 || cfg.parallel < 1:
		err := fmt.Errorf("runs and parallel must be at least 1")
		fmt.Fprintln(os.Stderr, err)
		return nil, err
	}
	return cfg, nil
}
