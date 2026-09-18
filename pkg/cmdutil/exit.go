package cmdutil

import "fmt"

// ExitCodeError carries a process exit code up to main, which exits with that
// code without printing an error banner. The runtime contract uses it so a
// script stays a transparent shell: its own stdout/stderr is the whole message
// and its exit code becomes z's. A status script that exits 3, or a command run
// through exec that fails, surfaces its real code rather than collapsing to 1.
type ExitCodeError struct {
	Code int
}

func (e ExitCodeError) Error() string {
	return fmt.Sprintf("exit code %d", e.Code)
}
