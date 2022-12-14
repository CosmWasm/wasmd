//go:build cgo

package cli

import (
	"fmt"

	wasmvm "github.com/CosmWasm/wasmvm"
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
			version, err := wasmvm.LibwasmvmVersion()
			if err != nil {
				return fmt.Errorf("error retrieving libwasmvm version: %w", err)
			}
			fmt.Println(version)
			return nil
		},
	}
	return cmd
}
