package app

import (
	"encoding/json"
	"time"

	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
	abci "github.com/cometbft/cometbft/abci/types"
	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	"github.com/cosmos/cosmos-sdk/baseapp"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	minttypes "github.com/cosmos/cosmos-sdk/x/mint/types"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	capabilitykeeper "github.com/cosmos/ibc-go/modules/capability/keeper"
	ibckeeper "github.com/cosmos/ibc-go/v8/modules/core/keeper"
)

func (app *WasmApp) GetIBCKeeper() *ibckeeper.Keeper {
	return app.IBCKeeper
}

func (app *WasmApp) GetScopedIBCKeeper() capabilitykeeper.ScopedKeeper {
	return app.ScopedIBCKeeper
}

func (app *WasmApp) GetBaseApp() *baseapp.BaseApp {
	return app.BaseApp
}

func (app *WasmApp) GetBankKeeper() bankkeeper.Keeper {
	return app.BankKeeper
}

func (app *WasmApp) GetStakingKeeper() *stakingkeeper.Keeper {
	return app.StakingKeeper
}

func (app *WasmApp) GetAccountKeeper() authkeeper.AccountKeeper {
	return app.AccountKeeper
}

func (app *WasmApp) GetWasmKeeper() wasmkeeper.Keeper {
	return app.WasmKeeper
}

// FundAccount is a utility function that funds an account by minting and sending the coins to the address.
func (app *WasmApp) FundAccount(ctx sdk.Context, addr sdk.AccAddress, amounts sdk.Coins) error {
	if err := app.BankKeeper.MintCoins(ctx, minttypes.ModuleName, amounts); err != nil {
		return err
	}

	return app.BankKeeper.SendCoinsFromModuleToAccount(ctx, minttypes.ModuleName, addr, amounts)
}

// FundModuleAccount is a utility function that funds a module account by minting and sending the coins to the address.
func (app *WasmApp) FundModuleAccount(ctx sdk.Context, recipientMod string, amounts sdk.Coins) error {
	if err := app.BankKeeper.MintCoins(ctx, minttypes.ModuleName, amounts); err != nil {
		return err
	}

	return app.BankKeeper.SendCoinsFromModuleToModule(ctx, minttypes.ModuleName, recipientMod, amounts)
}

// InitializeFromGenesisStates calls InitChain on the app using the provided genesis states.
// If any module genesis states are missing, defaults are used.
func (app *WasmApp) InitializeFromGenesisStates(ctx sdk.Context, genesisStates ...GenesisState) *WasmApp {
	return app.InitializeFromGenesisStatesWithTimeAndChainIDAndHeight(ctx, emptyTime, SimAppChainID, defaultInitialHeight, genesisStates...)
}

// InitializeFromGenesisStatesWithTime calls InitChain on the app using the provided genesis states and time.
// If any module genesis states are missing, defaults are used.
func (app *WasmApp) InitializeFromGenesisStatesWithTime(ctx sdk.Context, genTime time.Time, genesisStates ...GenesisState) *WasmApp {
	return app.InitializeFromGenesisStatesWithTimeAndChainIDAndHeight(ctx, genTime, SimAppChainID, defaultInitialHeight, genesisStates...)
}

// InitializeFromGenesisStatesWithTimeAndChainID calls InitChain on the app using the provided genesis states, time, and chain id.
// If any module genesis states are missing, defaults are used.
func (app *WasmApp) InitializeFromGenesisStatesWithTimeAndChainID(ctx sdk.Context, genTime time.Time, chainID string, genesisStates ...GenesisState) *WasmApp {
	return app.InitializeFromGenesisStatesWithTimeAndChainIDAndHeight(ctx, genTime, chainID, defaultInitialHeight, genesisStates...)
}

// InitializeFromGenesisStatesWithTimeAndChainIDAndHeight calls InitChain on the app using the provided genesis states and other parameters.
// If any module genesis states are missing, defaults are used.
func (app *WasmApp) InitializeFromGenesisStatesWithTimeAndChainIDAndHeight(ctx sdk.Context, genTime time.Time, chainID string, initialHeight int64, genesisStates ...GenesisState) *WasmApp {
	// Create a default genesis state and overwrite with provided values
	genesisState := app.DefaultGenesis()
	for _, state := range genesisStates {
		for k, v := range state {
			genesisState[k] = v
		}
	}

	// Initialize the chain
	stateBytes, err := json.Marshal(genesisState)
	if err != nil {
		panic(err)
	}
	app.InitChain(
		&abci.RequestInitChain{
			Time:          genTime,
			Validators:    []abci.ValidatorUpdate{},
			AppStateBytes: stateBytes,
			ChainId:       chainID,
			// Set consensus params, which is needed by x/feemarket
			ConsensusParams: &tmproto.ConsensusParams{
				Block: &tmproto.BlockParams{
					MaxBytes: 200000,
					MaxGas:   20000000,
				},
			},
			InitialHeight: initialHeight,
		},
	)
	app.Commit()
	app.BeginBlocker(ctx)
	return app
}
