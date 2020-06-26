package cli

import (
	"bufio"
	"strconv"

	"github.com/CosmWasm/wasmd/x/wasm/internal/types"
	"github.com/cosmos/cosmos-sdk/client/context"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/cosmos-sdk/x/auth"
	"github.com/cosmos/cosmos-sdk/x/auth/client/utils"
	"github.com/spf13/cobra"
)

// MigrateContractCmd will migrate a contract to a new code version
func MigrateContractCmd(cdc *codec.Codec) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "migrate [contract_addr_bech32] [new_code_id_int64] [json_encoded_migration_args]",
		Short: "Migrate a wasm contract to a new code version",
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			inBuf := bufio.NewReader(cmd.InOrStdin())
			txBldr := auth.NewTxBuilderFromCLI(inBuf).WithTxEncoder(utils.GetTxEncoder(cdc))
			cliCtx := context.NewCLIContextWithInput(inBuf).WithCodec(cdc)

			contractAddr, err := sdk.AccAddressFromBech32(args[0])
			if err != nil {
				return sdkerrors.Wrap(err, "contract")
			}

			// get the id of the code to instantiate
			codeID, err := strconv.ParseUint(args[1], 10, 64)
			if err != nil {
				return sdkerrors.Wrap(err, "code id")
			}

			migrateMsg := args[2]

			msg := types.MsgMigrateContract{
				Sender:     cliCtx.GetFromAddress(),
				Contract:   contractAddr,
				Code:       codeID,
				MigrateMsg: []byte(migrateMsg),
			}
			return utils.GenerateOrBroadcastMsgs(cliCtx, txBldr, []sdk.Msg{msg})
		},
	}
	return cmd
}

// UpdateContractAdminCmd sets an new admin for a contract
func UpdateContractAdminCmd(cdc *codec.Codec) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "set-contract-admin [contract_addr_bech32] [new_admin_addr_bech32]",
		Short: "Set new admin for a contract",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			inBuf := bufio.NewReader(cmd.InOrStdin())
			txBldr := auth.NewTxBuilderFromCLI(inBuf).WithTxEncoder(utils.GetTxEncoder(cdc))
			cliCtx := context.NewCLIContextWithInput(inBuf).WithCodec(cdc)

			contractAddr, err := sdk.AccAddressFromBech32(args[0])
			if err != nil {
				return sdkerrors.Wrap(err, "contract")
			}
			newAdmin, err := sdk.AccAddressFromBech32(args[1])
			if err != nil {
				return sdkerrors.Wrap(err, "new admin")
			}

			msg := types.MsgUpdateAdmin{
				Sender:   cliCtx.GetFromAddress(),
				Contract: contractAddr,
				NewAdmin: newAdmin,
			}
			return utils.GenerateOrBroadcastMsgs(cliCtx, txBldr, []sdk.Msg{msg})
		},
	}
	return cmd
}

// ClearContractAdminCmd clears an admin for a contract
func ClearContractAdminCmd(cdc *codec.Codec) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "clear-contract-admin [contract_addr_bech32]",
		Short: "Clears admin for a contract to prevent further migrations",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			inBuf := bufio.NewReader(cmd.InOrStdin())
			txBldr := auth.NewTxBuilderFromCLI(inBuf).WithTxEncoder(utils.GetTxEncoder(cdc))
			cliCtx := context.NewCLIContextWithInput(inBuf).WithCodec(cdc)

			contractAddr, err := sdk.AccAddressFromBech32(args[0])
			if err != nil {
				return sdkerrors.Wrap(err, "contract")
			}

			msg := types.MsgClearAdmin{
				Sender:   cliCtx.GetFromAddress(),
				Contract: contractAddr,
			}
			return utils.GenerateOrBroadcastMsgs(cliCtx, txBldr, []sdk.Msg{msg})
		},
	}
	return cmd
}
