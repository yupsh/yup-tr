package main

import (
	"context"
	"fmt"
	"io"

	command "github.com/gloo-foo/cmd-tr"
	gloo "github.com/gloo-foo/framework"
	"github.com/spf13/afero"
	"github.com/urfave/cli/v3"
)

// Error is the sentinel error type for the yup-tr wrapper. It lets every
// error path this package raises be matched with errors.Is.
type Error string

func (e Error) Error() string { return string(e) }

// ErrMissingOperand is returned when the required SET operands are absent:
// SET1 is always required, and SET2 is required unless deleting (-d).
const ErrMissingOperand Error = "missing operand"

const (
	flagDelete     = "delete"
	flagSqueeze    = "squeeze-repeats"
	flagComplement = "complement"
)

// usageText is the command's multi-line usage synopsis, shown in --help.
// cli/v3 indents the whole block by 3 spaces, so these lines are flush-left to
// stay aligned in the rendered output.
const usageText = `tr [OPTIONS] SET1 [SET2]

Translate, squeeze, and/or delete characters from standard
input, writing to standard output.`

// init replaces urfave/cli's default --version/-v flag with a --version-only
// flag, freeing the single-letter -v for command flags while still exposing
// the injected build version.
func init() {
	cli.VersionFlag = &cli.BoolFlag{Name: "version", Usage: "print version information and exit"}
}

// run builds and executes the tr CLI against the injected version and I/O,
// returning the process exit code. tr reads only standard input, so the
// filesystem is unused.
func run(version string, args []string, stdin io.Reader, stdout, stderr io.Writer, _ afero.Fs) int {
	cmd := newApp(version, stdin, stdout)
	cmd.Writer = stdout
	cmd.ErrWriter = stderr
	if err := cmd.Run(context.Background(), args); err != nil {
		_, _ = fmt.Fprintf(stderr, "tr: %v\n", err)
		return 1
	}
	return 0
}

func newApp(version string, stdin io.Reader, stdout io.Writer) *cli.Command {
	return &cli.Command{
		Name:            "tr",
		Version:         version,
		Usage:           "translate or delete characters",
		UsageText:       usageText,
		HideHelpCommand: true,
		// Keep exit handling in run() rather than letting urfave/cli call
		// os.Exit, so the exit code stays testable.
		ExitErrHandler: func(context.Context, *cli.Command, error) {},
		Flags:          flags(),
		Action:         action(stdin, stdout),
	}
}

func flags() []cli.Flag {
	return []cli.Flag{
		&cli.BoolFlag{Name: flagDelete, Aliases: []string{"d"}, Usage: "delete characters in SET1, do not translate"},
		&cli.BoolFlag{
			Name:    flagSqueeze,
			Aliases: []string{"s"},
			Usage:   "replace each sequence of a repeated character with a single occurrence",
		},
		&cli.BoolFlag{Name: flagComplement, Aliases: []string{"c"}, Usage: "use the complement of SET1"},
	}
}

func action(stdin io.Reader, stdout io.Writer) cli.ActionFunc {
	return func(_ context.Context, c *cli.Command) error {
		set1, set2, err := operands(c)
		if err != nil {
			return err
		}
		cmd := command.Tr(set1, set2, options(c)...)
		_, err = gloo.Run(gloo.ByteReaderSource([]io.Reader{stdin}), gloo.ByteWriteTo(stdout), cmd)
		return err
	}
}

// operands extracts SET1 and SET2 from the positional arguments. SET1 is always
// required. SET2 is required only for a plain translate; deleting (-d) or
// squeezing (-s) operate on SET1 alone, matching GNU tr.
func operands(c *cli.Command) (string, string, error) {
	if c.NArg() == 0 {
		return "", "", ErrMissingOperand
	}
	if c.NArg() < 2 && !c.Bool(flagDelete) && !c.Bool(flagSqueeze) {
		return "", "", ErrMissingOperand
	}
	return c.Args().Get(0), c.Args().Get(1), nil
}

// flagOption pairs a CLI flag name with the library option it enables.
type flagOption struct {
	name   string
	option any
}

func flagOptions() []flagOption {
	return []flagOption{
		{flagDelete, command.TrDelete},
		{flagSqueeze, command.TrSqueeze},
		{flagComplement, command.TrComplement},
	}
}

func options(c *cli.Command) []any {
	var opts []any
	for _, fo := range flagOptions() {
		if c.Bool(fo.name) {
			opts = append(opts, fo.option)
		}
	}
	return opts
}
