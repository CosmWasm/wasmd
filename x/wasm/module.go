package wasm

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"runtime/debug"
	"strings"

	wasmvm "github.com/CosmWasm/wasmvm/v3"
	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/spf13/cast"
	"github.com/spf13/cobra"

	"cosmossdk.io/core/appmodule"

	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/server"
	servertypes "github.com/cosmos/cosmos-sdk/server/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	simtypes "github.com/cosmos/cosmos-sdk/types/simulation"

	"github.com/CosmWasm/wasmd/x/wasm/client/cli"
	"github.com/CosmWasm/wasmd/x/wasm/exported"
	"github.com/CosmWasm/wasmd/x/wasm/keeper"
	"github.com/CosmWasm/wasmd/x/wasm/simulation"
	"github.com/CosmWasm/wasmd/x/wasm/types"
)

// Interface assertions to ensure AppModuleBasic and AppModule implement required interfaces
var (
	_ module.AppModuleBasic      = AppModuleBasic{}
	_ module.AppModuleSimulation = AppModule{}
)

// Module initialization flags for configuring wasm module behavior
const (
	// flagWasmMemoryCacheSize controls the size of in-memory cache for Wasm modules in MiB
	flagWasmMemoryCacheSize = "wasm.memory_cache_size"
	// flagWasmQueryGasLimit sets the maximum gas limit for smart contract queries
	flagWasmQueryGasLimit = "wasm.query_gas_limit"
	// flagWasmSimulationGasLimit sets the maximum gas limit for simulation transactions
	flagWasmSimulationGasLimit = "wasm.simulation_gas_limit"
	// flagWasmSkipWasmVMVersionCheck allows skipping the libwasmvm version compatibility check
	flagWasmSkipWasmVMVersionCheck = "wasm.skip_wasmvm_version_check"
)

// AppModuleBasic defines the basic application module used by the wasm module.
// This struct implements the module.AppModuleBasic interface and provides
// fundamental functionality like codec registration, CLI commands, and genesis handling.
type AppModuleBasic struct{}

// RegisterLegacyAminoCodec registers the wasm module's types with the legacy Amino codec.
// This is required for backward compatibility with older clients.
func (b AppModuleBasic) RegisterLegacyAminoCodec(amino *codec.LegacyAmino) {
	types.RegisterLegacyAminoCodec(amino)
}

// RegisterGRPCGatewayRoutes registers the gRPC Gateway routes for the wasm module.
// This enables HTTP/JSON API access to the module's query endpoints.
func (b AppModuleBasic) RegisterGRPCGatewayRoutes(clientCtx client.Context, serveMux *runtime.ServeMux) {
	err := types.RegisterQueryHandlerClient(context.Background(), serveMux, types.NewQueryClient(clientCtx))
	if err != nil {
		panic(err)
	}
}

// Name returns the wasm module's name as defined in the types package.
// This name is used for routing and identification throughout the Cosmos SDK.
func (AppModuleBasic) Name() string {
	return types.ModuleName
}

// DefaultGenesis returns default genesis state as raw bytes for the wasm module.
// This provides the initial state when a new blockchain is created.
func (AppModuleBasic) DefaultGenesis(cdc codec.JSONCodec) json.RawMessage {
	return cdc.MustMarshalJSON(&types.GenesisState{
		Params: types.DefaultParams(),
	})
}

// ValidateGenesis performs genesis state validation for the wasm module.
// This ensures that the genesis state is valid before the blockchain starts.
func (b AppModuleBasic) ValidateGenesis(marshaler codec.JSONCodec, _ client.TxEncodingConfig, message json.RawMessage) error {
	var data types.GenesisState
	err := marshaler.UnmarshalJSON(message, &data)
	if err != nil {
		return err
	}
	return types.ValidateGenesis(data)
}

// GetTxCmd returns the root transaction command for the wasm module.
// This provides CLI commands for creating and managing wasm transactions.
func (b AppModuleBasic) GetTxCmd() *cobra.Command {
	return cli.GetTxCmd()
}

