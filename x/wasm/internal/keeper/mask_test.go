package keeper

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	sdk "github.com/cosmos/cosmos-sdk/types"
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
	Msg []byte `json:"msg"`
}

func TestMaskSend(t *testing.T) {
	tempDir, err := ioutil.TempDir("", "wasm")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)
	ctx, accKeeper, keeper := CreateTestInput(t, false, tempDir)

	deposit := sdk.NewCoins(sdk.NewInt64Coin("denom", 100000))
	creator := createFakeFundedAccount(ctx, accKeeper, deposit)
	bob := createFakeFundedAccount(ctx, accKeeper, deposit)
	// _, _, fred := keyPubAddr()

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
	fmt.Println(contractAddr.String())

	// set owner to bob
	transfer := MaskHandleMsg{
		Change: &ownerPayload{
			Owner: bob,
		},
	}
	transferBz, err := json.Marshal(transfer)
	require.NoError(t, err)
	fmt.Println(string(transferBz))
	// TODO: switch order of args Instantiate vs Execute (caller/code vs contract/caller), (msg/coins vs coins/msg)
	_, err = keeper.Execute(ctx, contractAddr, creator, nil, transferBz)
	require.NoError(t, err)

	// bob can send contract's tokens to fred
}
