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

var (
	ErrCancelled = errors.New("cancelled")
)

func One[T any](
	ctx context.Context,
	items []T,
	iteratee func(item T, index int) string,
) (T, error) {
	var t T

	inputs := make([]string, 0, len(items))
	for index, item := range items {
		inputs = append(inputs, iteratee(item, index))
	}

	cmd := exec.CommandContext(
		ctx,
		"fzf",
		"--bind", "enter:become(echo {+n})",
	)

	var out bytes.Buffer
	cmd.Stdin = bytes.NewBufferString(strings.Join(inputs, "\n"))
	cmd.Stderr = os.Stderr
	cmd.Stdout = &out

	if err := cmd.Run(); err != nil {
		if cmd.ProcessState.ExitCode() == 130 {
			return t, ErrCancelled
		}

		return t, fmt.Errorf("error running command %q: %w", cmd.String(), err)
	}

	selected := strings.TrimSpace(out.String())
	index, err := strconv.Atoi(selected)
	if err != nil {
		return t, fmt.Errorf("error parsing zero-based index %q: %w", selected, err)
	}

	return items[index], nil
}
