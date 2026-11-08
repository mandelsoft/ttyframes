# ttyframes

`ttyframes` records and checks the behaviour of programs updating trailing lines of a
terminal (for example progress indicators): it runs Go main packages in a pseudo terminal,
splits their output into frames and compares the states (lines) and transitions with a
recorded behaviour (`expected.ttyframes.json`), classifying the differences.
See the README for the usage and [DESIGN.md](DESIGN.md) for the design decisions.

This project is created and maintained by Claude on behalf of its owner (see the README).
The conversation so far and the design decisions are imported below, read them before
working on the project.

## Structure

- `terminal`: line based terminal emulation with the output format of every character.
- `frames`: splitting an output into frames, masks for timing dependent values, states.
- `behaviour`: states and transitions of several runs, the numbers of frames per state and
  run (transient states), the JSON file format.
- `compare`: classification of differences, recognition of values (dates via Go layouts,
  durations, numbers).
- `runner`: listing, building and running main packages in a pseudo terminal (`script`).
- `cmd/ttyframes`: the command line tool (`record`, `check`).

## Commands

- `make build`, `make test` (or `go build ./...`, `go test ./...`), `go vet ./...`, `gofmt -l .`
- The tests running programs require the command `script` (util-linux).

## Maintenance Rules

For every change:

- Append the part of the conversation concerning this project (requests, decisions,
  outcome) to [CONVERSATION.md](CONVERSATION.md), in a section for the date.
- Keep [DESIGN.md](DESIGN.md) (decisions with reasons and alternatives, deferred topics),
  the README and the godoc consistent with the code.
- Every source file has a Ginkgo test file of its own, all changes are tested.
- Every change of the behaviour file format, also a compatible extension, gets a new
  format version (`behaviour.FormatVersion`), and older versions stay readable (see the
  section Behaviour File of DESIGN.md).
- Dependencies are limited to the standard library, `github.com/mandelsoft/ttycolors`
  and Ginkgo/Gomega. Ask before adding others.
- Commit and push only when requested. After pushing a new version, update the version
  used by `github.com/mandelsoft/ttyprogress` (`tools.go`, `make check-behaviour`) and
  check its examples.

## Conversation

@CONVERSATION.md

## Design Decisions

@DESIGN.md
