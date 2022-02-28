package cli

import (
	"fmt"
	"strconv"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/tx"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/gov/client/cli"
	govtypesv1beta1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"
	"github.com/spf13/cobra"

	"github.com/CosmWasm/wasmd/x/wasm/types"
)

func ProposalMigrateContractCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "migrate-contract [contract_addr_bech32] [new_code_id_int64] [json_encoded_migration_args]",
		Short: "Submit a migrate wasm contract to a new code version proposal",
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			src, err := parseMigrateContractArgs(args, clientCtx)
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
				return err
			}
			deposit, err := sdk.ParseCoinsNormalized(depositArg)
			if err != nil {
				return err
			}

			content := types.MigrateContractProposal{
				Title:       proposalTitle,
				Description: proposalDescr,
				Contract:    src.Contract,
				CodeID:      src.CodeID,
				Msg:         src.Msg,
			}

			msg, err := govtypesv1beta1.NewMsgSubmitProposal(&content, deposit, clientCtx.GetFromAddress())
			if err != nil {
				return err
			}
			if err = msg.ValidateBasic(); err != nil {
				return err
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	// proposal flags
	// type values must match the "ProposalHandler" "routes" in cli
	cmd.Flags().String(flagProposalType, "", "Permission of proposal, types: store-code/instantiate/migrate/update-admin/clear-admin/text/parameter_change/software_upgrade")
	return cmd
}

func ProposalSudoContractCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sudo-contract [contract_addr_bech32] [json_encoded_migration_args]",
		Short: "Submit a sudo wasm contract proposal (to call privileged commands)",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			contract := args[0]
			sudoMsg := []byte(args[1])

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
				return err
			}
			deposit, err := sdk.ParseCoinsNormalized(depositArg)
			if err != nil {
				return err
			}

			content := types.SudoContractProposal{
				Title:       proposalTitle,
				Description: proposalDescr,
				Contract:    contract,
				Msg:         sudoMsg,
			}

			msg, err := govtypesv1beta1.NewMsgSubmitProposal(&content, deposit, clientCtx.GetFromAddress())
			if err != nil {
				return err
			}
			if err = msg.ValidateBasic(); err != nil {
				return err
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	// proposal flagsExecute
	// type values must match the "ProposalHandler" "routes" in cli
	cmd.Flags().String(flagProposalType, "", "Permission of proposal, types: store-code/instantiate/migrate/update-admin/clear-admin/text/parameter_change/software_upgrade")
	return cmd
}

func ProposalUpdateContractAdminCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "set-contract-admin [contract_addr_bech32] [new_admin_addr_bech32]",
		Short: "Submit a new admin for a contract proposal",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			src, err := parseUpdateContractAdminArgs(args, clientCtx)
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

			content := types.UpdateAdminProposal{
				Title:       proposalTitle,
				Description: proposalDescr,
				Contract:    src.Contract,
				NewAdmin:    src.NewAdmin,
			}

			msg, err := govtypesv1beta1.NewMsgSubmitProposal(&content, deposit, clientCtx.GetFromAddress())
			if err != nil {
				return err
			}
			if err = msg.ValidateBasic(); err != nil {
				return err
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}
	// proposal flags
	// type values must match the "ProposalHandler" "routes" in cli
	cmd.Flags().String(flagProposalType, "", "Permission of proposal, types: store-code/instantiate/migrate/update-admin/clear-admin/text/parameter_change/software_upgrade")
	return cmd
}

func ProposalClearContractAdminCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "clear-contract-admin [contract_addr_bech32]",
		Short: "Submit a clear admin for a contract to prevent further migrations proposal",
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

			content := types.ClearAdminProposal{
				Title:       proposalTitle,
				Description: proposalDescr,
				Contract:    args[0],
			}

			msg, err := govtypesv1beta1.NewMsgSubmitProposal(&content, deposit, clientCtx.GetFromAddress())
			if err != nil {
				return err
			}
			if err = msg.ValidateBasic(); err != nil {
				return err
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}
	// proposal flags
	// type values must match the "ProposalHandler" "routes" in cli
	cmd.Flags().String(flagProposalType, "", "Permission of proposal, types: store-code/instantiate/migrate/update-admin/clear-admin/text/parameter_change/software_upgrade")
	return cmd
}

func ProposalPinCodesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pin-codes [code-ids]",
		Short: "Submit a pin code proposal for pinning a code to cache",
		Args:  cobra.MinimumNArgs(1),
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
			codeIds, err := parsePinCodesArgs(args)
			if err != nil {
				return err
			}

			content := types.PinCodesProposal{
				Title:       proposalTitle,
				Description: proposalDescr,
				CodeIDs:     codeIds,
			}

			msg, err := govtypesv1beta1.NewMsgSubmitProposal(&content, deposit, clientCtx.GetFromAddress())
			if err != nil {
				return err
			}
			if err = msg.ValidateBasic(); err != nil {
				return err
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}
	// proposal flags

	// type values must match the "ProposalHandler" "routes" in cli
	cmd.Flags().String(flagProposalType, "", "Permission of proposal, types: store-code/instantiate/migrate/update-admin/clear-admin/text/parameter_change/software_upgrade")
	return cmd
}

func parsePinCodesArgs(args []string) ([]uint64, error) {
	codeIDs := make([]uint64, len(args))
	for i, c := range args {
		codeID, err := strconv.ParseUint(c, 10, 64)
		if err != nil {
			return codeIDs, fmt.Errorf("code IDs: %s", err)
		}
		codeIDs[i] = codeID
	}
	return codeIDs, nil
}

func ProposalUnpinCodesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "unpin-codes [code-ids]",
		Short: "Submit a unpin code proposal for unpinning a code to cache",
		Args:  cobra.MinimumNArgs(1),
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
			codeIds, err := parsePinCodesArgs(args)
			if err != nil {
				return err
			}

			content := types.UnpinCodesProposal{
				Title:       proposalTitle,
				Description: proposalDescr,
				CodeIDs:     codeIds,
			}

			msg, err := govtypesv1beta1.NewMsgSubmitProposal(&content, deposit, clientCtx.GetFromAddress())
			if err != nil {
				return err
			}
			if err = msg.ValidateBasic(); err != nil {
				return err
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}
	// proposal flags
	// type values must match the "ProposalHandler" "routes" in cli
	cmd.Flags().String(flagProposalType, "", "Permission of proposal, types: store-code/instantiate/migrate/update-admin/clear-admin/text/parameter_change/software_upgrade")
	return cmd
}
