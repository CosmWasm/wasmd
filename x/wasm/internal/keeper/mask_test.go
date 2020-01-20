package keeper

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	wasmTypes "github.com/confio/go-cosmwasm/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/bank"
)

// MaskInitMsg is {}

type MaskHandleMsg struct {
	Reflect *reflectPayload `json:"reflectmsg,omitempty"`
	Change  *ownerPayload   `json:"changeowner,omitempty"`
}

type ownerPayload struct {
	Owner sdk.Address `json:"owner"`
}

type reflectPayload struct {
	Msg cosmosMsg `json:"msg"`
	// Msg wasmTypes.CosmosMsg `json:"msg"`
}

// replaces wasmTypes.CosmosMsg{
// TODO: fix upstream
type cosmosMsg struct {
	Send     *wasmTypes.SendMsg     `json:"send,omitempty"`
	Contract *wasmTypes.ContractMsg `json:"contract,omitempty"`
	Opaque   *wasmTypes.OpaqueMsg   `json:"opaque,omitempty"`
}

func TestMaskSend(t *testing.T) {
	tempDir, err := ioutil.TempDir("", "wasm")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)
	ctx, accKeeper, keeper := CreateTestInput(t, false, tempDir)

	deposit := sdk.NewCoins(sdk.NewInt64Coin("denom", 100000))
	creator := createFakeFundedAccount(ctx, accKeeper, deposit)
	bob := createFakeFundedAccount(ctx, accKeeper, deposit)
	_, _, fred := keyPubAddr()

	// upload code
	maskCode, err := ioutil.ReadFile("./testdata/mask.wasm")
	require.NoError(t, err)
	codeID, err := keeper.Create(ctx, creator, maskCode, "", "")
	require.NoError(t, err)
	require.Equal(t, uint64(1), codeID)

	// creator instantiates a contract and gives it tokens
	contractStart := sdk.NewCoins(sdk.NewInt64Coin("denom", 40000))
	contractAddr, err := keeper.Instantiate(ctx, creator, codeID, []byte("{}"), contractStart)
	require.NoError(t, err)
	require.NotEmpty(t, contractAddr)

	// set owner to bob
	transfer := MaskHandleMsg{
		Change: &ownerPayload{
			Owner: bob,
		},
	}
	transferBz, err := json.Marshal(transfer)
	require.NoError(t, err)
	// TODO: switch order of args Instantiate vs Execute (caller/code vs contract/caller), (msg/coins vs coins/msg)
	_, err = keeper.Execute(ctx, contractAddr, creator, nil, transferBz)
	require.NoError(t, err)

	// check some account values
	contractAcct := accKeeper.GetAccount(ctx, contractAddr)
	require.NotNil(t, contractAcct)
	require.Equal(t, contractAcct.GetCoins(), contractStart)
	bobAcct := accKeeper.GetAccount(ctx, bob)
	require.NotNil(t, bobAcct)
	require.Equal(t, bobAcct.GetCoins(), deposit)
	fredAcct := accKeeper.GetAccount(ctx, fred)
	require.Nil(t, fredAcct)

	// bob can send contract's tokens to fred (using SendMsg)
	// TODO: fix this upstream
	msg := cosmosMsg{
		Send: &wasmTypes.SendMsg{
			FromAddress: contractAddr.String(),
			ToAddress:   fred.String(),
			Amount: []wasmTypes.Coin{{
				Denom:  "denom",
				Amount: "15000",
			}},
		},
	}
	reflectSend := MaskHandleMsg{
		Reflect: &reflectPayload{
			Msg: msg,
		},
	}
	reflectSendBz, err := json.Marshal(reflectSend)
	require.NoError(t, err)
	// TODO: switch order of args Instantiate vs Execute (caller/code vs contract/caller), (msg/coins vs coins/msg)
	_, err = keeper.Execute(ctx, contractAddr, bob, nil, reflectSendBz)
	require.NoError(t, err)

	// fred got coins
	fredAcct = accKeeper.GetAccount(ctx, fred)
	require.NotNil(t, fredAcct)
	require.Equal(t, fredAcct.GetCoins(), sdk.NewCoins(sdk.NewInt64Coin("denom", 15000)))
	// contract lost them
	contractAcct = accKeeper.GetAccount(ctx, contractAddr)
	require.NotNil(t, contractAcct)
	require.Equal(t, contractAcct.GetCoins(), sdk.NewCoins(sdk.NewInt64Coin("denom", 25000)))

	// construct an opaque message
	var sdkSendMsg sdk.Msg = &bank.MsgSend{
		FromAddress: contractAddr,
		ToAddress:   fred,
		Amount:      sdk.NewCoins(sdk.NewInt64Coin("denom", 23000)),
	}
	opaque, err := ToOpaqueMsg(keeper.cdc, sdkSendMsg)
	require.NoError(t, err)
	reflectOpaque := MaskHandleMsg{
		Reflect: &reflectPayload{
			Msg: cosmosMsg{
				Opaque: opaque,
			},
		},
	}
	reflectOpaqueBz, err := json.Marshal(reflectOpaque)
	require.NoError(t, err)

	// TODO: switch order of args Instantiate vs Execute (caller/code vs contract/caller), (msg/coins vs coins/msg)
	_, err = keeper.Execute(ctx, contractAddr, bob, nil, reflectOpaqueBz)
	require.NoError(t, err)

	// fred got more coins
	fredAcct = accKeeper.GetAccount(ctx, fred)
	require.NotNil(t, fredAcct)
	require.Equal(t, fredAcct.GetCoins(), sdk.NewCoins(sdk.NewInt64Coin("denom", 38000)))
	// contract lost them
	contractAcct = accKeeper.GetAccount(ctx, contractAddr)
	require.NotNil(t, contractAcct)
	require.Equal(t, contractAcct.GetCoins(), sdk.NewCoins(sdk.NewInt64Coin("denom", 2000)))
}
