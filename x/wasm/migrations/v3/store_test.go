package v3_test

import (
	"bytes"
	"testing"

	"github.com/CosmWasm/wasmd/x/wasm/keeper"
	"github.com/CosmWasm/wasmd/x/wasm/keeper/wasmtesting"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/address"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/CosmWasm/wasmd/x/wasm/types"
)

func TestMigrate3To4(t *testing.T) {
	const AvailableCapabilities = "iterator,staking,stargate,cosmwasm_1_1"
	ctx, keepers := keeper.CreateTestInput(t, false, AvailableCapabilities)
	wasmKeeper := keepers.WasmKeeper

	deposit := sdk.NewCoins(sdk.NewInt64Coin("denom", 100000))
	creator := sdk.AccAddress(bytes.Repeat([]byte{1}, address.Len))
	keepers.Faucet.Fund(ctx, creator, deposit...)

	var mock wasmtesting.MockWasmer
	wasmtesting.MakeInstantiable(&mock)

	// contract with only address permission
	onlyAddrPermission := types.AccessTypeOnlyAddress.With(creator)
	contract1 := keeper.StoreRandomContractWithAccessConfig(t, ctx, keepers, &mock, &onlyAddrPermission)

	// contract with any addresses permission
	anyAddrsPermission := types.AccessTypeAnyOfAddresses.With(creator)
	contract2 := keeper.StoreRandomContractWithAccessConfig(t, ctx, keepers, &mock, &anyAddrsPermission)

	// contract with everybody permission
	everybodyPermission := types.AllowEverybody
	contract3 := keeper.StoreRandomContractWithAccessConfig(t, ctx, keepers, &mock, &everybodyPermission)

	// contract with nobody permission
	nobodyPermission := types.AllowNobody
	contract4 := keeper.StoreRandomContractWithAccessConfig(t, ctx, keepers, &mock, &nobodyPermission)

	// set only address permission params
	params := types.Params{
		CodeUploadAccess:             types.AccessTypeOnlyAddress.With(creator),
		InstantiateDefaultPermission: types.AccessTypeOnlyAddress,
	}
	err := wasmKeeper.SetParams(ctx, params)
	require.NoError(t, err)

	// when
	err = keeper.NewMigrator(*wasmKeeper, nil).Migrate3to4(ctx)

	// then
	require.NoError(t, err)

	expParams := types.Params{
		CodeUploadAccess:             types.AccessTypeAnyOfAddresses.With(creator),
		InstantiateDefaultPermission: types.AccessTypeAnyOfAddresses,
	}

	// params are migrated
	assert.Equal(t, expParams, wasmKeeper.GetParams(ctx))

	// access config for only address is migrated
	info1 := wasmKeeper.GetCodeInfo(ctx, contract1.CodeID)
	assert.Equal(t, anyAddrsPermission, info1.InstantiateConfig)

	// access config for any addresses is not migrated
	info2 := wasmKeeper.GetCodeInfo(ctx, contract2.CodeID)
	assert.Equal(t, anyAddrsPermission, info2.InstantiateConfig)

	// access config for allow everybody is not migrated
	info3 := wasmKeeper.GetCodeInfo(ctx, contract3.CodeID)
	assert.Equal(t, types.AllowEverybody, info3.InstantiateConfig)

	// access config for allow nobody is not migrated
	info4 := wasmKeeper.GetCodeInfo(ctx, contract4.CodeID)
	assert.Equal(t, types.AllowNobody, info4.InstantiateConfig)
}
