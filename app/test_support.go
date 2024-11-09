package app

import (
	bankkeeper "cosmossdk.io/x/bank/keeper"
	govkeeper "cosmossdk.io/x/gov/keeper"
	stakingkeeper "cosmossdk.io/x/staking/keeper"

	"github.com/cosmos/cosmos-sdk/baseapp"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"

	ibckeeper "github.com/cosmos/ibc-go/v9/modules/core/keeper"

	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
)

func (app *WasmApp) GetIBCKeeper() *ibckeeper.Keeper {
	return app.IBCKeeper
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
	return app.AuthKeeper
}

func (app *WasmApp) GetWasmKeeper() wasmkeeper.Keeper {
	return app.WasmKeeper
}

func (app *WasmApp) GetGovKeeper() govkeeper.Keeper {
	return app.GovKeeper
}
