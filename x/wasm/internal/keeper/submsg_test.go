package keeper

import (
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"testing"

	wasmvmtypes "github.com/CosmWasm/wasmvm/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
)

// test handing of submessages, very closely related to the reflect_test

// Try a simple send, no gas limit to for a sanity check before trying table tests
func TestDispatchSubMsgSuccessCase(t *testing.T) {
	ctx, keepers := CreateTestInput(t, false, ReflectFeatures, nil, nil)
	accKeeper, keeper, bankKeeper := keepers.AccountKeeper, keepers.WasmKeeper, keepers.BankKeeper

	deposit := sdk.NewCoins(sdk.NewInt64Coin("denom", 100000))
	contractStart := sdk.NewCoins(sdk.NewInt64Coin("denom", 40000))

	creator := createFakeFundedAccount(t, ctx, accKeeper, bankKeeper, deposit)
	creatorBalance := deposit.Sub(contractStart)
	_, _, fred := keyPubAddr()

	// upload code
	reflectCode, err := ioutil.ReadFile("./testdata/reflect.wasm")
	require.NoError(t, err)
	codeID, err := keeper.Create(ctx, creator, reflectCode, "", "", nil)
	require.NoError(t, err)
	require.Equal(t, uint64(1), codeID)

	// creator instantiates a contract and gives it tokens
	contractAddr, _, err := keeper.Instantiate(ctx, codeID, creator, nil, []byte("{}"), "reflect contract 1", contractStart)
	require.NoError(t, err)
	require.NotEmpty(t, contractAddr)

	// check some account values
	checkAccount(t, ctx, accKeeper, bankKeeper, contractAddr, contractStart)
	checkAccount(t, ctx, accKeeper, bankKeeper, creator, creatorBalance)
	checkAccount(t, ctx, accKeeper, bankKeeper, fred, nil)

	// creator can send contract's tokens to fred (using SendMsg)
	msg := wasmvmtypes.CosmosMsg{
		Bank: &wasmvmtypes.BankMsg{
			Send: &wasmvmtypes.SendMsg{
				ToAddress: fred.String(),
				Amount: []wasmvmtypes.Coin{{
					Denom:  "denom",
					Amount: "15000",
				}},
			},
		},
	}
	reflectSend := ReflectHandleMsg{
		ReflectSubCall: &reflectSubPayload{
			Msgs: []wasmvmtypes.SubMsg{{
				ID:  7,
				Msg: msg,
			}},
		},
	}
	reflectSendBz, err := json.Marshal(reflectSend)
	require.NoError(t, err)
	_, err = keeper.Execute(ctx, contractAddr, creator, reflectSendBz, nil)
	require.NoError(t, err)

	// fred got coins
	checkAccount(t, ctx, accKeeper, bankKeeper, fred, sdk.NewCoins(sdk.NewInt64Coin("denom", 15000)))
	// contract lost them
	checkAccount(t, ctx, accKeeper, bankKeeper, contractAddr, sdk.NewCoins(sdk.NewInt64Coin("denom", 25000)))
	checkAccount(t, ctx, accKeeper, bankKeeper, creator, creatorBalance)

	// TODO: query the reflect state to ensure the result was stored
	query := ReflectQueryMsg{
		SubCallResult: &SubCall{ID: 7},
	}
	queryBz, err := json.Marshal(query)
	require.NoError(t, err)
	queryRes, err := keeper.QuerySmart(ctx, contractAddr, queryBz)
	require.NoError(t, err)

	var res wasmvmtypes.Reply
	err = json.Unmarshal(queryRes, &res)
	require.NoError(t, err)
	assert.Equal(t, uint64(7), res.ID)
	assert.Empty(t, res.Result.Err)
	require.NotNil(t, res.Result.Ok)
	sub := res.Result.Ok
	assert.Empty(t, sub.Data)
	require.Len(t, sub.Events, 3)

	transfer := sub.Events[0]
	assert.Equal(t, "transfer", transfer.Type)
	assert.Equal(t, wasmvmtypes.EventAttribute{
		Key:   "recipient",
		Value: fred.String(),
	}, transfer.Attributes[0])

	sender := sub.Events[1]
	assert.Equal(t, "message", sender.Type)
	assert.Equal(t, wasmvmtypes.EventAttribute{
		Key:   "sender",
		Value: contractAddr.String(),
	}, sender.Attributes[0])

	// where does this come from?
	module := sub.Events[2]
	assert.Equal(t, "message", module.Type)
	assert.Equal(t, wasmvmtypes.EventAttribute{
		Key:   "module",
		Value: "bank",
	}, module.Attributes[0])

}
