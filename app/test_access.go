package app

import (
	"testing"

	"github.com/Finschia/finschia-sdk/baseapp"
	"github.com/Finschia/finschia-sdk/client"
	"github.com/Finschia/finschia-sdk/codec"
	bankkeeper "github.com/Finschia/finschia-sdk/x/bank/keeper"
	capabilitykeeper "github.com/Finschia/finschia-sdk/x/capability/keeper"
	stakingkeeper "github.com/Finschia/finschia-sdk/x/staking/keeper"
	ibctransferkeeper "github.com/Finschia/ibc-go/v3/modules/apps/transfer/keeper"
	ibckeeper "github.com/Finschia/ibc-go/v3/modules/core/keeper"

	"github.com/Finschia/wasmd/app/params"
	"github.com/Finschia/wasmd/x/wasm"
)

// Deprecated: use public app attributes directly
type TestSupport struct {
	t   testing.TB
	app *WasmApp
}

func NewTestSupport(t testing.TB, app *WasmApp) *TestSupport {
	return &TestSupport{t: t, app: app}
}

func (s TestSupport) IBCKeeper() *ibckeeper.Keeper {
	return s.app.IBCKeeper
}

func (s TestSupport) WasmKeeper() wasm.Keeper {
	return s.app.WasmKeeper
}

func (s TestSupport) AppCodec() codec.Codec {
	return s.app.appCodec
}

func (s TestSupport) ScopedWasmIBCKeeper() capabilitykeeper.ScopedKeeper {
	return s.app.ScopedWasmKeeper
}

func (s TestSupport) ScopeIBCKeeper() capabilitykeeper.ScopedKeeper {
	return s.app.ScopedIBCKeeper
}

func (s TestSupport) ScopedTransferKeeper() capabilitykeeper.ScopedKeeper {
	return s.app.ScopedTransferKeeper
}

func (s TestSupport) StakingKeeper() stakingkeeper.Keeper {
	return s.app.StakingKeeper
}

func (s TestSupport) BankKeeper() bankkeeper.Keeper {
	return s.app.BankKeeper
}

func (s TestSupport) TransferKeeper() ibctransferkeeper.Keeper {
	return s.app.TransferKeeper
}

func (s TestSupport) GetBaseApp() *baseapp.BaseApp {
	return s.app.BaseApp
}

func (s TestSupport) GetTxConfig() client.TxConfig {
	return params.MakeEncodingConfig().TxConfig
}
