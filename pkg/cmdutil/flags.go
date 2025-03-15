package cmdutil

import (
	"errors"

	"github.com/spf13/cobra"
)

func MarkFlagsRequired(cmd *cobra.Command, flags ...string) error {
	var errs []error
	for _, flag := range flags {
		if err := cmd.MarkFlagRequired(flag); err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	return nil
}
