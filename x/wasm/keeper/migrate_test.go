package keeper

import (
	"bytes"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/address"
	"github.com/stretchr/testify/require"

	legacytypes "github.com/CosmWasm/wasmd/x/wasm/types/legacy"
)

// integration testing of smart contract
func TestMigrate1To2(t *testing.T) {
	ctx, keepers := CreateTestInput(t, false, AvailableCapabilities)
	wasmKeeper := keepers.WasmKeeper

	deposit := sdk.NewCoins(sdk.NewInt64Coin("denom", 100000))
	creator := sdk.AccAddress(bytes.Repeat([]byte{1}, address.Len))
	keepers.Faucet.Fund(ctx, creator, deposit...)
	newLegacyContract(wasmKeeper, creator, ctx, t)

	// migrator
	migrator := NewMigrator(*wasmKeeper)
	migrator.Migrate1to2(ctx)
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

// go test -v -run ^TestAddContractCodeHistorySubStore$ github.com/CosmWasm/wasmd/x/wasm/keeper
func TestAddContractCodeHistorySubStore(t *testing.T) {
	ctx, keepers := CreateTestInput(t, false, AvailableCapabilities)
	wasmKeeper := keepers.WasmKeeper
	faucet := keepers.Faucet

	creator := getFundedAccount(ctx, faucet)

	// instantiate 3 legacy contracts
	legacyContract := newLegacyContract(wasmKeeper, creator, ctx, t)

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

func newLegacyContract(wasmkeeper *Keeper, creator sdk.AccAddress, ctx sdk.Context, t *testing.T) legacytypes.ContractInfo {
	t.Helper()

	contractAddress := RandomAccountAddress(t)
	contract := legacytypes.NewContractInfo(1, contractAddress, creator, creator, []byte("init"))
	wasmkeeper.SetLegacyContractInfo(ctx, contractAddress, contract)

	return contract
}
