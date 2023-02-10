package wasmplus

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"

	"github.com/dvsekhvalnov/jose2go/base64url"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	abci "github.com/tendermint/tendermint/abci/types"

	sdk "github.com/line/lbm-sdk/types"
	"github.com/line/lbm-sdk/types/module"
	authkeeper "github.com/line/lbm-sdk/x/auth/keeper"
	bankkeeper "github.com/line/lbm-sdk/x/bank/keeper"
	stakingkeeper "github.com/line/lbm-sdk/x/staking/keeper"
	"github.com/line/ostracon/crypto"
	"github.com/line/ostracon/crypto/ed25519"

	wasmkeeper "github.com/line/wasmd/x/wasm/keeper"
	wasmtypes "github.com/line/wasmd/x/wasm/types"
	"github.com/line/wasmd/x/wasmplus/keeper"
	"github.com/line/wasmd/x/wasmplus/types"
)

type testData struct {
	module        module.AppModule
	ctx           sdk.Context
	acctKeeper    authkeeper.AccountKeeper
	keeper        keeper.Keeper
	bankKeeper    bankkeeper.Keeper
	stakingKeeper stakingkeeper.Keeper
	faucet        *wasmkeeper.TestFaucet
}

func setupTest(t *testing.T) testData {
	ctx, keepers := keeper.CreateTestInput(t, false, "iterator,staking,stargate,cosmwasm_1_1")
	cdc := wasmkeeper.MakeTestCodec(t)
	data := testData{
		module:        NewAppModule(cdc, keepers.WasmKeeper, keepers.StakingKeeper, keepers.AccountKeeper, keepers.BankKeeper),
		ctx:           ctx,
		acctKeeper:    keepers.AccountKeeper,
		keeper:        *keepers.WasmKeeper,
		bankKeeper:    keepers.BankKeeper,
		stakingKeeper: keepers.StakingKeeper,
		faucet:        keepers.Faucet,
	}
	return data
}

func keyPubAddr() (crypto.PrivKey, crypto.PubKey, sdk.AccAddress) {
	key := ed25519.GenPrivKey()
	pub := key.PubKey()
	addr := sdk.AccAddress(pub.Address())
	return key, pub, addr
}

func mustLoad(path string) []byte {
	bz, err := os.ReadFile(path)
	if err != nil {
		panic(err)
	}
	return bz
}

var (
	_, _, addrAcc1 = keyPubAddr()
	addr1          = addrAcc1.String()
	testContract   = mustLoad("../wasm/keeper/testdata/hackatom.wasm")
	oldContract    = mustLoad("../wasm/testdata/escrow_0.7.wasm")
)

type initMsg struct {
	Verifier    sdk.AccAddress `json:"verifier"`
	Beneficiary sdk.AccAddress `json:"beneficiary"`
}

type emptyMsg struct{}

type state struct {
	Verifier    string `json:"verifier"`
	Beneficiary string `json:"beneficiary"`
	Funder      string `json:"funder"`
}

// ensures this returns a valid codeID and bech32 address and returns it
func parseStoreAndInitResponse(t *testing.T, data []byte) (uint64, string) {
	var res types.MsgStoreCodeAndInstantiateContractResponse
	require.NoError(t, res.Unmarshal(data))
	require.NotEmpty(t, res.CodeID)
	require.NotEmpty(t, res.Address)
	addr := res.Address
	codeID := res.CodeID
	// ensure this is a valid sdk address
	_, err := sdk.AccAddressFromBech32(addr)
	require.NoError(t, err)
	return codeID, addr
}

type prettyEvent struct {
	Type string
	Attr []sdk.Attribute
}

func prettyEvents(evts []abci.Event) string {
	res := make([]prettyEvent, len(evts))
	for i, e := range evts {
		res[i] = prettyEvent{
			Type: e.Type,
			Attr: prettyAttrs(e.Attributes),
		}
	}
	bz, err := json.MarshalIndent(res, "", "  ")
	if err != nil {
		panic(err)
	}
	return string(bz)
}

func prettyAttrs(attrs []abci.EventAttribute) []sdk.Attribute {
	pretty := make([]sdk.Attribute, len(attrs))
	for i, a := range attrs {
		pretty[i] = prettyAttr(a)
	}
	return pretty
}

func prettyAttr(attr abci.EventAttribute) sdk.Attribute {
	return sdk.NewAttribute(string(attr.Key), string(attr.Value))
}

func assertAttribute(t *testing.T, key string, value string, attr abci.EventAttribute) {
	t.Helper()
	assert.Equal(t, key, string(attr.Key), prettyAttr(attr))
	assert.Equal(t, value, string(attr.Value), prettyAttr(attr))
}

