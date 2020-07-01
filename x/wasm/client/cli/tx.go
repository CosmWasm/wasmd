package cli

import (
	"fmt"
	"io/ioutil"
	"strconv"

	wasmUtils "github.com/CosmWasm/wasmd/x/wasm/client/utils"
	"github.com/CosmWasm/wasmd/x/wasm/internal/types"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	flagTo      = "to"
	flagAmount  = "amount"
	flagSource  = "source"
	flagBuilder = "builder"
	flagLabel   = "label"
	flagAdmin   = "admin"
	flagNoAdmin = "no-admin"
)

// GetTxCmd returns the transaction commands for this module
func GetTxCmd(cliCtx client.Context) *cobra.Command {
	txCmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "Wasm transaction subcommands",
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}
	txCmd.AddCommand(
		flags.PostCommands(
			StoreCodeCmd(cliCtx),
			InstantiateContractCmd(cliCtx),
			ExecuteContractCmd(cliCtx),
			MigrateContractCmd(cliCtx),
			UpdateContractAdminCmd(cliCtx),
			ClearContractAdminCmd(cliCtx),
		)...,
	)
	return txCmd
}

// StoreCodeCmd will upload code to be reused.
func StoreCodeCmd(clientCtx client.Context) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "store [wasm file] --source [source] --builder [builder]",
		Short: "Upload a wasm binary",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx := clientCtx.InitWithInput(cmd.InOrStdin())

			// parse coins trying to be sent
			wasm, err := ioutil.ReadFile(args[0])
			if err != nil {
				return err
			}

			source := viper.GetString(flagSource)

			builder := viper.GetString(flagBuilder)

			// gzip the wasm file
			if wasmUtils.IsWasm(wasm) {
				wasm, err = wasmUtils.GzipIt(wasm)

				if err != nil {
					return err
				}
			} else if !wasmUtils.IsGzip(wasm) {
				return fmt.Errorf("invalid input file. Use wasm binary or gzip")
			}

			// build and sign the transaction, then broadcast to Tendermint
			msg := types.MsgStoreCode{
				Sender:       clientCtx.GetFromAddress(),
				WASMByteCode: wasm,
				Source:       source,
				Builder:      builder,
			}
			err = msg.ValidateBasic()

			if err != nil {
				return err
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), &msg)
		},
	}

	cmd.Flags().String(flagSource, "", "A valid URI reference to the contract's source code, optional")
	cmd.Flags().String(flagBuilder, "", "A valid docker tag for the build system, optional")
	return cmd
}

// InstantiateContractCmd will instantiate a contract from previously uploaded code.
func InstantiateContractCmd(clientCtx client.Context) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "instantiate [code_id_int64] [json_encoded_init_args]",
		Short: "Instantiate a wasm contract",
		Args:  cobra.RangeArgs(2, 3),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx := clientCtx.InitWithInput(cmd.InOrStdin())

			// get the id of the code to instantiate
			codeID, err := strconv.ParseUint(args[0], 10, 64)
			if err != nil {
				return err
			}

			amounstStr := viper.GetString(flagAmount)
			amount, err := sdk.ParseCoins(amounstStr)
			if err != nil {
				return err
			}

			label := viper.GetString(flagLabel)
			if label == "" {
				return fmt.Errorf("Label is required on all contracts")
			}

			initMsg := args[1]

			adminStr := viper.GetString(flagAdmin)
			var adminAddr sdk.AccAddress
			if len(adminStr) != 0 {
				adminAddr, err = sdk.AccAddressFromBech32(adminStr)
				if err != nil {
					return sdkerrors.Wrap(err, "admin")
				}
			}

			// build and sign the transaction, then broadcast to Tendermint
			msg := types.MsgInstantiateContract{
				Sender:    clientCtx.GetFromAddress(),
				Code:      codeID,
				Label:     label,
				InitFunds: amount,
				InitMsg:   []byte(initMsg),
				Admin:     adminAddr,
			}
			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), &msg)
		},
	}

	cmd.Flags().String(flagAmount, "", "Coins to send to the contract during instantiation")
	cmd.Flags().String(flagLabel, "", "A human-readable name for this contract in lists")
	cmd.Flags().String(flagAdmin, "", "Address of an admin")
	return cmd
}

// ExecuteContractCmd will instantiate a contract from previously uploaded code.
func ExecuteContractCmd(clientCtx client.Context) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "execute [contract_addr_bech32] [json_encoded_send_args]",
		Short: "Execute a command on a wasm contract",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx := clientCtx.InitWithInput(cmd.InOrStdin())

			// get the id of the code to instantiate
			contractAddr, err := sdk.AccAddressFromBech32(args[0])
			if err != nil {
				return err
			}

			amounstStr := viper.GetString(flagAmount)
			amount, err := sdk.ParseCoins(amounstStr)
			if err != nil {
				return err
			}

			execMsg := args[1]

			// build and sign the transaction, then broadcast to Tendermint
			msg := types.MsgExecuteContract{
				Sender:    clientCtx.GetFromAddress(),
				Contract:  contractAddr,
				SentFunds: amount,
				Msg:       []byte(execMsg),
			}
			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), &msg)
		},
	}

	cmd.Flags().String(flagAmount, "", "Coins to send to the contract along with command")
	return cmd
}
