package wasmplus_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	tmproto "github.com/tendermint/tendermint/proto/tendermint/types"

	sdk "github.com/Finschia/finschia-sdk/types"
	"github.com/Finschia/finschia-sdk/types/module"
	upgradetypes "github.com/Finschia/finschia-sdk/x/upgrade/types"

	"github.com/Finschia/wasmd/appplus"
	"github.com/Finschia/wasmd/x/wasm"
)

func TestAppPlusModuleMigrations(t *testing.T) {
	wasmApp := appplus.Setup(false)
	ctx := wasmApp.BaseApp.NewContext(false, tmproto.Header{})
	upgradeHandler := func(ctx sdk.Context, plan upgradetypes.Plan, fromVM module.VersionMap) (module.VersionMap, error) {
		return wasmApp.ModuleManager().RunMigrations(ctx, wasmApp.ModuleConfigurator(), fromVM)
	}
	fromVM := wasmApp.UpgradeKeeper.GetModuleVersionMap(ctx)
	fromVM[wasm.ModuleName] = 1 // start with initial version
	upgradeHandler(ctx, upgradetypes.Plan{Name: "testing"}, fromVM)
	// when
	gotVM, err := wasmApp.ModuleManager().RunMigrations(ctx, wasmApp.ModuleConfigurator(), fromVM)
	// then
	require.NoError(t, err)
	assert.Equal(t, uint64(1), gotVM[wasm.ModuleName])
}
