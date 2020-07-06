package keeper

import (
	"io/ioutil"
	"testing"

	"github.com/CosmWasm/wasmd/x/wasm/internal/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStoreCodeProposal(t *testing.T) {
	tempDir, err := ioutil.TempDir("", "wasm")
	require.NoError(t, err)

	ctx, keepers := CreateTestInput(t, false, tempDir, "staking", nil, nil)
	govKeeper, wasmKeeper := keepers.GovKeeper, keepers.WasmKeeper

	wasmCode, err := ioutil.ReadFile("./testdata/contract.wasm")
	require.NoError(t, err)

	var anyAddress sdk.AccAddress = make([]byte, sdk.AddrLen)

	src := types.StoreCodeProposal{
		WasmProposal: types.WasmProposal{
			Title:       "Foo",
			Description: "Bar",
		},
		Creator:      anyAddress,
		WASMByteCode: wasmCode,
		Source:       "https://example.com/mysource",
		Builder:      "foo/bar:v0.0.0",
	}

	// when stored
	storedProposal, err := govKeeper.SubmitProposal(ctx, &src)
	require.NoError(t, err)

	// then execute storedProposal
	handler := govKeeper.Router().GetRoute(storedProposal.ProposalRoute())

	err = handler(ctx, storedProposal.Content)
	require.NoError(t, err)
	cInfo := wasmKeeper.GetCodeInfo(ctx, 1)
	require.NotNil(t, cInfo)
	assert.Equal(t, anyAddress, cInfo.Creator)
	assert.Equal(t, "foo/bar:v0.0.0", cInfo.Builder)
	assert.Equal(t, "https://example.com/mysource", cInfo.Source)

	storedCode, err := wasmKeeper.GetByteCode(ctx, 1)
	require.NoError(t, err)
	assert.Equal(t, wasmCode, storedCode)
}
