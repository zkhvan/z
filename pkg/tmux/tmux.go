package tmux

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strconv"
	"strings"

	"github.com/samber/lo"
)

type Session struct {
	ID       string
	Attached int
	Name     string
}

func CurrentSessionID(ctx context.Context) (string, error) {
	cmd := exec.CommandContext(
		ctx,
		"tmux",
		"display-message",
		"-p", "#{session_id}",
	)

	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf(
			"error running %q: %w",
			cmd.String(),
			err,
		)
	}
	out = bytes.TrimSpace(out)

	return string(out), nil
}

// SwitchClient will switch the current tmux client to a different session.
//
// session.ID is preferred, but will use session.Name if session.ID is
// empty. If neither are specified, it will throw an error.
func SwitchClient(ctx context.Context, session Session) error {
	target := session.ID
	if len(target) == 0 {
		target = session.Name
	}
	if len(target) == 0 {
		return fmt.Errorf("invalid session")
	}

	cmd := exec.CommandContext(
		ctx,
		"tmux",
		"switch-client",
		"-t", target,
	)

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("error running %q: %w", cmd.String(), err)
	}

	return nil
}

func SwitchClientLast(ctx context.Context) error {
	cmd := exec.CommandContext(
		ctx,
		"tmux",
		"switch-client",
		"-l",
	)

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("error running %q: %w", cmd.String(), err)
	}

	return nil
}

type ListOptions struct {
	// ExcludeCurrentSession will filter out the currently active session. If
	// no session is attached, it will filter out the last active session.
	ExcludeCurrentSession bool
	// ExcludePopupSessions will filter out popup sessions (those with the
	// _popup_ prefix).
	ExcludePopupSessions bool
}

func ListSessions(ctx context.Context, opts *ListOptions) ([]Session, error) {
	if opts == nil {
		opts = &ListOptions{}
	}

	cmd := exec.CommandContext(
		ctx,
		"tmux",
		"list-sessions",
		"-F", "#{session_id}:#{session_attached}:#{session_name}",
	)

	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf(
			"error running command %q: %w",
			cmd.String(),
			err,
		)
	}

	var sessions []Session

	scanner := bufio.NewScanner(bytes.NewBuffer(out))
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Split(line, ":")

		// Expect the correct amount of parts based on the provided
		// session_format
		if len(parts) != 3 {
			return nil, fmt.Errorf("error parsing session_format: %q", line)
		}

		attached, err := strconv.Atoi(parts[1])
		if err != nil {
			return nil, fmt.Errorf("error parsing session_attached: %q", parts[1])
		}

		session := Session{
			ID:       parts[0],
			Attached: attached,
			Name:     parts[2],
		}

		sessions = append(sessions, session)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error scanning: %w", err)
	}

	if opts.ExcludePopupSessions {
		sessions = lo.Reject(sessions, func(s Session, _ int) bool {
			return IsPopupSession(s.Name)
		})
	}

	if opts.ExcludeCurrentSession {
		// If there's an error getting the current session, just ignore it.
		if currentSessionID, err := CurrentSessionID(ctx); err == nil {
			sessions = lo.Reject(sessions, func(s Session, _ int) bool {
				return s.ID == currentSessionID
			})
		}
	}

	return sessions, nil
}

type NewOptions struct {
	Name string
	Dir  string
}

func NewSession(ctx context.Context, opts *NewOptions) error {
	if opts == nil {
		opts = &NewOptions{}
	}

	cmd := exec.CommandContext(
		ctx,
		"tmux",
		"new-session",
		// Creates the session in the background. This prevents nested tmux
		// sessions.
		"-d",
		// Print the newly created session_id
		"-P", "-F", "#{session_id}",
	)

	if len(opts.Name) > 0 {
		cmd.Args = append(cmd.Args, "-s", opts.Name)
	}

	if len(opts.Dir) > 0 {
		cmd.Args = append(cmd.Args, "-c", opts.Dir)
	}

	output, err := cmd.Output()
	if err != nil {
		var exitError *exec.ExitError
		if errors.As(err, &exitError) {
			if !bytes.HasPrefix(exitError.Stderr, []byte("duplicate session")) {
				return fmt.Errorf("error running %q: %w", cmd.String(), err)
			}
		}
	}
	output = bytes.TrimSpace(output)

	session := Session{
		ID:   string(output),
		Name: opts.Name,
	}

	return SwitchClient(ctx, session)
}

