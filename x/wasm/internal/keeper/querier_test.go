package keeper

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"testing"

	"github.com/CosmWasm/wasmd/x/wasm/internal/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkErrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestQueryAllContractState(t *testing.T) {
	ctx, keepers := CreateTestInput(t, false, SupportedFeatures, nil, nil)
	keeper := keepers.WasmKeeper

	exampleContract := InstantiateHackatomExampleContract(t, ctx, keepers)
	contractAddr := exampleContract.Contract
	contractModel := []types.Model{
		{Key: []byte("foo"), Value: []byte(`"bar"`)},
		{Key: []byte{0x0, 0x1}, Value: []byte(`{"count":8}`)},
	}
	require.NoError(t, keeper.importContractState(ctx, contractAddr, contractModel))

	q := NewQuerier(keeper)
	specs := map[string]struct {
		srcQuery         *types.QueryAllContractStateRequest
		expModelContains []types.Model
		expErr           *sdkErrors.Error
	}{
		"query all": {
			srcQuery:         &types.QueryAllContractStateRequest{Address: contractAddr.String()},
			expModelContains: contractModel,
		},
		"query all with unknown address": {
			srcQuery: &types.QueryAllContractStateRequest{Address: RandomBech32AccountAddress(t)},
			expErr:   types.ErrNotFound,
		},
	}
	for msg, spec := range specs {
		t.Run(msg, func(t *testing.T) {
			got, err := q.AllContractState(sdk.WrapSDKContext(ctx), spec.srcQuery)
			require.True(t, spec.expErr.Is(err), err)
			if spec.expErr != nil {
				return
			}
			for _, exp := range spec.expModelContains {
				assert.Contains(t, got.Models, exp)
			}
		})
	}
}

func TestQuerySmartContractState(t *testing.T) {
	ctx, keepers := CreateTestInput(t, false, SupportedFeatures, nil, nil)
	keeper := keepers.WasmKeeper

	exampleContract := InstantiateHackatomExampleContract(t, ctx, keepers)
	contractAddr := exampleContract.Contract.String()

	q := NewQuerier(keeper)
	specs := map[string]struct {
		srcAddr  sdk.AccAddress
		srcQuery *types.QuerySmartContractStateRequest
		expResp  string
		expErr   *sdkErrors.Error
	}{
		"query smart": {
			srcQuery: &types.QuerySmartContractStateRequest{Address: contractAddr, QueryData: []byte(`{"verifier":{}}`)},
			expResp:  fmt.Sprintf(`{"verifier":"%s"}`, exampleContract.VerifierAddr.String()),
		},
		"query smart invalid request": {
			srcQuery: &types.QuerySmartContractStateRequest{Address: contractAddr, QueryData: []byte(`{"raw":{"key":"config"}}`)},
			expErr:   types.ErrQueryFailed,
		},
		"query smart with invalid json": {
			srcQuery: &types.QuerySmartContractStateRequest{Address: contractAddr, QueryData: []byte(`not a json string`)},
			expErr:   types.ErrQueryFailed,
		},
		"query smart with unknown address": {
			srcQuery: &types.QuerySmartContractStateRequest{Address: RandomBech32AccountAddress(t), QueryData: []byte(`{"verifier":{}}`)},
			expErr:   types.ErrNotFound,
		},
	}
	for msg, spec := range specs {
		t.Run(msg, func(t *testing.T) {
			got, err := q.SmartContractState(sdk.WrapSDKContext(ctx), spec.srcQuery)
			require.True(t, spec.expErr.Is(err), err)
			if spec.expErr != nil {
				return
			}
			assert.JSONEq(t, string(got.Data), spec.expResp)
		})
	}
}

