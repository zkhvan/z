package git

import (
	"bytes"
	"fmt"

	"github.com/zkhvan/z/pkg/exec"
)

var defaultExecutor exec.Interface = exec.New()

type Client struct {
	executor exec.Interface
}

func NewClient() *Client {
	return &Client{executor: defaultExecutor}
}

func (c *Client) SetExecutor(executor exec.Interface) *Client {
	c.executor = executor
	return c
}

// runCombined keeps git's own diagnostics in the error, which is what makes
// fail-loud failures actionable for the user.
func runCombined(cmd exec.Cmd) error {
	output, err := cmd.CombinedOutput()
	if err == nil {
		return nil
	}

	output = bytes.TrimSpace(output)
	if len(output) > 0 {
		return fmt.Errorf("error running command %q: %w: %s", cmd.String(), err, output)
	}
	return fmt.Errorf("error running command %q: %w", cmd.String(), err)
}
