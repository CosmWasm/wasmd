package v080_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	minttypes "github.com/cosmos/cosmos-sdk/x/mint/types"
	upgradetypes "github.com/cosmos/cosmos-sdk/x/upgrade/types"

	"github.com/CosmWasm/wasmd/app"
	"github.com/CosmWasm/wasmd/app/upgrades"
	v080 "github.com/CosmWasm/wasmd/app/upgrades/v080"
)

func TestUpgradeMovesProtocolPoolFundsToCommunityPool(t *testing.T) {
	poolAddr := authtypes.NewModuleAddress(v080.ProtocolPoolModuleName)
	escrowAddr := authtypes.NewModuleAddress(v080.ProtocolPoolEscrowAccount)

	specs := map[string]struct {
		poolFunds   sdk.Coins
		escrowFunds sdk.Coins
	}{
		"both accounts funded": {
			poolFunds:   sdk.NewCoins(sdk.NewInt64Coin(sdk.DefaultBondDenom, 1_000), sdk.NewInt64Coin("other", 7)),
			escrowFunds: sdk.NewCoins(sdk.NewInt64Coin(sdk.DefaultBondDenom, 300)),
		},
		"only escrow funded": {
			escrowFunds: sdk.NewCoins(sdk.NewInt64Coin(sdk.DefaultBondDenom, 300)),
		},
		"no funds": {},
	}
	for name, spec := range specs {
		t.Run(name, func(t *testing.T) {
			wasmApp := app.Setup(t)
			ctx, _ := wasmApp.NewContext(false).CacheContext()
			fund(t, wasmApp, ctx, poolAddr, spec.poolFunds)
			fund(t, wasmApp, ctx, escrowAddr, spec.escrowFunds)
			before, err := wasmApp.DistrKeeper.FeePool.Get(ctx)
			require.NoError(t, err)

			keepers := upgrades.AppKeepers{BankKeeper: wasmApp.BankKeeper, DistrKeeper: &wasmApp.DistrKeeper}
			handler := v080.CreateUpgradeHandler(wasmApp.ModuleManager, wasmApp.Configurator(), &keepers)
			fromVM, err := wasmApp.UpgradeKeeper.GetModuleVersionMap(ctx)
			require.NoError(t, err)

			// when
			_, err = handler(ctx, upgradetypes.Plan{Name: v080.UpgradeName}, fromVM)

			// then
			require.NoError(t, err)
			assert.True(t, wasmApp.BankKeeper.GetAllBalances(ctx, poolAddr).IsZero())
			assert.True(t, wasmApp.BankKeeper.GetAllBalances(ctx, escrowAddr).IsZero())
			after, err := wasmApp.DistrKeeper.FeePool.Get(ctx)
			require.NoError(t, err)
			moved := sdk.NewDecCoinsFromCoins(spec.poolFunds.Add(spec.escrowFunds...)...)
			assert.Equal(t, before.CommunityPool.Add(moved...), after.CommunityPool)
		})
	}
}

func fund(t *testing.T, wasmApp *app.WasmApp, ctx sdk.Context, addr sdk.AccAddress, coins sdk.Coins) {
	t.Helper()
	if coins.IsZero() {
		return
	}
	require.NoError(t, wasmApp.BankKeeper.MintCoins(ctx, minttypes.ModuleName, coins))
	require.NoError(t, wasmApp.BankKeeper.SendCoinsFromModuleToAccount(ctx, minttypes.ModuleName, addr, coins))
}
