package cli

import (
	"errors"
	"testing"

	tmcmd "github.com/cometbft/cometbft/cmd/cometbft/commands"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

func TestExtendUnsafeResetAllCmdPreservesResetError(t *testing.T) {
	resetError := errors.New("failed to reset consensus state")
	called := false
	reset := &cobra.Command{
		Use: tmcmd.ResetAllCmd.Use,
		RunE: func(*cobra.Command, []string) error {
			called = true
			return resetError
		},
	}
	comet := &cobra.Command{Use: "comet"}
	comet.AddCommand(reset)
	root := &cobra.Command{Use: "wasmd"}
	root.AddCommand(comet)

	ExtendUnsafeResetAllCmd(root)
	require.ErrorIs(t, reset.RunE(reset, nil), resetError)
	require.True(t, called)
}
