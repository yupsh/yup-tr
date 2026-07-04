// Command yup-tr is the CLI wrapper around github.com/gloo-foo/cmd-tr.
package main

import (
	clix "github.com/gloo-foo/cli"
	command "github.com/gloo-foo/cmd-tr"
	urf "github.com/urfave/cli/v3"
)

// version is the build version. It defaults to "dev" for local builds and is
// overridden at release time via the linker: -ldflags "-X main.version=<v>".
var version = "dev"

const (
	name           = "tr"
	flagDelete     = "delete"
	flagSqueeze    = "squeeze-repeats"
	flagComplement = "complement"
)

// Error is the package's sentinel error type, so every emitted error path is
// comparable with errors.Is.
type Error string

func (e Error) Error() string { return string(e) }

// ErrMissingOperand is emitted when the required SET operands are absent: SET1 is
// always required, and SET2 is required unless deleting (-d) or squeezing (-s).
const ErrMissingOperand Error = "missing operand"

// synopsis is the multi-line --help usage block; urfave/cli indents it three
// spaces, so the lines stay flush-left.
const synopsis = `tr [OPTIONS] SET1 [SET2]

Translate, squeeze, and/or delete characters from standard
input, writing to standard output.`

// spec declares the tr wrapper: a stdin filter whose operands are the SET1 and
// SET2 character sets.
var spec = clix.Spec{
	Name:     name,
	Summary:  "translate or delete characters",
	Synopsis: synopsis,
	Build:    build,
	Flags: []urf.Flag{
		&urf.BoolFlag{Name: flagDelete, Aliases: []string{"d"}, Usage: "delete characters in SET1, do not translate"},
		&urf.BoolFlag{
			Name:    flagSqueeze,
			Aliases: []string{"s"},
			Usage:   "replace each sequence of a repeated character with a single occurrence",
		},
		&urf.BoolFlag{Name: flagComplement, Aliases: []string{"c"}, Usage: "use the complement of SET1"},
	},
}

// build maps the invocation to tr's pipeline: standard input feeds tr, whose
// SET1 and SET2 operands drive the translation. SET1 is always required; SET2 is
// required only for a plain translate.
func build(inv clix.Invocation) (clix.Source, clix.Command, error) {
	c := inv.Args
	if missingOperands(c) {
		return nil, nil, ErrMissingOperand
	}
	from := command.TrSet(c.Args().Get(0))
	to := command.TrSet(c.Args().Get(1))
	return clix.Stdin(inv.Stdin), command.Tr(from, to, options(c)...), nil
}

// missingOperands reports whether the required SET operands are absent. SET1 is
// always required; SET2 is required only when neither deleting nor squeezing.
func missingOperands(c *urf.Command) bool {
	if c.NArg() == 0 {
		return true
	}
	return c.NArg() < 2 && !c.Bool(flagDelete) && !c.Bool(flagSqueeze)
}

// options folds the parsed flags into tr's option values.
func options(c *urf.Command) []any {
	var opts []any
	if c.Bool(flagDelete) {
		opts = append(opts, command.TrDelete)
	}
	if c.Bool(flagSqueeze) {
		opts = append(opts, command.TrSqueeze)
	}
	if c.Bool(flagComplement) {
		opts = append(opts, command.TrComplement)
	}
	return opts
}

// runMain is an indirection seam so main's wiring is testable without spawning
// the process; a test swaps it and restores it.
var runMain = clix.Main

func main() { runMain(spec, version) }
