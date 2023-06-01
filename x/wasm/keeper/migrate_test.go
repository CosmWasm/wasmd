package keeper

import (
	"bytes"
	"encoding/binary"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/address"
	"github.com/stretchr/testify/require"

	types "github.com/CosmWasm/wasmd/x/wasm/types"
	legacytypes "github.com/CosmWasm/wasmd/x/wasm/types/legacy"
)

func TestMigrateCodeFromLegacy(t *testing.T) {

	ctx, keepers := CreateTestInput(t, false, AvailableCapabilities)
	wasmkeeper := keepers.WasmKeeper
	migrator := NewMigrator(*wasmkeeper)
	creator := sdk.AccAddress(bytes.Repeat([]byte{1}, address.Len))
	id := uint64(12345)
	hash := []byte("12345")

	err := migrator.migrateCodeFromLegacy(ctx, creator, id, hash)
	require.NoError(t, err)

	// retrieve codeInfo per ID
	codeInfo := wasmkeeper.GetCodeInfo(ctx, id)
	require.NotEqual(t, codeInfo, nil, "Empty codeInfo after code migration")

	// check fields in codeInfo
	require.Equal(t, hash, codeInfo.CodeHash, "Wrong hash after code migration")
	require.Equal(t, creator.String(), codeInfo.Creator, "Wrong code creator after code migration")
	require.Equal(t, codeInfo.InstantiateConfig, wasmkeeper.getInstantiateAccessConfig(ctx).With(creator), "Wrong InstantiateAccessConfig after code migration")

}

// integration testing of smart contract
func TestMigrate1To2(t *testing.T) {
	ctx, keepers := CreateTestInput(t, false, AvailableCapabilities)
	wasmKeeper := keepers.WasmKeeper

	deposit := sdk.NewCoins(sdk.NewInt64Coin("denom", 100000))
	creator := sdk.AccAddress(bytes.Repeat([]byte{1}, address.Len))
	keepers.Faucet.Fund(ctx, creator, deposit...)
	newLegacyContract(wasmKeeper, ctx, creator, t)

	// migrator
	migrator := NewMigrator(*wasmKeeper)
	err := migrator.Migrate1to2(ctx)
	require.NoError(t, err)

	// label must equal address and no empty admin
	wasmKeeper.IterateContractInfo(ctx, func(addr sdk.AccAddress, info types.ContractInfo) bool {
		require.Equal(t, info.Label, addr.String())
		require.NotEqual(t, info.Admin, "")
		return false
	})
}

// test migrate legacy contract to include AbsoluteTxPosition
// go test -v -run ^TestMigrateAbsoluteTx$ github.com/CosmWasm/wasmd/x/wasm/keeper
func TestMigrateAbsoluteTx(t *testing.T) {
	ctx, keepers := CreateTestInput(t, false, AvailableCapabilities)
	wasmKeeper := keepers.WasmKeeper
	faucet := keepers.Faucet

	creator := getFundedAccount(ctx, faucet)

	// instantiate legacy contract
	legacyContract := newLegacyContract(wasmKeeper, ctx, creator, t)

	// migrator
	migrator := NewMigrator(*wasmKeeper)
	migrator.migrateAbsoluteTx(ctx, legacyContract)

	// check structure after migration not nil
	contractAddress := sdk.MustAccAddressFromBech32(legacyContract.Address)
	newContract := wasmKeeper.GetContractInfo(ctx, contractAddress)
	require.NotNil(t, newContract)
}

// go test -v -run ^TestAddContractCodeHistorySubStore$ github.com/CosmWasm/wasmd/x/wasm/keeper
func TestAddContractCodeHistorySubStore(t *testing.T) {
	ctx, keepers := CreateTestInput(t, false, AvailableCapabilities)
	wasmKeeper := keepers.WasmKeeper
	faucet := keepers.Faucet

	creator := getFundedAccount(ctx, faucet)

	// instantiate 3 legacy contracts
	legacyContract := newLegacyContract(wasmKeeper, ctx, creator, t)

	// migrator
	migrator := NewMigrator(*wasmKeeper)
	wasmKeeper.IterateLegacyContractInfo(ctx, func(contractInfo legacytypes.ContractInfo) bool {
		newContract := migrator.migrateAbsoluteTx(ctx, contractInfo)
		contractAddress := sdk.MustAccAddressFromBech32(contractInfo.Address)
		migrator.keeper.appendToContractHistory(ctx, contractAddress, newContract.InitialHistory(contractInfo.InitMsg))
		return false
	})

	// check query after migration is populated
	res := wasmKeeper.GetContractHistory(ctx, sdk.MustAccAddressFromBech32(legacyContract.Address))
	require.Equal(t, 1, len(res))
}

func getFundedAccount(ctx sdk.Context, faucet *TestFaucet) sdk.AccAddress {
	deposit := sdk.NewCoins(sdk.NewInt64Coin("uluna", 1000000))
	creator := sdk.AccAddress(bytes.Repeat([]byte{1}, address.Len))
	faucet.Fund(ctx, creator, deposit...)

	return creator
}

func newLegacyContract(wasmkeeper *Keeper, ctx sdk.Context, creator sdk.AccAddress, t *testing.T) legacytypes.ContractInfo {
	t.Helper()

	contractAddress := RandomAccountAddress(t)
	contract := legacytypes.NewContractInfo(1, contractAddress, creator, creator, []byte("init"))
	wasmkeeper.SetLegacyContractInfo(ctx, contractAddress, contract)

	contractAddress = RandomAccountAddress(t)
	contract = legacytypes.NewContractInfo(2, contractAddress, creator, sdk.AccAddress{}, []byte("init"))
	wasmkeeper.SetLegacyContractInfo(ctx, contractAddress, contract)

	return contract
}

// StoreCodeLegacy stores a legacy code info into the code info store
func newLegacyCode(wasmkeeper *Keeper, ctx sdk.Context, id uint64, creator sdk.AccAddress, hash []byte) legacytypes.CodeInfo {

	codeInfo := legacytypes.NewCodeInfo(id, creator, hash)
	wasmkeeper.SetLegacyCodeInfo(ctx, id, codeInfo)
	wasmkeeper.Logger(ctx).Debug("storing new contract", "code_id", id)

	return codeInfo
}

// newCodeInfoLegacy stores CodeInfo for the given codeID in legacy store
func newCodeInfoLegacy(wasmkeeper *Keeper, ctx sdk.Context, codeID uint64, codeInfo legacytypes.CodeInfo) {
	store := ctx.KVStore(wasmkeeper.storeKey)
	bz := wasmkeeper.cdc.MustMarshal(&codeInfo)
	store.Set(types.GetCodeKey(codeID), bz)
}

// SetLastCodeID sets last code id in legacy store
func setLastCodeIDLegacy(wasmkeeper *Keeper, ctx sdk.Context, id uint64) {
	store := ctx.KVStore(wasmkeeper.storeKey)
	bz := sdk.Uint64ToBigEndian(id)
	store.Set(legacytypes.LastCodeIDKey, bz)
}

// GetLastCodeID return last code ID from legacy store
func getLastCodeIDLegacy(wasmkeeper *Keeper, ctx sdk.Context) (uint64, error) {
	store := ctx.KVStore(wasmkeeper.storeKey)
	bz := store.Get(legacytypes.LastCodeIDKey)
	if bz == nil {
		// if it is not set we set it here
		// normally this would have been set
		// on genesis - but we don't have that
		// for legacy wasm
		setLastCodeIDLegacy(wasmkeeper, ctx, 1)
		return 1, nil
	}

	return binary.BigEndian.Uint64(bz), nil
}
