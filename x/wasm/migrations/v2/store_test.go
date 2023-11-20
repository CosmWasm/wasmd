package v2_test

import (
	"testing"

	dbm "github.com/cometbft/cometbft-db"
	"github.com/cometbft/cometbft/libs/log"
	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	"github.com/cosmos/cosmos-sdk/store"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	paramskeeper "github.com/cosmos/cosmos-sdk/x/params/keeper"
	paramstypes "github.com/cosmos/cosmos-sdk/x/params/types"

	"github.com/cometbft/cometbft/libs/rand"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/address"
	moduletestutil "github.com/cosmos/cosmos-sdk/types/module/testutil"

	"github.com/CosmWasm/wasmd/x/wasm"
	v2 "github.com/CosmWasm/wasmd/x/wasm/migrations/v2"
	"github.com/CosmWasm/wasmd/x/wasm/types"
)

func TestMigrate(t *testing.T) {
	cfg := moduletestutil.MakeTestEncodingConfig(wasm.AppModuleBasic{})
	cdc := cfg.Codec
	var (
		wasmStoreKey    = sdk.NewKVStoreKey(types.StoreKey)
		paramsStoreKey  = sdk.NewKVStoreKey(paramstypes.StoreKey)
		paramsTStoreKey = sdk.NewTransientStoreKey(paramstypes.TStoreKey)
		myAddress       = sdk.AccAddress(rand.Bytes(address.Len))
	)
	specs := map[string]struct {
		src v2.Params
	}{
		"one address": {
			src: v2.Params{
				CodeUploadAccess: v2.AccessConfig{
					Permission: v2.AccessTypeOnlyAddress,
					Address:    myAddress.String(),
				},
				InstantiateDefaultPermission: v2.AccessTypeNobody,
			},
		},
		"multiple addresses": {
			src: v2.Params{
				CodeUploadAccess: v2.AccessConfig{
					Permission: v2.AccessTypeAnyOfAddresses,
					Addresses:  []string{myAddress.String(), sdk.AccAddress(rand.Bytes(address.Len)).String()},
				},
				InstantiateDefaultPermission: v2.AccessTypeEverybody,
			},
		},
		"everybody": {
			src: v2.Params{
				CodeUploadAccess: v2.AccessConfig{
					Permission: v2.AccessTypeEverybody,
				},
				InstantiateDefaultPermission: v2.AccessTypeEverybody,
			},
		},
		"nobody": {
			src: v2.Params{
				CodeUploadAccess: v2.AccessConfig{
					Permission: v2.AccessTypeNobody,
				},
				InstantiateDefaultPermission: v2.AccessTypeNobody,
			},
		},
	}
	for name, spec := range specs {
		t.Run(name, func(t *testing.T) {
			paramsKeeper := paramskeeper.NewKeeper(cdc, cfg.Amino, paramsStoreKey, paramsTStoreKey)
			ctx := testContext(paramsTStoreKey, paramsStoreKey, wasmStoreKey)

			// register legacy parameters
			params := spec.src
			subspace := paramsKeeper.Subspace(types.ModuleName)
			subspace.WithKeyTable(v2.ParamKeyTable())
			subspace.SetParamSet(ctx, &params)

			// when
			require.NoError(t, v2.MigrateStore(ctx, wasmStoreKey, subspace, cdc))

			var res v2.Params
			bz := ctx.KVStore(wasmStoreKey).Get(types.ParamsKey)
			require.NoError(t, cdc.Unmarshal(bz, &res))
			assert.Equal(t, params, res)
		})
	}
}

func testContext(tkey storetypes.StoreKey, keys ...storetypes.StoreKey) sdk.Context {
	db := dbm.NewMemDB()
	cms := store.NewCommitMultiStore(db)
	for _, key := range keys {
		cms.MountStoreWithDB(key, storetypes.StoreTypeIAVL, db)
	}
	cms.MountStoreWithDB(tkey, storetypes.StoreTypeTransient, db)
	err := cms.LoadLatestVersion()
	if err != nil {
		panic(err)
	}
	ctx := sdk.NewContext(cms, tmproto.Header{}, false, log.NewNopLogger())

	return ctx
}
