package fd

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
)

type Options struct {
	Glob        *bool
	Hidden      *bool
	MaxDepth    *int
	NoIgnoreVCS *bool
	Path        *string
	Follow      *bool
}

func Run(ctx context.Context, pattern string, opts *Options) ([]string, error) {
	if opts == nil {
		opts = &Options{}
	}

	cmd := exec.CommandContext(
		ctx,
		"fd",
		pattern,
	)

	if opts.Glob != nil && *opts.Glob {
		cmd.Args = append(cmd.Args, "--glob")
	}

	if opts.Hidden != nil && *opts.Hidden {
		cmd.Args = append(cmd.Args, "--hidden")
	}

	if opts.MaxDepth != nil && *opts.MaxDepth > 0 {
		cmd.Args = append(cmd.Args, fmt.Sprintf("--max-depth=%d", *opts.MaxDepth))
	}

	if opts.NoIgnoreVCS != nil && *opts.NoIgnoreVCS {
		cmd.Args = append(cmd.Args, "--no-ignore-vcs")
	}

	if opts.Follow != nil && *opts.Follow {
		cmd.Args = append(cmd.Args, "--follow")
	}

	// If a path is specified, it is required to be after the pattern since
	// it's a positional argument.
	if opts.Path != nil && len(*opts.Path) > 0 {
		cmd.Args = append(cmd.Args, *opts.Path)
	}

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("error running command %q: %w", cmd.String(), err)
	}

	output = bytes.TrimSpace(output)

	var results []string
	for _, line := range bytes.Split(output, []byte("\n")) {
		results = append(results, string(line))
	}

	return results, nil
}