func TestQueryRawContractState(t *testing.T) {
	ctx, keepers := CreateTestInput(t, false, SupportedFeatures, nil, nil)
	keeper := keepers.WasmKeeper

	exampleContract := InstantiateHackatomExampleContract(t, ctx, keepers)
	contractAddr := exampleContract.Contract.String()
	contractModel := []types.Model{
		{Key: []byte("foo"), Value: []byte(`"bar"`)},
		{Key: []byte{0x0, 0x1}, Value: []byte(`{"count":8}`)},
	}
	require.NoError(t, keeper.importContractState(ctx, exampleContract.Contract, contractModel))

	q := NewQuerier(keeper)
	specs := map[string]struct {
		srcQuery *types.QueryRawContractStateRequest
		expData  []byte
		expErr   *sdkErrors.Error
	}{
		"query raw key": {
			srcQuery: &types.QueryRawContractStateRequest{Address: contractAddr, QueryData: []byte("foo")},
			expData:  []byte(`"bar"`),
		},
		"query raw binary key": {
			srcQuery: &types.QueryRawContractStateRequest{Address: contractAddr, QueryData: []byte{0x0, 0x1}},
			expData:  []byte(`{"count":8}`),
		},
		"query non-existent raw key": {
			srcQuery: &types.QueryRawContractStateRequest{Address: contractAddr, QueryData: []byte("not existing key")},
			expData:  nil,
		},
		"query empty raw key": {
			srcQuery: &types.QueryRawContractStateRequest{Address: contractAddr, QueryData: []byte("")},
			expData:  nil,
		},
		"query nil raw key": {
			srcQuery: &types.QueryRawContractStateRequest{Address: contractAddr},
			expData:  nil,
		},
		"query raw with unknown address": {
			srcQuery: &types.QueryRawContractStateRequest{Address: RandomBech32AccountAddress(t), QueryData: []byte("foo")},
			expErr:   types.ErrNotFound,
		},
	}
	for msg, spec := range specs {
		t.Run(msg, func(t *testing.T) {
			got, err := q.RawContractState(sdk.WrapSDKContext(ctx), spec.srcQuery)
			require.True(t, spec.expErr.Is(err), err)
			if spec.expErr != nil {
				return
			}
			assert.Equal(t, spec.expData, got.Data)
		})
	}
}

func TestQueryContractListByCodeOrdering(t *testing.T) {
	ctx, keepers := CreateTestInput(t, false, SupportedFeatures, nil, nil)
	accKeeper, keeper, bankKeeper := keepers.AccountKeeper, keepers.WasmKeeper, keepers.BankKeeper

	deposit := sdk.NewCoins(sdk.NewInt64Coin("denom", 1000000))
	topUp := sdk.NewCoins(sdk.NewInt64Coin("denom", 500))
	creator := createFakeFundedAccount(t, ctx, accKeeper, bankKeeper, deposit)
	anyAddr := createFakeFundedAccount(t, ctx, accKeeper, bankKeeper, topUp)

	wasmCode, err := ioutil.ReadFile("./testdata/hackatom.wasm")
	require.NoError(t, err)

	codeID, err := keeper.Create(ctx, creator, wasmCode, "", "", nil)
	require.NoError(t, err)

	_, _, bob := keyPubAddr()
	initMsg := HackatomExampleInitMsg{
		Verifier:    anyAddr,
		Beneficiary: bob,
	}
	initMsgBz, err := json.Marshal(initMsg)
	require.NoError(t, err)

	// manage some realistic block settings
	var h int64 = 10
	setBlock := func(ctx sdk.Context, height int64) sdk.Context {
		ctx = ctx.WithBlockHeight(height)
		meter := sdk.NewGasMeter(1000000)
		ctx = ctx.WithGasMeter(meter)
		ctx = ctx.WithBlockGasMeter(meter)
		return ctx
	}

	// create 10 contracts with real block/gas setup
	for i := range [10]int{} {
		// 3 tx per block, so we ensure both comparisons work
		if i%3 == 0 {
			ctx = setBlock(ctx, h)
			h++
		}
		_, err = keeper.Instantiate(ctx, codeID, creator, nil, initMsgBz, fmt.Sprintf("contract %d", i), topUp)
		require.NoError(t, err)
	}

	// query and check the results are properly sorted
	q := NewQuerier(keeper)
	res, err := q.ContractsByCode(sdk.WrapSDKContext(ctx), &types.QueryContractsByCodeRequest{CodeId: codeID})
	require.NoError(t, err)

	require.Equal(t, 10, len(res.ContractInfos))

	for i, contract := range res.ContractInfos {
		assert.Equal(t, fmt.Sprintf("contract %d", i), contract.Label)
		assert.NotEmpty(t, contract.Address)
		// ensure these are not shown
		assert.Nil(t, contract.Created)
	}
}

