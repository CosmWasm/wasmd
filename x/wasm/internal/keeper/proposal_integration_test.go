package keeper

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"io/ioutil"
	"os"
	"testing"

	"github.com/CosmWasm/wasmd/x/wasm/internal/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/gov"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStoreCodeProposal(t *testing.T) {
	tempDir, err := ioutil.TempDir("", "wasm")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	ctx, keepers := CreateTestInput(t, false, tempDir, "staking", nil, nil)
	govKeeper, wasmKeeper := keepers.GovKeeper, keepers.WasmKeeper

	wasmCode, err := ioutil.ReadFile("./testdata/contract.wasm")
	require.NoError(t, err)

	var anyAddress sdk.AccAddress = make([]byte, sdk.AddrLen)

	src := types.StoreCodeProposalFixture(func(p *types.StoreCodeProposal) {
		p.Creator = anyAddress
		p.WASMByteCode = wasmCode
		p.Source = "https://example.com/mysource"
		p.Builder = "foo/bar:v0.0.0"
	})

	// when stored
	storedProposal, err := govKeeper.SubmitProposal(ctx, &src)
	require.NoError(t, err)

	// and proposal execute
	handler := govKeeper.Router().GetRoute(storedProposal.ProposalRoute())
	err = handler(ctx, storedProposal.Content)
	require.NoError(t, err)

	// then
	cInfo := wasmKeeper.GetCodeInfo(ctx, 1)
	require.NotNil(t, cInfo)
	assert.Equal(t, anyAddress, cInfo.Creator)
	assert.Equal(t, "foo/bar:v0.0.0", cInfo.Builder)
	assert.Equal(t, "https://example.com/mysource", cInfo.Source)

	storedCode, err := wasmKeeper.GetByteCode(ctx, 1)
	require.NoError(t, err)
	assert.Equal(t, wasmCode, storedCode)
}

func TestInstantiateProposal(t *testing.T) {
	tempDir, err := ioutil.TempDir("", "wasm")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	ctx, keepers := CreateTestInput(t, false, tempDir, "staking", nil, nil)
	govKeeper, wasmKeeper := keepers.GovKeeper, keepers.WasmKeeper

	wasmCode, err := ioutil.ReadFile("./testdata/contract.wasm")
	require.NoError(t, err)

	require.NoError(t, wasmKeeper.importCode(ctx, 1,
		types.CodeInfoFixture(types.WithSHA256CodeHash(wasmCode)),
		wasmCode),
	)

	var (
		oneAddress   sdk.AccAddress = bytes.Repeat([]byte{0x1}, sdk.AddrLen)
		otherAddress sdk.AccAddress = bytes.Repeat([]byte{0x2}, sdk.AddrLen)
	)
	src := types.InstantiateContractProposalFixture(func(p *types.InstantiateContractProposal) {
		p.Code = 1
		p.Creator = oneAddress
		p.Admin = otherAddress
		p.Label = "testing"
	})

	// when stored
	storedProposal, err := govKeeper.SubmitProposal(ctx, &src)
	require.NoError(t, err)

	// and proposal execute
	handler := govKeeper.Router().GetRoute(storedProposal.ProposalRoute())
	err = handler(ctx, storedProposal.Content)
	require.NoError(t, err)

	// then
	contractAddr, err := sdk.AccAddressFromBech32("cosmos18vd8fpwxzck93qlwghaj6arh4p7c5n89uzcee5")
	require.NoError(t, err)

	cInfo := wasmKeeper.GetContractInfo(ctx, contractAddr)
	require.NotNil(t, cInfo)
	assert.Equal(t, uint64(1), cInfo.CodeID)
	assert.Equal(t, oneAddress, cInfo.Creator)
	assert.Equal(t, otherAddress, cInfo.Admin)
	assert.Equal(t, "testing", cInfo.Label)
	assert.Equal(t, src.InitMsg, cInfo.InitMsg)
}

