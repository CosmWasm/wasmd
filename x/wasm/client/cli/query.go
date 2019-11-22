package cli

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/context"
	"github.com/cosmos/cosmos-sdk/codec"

	"github.com/cosmwasm/wasmd/x/wasm/internal/keeper"
	"github.com/cosmwasm/wasmd/x/wasm/internal/types"
)

func GetQueryCmd(cdc *codec.Codec) *cobra.Command {
	queryCmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "Querying commands for the wasm module",
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}
	queryCmd.AddCommand(client.GetCommands(
		GetCmdListCode(cdc),
		GetCmdQueryCode(cdc),
	)...)
	return queryCmd
}

// GetCmdListCode lists all wasm code uploaded
func GetCmdListCode(cdc *codec.Codec) *cobra.Command {
	return &cobra.Command{
		Use:   "list-code",
		Short: "List all wasm bytecode on the chain",
		Long:  "List all wasm bytecode on the chain",
		Args:  cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			cliCtx := context.NewCLIContext().WithCodec(cdc)

			route := fmt.Sprintf("custom/%s/%s", types.QuerierRoute, keeper.QueryListCode)
			res, _, err := cliCtx.Query(route)
			if err != nil {
				return err
			}
			fmt.Println(res)
			return nil
		},
	}
}

// GetCmdQueryCode returns the bytecode for a given contract
func GetCmdQueryCode(cdc *codec.Codec) *cobra.Command {
	return &cobra.Command{
		Use:   "code [code_id] [output filename]",
		Short: "Downloads wasm bytecode for given code id",
		Long:  "Downloads wasm bytecode for given code id",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			cliCtx := context.NewCLIContext().WithCodec(cdc)

			codeID, err := strconv.ParseUint(args[0], 10, 64)
			if err != nil {
				return err
			}

			route := fmt.Sprintf("custom/%s/%s/%d", types.QuerierRoute, keeper.QueryGetCode, codeID)
			res, _, err := cliCtx.Query(route)
			if err != nil {
				return err
			}

			if len(res) == 0 {
				return fmt.Errorf("contract not found")
			}
			var code keeper.GetCodeResponse
			err = json.Unmarshal(res, &code)
			if err != nil {
				return err
			}

			fmt.Printf("Writing wasm code to %s\n", args[1])
			return ioutil.WriteFile(args[1], code.Code, 0644)
		},
	}
}