func CurrentSessionName(ctx context.Context) (string, error) {
	cmd := exec.CommandContext(
		ctx,
		"tmux",
		"display-message",
		"-p", "#{session_name}",
	)

	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf(
			"error running %q: %w",
			cmd.String(),
			err,
		)
	}
	out = bytes.TrimSpace(out)

	return string(out), nil
}

func HasSession(ctx context.Context, name string) bool {
	cmd := exec.CommandContext(
		ctx,
		"tmux",
		"has-session",
		"-t", name,
	)

	return cmd.Run() == nil
}

// NewSessionDetached creates a new tmux session in the background and returns
// it without switching the current client.
func NewSessionDetached(ctx context.Context, opts *NewOptions) (Session, error) {
	if opts == nil {
		opts = &NewOptions{}
	}

	cmd := exec.CommandContext(
		ctx,
		"tmux",
		"new-session",
		"-d",
		"-P", "-F", "#{session_id}",
	)

	if len(opts.Name) > 0 {
		cmd.Args = append(cmd.Args, "-s", opts.Name)
	}

	if len(opts.Dir) > 0 {
		cmd.Args = append(cmd.Args, "-c", opts.Dir)
	}

	output, err := cmd.Output()
	if err != nil {
		var exitError *exec.ExitError
		if errors.As(err, &exitError) {
			if !bytes.HasPrefix(exitError.Stderr, []byte("duplicate session")) {
				return Session{}, fmt.Errorf("error running %q: %w", cmd.String(), err)
			}
		}
	}
	output = bytes.TrimSpace(output)

	return Session{
		ID:   string(output),
		Name: opts.Name,
	}, nil
}

func SetSessionOption(ctx context.Context, target, key, value string) error {
	cmd := exec.CommandContext(
		ctx,
		"tmux",
		"set-option",
		"-t", target,
		key, value,
	)

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("error running %q: %w", cmd.String(), err)
	}

	return nil
}

func BindKey(ctx context.Context, keyTable, key string, args ...string) error {
	cmdArgs := []string{"bind-key", "-T", keyTable, key}
	cmdArgs = append(cmdArgs, args...)

	cmd := exec.CommandContext(ctx, "tmux", cmdArgs...)

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("error running %q: %w", cmd.String(), err)
	}

	return nil
}

type DisplayPopupOptions struct {
	Width        string
	Height       string
	ShellCommand string
}

func DisplayPopup(ctx context.Context, opts *DisplayPopupOptions) error {
	if opts == nil {
		opts = &DisplayPopupOptions{}
	}

	args := []string{"display-popup", "-E"}

	if len(opts.Width) > 0 {
		args = append(args, "-w", opts.Width)
	}
	if len(opts.Height) > 0 {
		args = append(args, "-h", opts.Height)
	}

	args = append(args, opts.ShellCommand)

	cmd := exec.CommandContext(ctx, "tmux", args...)

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("error running %q: %w", cmd.String(), err)
	}

	return nil
}

func KillSession(ctx context.Context, session Session) error {
	target := session.ID
	if len(target) == 0 {
		target = session.Name
	}
	if len(target) == 0 {
		return fmt.Errorf("invalid session")
	}

	// #nosec G204
	cmd := exec.CommandContext(
		ctx,
		"tmux",
		"kill-session",
		"-t", target,
	)

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("error running %q: %w", cmd.String(), err)
	}

	return nil
}
