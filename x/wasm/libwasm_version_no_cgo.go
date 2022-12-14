//go:build !cgo

package wasm

import (
	"github.com/spf13/cobra"
)

func checkLibwasmVersion(cmd *cobra.Command, args []string) error {
	panic("not implemented, please build with cgo enabled")
}
