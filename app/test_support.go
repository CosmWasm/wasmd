package app

import (
	"github.com/cosmos/cosmos-sdk/baseapp"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	capabilitykeeper "github.com/cosmos/cosmos-sdk/x/capability/keeper"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	ibckeeper "github.com/cosmos/ibc-go/v7/modules/core/keeper"

	"github.com/CosmWasm/wasmd/x/wasm"
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

func (app *WasmApp) GetWasmKeeper() wasm.Keeper {
	return app.WasmKeeper
}
