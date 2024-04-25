package tmux

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

type Session struct {
	ID       string
	Attached int
	Name     string
}

func ListSessions(ctx context.Context) ([]Session, error) {
	cmd := exec.CommandContext(ctx,
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

	return sessions, nil
}
