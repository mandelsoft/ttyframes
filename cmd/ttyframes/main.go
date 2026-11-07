// Command ttyframes records and checks the behaviour of programs
// updating trailing lines of a terminal, like progress indicators.
//
//	ttyframes record [options] <packages>
//	ttyframes check [options] <packages>
//
// The main packages matching the package patterns are built and run
// several times in a pseudo terminal. Their output is split into frames,
// and the states (lines) of the frames and their transitions are
// collected. record stores this behaviour in a file in the directory of
// every package, check compares it with the stored behaviour and reports
// the kind of differences.
package main

import (
	"fmt"
	"os"
)

func main() {
	os.Exit(run(os.Args[1:]))
}

// run executes the command and provides the exit code.
func run(args []string) int {
	if len(args) == 0 {
		usage()
		return 2
	}
	var cmd func(*config) int
	switch args[0] {
	case "record":
		cmd = record
	case "check":
		cmd = check
	case "help", "-h", "-help", "--help":
		usage()
		return 0
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n", args[0])
		usage()
		return 2
	}
	cfg, err := parseConfig(args[0], args[1:])
	if err != nil {
		return 2
	}
	return cmd(cfg)
}

func usage() {
	fmt.Fprintf(os.Stderr, `usage: ttyframes <command> [options] <packages>

commands:
  record   record the behaviour of the main packages
  check    compare the behaviour of the main packages with the recorded one

Use ttyframes <command> -h for the options.
`)
}
