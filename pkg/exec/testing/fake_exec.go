package testingexec

import (
	"context"
	"fmt"
	"io"
	"strings"
	"sync"

	"github.com/zkhvan/z/pkg/exec"
)

type FakeExec struct {
	CommandScript []FakeCommandAction
	CommandCalls  int

	mu sync.Mutex
}

var _ exec.Interface = &FakeExec{}

// FakeCommandAction is the function to be executed.
type FakeCommandAction func(cmd string, args ...string) exec.Cmd

func (fake *FakeExec) Command(name string, args ...string) exec.Cmd {
	return fake.nextCommand(name, args)
}

func (fake *FakeExec) CommandContext(_ context.Context, name string, args ...string) exec.Cmd {
	return fake.Command(name, args...)
}

func (fake *FakeExec) nextCommand(cmd string, args []string) exec.Cmd {
	fake.mu.Lock()
	defer fake.mu.Unlock()

	if len(fake.CommandScript)-1 < fake.CommandCalls {
		panic("no more commands to execute")
	}

	fakeCmd := fake.CommandScript[fake.CommandCalls](cmd, args...)
	fake.CommandCalls++

	// Check for an exact match of the command and args
	actualArgs := append([]string{cmd}, args...)
	fc, _ := fakeCmd.(*FakeCmd)
	if cmd != fc.Argv[0] {
		panic(fmt.Sprintf("expected command %v, got %v", fc.Argv[0], cmd))
	}
	if len(actualArgs) != len(fc.Argv) {
		panic(fmt.Sprintf("expected %d arguments, got %d", len(fc.Argv), len(actualArgs)))
	}
	for i, a := range actualArgs[1:] {
		if a != fc.Argv[i+1] {
			panic(fmt.Sprintf("expected argument %d to be %v, got %v", i+1, fc.Argv[i+1], a))
		}
	}

	return fakeCmd
}

type FakeCmd struct {
	Argv                  []string
	Dirs                  []string
	CombinedOutputScripts []FakeAction
	CombinedOutputCalls   int
	CombinedOutputLog     [][]string
	OutputScripts         []FakeAction
	OutputCalls           int
	OutputLog             [][]string
	RunScripts            []FakeAction
	RunCalls              int
	RunLog                [][]string
	Stdin                 io.Reader
	Stdout                io.Writer
	Stderr                io.Writer
	Env                   []string
	StderrPipeResponse    FakeStdIOPipeResponse
	StdoutPipeResponse    FakeStdIOPipeResponse
	StartResponse         error
	WaitResponse          error
}

var _ exec.Cmd = &FakeCmd{}

func NewFakeCmd(cmd string, args ...string) *FakeCmd {
	return &FakeCmd{
		Argv: append([]string{cmd}, args...),
	}
}

// FakeStdIOPipeResponse holds responses to use as fakes for the StdoutPipe and
// StderrPipe method calls
type FakeStdIOPipeResponse struct {
	ReadCloser io.ReadCloser
	Error      error
}

type FakeAction func() ([]byte, []byte, error)

func (f *FakeCmd) CombinedOutput() ([]byte, error) {
	if len(f.CombinedOutputScripts)-1 < f.CombinedOutputCalls {
		panic("no more CombinedOutput commands to execute")
	}
	if f.CombinedOutputLog == nil {
		f.CombinedOutputLog = [][]string{}
	}

	fakeAction := f.CombinedOutputScripts[f.CombinedOutputCalls]
	f.CombinedOutputLog = append(f.CombinedOutputLog, append([]string{}, f.Argv...))
	f.CombinedOutputCalls++

	stdout, _, err := fakeAction()
	return stdout, err
}

func (f *FakeCmd) Output() ([]byte, error) {
	if len(f.OutputScripts)-1 < f.OutputCalls {
		panic("no more Output commands to execute")
	}
	if f.OutputLog == nil {
		f.OutputLog = [][]string{}
	}

	fakeAction := f.OutputScripts[f.OutputCalls]
	f.OutputLog = append(f.OutputLog, append([]string{}, f.Argv...))
	f.OutputCalls++

	stdout, _, err := fakeAction()
	return stdout, err
}

func (f *FakeCmd) Run() error {
	if len(f.RunScripts)-1 < f.RunCalls {
		panic("no more Run commands to execute")
	}
	if f.RunLog == nil {
		f.RunLog = [][]string{}
	}

	fakeAction := f.RunScripts[f.RunCalls]
	f.RunLog = append(f.RunLog, append([]string{}, f.Argv...))
	f.RunCalls++

	stdout, stderr, err := fakeAction()
	if stdout != nil {
		_, _ = f.Stdout.Write(stdout)
	}
	if stderr != nil {
		_, _ = f.Stderr.Write(stderr)
	}

	return err
}

func (f *FakeCmd) SetDir(dir string) {
	f.Dirs = append(f.Dirs, dir)
}

func (f *FakeCmd) SetEnv(env []string) {
	f.Env = env
}

func (f *FakeCmd) SetStderr(out io.Writer) {
	f.Stderr = out
}

func (f *FakeCmd) SetStdin(in io.Reader) {
	f.Stdin = in
}

func (f *FakeCmd) SetStdout(out io.Writer) {
	f.Stdout = out
}

func (f *FakeCmd) Start() error {
	return f.StartResponse
}

func (f *FakeCmd) StderrPipe() (io.ReadCloser, error) {
	return f.StderrPipeResponse.ReadCloser, f.StderrPipeResponse.Error
}

func (f *FakeCmd) StdoutPipe() (io.ReadCloser, error) {
	return f.StdoutPipeResponse.ReadCloser, f.StdoutPipeResponse.Error
}

func (f *FakeCmd) Stop() {
	// no-op
}

func (f *FakeCmd) String() string {
	return strings.Join(f.Argv, " ")
}

func (f *FakeCmd) Wait() error {
	return f.WaitResponse
}
