package cli

import (
	"github.com/CosmWasm/wasmd/x/wasm/internal/types"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/tx"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/cosmos-sdk/x/gov/client/cli"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func ProposalStoreCodeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "wasm-store [wasm file] --source [source] --builder [builder] --title [text] --description [text] --run-as [address]",
		Short: "Submit a wasm binary proposal",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx := client.GetClientContextFromCmd(cmd)
			clientCtx, err := client.ReadTxCommandFlags(clientCtx, cmd.Flags())
			if err != nil {
				return err
			}

			src, err := parseStoreCodeArgs(args, clientCtx)
			if err != nil {
				return err
			}
			if len(viper.GetString(flagRunAs)) == 0 {
				return errors.New("run-as address is required")
			}
			runAsAddr, err := sdk.AccAddressFromBech32(viper.GetString(flagRunAs))
			if err != nil {
				return errors.Wrap(err, "run-as")
			}
			content := types.StoreCodeProposal{
				Title:                 viper.GetString(cli.FlagTitle),
				Description:           viper.GetString(cli.FlagDescription),
				RunAs:                 runAsAddr,
				WASMByteCode:          src.WASMByteCode,
				Source:                src.Source,
				Builder:               src.Builder,
				InstantiatePermission: src.InstantiatePermission,
			}

			deposit, err := sdk.ParseCoins(viper.GetString(cli.FlagDeposit))
			if err != nil {
				return err
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

	cmd.Flags().String(flagSource, "", "A valid URI reference to the contract's source code, optional")
	cmd.Flags().String(flagBuilder, "", "A valid docker tag for the build system, optional")
	cmd.Flags().String(flagRunAs, "", "The address that is stored as code creator")
	cmd.Flags().String(flagInstantiateByEverybody, "", "Everybody can instantiate a contract from the code, optional")
	cmd.Flags().String(flagInstantiateByAddress, "", "Only this address can instantiate a contract instance from the code, optional")

	// proposal flags
	cmd.Flags().String(cli.FlagTitle, "", "Title of proposal")
	cmd.Flags().String(cli.FlagDescription, "", "Description of proposal")
	cmd.Flags().String(cli.FlagDeposit, "", "Deposit of proposal")
	cmd.Flags().String(cli.FlagProposal, "", "Proposal file path (if this path is given, other proposal flags are ignored)")
	// type values must match the "ProposalHandler" "routes" in cli
	cmd.Flags().String(flagProposalType, "", "Permission of proposal, types: store-code/instantiate/migrate/update-admin/clear-admin/text/parameter_change/software_upgrade")
	return cmd
}

func ProposalInstantiateContractCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "instantiate-contract [code_id_int64] [json_encoded_init_args] --label [text] --title [text] --description [text] --run-as [address] --admin [address,optional] --amount [coins,optional]",
		Short: "Submit an instantiate wasm contract proposal",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx := client.GetClientContextFromCmd(cmd)
			clientCtx, err := client.ReadTxCommandFlags(clientCtx, cmd.Flags())
			if err != nil {
				return err
			}

			src, err := parseInstantiateArgs(args, clientCtx)
			if err != nil {
				return err
			}
			if len(viper.GetString(flagRunAs)) == 0 {
				return errors.New("creator address is required")
			}
			creator, err := sdk.AccAddressFromBech32(viper.GetString(flagRunAs))
			if err != nil {
				return errors.Wrap(err, "creator")
			}
			content := types.InstantiateContractProposal{
				Title:       viper.GetString(cli.FlagTitle),
				Description: viper.GetString(cli.FlagDescription),
				RunAs:       creator,
				Admin:       src.Admin,
				CodeID:      src.CodeID,
				Label:       src.Label,
				InitMsg:     src.InitMsg,
				InitFunds:   src.InitFunds,
			}

			deposit, err := sdk.ParseCoins(viper.GetString(cli.FlagDeposit))
			if err != nil {
				return err
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
	cmd.Flags().String(flagAmount, "", "Coins to send to the contract during instantiation")
	cmd.Flags().String(flagLabel, "", "A human-readable name for this contract in lists")
	cmd.Flags().String(flagAdmin, "", "Address of an admin")
	cmd.Flags().String(flagRunAs, "", "The address that pays the init funds. It is the creator of the contract and passed to the contract as sender on proposal execution")

	// proposal flags
	cmd.Flags().String(cli.FlagTitle, "", "Title of proposal")
	cmd.Flags().String(cli.FlagDescription, "", "Description of proposal")
	cmd.Flags().String(cli.FlagDeposit, "", "Deposit of proposal")
	cmd.Flags().String(cli.FlagProposal, "", "Proposal file path (if this path is given, other proposal flags are ignored)")
	// type values must match the "ProposalHandler" "routes" in cli
	cmd.Flags().String(flagProposalType, "", "Permission of proposal, types: store-code/instantiate/migrate/update-admin/clear-admin/text/parameter_change/software_upgrade")
	return cmd
}

func ProposalMigrateContractCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "migrate-contract [contract_addr_bech32] [new_code_id_int64] [json_encoded_migration_args]",
		Short: "Submit a migrate wasm contract to a new code version proposal",
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx := client.GetClientContextFromCmd(cmd)
			clientCtx, err := client.ReadTxCommandFlags(clientCtx, cmd.Flags())
			if err != nil {
				return err
			}

			src, err := parseMigrateContractArgs(args, clientCtx)
			if err != nil {
				return err
			}

			if len(viper.GetString(flagRunAs)) == 0 {
				return errors.New("run-as address is required")
			}
			runAs, err := sdk.AccAddressFromBech32(viper.GetString(flagRunAs))
			if err != nil {
				return errors.Wrap(err, "run-as")
			}

			content := types.MigrateContractProposal{
				Title:       viper.GetString(cli.FlagTitle),
				Description: viper.GetString(cli.FlagDescription),
				Contract:    src.Contract,
				CodeID:      src.CodeID,
				MigrateMsg:  src.MigrateMsg,
				RunAs:       runAs,
			}

			deposit, err := sdk.ParseCoins(viper.GetString(cli.FlagDeposit))
			if err != nil {
				return err
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
	cmd.Flags().String(flagRunAs, "", "The address that is passed as sender to the contract on proposal execution")

	// proposal flags
	cmd.Flags().String(cli.FlagTitle, "", "Title of proposal")
	cmd.Flags().String(cli.FlagDescription, "", "Description of proposal")
	cmd.Flags().String(cli.FlagDeposit, "", "Deposit of proposal")
	cmd.Flags().String(cli.FlagProposal, "", "Proposal file path (if this path is given, other proposal flags are ignored)")
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
			clientCtx := client.GetClientContextFromCmd(cmd)
			clientCtx, err := client.ReadTxCommandFlags(clientCtx, cmd.Flags())
			if err != nil {
				return err
			}

			src, err := parseUpdateContractAdminArgs(args, clientCtx)
			if err != nil {
				return err
			}

			content := types.UpdateAdminProposal{
				Title:       viper.GetString(cli.FlagTitle),
				Description: viper.GetString(cli.FlagDescription),
				Contract:    src.Contract,
				NewAdmin:    src.NewAdmin,
			}

			deposit, err := sdk.ParseCoins(viper.GetString(cli.FlagDeposit))
			if err != nil {
				return err
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
	// proposal flags
	cmd.Flags().String(cli.FlagTitle, "", "Title of proposal")
	cmd.Flags().String(cli.FlagDescription, "", "Description of proposal")
	cmd.Flags().String(cli.FlagDeposit, "", "Deposit of proposal")
	cmd.Flags().String(cli.FlagProposal, "", "Proposal file path (if this path is given, other proposal flags are ignored)")
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
			clientCtx := client.GetClientContextFromCmd(cmd)
			clientCtx, err := client.ReadTxCommandFlags(clientCtx, cmd.Flags())
			if err != nil {
				return err
			}

			contractAddr, err := sdk.AccAddressFromBech32(args[0])
			if err != nil {
				return sdkerrors.Wrap(err, "contract")
			}

			content := types.ClearAdminProposal{
				Title:       viper.GetString(cli.FlagTitle),
				Description: viper.GetString(cli.FlagDescription),
				Contract:    contractAddr,
			}

			deposit, err := sdk.ParseCoins(viper.GetString(cli.FlagDeposit))
			if err != nil {
				return err
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
	// proposal flags
	cmd.Flags().String(cli.FlagTitle, "", "Title of proposal")
	cmd.Flags().String(cli.FlagDescription, "", "Description of proposal")
	cmd.Flags().String(cli.FlagDeposit, "", "Deposit of proposal")
	cmd.Flags().String(cli.FlagProposal, "", "Proposal file path (if this path is given, other proposal flags are ignored)")
	// type values must match the "ProposalHandler" "routes" in cli
	cmd.Flags().String(flagProposalType, "", "Permission of proposal, types: store-code/instantiate/migrate/update-admin/clear-admin/text/parameter_change/software_upgrade")
	return cmd
}
