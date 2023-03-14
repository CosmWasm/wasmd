package keeper

import (
	"bytes"
	"encoding/json"
	"testing"

	dbm "github.com/cometbft/cometbft-db"
	"github.com/cometbft/cometbft/libs/log"
	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	"github.com/cosmos/cosmos-sdk/store"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/address"
	"github.com/stretchr/testify/require"

	"github.com/CosmWasm/wasmd/x/wasm/exported"
	"github.com/CosmWasm/wasmd/x/wasm/types"
)

type mockSubspace struct {
	ps types.Params
}

func newMockSubspace(ps types.Params) mockSubspace {
	return mockSubspace{ps: ps}
}

func (ms mockSubspace) GetParamSet(ctx sdk.Context, ps exported.ParamSet) {
	*ps.(*types.Params) = ms.ps
}

func TestMigrate1To2(t *testing.T) {
	ctx, keepers := CreateTestInput(t, false, AvailableCapabilities)
	wasmKeeper := keepers.WasmKeeper

	deposit := sdk.NewCoins(sdk.NewInt64Coin("denom", 100000))
	creator := sdk.AccAddress(bytes.Repeat([]byte{1}, address.Len))
	keepers.Faucet.Fund(ctx, creator, deposit...)
	example := StoreHackatomExampleContract(t, ctx, keepers)

	initMsg := HackatomExampleInitMsg{
		Verifier:    RandomAccountAddress(t),
		Beneficiary: RandomAccountAddress(t),
	}
	initMsgBz, err := json.Marshal(initMsg)
	require.NoError(t, err)

	em := sdk.NewEventManager()

	// create with no balance is also legal
	gotContractAddr1, _, err := keepers.ContractKeeper.Instantiate(ctx.WithEventManager(em), example.CodeID, creator, nil, initMsgBz, "demo contract 1", nil)
	ctx = ctx.WithBlockHeight(ctx.BlockHeight() + 1)
	gotContractAddr2, _, err := keepers.ContractKeeper.Instantiate(ctx.WithEventManager(em), example.CodeID, creator, nil, initMsgBz, "demo contract 1", nil)
	ctx = ctx.WithBlockHeight(ctx.BlockHeight() + 1)
	gotContractAddr3, _, err := keepers.ContractKeeper.Instantiate(ctx.WithEventManager(em), example.CodeID, creator, nil, initMsgBz, "demo contract 1", nil)

	info1 := wasmKeeper.GetContractInfo(ctx, gotContractAddr1)
	info2 := wasmKeeper.GetContractInfo(ctx, gotContractAddr2)
	info3 := wasmKeeper.GetContractInfo(ctx, gotContractAddr3)

	// remove key
	ctx.KVStore(wasmKeeper.storeKey).Delete(types.GetContractByCreatorSecondaryIndexKey(creator, info1.Created.Bytes(), gotContractAddr1))
	ctx.KVStore(wasmKeeper.storeKey).Delete(types.GetContractByCreatorSecondaryIndexKey(creator, info2.Created.Bytes(), gotContractAddr2))
	ctx.KVStore(wasmKeeper.storeKey).Delete(types.GetContractByCreatorSecondaryIndexKey(creator, info3.Created.Bytes(), gotContractAddr3))

	// legacy
	// migrator
	migrator := NewMigrator(*wasmKeeper, nil)
	err = migrator.Migrate1to2(ctx)
	require.NoError(t, err)

	// check new store
	var allContract []string
	wasmKeeper.IterateContractsByCreator(ctx, creator, func(addr sdk.AccAddress) bool {
		allContract = append(allContract, addr.String())
		return false
	})

	require.Equal(t, []string{gotContractAddr1.String(), gotContractAddr2.String(), gotContractAddr3.String()}, allContract)
}

func TestMigrate2To3(t *testing.T) {
	ctx, keepers := CreateTestInput(t, false, AvailableCapabilities)
	wasmKeeper := keepers.WasmKeeper

	storeKey := sdk.NewKVStoreKey("test-migration")
	tKey := sdk.NewTransientStoreKey("transient_test")
	ctx = defaultContext(storeKey, tKey)
	store := ctx.KVStore(storeKey)

	wasmKeeper.storeKey = storeKey
	legacySubspace := newMockSubspace(types.DefaultParams())

	//when
	migrator := NewMigrator(*wasmKeeper, legacySubspace)
	err := migrator.Migrate2to3(ctx)

	//then
	require.NoError(t, err)
	bz := store.Get(types.ParamsKey)

	var res types.Params
	require.NoError(t, wasmKeeper.cdc.Unmarshal(bz, &res))
	require.Equal(t, legacySubspace.ps, res)
}

// defaultContext creates a sdk.Context with a fresh MemDB that can be used in tests.
func defaultContext(key storetypes.StoreKey, tkey storetypes.StoreKey) sdk.Context {
	db := dbm.NewMemDB()
	cms := store.NewCommitMultiStore(db)
	cms.MountStoreWithDB(key, storetypes.StoreTypeIAVL, db)
	cms.MountStoreWithDB(tkey, storetypes.StoreTypeTransient, db)
	err := cms.LoadLatestVersion()
	if err != nil {
		panic(err)
	}
	ctx := sdk.NewContext(cms, tmproto.Header{}, false, log.NewNopLogger())

	return ctx
}