func assertCodeList(t *testing.T, q sdk.Querier, ctx sdk.Context, expectedNum int) {
	bz, sdkerr := q(ctx, []string{wasmkeeper.QueryListCode}, abci.RequestQuery{})
	require.NoError(t, sdkerr)

	if len(bz) == 0 {
		require.Equal(t, expectedNum, 0)
		return
	}

	var res []wasmtypes.CodeInfo
	err := json.Unmarshal(bz, &res)
	require.NoError(t, err)

	assert.Equal(t, expectedNum, len(res))
}

func assertCodeBytes(t *testing.T, q sdk.Querier, ctx sdk.Context, codeID uint64, expectedBytes []byte) {
	path := []string{wasmkeeper.QueryGetCode, fmt.Sprintf("%d", codeID)}
	bz, sdkerr := q(ctx, path, abci.RequestQuery{})
	require.NoError(t, sdkerr)

	if len(expectedBytes) == 0 {
		require.Equal(t, len(bz), 0, "%q", string(bz))
		return
	}
	var res map[string]interface{}
	err := json.Unmarshal(bz, &res)
	require.NoError(t, err)

	require.Contains(t, res, "data")
	b, err := base64url.Decode(res["data"].(string))
	require.NoError(t, err)
	assert.Equal(t, expectedBytes, b)
	assert.EqualValues(t, codeID, res["id"])
}

func assertContractList(t *testing.T, q sdk.Querier, ctx sdk.Context, codeID uint64, expContractAddrs []string) {
	bz, sdkerr := q(ctx, []string{wasmkeeper.QueryListContractByCode, fmt.Sprintf("%d", codeID)}, abci.RequestQuery{})
	require.NoError(t, sdkerr)

	if len(bz) == 0 {
		require.Equal(t, len(expContractAddrs), 0)
		return
	}

	var res []string
	err := json.Unmarshal(bz, &res)
	require.NoError(t, err)

	hasAddrs := make([]string, len(res))
	for i, r := range res {
		hasAddrs[i] = r
	}

	assert.Equal(t, expContractAddrs, hasAddrs)
}

func assertContractInfo(t *testing.T, q sdk.Querier, ctx sdk.Context, contractBech32Addr string, codeID uint64, creator sdk.AccAddress) {
	t.Helper()
	path := []string{wasmkeeper.QueryGetContract, contractBech32Addr}
	bz, sdkerr := q(ctx, path, abci.RequestQuery{})
	require.NoError(t, sdkerr)

	var res wasmtypes.ContractInfo
	err := json.Unmarshal(bz, &res)
	require.NoError(t, err)

	assert.Equal(t, codeID, res.CodeID)
	assert.Equal(t, creator.String(), res.Creator)
}

func assertContractState(t *testing.T, q sdk.Querier, ctx sdk.Context, contractBech32Addr string, expected state) {
	t.Helper()
	path := []string{wasmkeeper.QueryGetContractState, contractBech32Addr, wasmkeeper.QueryMethodContractStateAll}
	bz, sdkerr := q(ctx, path, abci.RequestQuery{})
	require.NoError(t, sdkerr)

	var res []wasmtypes.Model
	err := json.Unmarshal(bz, &res)
	require.NoError(t, err)
	require.Equal(t, 1, len(res), "#v", res)
	require.Equal(t, []byte("config"), []byte(res[0].Key))

	expectedBz, err := json.Marshal(expected)
	require.NoError(t, err)
	assert.Equal(t, expectedBz, res[0].Value)
}

func TestHandleStoreAndInstantiate(t *testing.T) {
	data := setupTest(t)
	creator := data.faucet.NewFundedRandomAccount(data.ctx, sdk.NewInt64Coin("denom", 100000))

	h := data.module.Route().Handler()
	q := data.module.LegacyQuerierHandler(nil)

	_, _, bob := keyPubAddr()
	_, _, fred := keyPubAddr()

	initMsg := initMsg{
		Verifier:    fred,
		Beneficiary: bob,
	}
	msgBz, err := json.Marshal(initMsg)
	require.NoError(t, err)

	// create with no balance is legal
	msg := &types.MsgStoreCodeAndInstantiateContract{
		Sender:       creator.String(),
		WASMByteCode: testContract,
		Msg:          msgBz,
		Label:        "contract for test",
		Funds:        nil,
	}
	res, err := h(data.ctx, msg)
	require.NoError(t, err)
	codeID, contractBech32Addr := parseStoreAndInitResponse(t, res.Data)

	require.Equal(t, uint64(1), codeID)
	require.Equal(t, "link14hj2tavq8fpesdwxxcu44rty3hh90vhujrvcmstl4zr3txmfvw9sgf2vn8", contractBech32Addr)
	// this should be standard x/wasm init event, nothing from contract
	require.Equal(t, 4, len(res.Events), prettyEvents(res.Events))
	assert.Equal(t, "store_code", res.Events[0].Type)
	assertAttribute(t, "code_id", "1", res.Events[0].Attributes[1])
	assert.Equal(t, "message", res.Events[1].Type)
	assertAttribute(t, "module", "wasm", res.Events[1].Attributes[0])
	assert.Equal(t, "instantiate", res.Events[2].Type)
	assertAttribute(t, "_contract_address", contractBech32Addr, res.Events[2].Attributes[0])
	assertAttribute(t, "code_id", "1", res.Events[2].Attributes[1])
	assert.Equal(t, "wasm", res.Events[3].Type)
	assertAttribute(t, "_contract_address", contractBech32Addr, res.Events[3].Attributes[0])

	assertCodeList(t, q, data.ctx, 1)
	assertCodeBytes(t, q, data.ctx, 1, testContract)

	assertContractList(t, q, data.ctx, 1, []string{contractBech32Addr})
	assertContractInfo(t, q, data.ctx, contractBech32Addr, 1, creator)
	assertContractState(t, q, data.ctx, contractBech32Addr, state{
		Verifier:    fred.String(),
		Beneficiary: bob.String(),
		Funder:      creator.String(),
	})
}

