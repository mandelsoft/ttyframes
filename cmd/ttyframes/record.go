package main

import (
	"fmt"
	"os"

	"github.com/mandelsoft/ttyframes/behaviour"
	"github.com/mandelsoft/ttyframes/runner"
)

// record records the behaviour of the packages.
func record(cfg *config) int {
	pkgs, errs, err := forPackages(cfg, func(pkg runner.Package, bindir string) error {
		b, err := capture(cfg, pkg, bindir)
		if err != nil {
			return err
		}
		f, err := os.Create(expectedFile(cfg, pkg))
		if err != nil {
			return err
		}
		if err := behaviour.Write(f, b); err != nil {
			f.Close()
			return err
		}
		return f.Close()
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	code := 0
	for i, pkg := range pkgs {
		if errs[i] != nil {
			fmt.Printf("%-9s %s: %s\n", "ERROR", name(pkg), errs[i])
			code = 1
		} else {
			fmt.Printf("%-9s %s\n", "RECORDED", name(pkg))
		}
	}
	return code
}
