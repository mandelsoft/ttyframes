package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/mandelsoft/ttyframes/behaviour"
	"github.com/mandelsoft/ttyframes/frames"
	"github.com/mandelsoft/ttyframes/runner"
)

// capture builds a package and provides its behaviour merged from
// the configured number of runs.
func capture(cfg *config, pkg runner.Package, bindir string) (*behaviour.Behaviour, error) {
	bin, err := runner.Build(pkg, bindir)
	if err != nil {
		return nil, err
	}
	b := behaviour.New()
	for i := 0; i < cfg.runs; i++ {
		out, err := runner.Run(bin, pkg.Dir, cfg.options)
		if err != nil {
			return nil, err
		}
		b.Merge(behaviour.Analyze(frames.Split(out, cfg.masks)))
	}
	return b, nil
}

// forPackages lists the packages and calls f for every package in
// parallel. The results are provided in the order of the packages.
func forPackages[T any](cfg *config, f func(pkg runner.Package, bindir string) T) ([]runner.Package, []T, error) {
	wd, err := os.Getwd()
	if err != nil {
		return nil, nil, err
	}
	pkgs, err := runner.List(wd, cfg.patterns...)
	if err != nil {
		return nil, nil, err
	}
	if len(pkgs) == 0 {
		return nil, nil, fmt.Errorf("no main packages found for %v", cfg.patterns)
	}
	bindir, err := os.MkdirTemp("", "ttyframes")
	if err != nil {
		return nil, nil, err
	}
	defer os.RemoveAll(bindir)

	results := make([]T, len(pkgs))
	sem := make(chan struct{}, cfg.parallel)
	var wg sync.WaitGroup
	for i, pkg := range pkgs {
		wg.Add(1)
		go func() {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			results[i] = f(pkg, bindir)
		}()
	}
	wg.Wait()
	return pkgs, results, nil
}

// name provides the name of a package used in reports:
// its directory relative to the working directory.
func name(pkg runner.Package) string {
	wd, err := os.Getwd()
	if err == nil {
		if rel, err := filepath.Rel(wd, pkg.Dir); err == nil {
			return rel
		}
	}
	return pkg.ImportPath
}

// expectedFile provides the path of the expected behaviour of a package.
func expectedFile(cfg *config, pkg runner.Package) string {
	return filepath.Join(pkg.Dir, cfg.file)
}