func TestErrorsCreateAndInstantiate(t *testing.T) {
	// init messages
	_, _, bob := keyPubAddr()
	_, _, fred := keyPubAddr()
	initMsg := initMsg{
		Verifier:    fred,
		Beneficiary: bob,
	}
	validInitMsgBz, err := json.Marshal(initMsg)
	require.NoError(t, err)

	invalidInitMsgBz, err := json.Marshal(emptyMsg{})

	expectedContractBech32Addr := "link14hj2tavq8fpesdwxxcu44rty3hh90vhujrvcmstl4zr3txmfvw9sgf2vn8"

	// test cases
	cases := map[string]struct {
		msg           sdk.Msg
		isValid       bool
		expectedCodes int
		expectedBytes []byte
	}{
		"empty": {
			msg:           &types.MsgStoreCodeAndInstantiateContract{},
			isValid:       false,
			expectedCodes: 0,
			expectedBytes: nil,
		},
		"valid one": {
			msg: &types.MsgStoreCodeAndInstantiateContract{
				Sender:       addr1,
				WASMByteCode: testContract,
				Msg:          validInitMsgBz,
				Label:        "foo",
				Funds:        nil,
			},
			isValid:       true,
			expectedCodes: 1,
			expectedBytes: testContract,
		},
		"invalid wasm": {
			msg: &types.MsgStoreCodeAndInstantiateContract{
				Sender:       addr1,
				WASMByteCode: []byte("foobar"),
				Msg:          validInitMsgBz,
				Label:        "foo",
				Funds:        nil,
			},
			isValid:       false,
			expectedCodes: 0,
			expectedBytes: nil,
		},
		"old wasm (0.7)": {
			msg: &types.MsgStoreCodeAndInstantiateContract{
				Sender:       addr1,
				WASMByteCode: oldContract,
				Msg:          validInitMsgBz,
				Label:        "foo",
				Funds:        nil,
			},
			isValid:       false,
			expectedCodes: 0,
			expectedBytes: nil,
		},
		"invalid init message": {
			msg: &types.MsgStoreCodeAndInstantiateContract{
				Sender:       addr1,
				WASMByteCode: testContract,
				Msg:          invalidInitMsgBz,
				Label:        "foo",
				Funds:        nil,
			},
			isValid:       false,
			expectedCodes: 1,
			expectedBytes: testContract,
		},
	}

	for name, tc := range cases {
		tc := tc
		t.Run(name, func(t *testing.T) {
			data := setupTest(t)

			h := data.module.Route().Handler()
			q := data.module.LegacyQuerierHandler(nil)

			// asserting response
			res, err := h(data.ctx, tc.msg)
			if tc.isValid {
				require.NoError(t, err)
				codeID, contractBech32Addr := parseStoreAndInitResponse(t, res.Data)
				require.Equal(t, uint64(1), codeID)
				require.Equal(t, expectedContractBech32Addr, contractBech32Addr)

			} else {
				require.Error(t, err, "%#v", res)
			}

			// asserting code state
			assertCodeList(t, q, data.ctx, tc.expectedCodes)
			assertCodeBytes(t, q, data.ctx, 1, tc.expectedBytes)

			// asserting contract state
			if tc.isValid {
				assertContractList(t, q, data.ctx, 1, []string{expectedContractBech32Addr})
				assertContractInfo(t, q, data.ctx, expectedContractBech32Addr, 1, addrAcc1)
				assertContractState(t, q, data.ctx, expectedContractBech32Addr, state{
					Verifier:    fred.String(),
					Beneficiary: bob.String(),
					Funder:      addrAcc1.String(),
				})
			} else {
				assertContractList(t, q, data.ctx, 0, []string{})
			}
		})
	}
}
