package main

import (
	"context"
	"errors"
	"strings"
	"testing"

	clix "github.com/gloo-foo/cli"
	"github.com/spf13/afero"
	urf "github.com/urfave/cli/v3"
)

// parse runs args through a bare command carrying the wrapper's flags and
// returns the parsed accessor, so flag-dependent helpers are tested against real
// parsed flags.
func parse(t *testing.T, args ...string) *urf.Command {
	t.Helper()
	var got *urf.Command
	app := &urf.Command{
		Name:   name,
		Flags:  spec.Flags,
		Action: func(_ context.Context, c *urf.Command) error { got = c; return nil },
	}
	if err := app.Run(context.Background(), args); err != nil {
		t.Fatalf("parse: %v", err)
	}
	return got
}

func invocation(t *testing.T, args ...string) clix.Invocation {
	return clix.Invocation{Args: parse(t, args...), Stdin: strings.NewReader(""), Fs: afero.NewMemMapFs()}
}

func TestOptions(t *testing.T) {
	cases := []struct {
		name string
		args []string
		want int
	}{
		{"none", []string{name, "a", "b"}, 0},
		{"delete", []string{name, "-d", "a"}, 1},
		{"squeeze", []string{name, "-s", "a"}, 1},
		{"complement", []string{name, "-c", "a", "b"}, 1},
		{"all", []string{name, "-d", "-s", "-c", "a"}, 3},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := len(options(parse(t, tc.args...))); got != tc.want {
				t.Fatalf("options len=%d, want %d", got, tc.want)
			}
		})
	}
}

func TestBuild_MissingOperands(t *testing.T) {
	// no SET at all; and only SET1 while a plain translate needs SET2.
	cases := [][]string{{name}, {name, "abc"}}
	for _, args := range cases {
		src, filter, err := build(invocation(t, args...))
		if !errors.Is(err, ErrMissingOperand) {
			t.Fatalf("args=%v: err=%v, want ErrMissingOperand", args, err)
		}
		if src != nil || filter != nil {
			t.Fatalf("args=%v: src=%v filter=%v, want both nil", args, src, filter)
		}
	}
}

func TestErrMissingOperand_Message(t *testing.T) {
	if got := ErrMissingOperand.Error(); got != string(ErrMissingOperand) {
		t.Fatalf("message=%q, want %q", got, string(ErrMissingOperand))
	}
}

func TestBuild_TranslateAndSingleSet(t *testing.T) {
	cases := [][]string{
		{name, "abc", "xyz"}, // plain translate
		{name, "-d", "abc"},  // delete needs only SET1
		{name, "-s", "abc"},  // squeeze needs only SET1
	}
	for _, args := range cases {
		src, filter, err := build(invocation(t, args...))
		if err != nil || src == nil || filter == nil {
			t.Fatalf("args=%v: src=%v filter=%v err=%v", args, src, filter, err)
		}
	}
}

func Test_main(t *testing.T) {
	orig := runMain
	t.Cleanup(func() { runMain = orig })
	var gotName clix.Name
	runMain = func(s clix.Spec, _ clix.Version) { gotName = s.Name }
	main()
	if gotName != name {
		t.Fatalf("main used spec %q, want %s", gotName, name)
	}
}
