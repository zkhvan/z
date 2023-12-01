package main

import (
	"fmt"
	"os"

	"github.com/zkhvan/z/internal/build"
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

	fmt.Printf("date: %q, version: %q", buildDate, buildVersion)

	return exitOK
}