func TestQueryContractHistory(t *testing.T) {
	ctx, keepers := CreateTestInput(t, false, SupportedFeatures, nil, nil)
	keeper := keepers.WasmKeeper

	var (
		otherAddr sdk.AccAddress = bytes.Repeat([]byte{0x2}, sdk.AddrLen)
	)

	specs := map[string]struct {
		srcQueryAddr sdk.AccAddress
		srcHistory   []types.ContractCodeHistoryEntry
		expContent   []types.ContractCodeHistoryEntry
	}{
		"response with internal fields cleared": {
			srcHistory: []types.ContractCodeHistoryEntry{{
				Operation: types.ContractCodeHistoryOperationTypeGenesis,
				CodeID:    firstCodeID,
				Updated:   types.NewAbsoluteTxPosition(ctx),
				Msg:       []byte(`"init message"`),
			}},
			expContent: []types.ContractCodeHistoryEntry{{
				Operation: types.ContractCodeHistoryOperationTypeGenesis,
				CodeID:    firstCodeID,
				Msg:       []byte(`"init message"`),
			}},
		},
		"response with multiple entries": {
			srcHistory: []types.ContractCodeHistoryEntry{{
				Operation: types.ContractCodeHistoryOperationTypeInit,
				CodeID:    firstCodeID,
				Updated:   types.NewAbsoluteTxPosition(ctx),
				Msg:       []byte(`"init message"`),
			}, {
				Operation: types.ContractCodeHistoryOperationTypeMigrate,
				CodeID:    2,
				Updated:   types.NewAbsoluteTxPosition(ctx),
				Msg:       []byte(`"migrate message 1"`),
			}, {
				Operation: types.ContractCodeHistoryOperationTypeMigrate,
				CodeID:    3,
				Updated:   types.NewAbsoluteTxPosition(ctx),
				Msg:       []byte(`"migrate message 2"`),
			}},
			expContent: []types.ContractCodeHistoryEntry{{
				Operation: types.ContractCodeHistoryOperationTypeInit,
				CodeID:    firstCodeID,
				Msg:       []byte(`"init message"`),
			}, {
				Operation: types.ContractCodeHistoryOperationTypeMigrate,
				CodeID:    2,
				Msg:       []byte(`"migrate message 1"`),
			}, {
				Operation: types.ContractCodeHistoryOperationTypeMigrate,
				CodeID:    3,
				Msg:       []byte(`"migrate message 2"`),
			}},
		},
		"unknown contract address": {
			srcQueryAddr: otherAddr,
			srcHistory: []types.ContractCodeHistoryEntry{{
				Operation: types.ContractCodeHistoryOperationTypeGenesis,
				CodeID:    firstCodeID,
				Updated:   types.NewAbsoluteTxPosition(ctx),
				Msg:       []byte(`"init message"`),
			}},
			expContent: nil,
		},
	}
	for msg, spec := range specs {
		t.Run(msg, func(t *testing.T) {
			_, _, myContractAddr := keyPubAddr()
			keeper.appendToContractHistory(ctx, myContractAddr, spec.srcHistory...)

			queryContractAddr := spec.srcQueryAddr
			if queryContractAddr == nil {
				queryContractAddr = myContractAddr
			}
			req := &types.QueryContractHistoryRequest{Address: queryContractAddr.String()}

			// when
			q := NewQuerier(keeper)
			got, err := q.ContractHistory(sdk.WrapSDKContext(ctx), req)

			// then
			if spec.expContent == nil {
				require.Error(t, types.ErrEmpty)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, spec.expContent, got.Entries)
		})
	}
}

func TestQueryCodeList(t *testing.T) {
	wasmCode, err := ioutil.ReadFile("./testdata/hackatom.wasm")
	require.NoError(t, err)

	specs := map[string]struct {
		codeIDs []uint64
	}{
		"none": {},
		"no gaps": {
			codeIDs: []uint64{1, 2, 3},
		},
		"with gaps": {
			codeIDs: []uint64{2, 4, 6},
		},
	}

	for msg, spec := range specs {
		t.Run(msg, func(t *testing.T) {
			ctx, keepers := CreateTestInput(t, false, SupportedFeatures, nil, nil)
			keeper := keepers.WasmKeeper

			for _, codeID := range spec.codeIDs {
				require.NoError(t, keeper.importCode(ctx, codeID,
					types.CodeInfoFixture(types.WithSHA256CodeHash(wasmCode)),
					wasmCode),
				)
			}
			// when
			q := NewQuerier(keeper)
			got, err := q.Codes(sdk.WrapSDKContext(ctx), nil)

			// then
			if len(spec.codeIDs) == 0 {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)

			require.Len(t, got.CodeInfos, len(spec.codeIDs))
			for i, exp := range spec.codeIDs {
				assert.EqualValues(t, exp, got.CodeInfos[i].CodeID)
			}
		})
	}
}
