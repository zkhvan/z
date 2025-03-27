package cmd

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"

	"github.com/zkhvan/z/pkg/cmdutil"
)

// DefaultPluginHandler implements PluginHandler
type DefaultPluginHandler struct {
	ValidPrefixes []string
}

// NewDefaultPluginHandler instantiates the DefaultPluginHandler with a list
// of given filename prefixes used to identify valid plugin filenames.
func NewDefaultPluginHandler(validPrefixes []string) *DefaultPluginHandler {
	return &DefaultPluginHandler{
		ValidPrefixes: validPrefixes,
	}
}

// Lookup implements PluginHandler
func (h *DefaultPluginHandler) Lookup(filename string) (string, bool) {
	for _, prefix := range h.ValidPrefixes {
		path, err := exec.LookPath(fmt.Sprintf("%s-%s", prefix, filename))
		if shouldSkipOnLookPathErr(err) || len(path) == 0 {
			continue
		}
		return path, true
	}
	return "", false
}

// Execute implements PluginHandler
func (h *DefaultPluginHandler) Execute(executablePath string, cmdArgs, environment []string) error {
	// Windows does not support exec syscall.
	if runtime.GOOS == "windows" {
		cmd := Command(executablePath, cmdArgs...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Stdin = os.Stdin
		cmd.Env = environment
		err := cmd.Run()
		if err == nil {
			os.Exit(0)
		}
		return err
	}

	// invoke cmd binary relaying the environment and args given append
	// executablePath to cmdArgs, as execve will make first argument the
	// "binary name".
	return syscall.Exec(executablePath, append([]string{executablePath}, cmdArgs...), environment)
}

// HandlePluginCommand receives a pluginHandler and command-line arguments and
// attempts to find a plugin executable on the PATH that satisfies the given
// arguments.
func HandlePluginCommand(handler cmdutil.PluginHandler, args []string, exactMatch bool) error {
	var remainingArgs []string // all "non-flag" arguments
	for _, arg := range args {
		if strings.HasPrefix(arg, "-") {
			break
		}
		remainingArgs = append(remainingArgs, strings.ReplaceAll(arg, "-", "_"))
	}

	if len(remainingArgs) == 0 {
		// the length of cmdArgs is at least 1
		return fmt.Errorf("flags cannot be placed before plugin name: %s", args[0])
	}

	foundBinaryPath := ""

	// attempt to find binary, starting at longest possible name with given
	// cmdArgs
	for len(remainingArgs) > 0 {
		path, found := handler.Lookup(strings.Join(remainingArgs, "-"))
		if !found {
			if exactMatch {
				// if exactMatch is true, we shouldn't continue searching with
				// shorter names. this is especially for not searching
				// z-create plugin when z-create-foo plugin is not found.
				break
			}
			remainingArgs = remainingArgs[:len(remainingArgs)-1]
			continue
		}

		foundBinaryPath = path
		break
	}

	if len(foundBinaryPath) == 0 {
		return nil
	}

	// invoke cmd binary relaying the current environment and args given
	if err := handler.Execute(foundBinaryPath, args[len(remainingArgs):], os.Environ()); err != nil {
		return err
	}

	return nil
}

func Command(name string, arg ...string) *exec.Cmd {
	cmd := &exec.Cmd{
		Path: name,
		Args: append([]string{name}, arg...),
	}
	if filepath.Base(name) == name {
		lp, err := exec.LookPath(name)
		if lp != "" && !shouldSkipOnLookPathErr(err) {
			// Update cmd.Path even if err is non-nil. If err is ErrDot
			// (especially on Windows), lp may include a resolved extension
			// (like .exe or .bat) that should be preserved.
			cmd.Path = lp
		}
	}
	return cmd
}

func shouldSkipOnLookPathErr(err error) bool {
	return err != nil && !errors.Is(err, exec.ErrDot)
}
