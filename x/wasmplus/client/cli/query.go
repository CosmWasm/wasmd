package cli

import (
	"context"
	"encoding/base64"

	"github.com/spf13/cobra"
	flag "github.com/spf13/pflag"

	"github.com/Finschia/finschia-sdk/client"
	"github.com/Finschia/finschia-sdk/client/flags"

	wasmcli "github.com/Finschia/wasmd/x/wasm/client/cli"
	"github.com/Finschia/wasmd/x/wasmplus/types"
)

func GetQueryCmd() *cobra.Command {
	queryCmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "Querying commands for the wasm module",
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}
	queryCmd.AddCommand(
		wasmcli.GetCmdListCode(),
		wasmcli.GetCmdListContractByCode(),
		wasmcli.GetCmdQueryCode(),
		wasmcli.GetCmdQueryCodeInfo(),
		wasmcli.GetCmdGetContractInfo(),
		wasmcli.GetCmdGetContractHistory(),
		wasmcli.GetCmdGetContractState(),
		wasmcli.GetCmdListPinnedCode(),
		wasmcli.GetCmdLibVersion(),
		wasmcli.GetCmdQueryParams(),
		wasmcli.GetCmdBuildAddress(),
		GetCmdListInactiveContracts(),
		GetCmdIsInactiveContract(),
	)
	return queryCmd
}

// sdk ReadPageRequest expects binary, but we encoded to base64 in our marshaller
func withPageKeyDecoded(flagSet *flag.FlagSet) *flag.FlagSet {
	encoded, err := flagSet.GetString(flags.FlagPageKey)
	if err != nil {
		panic(err.Error())
	}
	raw, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		panic(err.Error())
	}
	err = flagSet.Set(flags.FlagPageKey, string(raw))
	if err != nil {
		panic(err.Error())
	}
	return flagSet
}

func GetCmdListInactiveContracts() *cobra.Command {
	cmd := &cobra.Command{
		Use:  "inactive-contracts",
		Long: "List all inactive contracts",
		Args: cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			pageReq, err := client.ReadPageRequest(withPageKeyDecoded(cmd.Flags()))
			if err != nil {
				return err
			}
			queryClient := types.NewQueryClient(clientCtx)
			res, err := queryClient.InactiveContracts(
				context.Background(),
				&types.QueryInactiveContractsRequest{
					Pagination: pageReq,
				},
			)
			if err != nil {
				return err
			}
			return clientCtx.PrintProto(res)
		},
	}
	flags.AddQueryFlagsToCmd(cmd)
	flags.AddPaginationFlagsToCmd(cmd, "list of inactive contracts")
	return cmd
}

func GetCmdIsInactiveContract() *cobra.Command {
	cmd := &cobra.Command{
		Use:  "is-inactive [bech32_address]",
		Long: "Check if inactive contract or not",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)
			res, err := queryClient.InactiveContract(
				context.Background(),
				&types.QueryInactiveContractRequest{
					Address: args[0],
				},
			)
			if err != nil {
				return err
			}
			return clientCtx.PrintProto(res)
		},
	}
	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}
