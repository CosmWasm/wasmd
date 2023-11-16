package v2_test

import (
	"testing"

	"github.com/cometbft/cometbft/libs/rand"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	storetypes "cosmossdk.io/store/types"

	"github.com/cosmos/cosmos-sdk/runtime"
	"github.com/cosmos/cosmos-sdk/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/address"
	moduletestutil "github.com/cosmos/cosmos-sdk/types/module/testutil"

	"github.com/CosmWasm/wasmd/x/wasm"
	"github.com/CosmWasm/wasmd/x/wasm/exported"
	v2 "github.com/CosmWasm/wasmd/x/wasm/migrations/v2"
	"github.com/CosmWasm/wasmd/x/wasm/types"
)

type mockSubspace struct {
	ps v2.Params
}

func newMockSubspace(ps v2.Params) mockSubspace {
	return mockSubspace{ps: ps}
}

func (ms mockSubspace) GetParamSet(ctx sdk.Context, ps exported.ParamSet) {
	*ps.(*v2.Params) = ms.ps
}

func TestMigrate(t *testing.T) {
	cdc := moduletestutil.MakeTestEncodingConfig(wasm.AppModuleBasic{}).Codec
	storeKey := storetypes.NewKVStoreKey(types.StoreKey)
	tKey := storetypes.NewTransientStoreKey("transient_test")
	ctx := testutil.DefaultContext(storeKey, tKey)
	store := ctx.KVStore(storeKey)

	myAddress := sdk.AccAddress(rand.Bytes(address.Len))
	params := v2.Params{
		CodeUploadAccess: v2.AccessConfig{
			Permission: v2.AccessTypeOnlyAddress,
			Address:    myAddress.String(),
		},
		InstantiateDefaultPermission: v2.AccessTypeNobody,
	}
	legacySubspace := newMockSubspace(params)
	// when
	require.NoError(t, v2.MigrateStore(ctx, runtime.NewKVStoreService(storeKey), legacySubspace, cdc))

	var res v2.Params
	bz := store.Get(types.ParamsKey)
	require.NoError(t, cdc.Unmarshal(bz, &res))
	assert.Equal(t, params, res)
}
