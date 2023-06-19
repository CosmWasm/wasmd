package main

import (
	"cosmossdk.io/log"
	"github.com/CosmWasm/wasmd/app"
	svrcmd "github.com/cosmos/cosmos-sdk/server/cmd"
	"os"
)

func main() {
	rootCmd := NewRootCmd()

	if err := svrcmd.Execute(rootCmd, "", app.DefaultNodeHome); err != nil {
		log.NewLogger(rootCmd.OutOrStderr()).Error("failure when running app", "err", err)
		os.Exit(1)
	}
}
