package main

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/mandelsoft/ttyframes/behaviour"
	"github.com/mandelsoft/ttyframes/compare"
	"github.com/mandelsoft/ttyframes/frames"
	"github.com/mandelsoft/ttyframes/runner"
)

// checked is the result of checking a package.
type checked struct {
	expected, actual *behaviour.Behaviour
	result           *compare.Result
	err              error
	missing          bool
}

// check compares the behaviour of the packages with the expected one.
func check(cfg *config) int {
	pkgs, results, err := forPackages(cfg, func(pkg runner.Package, bindir string) checked {
		f, err := os.Open(expectedFile(cfg, pkg))
		if err != nil {
			return checked{err: err, missing: os.IsNotExist(err)}
		}
		expected, err := behaviour.Read(f)
		f.Close()
		if err != nil {
			return checked{err: fmt.Errorf("%s: %w", cfg.file, err)}
		}
		actual, err := capture(cfg, pkg, bindir)
		if err != nil {
			return checked{err: err}
		}
		return checked{expected: expected, actual: actual, result: compare.Compare(expected, actual)}
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	code := 0
	counts := map[string]int{}
	for i, pkg := range pkgs {
		if report(os.Stdout, cfg, name(pkg), results[i], counts) {
			code = 1
		}
	}
	summary(os.Stdout, counts)
	return code
}

// report reports the result for a package and
// provides whether it is a failure.
func report(w io.Writer, cfg *config, name string, c checked, counts map[string]int) bool {
	switch {
	case c.missing:
		counts["MISSING"]++
		fmt.Fprintf(w, "%-9s %s: no %s, record it with ttyframes record\n", "MISSING", name, cfg.file)
		return true
	case c.err != nil:
		counts["ERROR"]++
		fmt.Fprintf(w, "%-9s %s: %s\n", "ERROR", name, c.err)
		return true
	}
	r := c.result
	counts[r.Category.Name]++
	fmt.Fprintf(w, "%-9s %s: frames %s/%s, states %d/%d (only expected %d, only actual %d), transitions %d/%d (only expected %d, only actual %d)\n",
		r.Category, name, frameCounts(c.expected), frameCounts(c.actual),
		len(c.expected.States), len(c.actual.States), len(r.OnlyExpected), len(r.OnlyActual),
		len(c.expected.Transitions), len(c.actual.Transitions), r.TransitionsOnlyExpected, r.TransitionsOnlyActual)
	if r.Category.Details || cfg.verbose {
		details(w, cfg, r)
	}
	return failure(cfg, r.Category)
}

// failure checks whether a category is a failure.
func failure(cfg *config, c *compare.Category) bool {
	return c.Failure || (cfg.strict && c == compare.Values)
}

// details reports the differing states.
func details(w io.Writer, cfg *config, r *compare.Result) {
	for i, c := range r.Changes {
		if i >= cfg.details {
			fmt.Fprintf(w, "          ... %d more format changes\n", len(r.Changes)-i)
			break
		}
		fmt.Fprintf(w, "          format of %q\n            expected %s\n            actual   %s\n", c.Actual.Text(), c.Expected.Runs(), c.Actual.Runs())
	}
	if len(r.Changes) > 0 {
		return
	}
	states := func(title string, s []frames.State) {
		for i, e := range s {
			if i >= cfg.details {
				fmt.Fprintf(w, "          ... %d more\n", len(s)-i)
				break
			}
			fmt.Fprintf(w, "          %s %s\n", title, e.Key())
		}
	}
	states("only expected:", r.OnlyExpected)
	states("only actual:  ", r.OnlyActual)
}

func frameCounts(b *behaviour.Behaviour) string {
	var r []string
	for _, n := range b.Frames {
		r = append(r, fmt.Sprint(n))
	}
	if len(r) == 0 {
		return "-"
	}
	return strings.Join(r, ",")
}

// summary reports the number of packages per category.
func summary(w io.Writer, counts map[string]int) {
	var parts []string
	for _, c := range compare.Categories {
		if n := counts[c.Name]; n > 0 {
			parts = append(parts, fmt.Sprintf("%d %s", n, c.Name))
		}
	}
	for _, n := range []string{"MISSING", "ERROR"} {
		if counts[n] > 0 {
			parts = append(parts, fmt.Sprintf("%d %s", counts[n], n))
		}
	}
	fmt.Fprintf(w, "\n%s\n", strings.Join(parts, ", "))
}
