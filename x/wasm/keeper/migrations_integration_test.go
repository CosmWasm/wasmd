package keeper_test

import (
	"testing"

	"github.com/CosmWasm/wasmd/x/wasm/types"

	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	upgradetypes "github.com/cosmos/cosmos-sdk/x/upgrade/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/CosmWasm/wasmd/app"
	"github.com/CosmWasm/wasmd/x/wasm"
)

func TestModuleMigrations(t *testing.T) {
	addr := "cosmos1vx8knpllrj7n963p9ttd80w47kpacrhuts497x"
	address, err := sdk.AccAddressFromBech32(addr)

	require.NoError(t, err)

	wasmApp := app.Setup(t)
	upgradeHandler := func(ctx sdk.Context, plan upgradetypes.Plan, fromVM module.VersionMap) (module.VersionMap, error) { //nolint:unparam
		return wasmApp.ModuleManager.RunMigrations(ctx, wasmApp.Configurator(), fromVM)
	}

	specs := map[string]struct {
		setup        func(ctx sdk.Context)
		startVersion uint64
		exp          types.Params
	}{
		"with legacy params migrated": {
			startVersion: 1,
			setup: func(ctx sdk.Context) {
				params := types.Params{
					CodeUploadAccess:             types.AllowNobody,
					InstantiateDefaultPermission: types.AccessTypeNobody,
				}
				sp, _ := wasmApp.ParamsKeeper.GetSubspace(types.ModuleName)
				sp.SetParamSet(ctx, &params)
			},
			exp: types.Params{
				CodeUploadAccess:             types.AllowNobody,
				InstantiateDefaultPermission: types.AccessTypeNobody,
			},
		},
		"fresh from genesis": {
			startVersion: wasmApp.ModuleManager.GetVersionMap()[types.ModuleName], // latest
			setup:        func(ctx sdk.Context) {},
			exp:          types.DefaultParams(),
		},
		"with legacy params and access config migrated": {
			startVersion: 1,
			setup: func(ctx sdk.Context) {
				params := types.Params{
					CodeUploadAccess:             types.AccessTypeOnlyAddress.With(address),
					InstantiateDefaultPermission: types.AccessTypeOnlyAddress,
				}
				sp, _ := wasmApp.ParamsKeeper.GetSubspace(types.ModuleName)
				sp.SetParamSet(ctx, &params)
			},
			exp: types.Params{
				CodeUploadAccess:             types.AccessTypeAnyOfAddresses.With(address),
				InstantiateDefaultPermission: types.AccessTypeAnyOfAddresses,
			},
		},
	}
	for name, spec := range specs {
		t.Run(name, func(t *testing.T) {
			ctx, _ := wasmApp.BaseApp.NewContext(false, tmproto.Header{}).CacheContext()
			spec.setup(ctx)

			fromVM := wasmApp.UpgradeKeeper.GetModuleVersionMap(ctx)
			fromVM[wasm.ModuleName] = spec.startVersion
			_, err := upgradeHandler(ctx, upgradetypes.Plan{Name: "testing"}, fromVM)
			require.NoError(t, err)

			// when
			gotVM, err := wasmApp.ModuleManager.RunMigrations(ctx, wasmApp.Configurator(), fromVM)

			// then
			require.NoError(t, err)
			var expModuleVersion uint64 = 4
			assert.Equal(t, expModuleVersion, gotVM[wasm.ModuleName])
			gotParams := wasmApp.WasmKeeper.GetParams(ctx)
			assert.Equal(t, spec.exp, gotParams)

		})
	}
}
