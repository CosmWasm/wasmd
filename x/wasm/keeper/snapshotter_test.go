package keeper

import (
	"encoding/json"
	"fmt"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	dbm "github.com/tendermint/tm-db"

	"github.com/CosmWasm/wasmd/x/wasm/types"
)

func TestSnapshotRestoreMovesFile(t *testing.T) {
	// we hack this to "fake" copying over all the iavl data
	sharedDB := dbm.NewMemDB()

	ctx, keepers := createTestInput(t, false, SupportedFeatures, types.DefaultWasmConfig(), sharedDB)
	deposit := sdk.NewCoins(sdk.NewInt64Coin("denom", 100000))
	creator := keepers.Faucet.NewFundedAccount(ctx, deposit...)
	_, _, bob := keyPubAddr()
	_, _, fred := keyPubAddr()

	// create a contact
	codeID, err := keepers.ContractKeeper.Create(ctx, creator, hackatomWasm, nil)
	require.NoError(t, err)
	require.Equal(t, uint64(1), codeID)

	// instantiate it
	initMsg := HackatomExampleInitMsg{
		Verifier:    fred,
		Beneficiary: bob,
	}
	initMsgBz, err := json.Marshal(initMsg)
	require.NoError(t, err)
	contractAddr, _, err := keepers.ContractKeeper.Instantiate(ctx, codeID, creator, nil, initMsgBz, "demo contract 1", deposit)
	require.NoError(t, err)
	require.Equal(t, "cosmos14hj2tavq8fpesdwxxcu44rty3hh90vhujrvcmstl4zr3txmfvw9s4hmalr", contractAddr.String())

	// successfully query it
	queryBz := []byte(`{"verifier":{}}`)
	res, err := keepers.WasmKeeper.QuerySmart(ctx, contractAddr, queryBz)
	require.NoError(t, err)
	expected := fmt.Sprintf(`{"verifier":"%s"}`, fred.String())
	assert.JSONEq(t, string(res), expected)

	// failed attempt to copy state
	// // now, we make a new app with a copy of the "iavl" db, but no contracts
	// copyCtx, copyKeepers := createTestInput(t, false, SupportedFeatures, types.DefaultWasmConfig(), sharedDB)

	// // contract exists
	// info := copyKeepers.WasmKeeper.GetContractInfo(ctx, contractAddr)
	// require.NotNil(t, info)
	// require.Equal(t, info.CodeID, codeID)

	// // querying the existing contract errors, as there is no wasm file
	// res, err = copyKeepers.WasmKeeper.QuerySmart(copyCtx, contractAddr, queryBz)
	// require.Error(t, err)

}
