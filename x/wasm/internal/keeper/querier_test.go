package keeper

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmwasm/wasmd/x/wasm/internal/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	abci "github.com/tendermint/tendermint/abci/types"
)

func TestQueryContractState(t *testing.T) {
	type model struct {
		Key   string `json:"key"`
		Value string `json:"val"`
	}

	tempDir, err := ioutil.TempDir("", "wasm")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)
	ctx, accKeeper, keeper := CreateTestInput(t, false, tempDir)

	deposit := sdk.NewCoins(sdk.NewInt64Coin("denom", 100000))
	topUp := sdk.NewCoins(sdk.NewInt64Coin("denom", 5000))
	creator := createFakeFundedAccount(ctx, accKeeper, deposit.Add(deposit))
	anyAddr := createFakeFundedAccount(ctx, accKeeper, topUp)

	wasmCode, err := ioutil.ReadFile("./testdata/contract.wasm")
	require.NoError(t, err)

	contractID, err := keeper.Create(ctx, creator, wasmCode)
	require.NoError(t, err)

	_, _, bob := keyPubAddr()
	initMsg := InitMsg{
		Verifier:    anyAddr,
		Beneficiary: bob,
	}
	initMsgBz, err := json.Marshal(initMsg)
	require.NoError(t, err)

	addr, err := keeper.Instantiate(ctx, creator, contractID, initMsgBz, deposit)
	require.NoError(t, err)
	require.Equal(t, "cosmos18vd8fpwxzck93qlwghaj6arh4p7c5n89uzcee5", addr.String())

	contractModel := []types.Model{
		{Key: "foo", Value: "bar"},
		{Key: string([]byte{0x0, 0x1}), Value: string([]byte{0x2, 0x3})},
	}
	keeper.setContractState(ctx, addr, contractModel)

	q := NewQuerier(keeper)
	specs := map[string]struct {
		srcPath          []string
		srcReq           abci.RequestQuery
		expModelLen      int
		expModelContains []model
		expErr           sdk.Error
	}{
		"query all": {
			srcPath:     []string{QueryGetContractState, addr.String(), QueryMethodContractStateAll},
			expModelLen: 3,
			expModelContains: []model{
				{Key: "foo", Value: "bar"},
				{Key: string([]byte{0x0, 0x1}), Value: string([]byte{0x2, 0x3})},
			},
		},
		"query raw key": {
			srcPath:          []string{QueryGetContractState, addr.String(), QueryMethodContractStateRaw},
			srcReq:           abci.RequestQuery{Data: []byte("foo")},
			expModelLen:      1,
			expModelContains: []model{{Key: "foo", Value: "bar"}},
		},
		"query raw binary key": {
			srcPath:          []string{QueryGetContractState, addr.String(), QueryMethodContractStateRaw},
			srcReq:           abci.RequestQuery{Data: []byte{0x0, 0x1}},
			expModelLen:      1,
			expModelContains: []model{{Key: string([]byte{0x0, 0x1}), Value: string([]byte{0x2, 0x3})}},
		},
		"query smart": {
			srcPath:     []string{QueryGetContractState, addr.String(), QueryMethodContractStateSmart},
			srcReq:      abci.RequestQuery{Data: []byte(`{"raw":{"key":"config"}}`)},
			expModelLen: 1,
			//expModelContains: []model{}, // stopping here as contract internals are not stable
		},
		"query unknown raw key": {
			srcPath:     []string{QueryGetContractState, addr.String(), QueryMethodContractStateRaw},
			srcReq:      abci.RequestQuery{Data: []byte("unknown")},
			expModelLen: 0,
		},
		"query empty raw key": {
			srcPath:     []string{QueryGetContractState, addr.String(), QueryMethodContractStateRaw},
			expModelLen: 0,
		},
		"query raw with unknown address": {
			srcPath:     []string{QueryGetContractState, anyAddr.String(), QueryMethodContractStateRaw},
			expModelLen: 0,
		},
		"query all with unknown address": {
			srcPath:     []string{QueryGetContractState, anyAddr.String(), QueryMethodContractStateAll},
			expModelLen: 0,
		},
		"query smart with unknown address": {
			srcPath:     []string{QueryGetContractState, anyAddr.String(), QueryMethodContractStateSmart},
			expModelLen: 0,
			expErr:      types.ErrNotFound("contract"),
		},
	}

	for msg, spec := range specs {
		t.Run(msg, func(t *testing.T) {
			binResult, err := q(ctx, spec.srcPath, spec.srcReq)
			require.Equal(t, spec.expErr, err, "foo")
			// require.Equal(t, spec.expErr, err, err.Error())
			// then
			var r []model
			if spec.expErr == nil {
				require.NoError(t, json.Unmarshal(binResult, &r))
				require.NotNil(t, r)
			}
			require.Len(t, r, spec.expModelLen)
			// and in result set
			for _, v := range spec.expModelContains {
				assert.Contains(t, r, v)
			}
		})
	}
}
