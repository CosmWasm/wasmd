package v2_test

import (
	"testing"

	dbm "github.com/cosmos/cosmos-db"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"cosmossdk.io/log"
	"cosmossdk.io/math/unsafe"
	"cosmossdk.io/store"
	"cosmossdk.io/store/metrics"
	storetypes "cosmossdk.io/store/types"
	paramskeeper "cosmossdk.io/x/params/keeper"
	paramstypes "cosmossdk.io/x/params/types"

	codectestutil "github.com/cosmos/cosmos-sdk/codec/testutil"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/address"
	moduletestutil "github.com/cosmos/cosmos-sdk/types/module/testutil"

	"github.com/CosmWasm/wasmd/x/wasm"
	v2 "github.com/CosmWasm/wasmd/x/wasm/migrations/v2"
	"github.com/CosmWasm/wasmd/x/wasm/types"
)

func TestMigrate(t *testing.T) {
	cfg := moduletestutil.MakeTestEncodingConfig(codectestutil.CodecOptions{}, wasm.AppModule{})
	cdc := cfg.Codec
	var (
		wasmStoreKey    = storetypes.NewKVStoreKey(types.StoreKey)
		paramsStoreKey  = storetypes.NewKVStoreKey(paramstypes.StoreKey)
		paramsTStoreKey = storetypes.NewTransientStoreKey(paramstypes.TStoreKey)
		myAddress       = sdk.AccAddress(unsafe.Bytes(address.Len))
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
					Addresses:  []string{myAddress.String(), sdk.AccAddress(unsafe.Bytes(address.Len)).String()},
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
			ctx := defaultContextWithKeys(
				map[string]*storetypes.KVStoreKey{
					paramstypes.StoreKey: paramsStoreKey,
					types.StoreKey:       wasmStoreKey,
				},
				map[string]*storetypes.TransientStoreKey{
					paramstypes.TStoreKey: paramsTStoreKey,
				},
				nil,
			)

			// register legacy parameters
			params := spec.src
			subspace := paramsKeeper.Subspace(types.ModuleName)
			subspace.WithKeyTable(v2.ParamKeyTable())
			subspace.SetParamSet(ctx, &params)

			// when
			require.NoError(t, v2.MigrateStore(ctx, runtime.NewKVStoreService(wasmStoreKey), subspace, cdc))

			var res v2.Params
			bz := ctx.KVStore(wasmStoreKey).Get(types.ParamsKey)
			require.NoError(t, cdc.Unmarshal(bz, &res))
			assert.Equal(t, params, res)
		})
	}
}

func defaultContextWithKeys(
	keys map[string]*storetypes.KVStoreKey,
	transKeys map[string]*storetypes.TransientStoreKey,
	memKeys map[string]*storetypes.MemoryStoreKey,
) sdk.Context {
	db := dbm.NewMemDB()
	cms := store.NewCommitMultiStore(db, log.NewNopLogger(), metrics.NewNoOpMetrics())

	for _, key := range keys {
		cms.MountStoreWithDB(key, storetypes.StoreTypeIAVL, db)
	}

	for _, tKey := range transKeys {
		cms.MountStoreWithDB(tKey, storetypes.StoreTypeTransient, db)
	}

	for _, memkey := range memKeys {
		cms.MountStoreWithDB(memkey, storetypes.StoreTypeMemory, db)
	}

	err := cms.LoadLatestVersion()
	if err != nil {
		panic(err)
	}

	return sdk.NewContext(cms, false, log.NewNopLogger())
}
