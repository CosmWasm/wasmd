package wasm

import (
	"context"
	"encoding/json"
	"math/rand"

	"github.com/CosmWasm/wasmd/x/wasm/client/cli"
	"github.com/CosmWasm/wasmd/x/wasm/client/rest"
	"github.com/CosmWasm/wasmd/x/wasm/internal/keeper"
	"github.com/CosmWasm/wasmd/x/wasm/internal/types"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	simtypes "github.com/cosmos/cosmos-sdk/types/simulation"
	"github.com/gogo/protobuf/grpc"
	"github.com/gorilla/mux"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/spf13/cobra"
	abci "github.com/tendermint/tendermint/abci/types"
)

var (
	_ module.AppModule      = AppModule{}
	_ module.AppModuleBasic = AppModuleBasic{}
)

// AppModuleBasic defines the basic application module used by the wasm module.
type AppModuleBasic struct{}

func (b AppModuleBasic) RegisterLegacyAminoCodec(amino *codec.LegacyAmino) {
	RegisterCodec(amino)
}

func (b AppModuleBasic) RegisterGRPCRoutes(clientCtx client.Context, serveMux *runtime.ServeMux) {
	types.RegisterQueryHandlerClient(context.Background(), serveMux, types.NewQueryClient(clientCtx))
}

// Name returns the wasm module's name.
func (AppModuleBasic) Name() string {
	return ModuleName
}

// DefaultGenesis returns default genesis state as raw bytes for the wasm
// module.
func (AppModuleBasic) DefaultGenesis(cdc codec.JSONMarshaler) json.RawMessage {
	return cdc.MustMarshalJSON(&GenesisState{
		Params: DefaultParams(),
	})
}

// ValidateGenesis performs genesis state validation for the wasm module.
func (b AppModuleBasic) ValidateGenesis(marshaler codec.JSONMarshaler, config client.TxEncodingConfig, message json.RawMessage) error {
	var data GenesisState
	err := marshaler.UnmarshalJSON(message, &data)
	if err != nil {
		return err
	}
	return ValidateGenesis(data)
}

// RegisterRESTRoutes registers the REST routes for the wasm module.
func (AppModuleBasic) RegisterRESTRoutes(cliCtx client.Context, rtr *mux.Router) {
	rest.RegisterRoutes(cliCtx, rtr)
}

// GetTxCmd returns the root tx command for the wasm module.
func (b AppModuleBasic) GetTxCmd() *cobra.Command {
	return cli.GetTxCmd()
}

// GetQueryCmd returns no root query command for the wasm module.
func (b AppModuleBasic) GetQueryCmd() *cobra.Command {
	return cli.GetQueryCmd()
}

// RegisterInterfaceTypes implements InterfaceModule
func (b AppModuleBasic) RegisterInterfaces(registry cdctypes.InterfaceRegistry) {
	types.RegisterInterfaces(registry)
}

//____________________________________________________________________________

// AppModule implements an application module for the wasm module.
type AppModule struct {
	AppModuleBasic
	keeper Keeper
}

func (am AppModule) RegisterServices(configurator module.Configurator) {
	types.RegisterQueryServer(configurator.QueryServer(), NewQuerier(am.keeper))
}

func (am AppModule) LegacyQuerierHandler(amino *codec.LegacyAmino) sdk.Querier {
	return keeper.NewLegacyQuerier(am.keeper)
}

func (am AppModule) RegisterQueryService(server grpc.Server) {
	types.RegisterQueryServer(server, NewQuerier(am.keeper))
}

// NewAppModule creates a new AppModule object
func NewAppModule(keeper Keeper) AppModule {
	return AppModule{
		AppModuleBasic: AppModuleBasic{},
		keeper:         keeper,
	}
}

// Name returns the wasm module's name.
func (AppModule) Name() string {
	return ModuleName
}

// RegisterInvariants registers the wasm module invariants.
func (am AppModule) RegisterInvariants(ir sdk.InvariantRegistry) {}

// Route returns the message routing key for the wasm module.
func (am AppModule) Route() sdk.Route {
	return sdk.NewRoute(RouterKey, NewHandler(am.keeper))
}

// QuerierRoute returns the wasm module's querier route name.
func (AppModule) QuerierRoute() string {
	return QuerierRoute
}

// InitGenesis performs genesis initialization for the wasm module. It returns
// no validator updates.
func (am AppModule) InitGenesis(ctx sdk.Context, cdc codec.JSONMarshaler, data json.RawMessage) []abci.ValidatorUpdate {
	var genesisState GenesisState
	cdc.MustUnmarshalJSON(data, &genesisState)
	if err := InitGenesis(ctx, am.keeper, genesisState); err != nil {
		panic(err)
	}
	return []abci.ValidatorUpdate{}
}

// ExportGenesis returns the exported genesis state as raw bytes for the wasm
// module.
func (am AppModule) ExportGenesis(ctx sdk.Context, cdc codec.JSONMarshaler) json.RawMessage {
	gs := ExportGenesis(ctx, am.keeper)
	return cdc.MustMarshalJSON(gs)
}

// BeginBlock returns the begin blocker for the wasm module.
func (am AppModule) BeginBlock(_ sdk.Context, _ abci.RequestBeginBlock) {}

// EndBlock returns the end blocker for the wasm module. It returns no validator
// updates.
func (AppModule) EndBlock(_ sdk.Context, _ abci.RequestEndBlock) []abci.ValidatorUpdate {
	return []abci.ValidatorUpdate{}
}

//____________________________________________________________________________

// AppModuleSimulation functions

// GenerateGenesisState creates a randomized GenState of the bank module.
func (AppModule) GenerateGenesisState(simState *module.SimulationState) {
}

// ProposalContents doesn't return any content functions for governance proposals.
func (AppModule) ProposalContents(simState module.SimulationState) []simtypes.WeightedProposalContent {
	return nil
}

// RandomizedParams creates randomized bank param changes for the simulator.
func (AppModule) RandomizedParams(r *rand.Rand) []simtypes.ParamChange {
	return nil
}

// RegisterStoreDecoder registers a decoder for supply module's types
func (am AppModule) RegisterStoreDecoder(sdr sdk.StoreDecoderRegistry) {
}

// WeightedOperations returns the all the gov module operations with their respective weights.
func (am AppModule) WeightedOperations(simState module.SimulationState) []simtypes.WeightedOperation {
	return nil
}
