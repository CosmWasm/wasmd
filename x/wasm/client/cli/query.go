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
	sdk "github.com/cosmos/cosmos-sdk/types"

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
		GetCmdListContracts(cdc),
		GetCmdGetContractInfo(cdc),
		GetCmdGetContractState(cdc),
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
			fmt.Println(string(res))
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

			fmt.Printf("Downloading wasm code to %s\n", args[1])
			return ioutil.WriteFile(args[1], code.Code, 0644)
		},
	}
}

// GetCmdListContracts lists all instantiated contracts
func GetCmdListContracts(cdc *codec.Codec) *cobra.Command {
	return &cobra.Command{
		Use:   "list-contracts",
		Short: "List addresses of all instantiated contracts on the chain",
		Long:  "List addresses of all instantiated contracts on the chain",
		Args:  cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			cliCtx := context.NewCLIContext().WithCodec(cdc)

			route := fmt.Sprintf("custom/%s/%s", types.QuerierRoute, keeper.QueryListContracts)
			res, _, err := cliCtx.Query(route)
			if err != nil {
				return err
			}
			fmt.Println(string(res))
			return nil
		},
	}
}

// GetCmdGetContractInfo gets details about a given contract
func GetCmdGetContractInfo(cdc *codec.Codec) *cobra.Command {
	return &cobra.Command{
		Use:   "contract [bech32_address]",
		Short: "Prints out metadata of a contract given its address",
		Long:  "Prints out metadata of a contract given its address",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cliCtx := context.NewCLIContext().WithCodec(cdc)

			addr, err := sdk.AccAddressFromBech32(args[0])
			if err != nil {
				return err
			}

			route := fmt.Sprintf("custom/%s/%s/%s", types.QuerierRoute, keeper.QueryGetContract, addr.String())
			res, _, err := cliCtx.Query(route)
			if err != nil {
				return err
			}
			fmt.Println(string(res))
			return nil
		},
	}
}

// GetCmdGetContractState dumps full internal state of a given contract
func GetCmdGetContractState(cdc *codec.Codec) *cobra.Command {
	return &cobra.Command{
		Use:   "contract-state [bech32_address]",
		Short: "Prints out internal state of a contract given its address",
		Long:  "Prints out internal state of a contract given its address",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cliCtx := context.NewCLIContext().WithCodec(cdc)

			addr, err := sdk.AccAddressFromBech32(args[0])
			if err != nil {
				return err
			}

			route := fmt.Sprintf("custom/%s/%s/%s", types.QuerierRoute, keeper.QueryGetContractState, addr.String())
			res, _, err := cliCtx.Query(route)
			if err != nil {
				return err
			}
			fmt.Println(string(res))
			return nil
		},
	}
}
