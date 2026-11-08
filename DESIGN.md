# Design Decisions

This document records the design decisions implemented by `ttyframes`, the
reasons for them and the alternatives considered. The tool emerged from checking
manually whether the animated examples of `ttyprogress` were unchanged after
reworking `ttycolors`, and it keeps the decisions taken there.

## Purpose

**Decision:** Compare the visible behaviour of animated terminal output during the
animation and classify the differences.

**Reason:** The output depends on the timing (frequency of redraws, sampled values,
combinations of independently updated elements), so it is never identical between runs.
A check must tell differences caused by the timing from changed behaviour.

**Alternatives:**
- *Byte-wise comparison of the output:* never identical, and identical screens can be
  produced by different escape sequences.
- *Comparison of the final screen:* was the first approach. It missed a real
  regression: a line bar lost the color of a part of its line only while running,
  the final screen was correct.

## Separate Project and Go Implementation

**Decision:** A separate Go module `github.com/mandelsoft/ttyframes`, used by other
modules as pinned tool dependency.

**Reason:** The check is useful for every module with animated terminal output.
Go is the language of the modules using it.

**Alternatives:**
- *A tool inside `ttyprogress` (`hack/...`):* not reusable by other modules.
- *Python scripts (the first prototype):* an additional runtime, which is not wanted
  for the tooling of Go modules.

## Name

**Decision:** `ttyframes`, files with the suffix `.ttyframes.json`.

**Reason:** It fits the family of modules (`ttycolors`, `ttyprogress`) and names
the subject: the frames shown on a terminal.

**Alternatives:** `ttyreplay` (suggests replaying the output), `ttygolden`
(describes the method instead of the subject).

## Integration via `tools.go`

**Decision:** Modules pin the tool with a `tools.go` file (build tag `tools`)
and run it with `go run`.

**Reason:** The modules use Go 1.23, the version of the tool is pinned by `go.mod`.

**Alternatives:**
- *`tool` directive (`go get -tool`, `go tool ttyframes`):* requires Go 1.24 (deferred).
- *`go install`:* the version is not pinned by the module.

## Pseudo Terminal via `script`

**Decision:** The programs are run with `script -qec` (util-linux), the size of the
pseudo terminal is fixed with `stty`.

**Reason:** Programs only generate formatted output for a terminal. `script` needs no
Go dependency. A fixed size makes the output independent of the calling terminal.

**Alternatives:**
- *`github.com/creack/pty`:* portable, but an additional dependency (kept as option).
- *Forcing colors by an environment variable:* programs may behave differently without
  a terminal, and it is not generic.

## Fixed Random Values

**Decision:** The runs use `GODEBUG=randautoseed=0`.

**Reason:** The random values of `math/rand` are identical for every run, which reduces
the differences between runs. It was used for the manual checks, where it made the
results of most examples reproducible.

**Alternatives:** Seeding inside the programs would also cover `math/rand/v2`, but it is
the responsibility of the programs, not of the tool.

## Line Based Terminal Emulation

**Decision:** An own terminal emulation (package `terminal`) keeping all lines
(no screen size, no scrolling) and the output format of every character. It supports
cursor movements within and between lines, erasing lines and the display, and output
format sequences. Other escape sequences are ignored. Data may be split at arbitrary
positions, incomplete sequences and characters are kept until the next write.

**Reason:** Programs updating trailing lines only use these sequences. Keeping all lines
avoids dependencies on the terminal size.

**Alternatives:**
- *A full VT emulation library:* an additional dependency for features not needed.
- *Rendering with terminalizer (used for demo GIFs):* requires Node.js and provides
  no structured data.

## Canonical Output Formats

**Decision:** The output format of a character is described by the code of every output
format area (for example `hl=1,fg=32`), computed with `ansi.ScanCodes` of `ttycolors`.

**Reason:** Different encodings of the same rendition (combined or separate sequences,
redundant codes) describe the same screen and must compare equal.

**Alternatives:** Comparing the raw escape sequences, which reports differences
without visible effect.

## Frames

**Decision:** A frame is the screen content before every redraw. A redraw starts with a
cursor movement up (or to a previous line) following other output than cursor movements
or erasing.

**Reason:** Programs updating trailing lines move the cursor up to the first line to
update and redraw from there. The rule generalizes the pattern used for the manual checks
(`ESC[1A ESC[2K` not preceded by `ESC[2K`).

**Alternatives:**
- *A frame per write:* writes are split arbitrarily, so partial frames would be compared.
- *Time based snapshots:* the captured output has no timestamps, and the timing would be
  an additional source of differences.

## States and Transitions

**Decision:** Every line of every frame is a state. Transitions relate the lines at the
same position of consecutive frames with the same number of lines. The behaviour is the
set of states and transitions.

**Reason:** Lines are mostly updated independently. Whole frames as states would
multiply the combinations, and sampling differences would dominate.

**Alternatives:** Frames as states, sequences of frames (both depend on the timing).

