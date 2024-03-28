package main

import (
	"fmt"
	"os"

	"github.com/zkhvan/z/internal/build"
	"github.com/zkhvan/z/pkg/cmd"
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

	f := factory.New(buildVersion)
	stderr := f.IOStreams.ErrOut

	rootCmd, err := cmd.NewCmdRoot(f, buildVersion, buildDate)
	if err != nil {
		fmt.Fprintf(stderr, "failed to create root command: %s\n", err)
		return exitErr
	}

	if _, err := rootCmd.ExecuteContextC(signal.Notify()); err != nil {
		fmt.Fprintln(stderr, err)
		return exitErr
	}

	return exitOK
}
