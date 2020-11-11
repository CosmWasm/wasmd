package main_test

import (
	"fmt"
	"testing"

	wasmd "github.com/CosmWasm/wasmd/cmd/wasmd"
	"github.com/stretchr/testify/require"

	"github.com/cosmos/cosmos-sdk/x/genutil/client/cli"
)

func TestInitCmd(t *testing.T) {
	rootCmd, _ := wasmd.NewRootCmd()
	rootCmd.SetArgs([]string{
		"init",        // Test the init cmd
		"simapp-test", // Moniker
		fmt.Sprintf("--%s=%s", cli.FlagOverwrite, "true"), // Overwrite genesis.json, in case it already exists
	})

	err := wasmd.Execute(rootCmd)
	require.NoError(t, err)
}
