package main

import (
	"fmt"
	"os"
	"path"

	"github.com/CosmWasm/wasmd/app"
	"github.com/CosmWasm/wasmd/x/wasm"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/keys"
	"github.com/cosmos/cosmos-sdk/client/rpc"
	"github.com/cosmos/cosmos-sdk/simapp"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/version"
	authclient "github.com/cosmos/cosmos-sdk/x/auth/client"
	authcmd "github.com/cosmos/cosmos-sdk/x/auth/client/cli"
	"github.com/cosmos/cosmos-sdk/x/auth/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	bankcmd "github.com/cosmos/cosmos-sdk/x/bank/client/cli"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/tendermint/tendermint/libs/cli"
)

var (
	// todo: NOTE this is simapp: MakeEncodingConfig creates an EncodingConfig for an amino based test configuration.
	encodingConfig = simapp.MakeEncodingConfig()
	initClientCtx  = client.Context{}.
		WithJSONMarshaler(encodingConfig.Marshaler).
		WithTxGenerator(encodingConfig.TxGenerator).
		WithCodec(encodingConfig.Amino).
		WithInput(os.Stdin).
		WithAccountRetriever(types.NewAccountRetriever(encodingConfig.Marshaler))
)

func init() {
	// TODO: this seems to be some legacy setup left over or WIP
	authclient.Codec = encodingConfig.Marshaler
	wasm.RegisterCodec(encodingConfig.Amino)
	wasm.RegisterInterfaces(encodingConfig.InterfaceRegistry)
}

func main() {
	// Configure cobra to sort commands
	cobra.EnableCommandSorting = false

	// Read in the configuration file for the sdk
	config := sdk.GetConfig()
	config.SetBech32PrefixForAccount(sdk.Bech32PrefixAccAddr, sdk.Bech32PrefixAccPub)
	config.SetBech32PrefixForValidator(sdk.Bech32PrefixValAddr, sdk.Bech32PrefixValPub)
	config.SetBech32PrefixForConsensusNode(sdk.Bech32PrefixConsAddr, sdk.Bech32PrefixConsPub)
	config.Seal()

	// TODO: setup keybase, viper object, etc. to be passed into
	// the below functions and eliminate global vars, like we do
	// with the cdc

	rootCmd := &cobra.Command{
		Use:   "wasmcli",
		Short: "Command line interface for interacting with wasmd",
	}

	// Add --chain-id to persistent flags and mark it required
	rootCmd.PersistentFlags().String(flags.FlagChainID, "", "Chain ID of tendermint node")
	rootCmd.PersistentPreRunE = func(_ *cobra.Command, _ []string) error {
		return initConfig(rootCmd)
	}

	// Construct Root Command
	rootCmd.AddCommand(
		rpc.StatusCommand(),
		queryCmd(), // TODO: note this is simapp encoding config
		txCmd(),    // TODO: note this is simapp encoding config
		flags.LineBreak,
		flags.LineBreak,
		keys.Commands(),
		flags.LineBreak,
		version.Cmd,
		flags.NewCompletionCmd(rootCmd, true),
	)

	// Add flags and prefix all env exposed with WM
	executor := cli.PrepareMainCmd(rootCmd, "WM", app.DefaultCLIHome)

	err := executor.Execute()
	if err != nil {
		fmt.Printf("Failed executing CLI command: %s, exiting...\n", err)
		os.Exit(1)
	}
}

func queryCmd() *cobra.Command {
	queryCmd := &cobra.Command{
		Use:                        "query",
		Aliases:                    []string{"q"},
		Short:                      "Querying subcommands",
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	queryCmd.AddCommand(
		authcmd.GetAccountCmd(encodingConfig.Amino),
		flags.LineBreak,
		rpc.ValidatorCommand(encodingConfig.Amino),
		rpc.BlockCommand(),
		authcmd.QueryTxsByEventsCmd(encodingConfig.Amino),
		authcmd.QueryTxCmd(encodingConfig.Amino),
		flags.LineBreak,
	)

	app.ModuleBasics.AddQueryCommands(queryCmd, initClientCtx)

	return queryCmd
}

func txCmd() *cobra.Command {
	txCmd := &cobra.Command{
		Use:                        "tx",
		Short:                      "Transactions subcommands",
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	txCmd.AddCommand(
		bankcmd.NewSendTxCmd(initClientCtx),
		flags.LineBreak,
		authcmd.GetSignCommand(initClientCtx),
		//authcmd.GetSignBatchCommand(cdc), // TODO: do we want batch?
		authcmd.GetMultiSignCommand(initClientCtx),
		flags.LineBreak,
		authcmd.GetBroadcastCommand(initClientCtx),
		authcmd.GetEncodeCommand(initClientCtx),
		authcmd.GetDecodeCommand(initClientCtx),
		// TODO: I think it is safe to remove
		// authcmd.GetDecodeTxCmd(cdc),
		flags.LineBreak,
	)

	// add modules' tx commands
	app.ModuleBasics.AddTxCommands(txCmd, initClientCtx)

	// remove auth and bank commands as they're mounted under the root tx command
	var cmdsToRemove []*cobra.Command

	for _, cmd := range txCmd.Commands() {
		if cmd.Use == authtypes.ModuleName || cmd.Use == banktypes.ModuleName {
			cmdsToRemove = append(cmdsToRemove, cmd)
		}
	}

	txCmd.RemoveCommand(cmdsToRemove...)

	return txCmd
}

func initConfig(cmd *cobra.Command) error {
	home, err := cmd.PersistentFlags().GetString(cli.HomeFlag)
	if err != nil {
		return err
	}

	cfgFile := path.Join(home, "config", "config.toml")
	if _, err := os.Stat(cfgFile); err == nil {
		viper.SetConfigFile(cfgFile)

		if err := viper.ReadInConfig(); err != nil {
			return err
		}
	}
	if err := viper.BindPFlag(flags.FlagChainID, cmd.PersistentFlags().Lookup(flags.FlagChainID)); err != nil {
		return err
	}
	if err := viper.BindPFlag(cli.EncodingFlag, cmd.PersistentFlags().Lookup(cli.EncodingFlag)); err != nil {
		return err
	}
	return viper.BindPFlag(cli.OutputFlag, cmd.PersistentFlags().Lookup(cli.OutputFlag))
}