// GetQueryCmd returns the root query command for the wasm module.
// This provides CLI commands for querying wasm module state.
func (b AppModuleBasic) GetQueryCmd() *cobra.Command {
	return cli.GetQueryCmd()
}

// RegisterInterfaces registers the wasm module's interfaces with the interface registry.
// This enables proper serialization and deserialization of module types.
func (b AppModuleBasic) RegisterInterfaces(registry cdctypes.InterfaceRegistry) {
	types.RegisterInterfaces(registry)
}

// ____________________________________________________________________________
// AppModule implementation

// Interface assertion to ensure AppModule implements the appmodule.AppModule interface
var _ appmodule.AppModule = AppModule{}

// AppModule implements an application module for the wasm module.
// This struct contains all the dependencies and state needed to run the wasm module
// within a Cosmos SDK application.
type AppModule struct {
	AppModuleBasic
	cdc                codec.Codec               // Codec for serialization/deserialization
	keeper             *keeper.Keeper            // Main keeper for wasm module state management
	validatorSetSource keeper.ValidatorSetSource // Source for validator set information
	accountKeeper      types.AccountKeeper       // Account keeper for simulation purposes
	bankKeeper         simulation.BankKeeper     // Bank keeper for simulation purposes
	router             keeper.MessageRouter      // Message router for handling wasm messages
	// legacySubspace is used solely for migration of x/params managed parameters
	// This field is deprecated and will be removed in future versions
	legacySubspace exported.Subspace
}

// NewAppModule creates a new AppModule object with all required dependencies.
// This constructor initializes the module with the necessary keepers and configuration.
func NewAppModule(
	cdc codec.Codec,
	keeper *keeper.Keeper,
	validatorSetSource keeper.ValidatorSetSource,
	ak types.AccountKeeper,
	bk simulation.BankKeeper,
	router *baseapp.MsgServiceRouter,
	ss exported.Subspace,
) AppModule {
	return AppModule{
		AppModuleBasic:     AppModuleBasic{},
		cdc:                cdc,
		keeper:             keeper,
		validatorSetSource: validatorSetSource,
		accountKeeper:      ak,
		bankKeeper:         bk,
		router:             router,
		legacySubspace:     ss,
	}
}

// IsOnePerModuleType implements the depinject.OnePerModuleType interface.
// This is a marker method that indicates this module should have only one instance per type.
func (am AppModule) IsOnePerModuleType() { // marker
}

// IsAppModule implements the appmodule.AppModule interface.
// This is a marker method that indicates this struct implements the AppModule interface.
func (am AppModule) IsAppModule() { // marker
}

// ConsensusVersion is a sequence number for state-breaking changes of the module.
// It should be incremented on each consensus-breaking change introduced by the module.
// To avoid wrong/empty versions, the initial version should be set to 1.
// Current version: 4 - indicates this module has had 4 major consensus-breaking changes.
func (AppModule) ConsensusVersion() uint64 { return 4 }

// RegisterServices registers the gRPC services for the wasm module.
// This includes message server, query server, and migration handlers for state upgrades.
func (am AppModule) RegisterServices(cfg module.Configurator) {
	// Register message server for handling wasm transactions
	types.RegisterMsgServer(cfg.MsgServer(), keeper.NewMsgServerImpl(am.keeper))
	// Register query server for handling wasm queries
	types.RegisterQueryServer(cfg.QueryServer(), keeper.Querier(am.keeper))

	// Register migration handlers for state upgrades
	m := keeper.NewMigrator(*am.keeper, am.legacySubspace)
	err := cfg.RegisterMigration(types.ModuleName, 1, m.Migrate1to2)
	if err != nil {
		panic(err)
	}
	err = cfg.RegisterMigration(types.ModuleName, 2, m.Migrate2to3)
	if err != nil {
		panic(err)
	}
	err = cfg.RegisterMigration(types.ModuleName, 3, m.Migrate3to4)
	if err != nil {
		panic(err)
	}
}

