package main

import (
	"os"

	"github.com/CosmWasm/wasmd/app"
	"github.com/cosmos/cosmos-sdk/server"
)

func main() {
	rootCmd, _ := NewRootCmd()

	if err := Execute(rootCmd, app.DefaultNodeHome); err != nil {
		switch e := err.(type) {
		case server.ErrorCode:
			os.Exit(e.Code)

		default:
			os.Exit(1)
		}
	}
}
