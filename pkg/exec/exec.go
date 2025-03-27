package exec

import (
	"context"
	"io"
	osexec "os/exec"
	"syscall"
	"time"
)

type Interface interface {
	// Command returns a Cmd instance which can be used to run a command.
	// This follows the pattern of package os/exec.
	Command(name string, arg ...string) Cmd

	// CommandContext returns a Cmd instance which can be used to run a command.
	CommandContext(ctx context.Context, name string, arg ...string) Cmd
}

// Cmd is an interface that mimics the os/exec.Cmd type. As more functionality
// is needed, this interface will be expanded.
//
// Since os/exec.Cmd is a struct, we need to replace fields with get/set
// methods.
type Cmd interface {
	// Run runs the command and waits for it to complete.
	Run() error

	// CombinedOutput runs the command and returns its combined output.
	CombinedOutput() ([]byte, error)

	// Output runs the command and returns its output.
	Output() ([]byte, error)

	// Start starts the command but does not wait for it to complete.
	Start() error

	// Wait waits for the command to complete and returns the exit code.
	Wait() error

	// Stops the command by sending SIGTERM and then SIGKILL after a timeout (10
	// seconds).
	//
	// If the command does not terminate after the timeout, it will be killed
	// with SIGKILL.
	Stop()

	// SetDir sets the working directory of the command.
	SetDir(dir string)

	// SetEnv sets the environment variables of the command.
	SetEnv(env []string)

	SetStdin(in io.Reader)
	SetStdout(out io.Writer)
	SetStderr(out io.Writer)

	// StdoutPipe returns a pipe that will be connected to the command's
	// standard output when the command starts.
	StdoutPipe() (io.ReadCloser, error)

	// StderrPipe returns a pipe that will be connected to the command's
	// standard error when the command starts.
	StderrPipe() (io.ReadCloser, error)

	// String returns a human-readable description of the command.
	String() string
}

type executor struct{}

func New() Interface {
	return &executor{}
}

// Command implements Interface.
func (e *executor) Command(name string, arg ...string) Cmd {
	return (*cmdWrapper)(osexec.Command(name, arg...))
}

// CommandContext implements Interface.
func (e *executor) CommandContext(ctx context.Context, name string, arg ...string) Cmd {
	return (*cmdWrapper)(osexec.CommandContext(ctx, name, arg...))
}

type cmdWrapper osexec.Cmd

var _ Cmd = (*cmdWrapper)(nil)

func (cmd *cmdWrapper) CombinedOutput() ([]byte, error) {
	return (*osexec.Cmd)(cmd).CombinedOutput()
}

func (cmd *cmdWrapper) Output() ([]byte, error) {
	return (*osexec.Cmd)(cmd).Output()
}

func (cmd *cmdWrapper) Run() error {
	return (*osexec.Cmd)(cmd).Run()
}

func (cmd *cmdWrapper) SetDir(dir string) {
	cmd.Dir = dir
}

func (cmd *cmdWrapper) SetEnv(env []string) {
	cmd.Env = env
}

func (cmd *cmdWrapper) SetStderr(out io.Writer) {
	cmd.Stderr = out
}

func (cmd *cmdWrapper) SetStdin(in io.Reader) {
	cmd.Stdin = in
}

func (cmd *cmdWrapper) SetStdout(out io.Writer) {
	cmd.Stdout = out
}

func (cmd *cmdWrapper) Start() error {
	return (*osexec.Cmd)(cmd).Start()
}

func (cmd *cmdWrapper) StderrPipe() (io.ReadCloser, error) {
	return (*osexec.Cmd)(cmd).StderrPipe()
}

func (cmd *cmdWrapper) StdoutPipe() (io.ReadCloser, error) {
	return (*osexec.Cmd)(cmd).StdoutPipe()
}

func (cmd *cmdWrapper) Stop() {
	c := (*osexec.Cmd)(cmd)

	if c.Process == nil {
		return
	}

	_ = c.Process.Signal(syscall.SIGTERM)

	time.AfterFunc(10*time.Second, func() {
		if !c.ProcessState.Exited() {
			_ = c.Process.Signal(syscall.SIGKILL)
		}
	})
}

func (cmd *cmdWrapper) Wait() error {
	return (*osexec.Cmd)(cmd).Wait()
}

func (cmd *cmdWrapper) String() string {
	return (*osexec.Cmd)(cmd).String()
}
