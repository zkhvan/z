package fzf

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

var ErrCanceled = errors.New("canceled")

type Binding[T any] struct {
	Key    string
	Action func(item T) error
}

type Options[T any] struct {
	Iterator func(item T, index int) string
	Bindings []Binding[T]
	Header   string
}

type Option[T any] func(*Options[T])

func WithIterator[T any](iterator func(item T, index int) string) Option[T] {
	return func(opts *Options[T]) {
		opts.Iterator = iterator
	}
}

func WithBinding[T any](key string, action func(item T) error) Option[T] {
	return func(opts *Options[T]) {
		opts.Bindings = append(opts.Bindings, Binding[T]{
			Key:    key,
			Action: action,
		})
	}
}

func WithHeader[T any](header string) Option[T] {
	return func(opts *Options[T]) {
		opts.Header = header
	}
}

func One[T any](
	ctx context.Context,
	items []T,
	options ...Option[T],
) (T, error) {
	var t T

	opts := &Options[T]{}
	for _, option := range options {
		option(opts)
	}

	if opts.Iterator == nil {
		opts.Iterator = func(item T, _ int) string {
			return fmt.Sprintf("%+v", item)
		}
	}

	inputs := make([]string, 0, len(items))
	for index, item := range items {
		inputs = append(inputs, opts.Iterator(item, index))
	}

	args := []string{"--bind", "enter:become(echo {+n})"}
	if len(opts.Bindings) > 0 {
		for _, binding := range opts.Bindings {
			args = append(args, "--bind")
			args = append(args, fmt.Sprintf("%s:become(echo {+n} %[1]s)", binding.Key))
		}
	}

	if opts.Header != "" {
		args = append(args, "--header", opts.Header)
	}

	cmd := exec.CommandContext(
		ctx,
		"fzf",
		args...,
	)

	var out bytes.Buffer
	cmd.Stdin = bytes.NewBufferString(strings.Join(inputs, "\n"))
	cmd.Stderr = os.Stderr
	cmd.Stdout = &out

	if err := cmd.Run(); err != nil {
		if cmd.ProcessState.ExitCode() == 130 {
			return t, ErrCanceled
		}

		return t, fmt.Errorf("error running command %q: %w", cmd.String(), err)
	}

	selected := strings.TrimSpace(out.String())
	parts := strings.Split(selected, " ")
	index, err := strconv.Atoi(parts[0])
	if err != nil {
		return t, fmt.Errorf("error parsing zero-based index %q: %w", selected, err)
	}

	if len(parts) > 1 {
		for _, binding := range opts.Bindings {
			if binding.Key == parts[1] {
				return items[index], binding.Action(items[index])
			}
		}
	}

	return items[index], nil
}
