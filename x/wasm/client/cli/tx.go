package client

import (
	"bufio"
	// "strconv"
	"io/ioutil"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/context"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth"
	"github.com/cosmos/cosmos-sdk/x/auth/client/utils"

	"github.com/cosmwasm/wasmd/x/wasm/internal/types"
)

const (
	flagTo     = "to"
	flagAmount = "amount"
)

// GetTxCmd returns the transaction commands for this module
func GetTxCmd(cdc *codec.Codec) *cobra.Command {
	txCmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "Wasm transaction subcommands",
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}
	txCmd.AddCommand(
		StoreCodeCmd(cdc),
		// InstantiateContractCmd(cdc),
		// ExecuteContractCmd(cdc),
	)
	return txCmd
}

// StoreCodeCmd will upload code to be reused.
func StoreCodeCmd(cdc *codec.Codec) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "store [from_key_or_address] [wasm file]",
		Short: "Upload a wasm binary",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			inBuf := bufio.NewReader(cmd.InOrStdin())
			txBldr := auth.NewTxBuilderFromCLI(inBuf).WithTxEncoder(utils.GetTxEncoder(cdc))
			cliCtx := context.NewCLIContextWithInputAndFrom(inBuf, args[0]).WithCodec(cdc)

			// parse coins trying to be sent
			wasm, err := ioutil.ReadFile(args[1])
			if err != nil {
				return err
			}

			// build and sign the transaction, then broadcast to Tendermint
			msg := types.MsgStoreCode{
				Sender:       cliCtx.GetFromAddress(),
				WASMByteCode: wasm,
			}
			return utils.GenerateOrBroadcastMsgs(cliCtx, txBldr, []sdk.Msg{msg})
		},
	}

	cmd = client.PostCommands(cmd)[0]

	return cmd
}

// // InstantiateContractCmd will instantiate a contract from previously uploaded code.
// func InstantiateContractCmd(cdc *codec.Codec) *cobra.Command {
// 	cmd := &cobra.Command{
// 		Use:   "create [from_key_or_address] [code_id_int64] [coins] [json_encoded_init_args]",
// 		Short: "Instantiate a wasm contract",
// 		Args:  cobra.ExactArgs(4),
// 		RunE: func(cmd *cobra.Command, args []string) error {
// 			txBldr := auth.NewTxBuilderFromCLI().WithTxEncoder(utils.GetTxEncoder(cdc))
// 			cliCtx := context.NewCLIContextWithFrom(args[0]).
// 				WithCodec(cdc).
// 				WithAccountDecoder(cdc)

// 			// get the id of the code to instantiate
// 			codeID, err := strconv.Atoi(args[1])
// 			if err != nil {
// 				return err
// 			}

// 			// parse coins trying to be sent
// 			coins, err := sdk.ParseCoins(args[2])
// 			if err != nil {
// 				return err
// 			}

// 			initMsg := args[3]

// 			// build and sign the transaction, then broadcast to Tendermint
// 			msg := MsgCreateContract{
// 				Sender:    cliCtx.GetFromAddress(),
// 				Code:      CodeID(codeID),
// 				InitFunds: coins,
// 				InitMsg:   []byte(initMsg),
// 			}
// 			return utils.GenerateOrBroadcastMsgs(cliCtx, txBldr, []sdk.Msg{msg})
// 		},
// 	}

// 	cmd = client.PostCommands(cmd)[0]

// 	return cmd
// }

// // ExecuteContractCmd will instantiate a contract from previously uploaded code.
// func ExecuteContractCmd(cdc *codec.Codec) *cobra.Command {
// 	cmd := &cobra.Command{
// 		Use:   "send [from_key_or_address] [contract_addr_bech32] [coins] [json_encoded_send_args]",
// 		Short: "Instantiate a wasm contract",
// 		Args:  cobra.ExactArgs(4),
// 		RunE: func(cmd *cobra.Command, args []string) error {
// 			txBldr := auth.NewTxBuilderFromCLI().WithTxEncoder(utils.GetTxEncoder(cdc))
// 			cliCtx := context.NewCLIContextWithFrom(args[0]).
// 				WithCodec(cdc).
// 				WithAccountDecoder(cdc)

// 			// get the id of the code to instantiate
// 			contractAddr, err := sdk.AccAddressFromBech32(args[1])
// 			if err != nil {
// 				return err
// 			}

// 			// parse coins trying to be sent
// 			coins, err := sdk.ParseCoins(args[2])
// 			if err != nil {
// 				return err
// 			}

// 			sendMsg := args[3]

// 			// build and sign the transaction, then broadcast to Tendermint
// 			msg := MsgSendContract{
// 				Sender:   cliCtx.GetFromAddress(),
// 				Contract: contractAddr,
// 				Payment:  coins,
// 				Msg:      []byte(sendMsg),
// 			}
// 			return utils.GenerateOrBroadcastMsgs(cliCtx, txBldr, []sdk.Msg{msg})
// 		},
// 	}

// 	cmd = client.PostCommands(cmd)[0]

// 	return cmd
// }
