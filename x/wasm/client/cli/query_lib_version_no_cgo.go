//go:build !cgo

package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

// GetCmdLibVersion gets current libwasmvm version.
func GetCmdLibVersion() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "libwasmvm-version",
		Short:   "Get libwasmvm version",
		Long:    "Get libwasmvm version",
		Aliases: []string{"lib-version"},
		Args:    cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			return fmt.Errorf("not implemented, please build with cgo enabled")
		},
	}
	return cmd
}
