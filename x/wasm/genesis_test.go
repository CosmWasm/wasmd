package wasm

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

type contractState struct {
}

func TestInitGenesis(t *testing.T) {
	data, cleanup := setupTest(t)
	defer cleanup()

	deposit := sdk.NewCoins(sdk.NewInt64Coin("denom", 100000))
	topUp := sdk.NewCoins(sdk.NewInt64Coin("denom", 5000))
	creator := createFakeFundedAccount(data.ctx, data.acctKeeper, deposit.Add(deposit...))
	fred := createFakeFundedAccount(data.ctx, data.acctKeeper, topUp)

	h := data.module.NewHandler()
	q := data.module.NewQuerierHandler()

	t.Log("fail with invalid source url")
	msg := MsgStoreCode{
		Sender:       creator,
		WASMByteCode: testContract,
		Source:       "someinvalidurl",
		Builder:      "",
	}

	sdkerr := msg.ValidateBasic()
	require.Error(t, sdkerr)

	res, err := h(data.ctx, msg)
	require.NotNil(t, err)

	t.Log("fail with relative source url")
	msg = MsgStoreCode{
		Sender:       creator,
		WASMByteCode: testContract,
		Source:       "./testdata/escrow.wasm",
		Builder:      "",
	}

	sdkerr = msg.ValidateBasic()
	require.Error(t, sdkerr)

	res, err = h(data.ctx, msg)
	require.NotNil(t, err)

	t.Log("fail with unreachable source url")
	msg = MsgStoreCode{
		Sender:       creator,
		WASMByteCode: testContract,
		Source:       "https://github.com/cosmwasm/wasmddddddrandom",
		Builder:      "",
	}

	sdkerr = msg.ValidateBasic()
	require.Error(t, sdkerr)

	res, err = h(data.ctx, msg)
	require.NotNil(t, err)

	t.Log("fail with invalid build tag")
	msg = MsgStoreCode{
		Sender:       creator,
		WASMByteCode: testContract,
		Source:       "",
		Builder:      "somerandombuildtag-0.6.2",
	}

	sdkerr = msg.ValidateBasic()
	require.Error(t, sdkerr)

	res, err = h(data.ctx, msg)
	require.NotNil(t, err)

	t.Log("no error with valid source and build tag")
	msg = MsgStoreCode{
		Sender:       creator,
		WASMByteCode: testContract,
		Source:       "https://github.com/cosmwasm/wasmd/blob/master/x/wasm/testdata/escrow.wasm",
		Builder:      "cosmwasm-opt:0.5.2",
	}
	sdkerr = msg.ValidateBasic()
	require.NoError(t, sdkerr)

	res, err = h(data.ctx, msg)
	require.Nil(t, err)
	require.Equal(t, res.Data, []byte("1"))

	_, _, bob := keyPubAddr()
	initMsg := initMsg{
		Verifier:    fred,
		Beneficiary: bob,
	}
	initMsgBz, err := json.Marshal(initMsg)
	require.NoError(t, err)

	initCmd := MsgInstantiateContract{
		Sender:    creator,
		Code:      1,
		InitMsg:   initMsgBz,
		InitFunds: deposit,
	}
	res, err = h(data.ctx, initCmd)
	require.Nil(t, err)
	contractAddr := sdk.AccAddress(res.Data)

	execCmd := MsgExecuteContract{
		Sender:    fred,
		Contract:  contractAddr,
		Msg:       []byte("{}"),
		SentFunds: topUp,
	}
	res, err = h(data.ctx, execCmd)
	require.Nil(t, err)

	// ensure all contract state is as after init
	assertCodeList(t, q, data.ctx, 1)
	assertCodeBytes(t, q, data.ctx, 1, testContract)

	assertContractList(t, q, data.ctx, []string{contractAddr.String()})
	assertContractInfo(t, q, data.ctx, contractAddr, 1, creator)
	assertContractState(t, q, data.ctx, contractAddr, state{
		Verifier:    []byte(fred),
		Beneficiary: []byte(bob),
		Funder:      []byte(creator),
	})

	// export into genstate
	genState := ExportGenesis(data.ctx, data.keeper)

	// create new app to import genstate into
	newData, newCleanup := setupTest(t)
	defer newCleanup()
	q2 := newData.module.NewQuerierHandler()

	// initialize new app with genstate
	InitGenesis(newData.ctx, newData.keeper, genState)

	// run same checks again on newdata, to make sure it was reinitialized correctly
	assertCodeList(t, q2, newData.ctx, 1)
	assertCodeBytes(t, q2, newData.ctx, 1, testContract)

	assertContractList(t, q2, newData.ctx, []string{contractAddr.String()})
	assertContractInfo(t, q2, newData.ctx, contractAddr, 1, creator)
	assertContractState(t, q2, newData.ctx, contractAddr, state{
		Verifier:    []byte(fred),
		Beneficiary: []byte(bob),
		Funder:      []byte(creator),
	})
}
