package keeper

import (
	"bytes"
	"encoding/json"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/address"
	"github.com/stretchr/testify/require"

	"github.com/CosmWasm/wasmd/x/wasm/types"
	legacytypes "github.com/CosmWasm/wasmd/x/wasm/types/legacy"
)

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
	gotContractAddr1, _, err := keepers.ContractKeeper.Instantiate(ctx.WithEventManager(em), example.CodeID, creator, nil, initMsgBz, nil)
	ctx = ctx.WithBlockHeight(ctx.BlockHeight() + 1)
	gotContractAddr2, _, err := keepers.ContractKeeper.Instantiate(ctx.WithEventManager(em), example.CodeID, creator, nil, initMsgBz, nil)
	ctx = ctx.WithBlockHeight(ctx.BlockHeight() + 1)
	gotContractAddr3, _, err := keepers.ContractKeeper.Instantiate(ctx.WithEventManager(em), example.CodeID, creator, nil, initMsgBz, nil)

	info1 := wasmKeeper.GetContractInfo(ctx, gotContractAddr1)
	info2 := wasmKeeper.GetContractInfo(ctx, gotContractAddr2)
	info3 := wasmKeeper.GetContractInfo(ctx, gotContractAddr3)

	// remove key
	ctx.KVStore(wasmKeeper.storeKey).Delete(types.GetContractByCreatorSecondaryIndexKey(creator, info1.Created.Bytes(), gotContractAddr1))
	ctx.KVStore(wasmKeeper.storeKey).Delete(types.GetContractByCreatorSecondaryIndexKey(creator, info2.Created.Bytes(), gotContractAddr2))
	ctx.KVStore(wasmKeeper.storeKey).Delete(types.GetContractByCreatorSecondaryIndexKey(creator, info3.Created.Bytes(), gotContractAddr3))

	// migrator
	migrator := NewMigrator(*wasmKeeper)
	migrator.Migrate1to2(ctx)

	// check new store
	var allContract []string
	wasmKeeper.IterateContractsByCreator(ctx, creator, func(addr sdk.AccAddress) bool {
		allContract = append(allContract, addr.String())
		return false
	})

	require.Equal(t, []string{gotContractAddr1.String(), gotContractAddr2.String(), gotContractAddr3.String()}, allContract)
}

// test migrate legacy contract to include AbsoluteTxPosition
// go test -v -run ^TestMigrateAbsoluteTx$ github.com/CosmWasm/wasmd/x/wasm/keeper
func TestMigrateAbsoluteTx(t *testing.T) {
	ctx, keepers := CreateTestInput(t, false, AvailableCapabilities)
	wasmKeeper := keepers.WasmKeeper
	faucet := keepers.Faucet

	creator := getFundedAccount(ctx, faucet)

	// instantiate legacy contract
	legacyContract := newLegacyContract(wasmKeeper, creator, ctx, t)

	// migrator
	migrator := NewMigrator(*wasmKeeper)
	migrator.migrateAbsoluteTx(ctx, legacyContract)

	// check structure after migration not nil
	contractAddress := sdk.MustAccAddressFromBech32(legacyContract.Address)
	newContract := wasmKeeper.GetContractInfo(ctx, contractAddress)
	require.NotNil(t, newContract)
}

func getFundedAccount(ctx sdk.Context, faucet *TestFaucet) sdk.AccAddress {
	deposit := sdk.NewCoins(sdk.NewInt64Coin("uluna", 1000000))
	creator := sdk.AccAddress(bytes.Repeat([]byte{1}, address.Len))
	faucet.Fund(ctx, creator, deposit...)

	return creator
}

func newLegacyContract(wasmkeeper *Keeper, creator sdk.AccAddress, ctx sdk.Context, t *testing.T) legacytypes.ContractInfo {
	t.Helper()

	contractAddress := RandomAccountAddress(t)
	contract := legacytypes.NewContractInfo(1, contractAddress, creator, creator, []byte("init"))
	wasmkeeper.SetLegacyContractInfo(ctx, contractAddress, contract)

	return contract
}