## Masks

**Decision:** Timing dependent values are replaced by masks (default: elapsed seconds
`\b\d+s\b` → `Ns`, percentages `\b\d+%` → `N%`). Additional masks can be given with
`-mask`. Masks are applied to the complete text of a line, a replacement gets the output
format of the first replaced character.

**Reason:** Such values differ between runs without a change of the behaviour. Applying
the masks to complete lines also masks values spanning output format changes, for example
a percentage partly shown in reverse video by a line bar. Masking the texts of the
segments separately missed such values.

**Alternatives:** Masking segments separately (see above). Storing the masks in the
behaviour file was not done: they are part of the call (for example the Makefile).

## Several Runs

**Decision:** The behaviour of several runs (default 3) is merged, for recording as well
as for checking. The numbers of frames are only kept as information.

**Reason:** A single run samples only some of the possible states.

**Alternatives:** A single run (more sampling differences).

## Classification

**Decision:** The categories `SAME`, `TIMING`, `SAMPLING`, `VALUES`, `TRANSIENT`, `FORMAT` and
`TEXT`, checked in this order. `FORMAT` and `TEXT` are failures, `VALUES` with the option
`-strict` (see Transient States for `TRANSIENT`). For `VALUES`, `TRANSIENT`, `FORMAT` and
`TEXT`, the differing states are reported. The categories are values with flags for failures
and details, not an enumeration.

- `TIMING`: states and transitions are identical after normalizing white space.
- `SAMPLING`: the sets of normalized segments (texts with output format) are identical.
- `VALUES`: the sets of normalized segments are identical after masking all values
  (see Random Values of Programs and Dates and Durations).
- `TRANSIENT`: the comparison without the differing transient states provides one of the
  categories before (see Transient States).
- `FORMAT`: a state only found in one behaviour has the same normalized text (up to the
  values of numbers) as a state only found in the other behaviour, but other output formats.
- `TEXT`: all other differences.

**Reason:** These are the kinds of differences found by the manual checks: shifted columns
(values of other lengths), other combinations of independently updated elements, and real
changes like the lost color of a line bar or a fixed bug in a group of bars.

**Alternatives:** A plain diff of states (cannot distinguish the kinds), thresholds for the
number of differences (no meaning).

## Random Values of Programs

**Decision:** Differences, which disappear when all values are masked, are a category of
their own (`VALUES`). It is no failure by default, but its differing states are reported.
The option `-strict` makes it a failure. A value is an optionally signed number followed by
further numbers separated by `.`, `,`, `:`, `/` or `-`, and it is masked as a whole.

**Reason:** Programs, especially examples, use random values, for example the number of
steps or the duration of a task. If several goroutines share a random number generator,
the scheduling decides which task gets which value, even with a fixed seed, and the values
of `math/rand/v2` cannot be fixed at all. The checked programs must not be changed for the
check: examples are documentation for their users and should not contain hooks of a test
environment. The variation of such values is recognized generically, without configuration.
The loss of detecting changes of numbers only is made visible by reporting the states,
and `-strict` covers programs whose numbers must be reproducible.

Masking a value as a whole accepts floating point numbers, dates, times and durations with a
different number of parts (for example `1:05` and `1:02:03`), which masking every number
separately would distinguish. The separators also join unrelated numbers (for example a
range `1-3` or a version `1.2.3`), which is acceptable for a category, which is no failure.

**Alternatives:**
- *Masking every number separately (the first version):* distinguishes values with a
  different number of parts, for example durations below and above an hour.
- *Separate patterns for every kind of value:* more precise, but the kinds of values shown
  by programs are open-ended, and the distinction is not needed for the classification.
- *Random sources per task with a seed given by the tool (for example `TTYFRAMES_SEED`):*
  makes the values reproducible, but requires changes of the checked programs, and the
  scheduling of goroutines still influences the timing.
- *Additional masks for the varying values (option `-mask`):* masks them for all
  programs of a check, also where they are relevant.
- *Masks per program in a configuration file next to the behaviour file:* precise, but
  requires a configuration per program and hides the changes completely (deferred).

## Transient States

**Decision:** A state is *transient*, if it is shown in at most one frame of a run and not in
all runs: a snapshot shown by chance. The behaviour keeps the numbers of frames showing a
state per run. If the comparison without the differing transient states (of both behaviours)
provides a category, which is no failure, the result is `TRANSIENT`, with this category as
residual category. It is no failure; with `-strict`, it is a failure if the residual category
is `VALUES`. The transient and differing states are reported.

**Reason:** Programs show short-lived intermediate states, which are seen only if a redraw
happens to fall into them: a line set up step by step (an element added, then variables
set, then started), or an element already complete, but not yet replaced by its final text.
The examples of `ttyprogress` (`variables`, `multi`) failed sporadically as `TEXT` because of
such states, although their behaviour did not change. Real changes are repeated by every
redraw and shown by every run. The checked programs must not be changed for the check (see
Random Values of Programs).

