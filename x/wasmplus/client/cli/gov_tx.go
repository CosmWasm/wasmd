package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/line/lbm-sdk/client"
	"github.com/line/lbm-sdk/client/tx"
	sdk "github.com/line/lbm-sdk/types"
	"github.com/line/lbm-sdk/x/gov/client/cli"
	govtypes "github.com/line/lbm-sdk/x/gov/types"

	"github.com/line/wasmd/x/wasmplus/types"
)

func ProposalDeactivateContractCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "deactivate-contract [contract_addr_bech32]",
		Short: "Deactivate the contract. This contract will not be executed after that.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			proposalTitle, err := cmd.Flags().GetString(cli.FlagTitle)
			if err != nil {
				return fmt.Errorf("proposal title: %s", err)
			}
			proposalDescr, err := cmd.Flags().GetString(cli.FlagDescription)
			if err != nil {
				return fmt.Errorf("proposal description: %s", err)
			}
			depositArg, err := cmd.Flags().GetString(cli.FlagDeposit)
			if err != nil {
				return fmt.Errorf("deposit: %s", err)
			}
			deposit, err := sdk.ParseCoinsNormalized(depositArg)
			if err != nil {
				return err
			}

			content := types.DeactivateContractProposal{
				Title:       proposalTitle,
				Description: proposalDescr,
				Contract:    args[0],
			}

			msg, err := govtypes.NewMsgSubmitProposal(&content, deposit, clientCtx.GetFromAddress())
			if err != nil {
				return err
			}
			if err = msg.ValidateBasic(); err != nil {
				return err
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	cmd.Flags().String(cli.FlagTitle, "", "Title of proposal")
	cmd.Flags().String(cli.FlagDescription, "", "Description of proposal")
	cmd.Flags().String(cli.FlagDeposit, "", "Deposit of proposal")

	return cmd
}

func ProposalActivateContractCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "activate-contract [contract_addr_bech32]",
		Short: "Activate the inactive contract. This contract will be executed after that.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			proposalTitle, err := cmd.Flags().GetString(cli.FlagTitle)
			if err != nil {
				return fmt.Errorf("proposal title: %s", err)
			}
			proposalDescr, err := cmd.Flags().GetString(cli.FlagDescription)
			if err != nil {
				return fmt.Errorf("proposal description: %s", err)
			}
			depositArg, err := cmd.Flags().GetString(cli.FlagDeposit)
			if err != nil {
				return fmt.Errorf("deposit: %s", err)
			}
			deposit, err := sdk.ParseCoinsNormalized(depositArg)
			if err != nil {
				return err
			}

			content := types.ActivateContractProposal{
				Title:       proposalTitle,
				Description: proposalDescr,
				Contract:    args[0],
			}

			msg, err := govtypes.NewMsgSubmitProposal(&content, deposit, clientCtx.GetFromAddress())
			if err != nil {
				return err
			}
			if err = msg.ValidateBasic(); err != nil {
				return err
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	cmd.Flags().String(cli.FlagTitle, "", "Title of proposal")
	cmd.Flags().String(cli.FlagDescription, "", "Description of proposal")
	cmd.Flags().String(cli.FlagDeposit, "", "Deposit of proposal")

	return cmd
}