func TestMigrateProposal(t *testing.T) {
	tempDir, err := ioutil.TempDir("", "wasm")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	ctx, keepers := CreateTestInput(t, false, tempDir, "staking", nil, nil)
	govKeeper, wasmKeeper := keepers.GovKeeper, keepers.WasmKeeper

	wasmCode, err := ioutil.ReadFile("./testdata/contract.wasm")
	require.NoError(t, err)

	codeInfoFixture := types.CodeInfoFixture(types.WithSHA256CodeHash(wasmCode))
	require.NoError(t, wasmKeeper.importCode(ctx, 1, codeInfoFixture, wasmCode))
	require.NoError(t, wasmKeeper.importCode(ctx, 2, codeInfoFixture, wasmCode))

	var (
		anyAddress   sdk.AccAddress = bytes.Repeat([]byte{0x1}, sdk.AddrLen)
		otherAddress sdk.AccAddress = bytes.Repeat([]byte{0x2}, sdk.AddrLen)
		contractAddr                = contractAddress(1, 1)
	)

	contractInfoFixture := types.ContractInfoFixture(func(c *types.ContractInfo) {
		c.Label = "testing"
		c.Admin = anyAddress
	})
	key, err := hex.DecodeString("636F6E666967")
	require.NoError(t, err)
	m := types.Model{Key: key, Value: []byte(`{"verifier":"AAAAAAAAAAAAAAAAAAAAAAAAAAA=","beneficiary":"AAAAAAAAAAAAAAAAAAAAAAAAAAA=","funder":"AQEBAQEBAQEBAQEBAQEBAQEBAQE="}`)}
	require.NoError(t, wasmKeeper.importContract(ctx, contractAddr, &contractInfoFixture, []types.Model{m}))

	migMsg := struct {
		Verifier sdk.AccAddress `json:"verifier"`
	}{Verifier: otherAddress}
	migMsgBz, err := json.Marshal(migMsg)
	require.NoError(t, err)

	src := types.MigrateContractProposal{
		WasmProposal: types.WasmProposal{
			Title:       "Foo",
			Description: "Bar",
		},
		Code:       2,
		Contract:   contractAddr,
		MigrateMsg: migMsgBz,
		Sender:     otherAddress,
	}

	// when stored
	storedProposal, err := govKeeper.SubmitProposal(ctx, &src)
	require.NoError(t, err)

	// and proposal execute
	handler := govKeeper.Router().GetRoute(storedProposal.ProposalRoute())
	err = handler(ctx, storedProposal.Content)
	require.NoError(t, err)

	// then
	require.NoError(t, err)
	cInfo := wasmKeeper.GetContractInfo(ctx, contractAddr)
	require.NotNil(t, cInfo)
	assert.Equal(t, uint64(2), cInfo.CodeID)
	assert.Equal(t, uint64(1), cInfo.PreviousCodeID)
	assert.Equal(t, anyAddress, cInfo.Admin)
	assert.Equal(t, "testing", cInfo.Label)
}

func TestAdminProposals(t *testing.T) {
	var (
		anyAddress   sdk.AccAddress = bytes.Repeat([]byte{0x1}, sdk.AddrLen)
		otherAddress sdk.AccAddress = bytes.Repeat([]byte{0x2}, sdk.AddrLen)
		contractAddr                = contractAddress(1, 1)
	)
	wasmCode, err := ioutil.ReadFile("./testdata/contract.wasm")
	require.NoError(t, err)

	specs := map[string]struct {
		state       types.ContractInfo
		srcProposal gov.Content
		expAdmin    sdk.AccAddress
	}{
		"update with different admin": {
			state: types.ContractInfoFixture(),
			srcProposal: &types.UpdateAdminProposal{
				WasmProposal: types.WasmProposal{
					Title:       "Foo",
					Description: "Bar",
				},
				Contract: contractAddr,
				Sender:   anyAddress,
				NewAdmin: otherAddress,
			},
			expAdmin: otherAddress,
		},
		"update with old admin empty": {
			state: types.ContractInfoFixture(func(info *types.ContractInfo) {
				info.Admin = nil
			}),
			srcProposal: &types.UpdateAdminProposal{
				WasmProposal: types.WasmProposal{
					Title:       "Foo",
					Description: "Bar",
				},
				Contract: contractAddr,
				Sender:   anyAddress,
				NewAdmin: otherAddress,
			},
			expAdmin: otherAddress,
		},
		"clear admin": {
			state: types.ContractInfoFixture(),
			srcProposal: &types.ClearAdminProposal{
				WasmProposal: types.WasmProposal{
					Title:       "Foo",
					Description: "Bar",
				},
				Contract: contractAddr,
				Sender:   anyAddress,
			},
			expAdmin: nil,
		},
		"clear with old admin empty": {
			state: types.ContractInfoFixture(func(info *types.ContractInfo) {
				info.Admin = nil
			}),
			srcProposal: &types.ClearAdminProposal{
				WasmProposal: types.WasmProposal{
					Title:       "Foo",
					Description: "Bar",
				},
				Contract: contractAddr,
				Sender:   anyAddress,
			},
			expAdmin: nil,
		},
	}
	for msg, spec := range specs {
		t.Run(msg, func(t *testing.T) {
			tempDir, err := ioutil.TempDir("", "wasm")
			require.NoError(t, err)
			defer os.RemoveAll(tempDir)
			ctx, keepers := CreateTestInput(t, false, tempDir, "staking", nil, nil)
			govKeeper, wasmKeeper := keepers.GovKeeper, keepers.WasmKeeper

			codeInfoFixture := types.CodeInfoFixture(types.WithSHA256CodeHash(wasmCode))
			require.NoError(t, wasmKeeper.importCode(ctx, 1, codeInfoFixture, wasmCode))

			require.NoError(t, wasmKeeper.importContract(ctx, contractAddr, &spec.state, []types.Model{}))
			// when stored
			storedProposal, err := govKeeper.SubmitProposal(ctx, spec.srcProposal)
			require.NoError(t, err)

			// and execute proposal
			handler := govKeeper.Router().GetRoute(storedProposal.ProposalRoute())
			err = handler(ctx, storedProposal.Content)
			require.NoError(t, err)

			// then
			cInfo := wasmKeeper.GetContractInfo(ctx, contractAddr)
			require.NotNil(t, cInfo)
			assert.Equal(t, spec.expAdmin, cInfo.Admin)
		})
	}
}
