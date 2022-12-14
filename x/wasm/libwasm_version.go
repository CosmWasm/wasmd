//go:build cgo

package wasm

import (
	"fmt"
	"strings"

	wasmvm "github.com/CosmWasm/wasmvm"
	"github.com/spf13/cobra"
)

func checkLibwasmVersion(cmd *cobra.Command, args []string) error {
	wasmVersion, err := wasmvm.LibwasmvmVersion()
	if err != nil {
		return fmt.Errorf("unable to retrieve libwasmversion %w", err)
	}
	wasmExpectedVersion := getExpectedLibwasmVersion()
	if wasmExpectedVersion == "" {
		return fmt.Errorf("wasmvm module not exist")
	}
	if !strings.Contains(wasmExpectedVersion, wasmVersion) {
		return fmt.Errorf("libwasmversion mismatch. got: %s; expected: %s", wasmVersion, wasmExpectedVersion)
	}
	return nil
}
