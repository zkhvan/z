package plugin

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/spf13/cobra"
	"github.com/zkhvan/z/pkg/iolib"
)

var (
	ValidPluginFilenamePrefixes = []string{"z"}
)

// pathVerifier receives a path and determines if it is valid or not
type PathVerifier interface {
	// Verify determines if a given path is valid
	Verify(path string) []error
}

type CommandOverrideVerifier struct {
	root        *cobra.Command
	seenPlugins map[string]string
}

// Verify implements PathVerifier and determines if a given path
// is valid depending on whether or not it overwrites an existing
// z command path, or a previously seen plugin.
func (v *CommandOverrideVerifier) Verify(path string) []error {
	if v.root == nil {
		return []error{fmt.Errorf("unable to verify path with nil root")}
	}

	// extract the plugin binary name
	segs := strings.Split(path, "/")
	binName := segs[len(segs)-1]

	cmdPath := strings.Split(binName, "-")
	if len(cmdPath) > 1 {
		// the first argument is always "z" for a plugin binary
		cmdPath = cmdPath[1:]
	}

	errors := []error{}

	if isExec, err := isExecutable(path); err == nil && !isExec {
		errors = append(errors, fmt.Errorf("warning: %s identified as a z plugin, but it is not executable", path))
	} else if err != nil {
		errors = append(errors, fmt.Errorf("error: unable to identify %s as an executable file: %v", path, err))
	}

	if existingPath, ok := v.seenPlugins[binName]; ok {
		errors = append(errors, fmt.Errorf("warning: %s is overshadowed by a similarly named plugin: %s", path, existingPath))
	} else {
		v.seenPlugins[binName] = path
	}

	if cmd, _, err := v.root.Find(cmdPath); err == nil {
		errors = append(errors, fmt.Errorf("warning: %s overwrites existing command: %q", binName, cmd.CommandPath()))
	}

	return errors
}

func isExecutable(fullPath string) (bool, error) {
	info, err := os.Stat(fullPath)
	if err != nil {
		return false, err
	}

	if runtime.GOOS == "windows" {
		fileExt := strings.ToLower(filepath.Ext(fullPath))

		switch fileExt {
		case ".bat", ".cmd", ".com", ".exe", ".ps1":
			return true, nil
		}
		return false, nil
	}

	if m := info.Mode(); !m.IsDir() && m&0111 != 0 {
		return true, nil
	}

	return false, nil
}

// uniquePathsList deduplicates a given slice of strings without
// sorting or otherwise altering its order in any way.
func uniquePathsList(paths []string) []string {
	seen := map[string]bool{}
	newPaths := []string{}
	for _, p := range paths {
		if seen[p] {
			continue
		}
		seen[p] = true
		newPaths = append(newPaths, p)
	}
	return newPaths
}

func hasValidPrefix(filepath string, validPrefixes []string) bool {
	for _, prefix := range validPrefixes {
		if !strings.HasPrefix(filepath, prefix+"-") {
			continue
		}
		return true
	}
	return false
}

type PluginListOptions struct {
	Verifier PathVerifier
	NameOnly bool

	PluginPaths []string

	iolib.IOStreams
}

func (o *PluginListOptions) Complete(cmd *cobra.Command) {
	o.Verifier = &CommandOverrideVerifier{
		root:        cmd.Root(),
		seenPlugins: make(map[string]string),
	}

	o.PluginPaths = filepath.SplitList(os.Getenv("PATH"))
}

func (o *PluginListOptions) Run() error {
	plugins, pluginErrors := o.ListPlugins()

	if len(plugins) > 0 {
		fmt.Fprintf(o.Out, "The following compatible plugins are available:\n\n")
	} else {
		pluginErrors = append(pluginErrors, fmt.Errorf("error: unable to find any z plugins in your PATH"))
	}

	pluginWarnings := 0
	for _, pluginPath := range plugins {
		if o.NameOnly {
			fmt.Fprintf(o.Out, "%s\n", filepath.Base(pluginPath))
		} else {
			fmt.Fprintf(o.Out, "%s\n", pluginPath)
		}
		if errs := o.Verifier.Verify(pluginPath); len(errs) != 0 {
			for _, err := range errs {
				fmt.Fprintf(o.ErrOut, "  - %s\n", err)
				pluginWarnings++
			}
		}
	}

	if pluginWarnings > 0 {
		if pluginWarnings == 1 {
			pluginErrors = append(pluginErrors, fmt.Errorf("error: one plugin warning was found"))
		} else {
			pluginErrors = append(pluginErrors, fmt.Errorf("error: %v plugin warnings were found", pluginWarnings))
		}
	}
	if len(pluginErrors) > 0 {
		errs := bytes.NewBuffer(nil)
		for _, e := range pluginErrors {
			fmt.Fprintln(errs, e)
		}
		return fmt.Errorf("%s", errs.String())
	}

	return nil
}

// ListPlugins returns list of plugin paths.
func (o *PluginListOptions) ListPlugins() ([]string, []error) {
	plugins := []string{}
	errors := []error{}

	for _, dir := range uniquePathsList(o.PluginPaths) {
		if len(strings.TrimSpace(dir)) == 0 {
			continue
		}

		files, err := os.ReadDir(dir)
		if err != nil {
			if _, ok := err.(*os.PathError); ok {
				fmt.Fprintf(o.ErrOut, "Unable to read directory %q from your PATH: %v. Skipping...\n", dir, err)
				continue
			}

			errors = append(errors, fmt.Errorf("error: unable to read directory %q in your PATH: %v", dir, err))
			continue
		}

		for _, f := range files {
			if f.IsDir() {
				continue
			}
			if !hasValidPrefix(f.Name(), ValidPluginFilenamePrefixes) {
				continue
			}

			plugins = append(plugins, filepath.Join(dir, f.Name()))
		}
	}

	return plugins, errors
}
