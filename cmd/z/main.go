package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/zkhvan/z/internal/build"
	"github.com/zkhvan/z/pkg/cmd"
	"github.com/zkhvan/z/pkg/cmdutil"
	"github.com/zkhvan/z/pkg/factory"
	"github.com/zkhvan/z/pkg/signal"
)

type exitCode int

const (
	exitOK     exitCode = 0
	exitErr    exitCode = 1
	exitCancel exitCode = 2
)

func main() {
	code := run()
	os.Exit(int(code))
}

func run() exitCode {
	buildDate := build.Date
	buildVersion := build.Version

	f, err := factory.New(buildVersion)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return exitErr
	}
	stderr := f.IOStreams.ErrOut

	rootCmd, err := cmd.NewCmdRoot(f, buildVersion, buildDate)
	if err != nil {
		fmt.Fprintf(stderr, "failed to create root command: %s\n", err)
		return exitErr
	}

	if _, err := rootCmd.ExecuteContextC(signal.Notify()); err != nil {
		// A runtime verb surfaces its script's exit code verbatim; the script's own
		// output already said everything, so print no banner.
		var codeErr cmdutil.ExitCodeError
		if errors.As(err, &codeErr) {
			return exitCode(codeErr.Code)
		}
		fmt.Fprintln(stderr, err)
		return exitErr
	}

	return exitOK
}