// RegisterInvariants registers the wasm module invariants.
// Currently, no invariants are registered for the wasm module.
// Invariants are checks that should always be true and help detect state corruption.
func (am AppModule) RegisterInvariants(_ sdk.InvariantRegistry) {} // nolint: staticcheck // deprecated interface

// QuerierRoute returns the wasm module's querier route name.
// This is used for routing queries to the correct module handler.
func (AppModule) QuerierRoute() string {
	return types.QuerierRoute
}

// InitGenesis performs genesis initialization for the wasm module.
// This function is called when the blockchain starts and sets up the initial state.
// It returns no validator updates as the wasm module doesn't manage validators.
func (am AppModule) InitGenesis(ctx sdk.Context, cdc codec.JSONCodec, data json.RawMessage) []abci.ValidatorUpdate {
	var genesisState types.GenesisState
	cdc.MustUnmarshalJSON(data, &genesisState)
	validators, err := keeper.InitGenesis(ctx, am.keeper, genesisState)
	if err != nil {
		panic(err)
	}
	return validators
}

// ExportGenesis returns the exported genesis state as raw bytes for the wasm module.
// This function is used to export the current state for genesis file generation.
func (am AppModule) ExportGenesis(ctx sdk.Context, cdc codec.JSONCodec) json.RawMessage {
	gs := keeper.ExportGenesis(ctx, am.keeper)
	return cdc.MustMarshalJSON(gs)
}

// ____________________________________________________________________________
// AppModuleSimulation functions

// GenerateGenesisState creates a randomized GenState of the wasm module.
// This is used for simulation testing to generate random initial states.
func (AppModule) GenerateGenesisState(simState *module.SimulationState) {
	simulation.RandomizedGenState(simState)
}

// ProposalMsgs returns msgs used for governance proposals for simulations.
// This provides sample messages that can be used in governance proposal simulations.
func (am AppModule) ProposalMsgs(simState module.SimulationState) []simtypes.WeightedProposalMsg {
	return simulation.ProposalMsgs(am.bankKeeper, am.keeper)
}

// RegisterStoreDecoder registers a decoder for wasm module's types.
// Currently, no store decoder is registered for the wasm module.
func (am AppModule) RegisterStoreDecoder(_ simtypes.StoreDecoderRegistry) {
}

// WeightedOperations returns all the wasm module operations with their respective weights.
// This is used for simulation testing to determine the frequency of different operations.
func (am AppModule) WeightedOperations(simState module.SimulationState) []simtypes.WeightedOperation {
	return simulation.WeightedOperations(simState.AppParams, am.accountKeeper, am.bankKeeper, am.keeper)
}

// ____________________________________________________________________________
// Module initialization and configuration functions

// AddModuleInitFlags implements servertypes.ModuleInitFlags interface.
// This function adds command-line flags for configuring the wasm module during node startup.
func AddModuleInitFlags(startCmd *cobra.Command) {
	defaults := types.DefaultNodeConfig()

	// Add flag for memory cache size configuration
	startCmd.Flags().Uint32(flagWasmMemoryCacheSize, defaults.MemoryCacheSize,
		"Sets the size in MiB (NOT bytes) of an in-memory cache for Wasm modules. Set to 0 to disable.")

	// Add flag for query gas limit configuration
	startCmd.Flags().Uint64(flagWasmQueryGasLimit, defaults.SmartQueryGasLimit,
		"Set the max gas that can be spent on executing a query with a Wasm contract")

	// Add flag for simulation gas limit configuration
	startCmd.Flags().String(flagWasmSimulationGasLimit, "",
		"Set the max gas that can be spent when executing a simulation TX")

	// Add flag to skip wasmvm version check
	startCmd.Flags().Bool(flagWasmSkipWasmVMVersionCheck, false,
		"Skip check that ensures that libwasmvm version (the Rust project) and wasmvm version (the Go project) match")

	// Pre-run check function to validate wasmvm version compatibility
	preCheck := func(cmd *cobra.Command, _ []string) error {
		skip, err := cmd.Flags().GetBool(flagWasmSkipWasmVMVersionCheck)
		if err != nil {
			return fmt.Errorf("unable to read skip flag value: %w", err)
		}
		if skip {
			cmd.Println("libwasmvm version check skipped")
			return nil
		}
		return CheckLibwasmVersion(getExpectedLibwasmVersion())
	}
	startCmd.PreRunE = chainPreRuns(preCheck, startCmd.PreRunE)
}