**Alternatives:**
- *Transient as "shown in a single frame" only (the first version):* a deterministic program
  may show every state in a single frame (for example a sequence of steps), so that all its
  differences, including real changes, would be removed. Requiring that the state is not
  shown by all runs distinguishes chance from regular behaviour.
- *Only the maximum number of frames per state:* cannot tell whether a state was shown by all
  runs.
- *Additional states being the beginning of a known state, plus a mask for bar contents
  configured by the checking project:* two special rules for one kind of difference, and the
  mask would also hide real changes of the rendering of bars.
- *Changing the examples (setting the variables before adding an element):* examples are
  documentation, the test tooling has to be tolerant.

A sporadic error shown only in a single frame of some runs is tolerated, but reported. With
a single run (`-runs 1`), no state is transient.

## Dates and Durations

**Decision:** Dates and times are recognized according to the layouts of the Go package `time`:
all standard layouts and additional layouts given with `-layout` in Go layout notation. The
regular expressions are derived from the layouts by converting their elements (for example
`2006`, `Jan`, `Mon`, `MST`, `Z07:00`, `.000`, `PM`). Durations are recognized in the format of
`time.Duration`. Dates and durations are masked before the other values. They are replaced by a
digit first, so that the values pattern joins them with adjacent parts (for example a fraction
not covered by a layout) to a single value. Space padded elements (`_2`) at the beginning of a
layout start with their digits, the padding is kept as text and normalized as white space.

**Reason:** Dates with names (`Sat, 07 Nov 2026 09:05:03 CET`), time zones (`Z`, `+01:00`) or
`AM`/`PM` and durations with units (`1h2m3.5s`, `250ms`) are not covered by the pattern for
numbers. Deriving the patterns from layouts covers exactly the formats supported by Go, and
programs with own formats can describe them in the notation they use to produce them. Names
are only masked as part of a date, so that changed texts containing names are still detected.

**Alternatives:**
- *Hand-written regular expressions per format:* error-prone and incomplete, own formats
  would require regular expressions instead of layouts.
- *Masking names of months, weekdays and time zones everywhere:* hides changes of texts
  containing such words.
- *Parsing every substring with `time.Parse`:* expensive, and the substrings to check would
  have to be found first.

## Behaviour File

**Decision:** A JSON file with a version marker (`"format": "ttyframes/v1"`), sorted states
and sorted transitions, transitions are given with their complete states. Output formats
are objects of codes indexed by the names of the output format areas. The tool writes every
state and transition in a line of its own, but reads any valid JSON with this structure.
The file is stored in the package directory with the name `expected.ttyframes.json`.

**Reason:** The behaviour can be evaluated by other tools (for example `jq`) without a
parser for an own format, and JSON is supported by the standard library. The layout with a
line per element keeps the differences reviewed with `git diff` small and readable, like
sorting and complete states do. The version marker allows changes of the format. The name
says that the file is the reference of the comparison, to which tool it belongs and which
format it has, and it is found next to the program it describes.

The numbers of frames per state (`"frames"` of a state) are optional: files written before
they were introduced are still read, their states are never transient. The version marker
was kept (`ttyframes/v1`), because the extension is compatible for reading old files, and the
tool is pinned by the checking projects. Tool versions before this extension reject a newer
file because of the unknown attribute; a new version marker would not help them, because
they decode the file strictly before checking the marker.

Every further change of the format, also a compatible extension, gets a new version marker,
and the tool keeps reading the older versions. Together with the check of the marker before
the content, an older tool then reports a newer file as unsupported format instead of an
unqualified read error. The extension by the numbers of frames is the only one without a new
marker, because it was introduced before the check order.

The version marker is checked before the content: a file is first read for its marker only,
and only a supported version is decoded strictly (unknown attributes are errors). Files of
future versions are therefore reported as unsupported format instead of failing on their new
attributes.

**Alternatives:**
- *An own line based text format (the first version):* very readable, but requires an own
  parser for every tool evaluating it.
- *YAML:* similarly readable, but requires an additional dependency.
- *Indented JSON (`json.MarshalIndent`):* spreads a state over many lines, the differences
  are hard to read.
- *Output formats as strings like `fg=32,rev=7`:* require parsing by other tools.
- *Transitions referring to state indices:* a new state shifts all indices.
- *`expected.ttyframes` (without `.json`):* the format is not recognized by editors and tools.
  *`.txt`:* no relation to the tool. *`.golden`:* the Go convention for reference files,
  but generic. *`demo.ttyframes.json`:* suggests a demonstration.
- *A central `testdata` directory:* separates the reference from the program.

## Deferred Topics

- Using the `tool` directive of Go 1.24.
- A portable pseudo terminal (`github.com/creack/pty`).
- Checking programs given as command instead of Go packages.
- Full screen emulation (screen size, scrolling) for programs redrawing the whole screen.
- Masks per program in a configuration file next to the behaviour file.
