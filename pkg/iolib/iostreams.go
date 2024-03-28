package iolib

import (
	"io"
)

type IOStreams struct {
	In     io.Reader // think os.Stdin
	Out    io.Writer // think os.Stdout
	ErrOut io.Writer // think os.Stderr
}