// ReadNodeConfig reads the node-specific configuration from application options.
// This function parses the command-line flags and environment variables to configure the wasm module.
func ReadNodeConfig(opts servertypes.AppOptions) (types.NodeConfig, error) {
	cfg := types.DefaultNodeConfig()
	var err error

	// Parse memory cache size configuration
	if v := opts.Get(flagWasmMemoryCacheSize); v != nil {
		if cfg.MemoryCacheSize, err = cast.ToUint32E(v); err != nil {
			return cfg, err
		}
	}

	// Parse query gas limit configuration
	if v := opts.Get(flagWasmQueryGasLimit); v != nil {
		if cfg.SmartQueryGasLimit, err = cast.ToUint64E(v); err != nil {
			return cfg, err
		}
	}

	// Parse simulation gas limit configuration
	if v := opts.Get(flagWasmSimulationGasLimit); v != nil {
		if raw, ok := v.(string); !ok || raw != "" {
			limit, err := cast.ToUint64E(v) // non empty string set
			if err != nil {
				return cfg, err
			}
			cfg.SimulationGasLimit = &limit
		}
	}

	// Attach contract debugging to global "trace" flag
	if v := opts.Get(server.FlagTrace); v != nil {
		if cfg.ContractDebugMode, err = cast.ToBoolE(v); err != nil {
			return cfg, err
		}
	}
	return cfg, nil
}

// getExpectedLibwasmVersion retrieves the expected libwasmvm version from go.mod dependencies.
// This function reads the build information to determine the expected version of the wasmvm library.
func getExpectedLibwasmVersion() string {
	buildInfo, ok := debug.ReadBuildInfo()
	if !ok {
		panic("can't read build info")
	}
	for _, d := range buildInfo.Deps {
		if d.Path != "github.com/CosmWasm/wasmvm/v3" {
			continue
		}
		if d.Replace != nil {
			return d.Replace.Version
		}
		return d.Version
	}
	return ""
}

// CheckLibwasmVersion ensures that the libwasmvm version loaded at runtime matches the version
// of the github.com/CosmWasm/wasmvm dependency in go.mod. This is useful when dealing with
// shared libraries that are copied or moved from their default location, e.g. when building the node
// on one machine and deploying it to other machines.
//
// Usually the libwasmvm version (the Rust project) and wasmvm version (the Go project) match. However,
// there are situations in which this is not the case. This can be during development or if one of the
// two is patched. In such cases it is advised to not execute the check.
//
// An alternative method to obtain the libwasmvm version loaded at runtime is executing
// `wasmd query wasm libwasmvm-version`.
func CheckLibwasmVersion(wasmExpectedVersion string) error {
	if wasmExpectedVersion == "" {
		return errors.New("wasmvm module not exist")
	}
	wasmVersion, err := wasmvm.LibwasmvmVersion()
	if err != nil {
		return fmt.Errorf("unable to retrieve libwasmversion %w", err)
	}
	if !strings.Contains(wasmExpectedVersion, wasmVersion) {
		return fmt.Errorf("libwasmversion mismatch. got: %s; expected: %s", wasmVersion, wasmExpectedVersion)
	}
	return nil
}

// preRunFn defines the type for pre-run functions that can be chained together
type preRunFn func(cmd *cobra.Command, args []string) error

// chainPreRuns chains multiple pre-run functions together.
// This function executes each pre-run function in sequence, stopping if any returns an error.
// It's used to combine multiple validation or setup functions that need to run before command execution.
func chainPreRuns(pfns ...preRunFn) preRunFn {
	return func(cmd *cobra.Command, args []string) error {
		for _, pfn := range pfns {
			if pfn != nil {
				if err := pfn(cmd, args); err != nil {
					return err
				}
			}
		}
		return nil
	}
}
