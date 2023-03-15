package keeper_test

import (
	"crypto/sha256"
	_ "embed"
	"testing"

	"github.com/cosmos/cosmos-sdk/testutil/testdata"
	sdk "github.com/cosmos/cosmos-sdk/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"

	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/CosmWasm/wasmd/app"
	"github.com/CosmWasm/wasmd/x/wasm/types"
)

//go:embed testdata/reflect.wasm
var wasmContract []byte

func TestStoreCode(t *testing.T) {
	wasmApp := app.Setup(t)
	ctx := wasmApp.BaseApp.NewContext(false, tmproto.Header{})
	_, _, sender := testdata.KeyTestPubAddr()
	msg := types.MsgStoreCodeFixture(func(m *types.MsgStoreCode) {
		m.WASMByteCode = wasmContract
		m.Sender = sender.String()
	})

	// when
	rsp, err := wasmApp.MsgServiceRouter().Handler(msg)(ctx, msg)

	// then
	require.NoError(t, err)
	var result types.MsgStoreCodeResponse
	require.NoError(t, wasmApp.AppCodec().Unmarshal(rsp.Data, &result))
	assert.Equal(t, uint64(1), result.CodeID)
	expHash := sha256.Sum256(wasmContract)
	assert.Equal(t, expHash[:], result.Checksum)
	// and
	info := wasmApp.WasmKeeper.GetCodeInfo(ctx, 1)
	assert.NotNil(t, info)
	assert.Equal(t, expHash[:], info.CodeHash)
	assert.Equal(t, sender.String(), info.Creator)
	assert.Equal(t, types.DefaultParams().InstantiateDefaultPermission.With(sender), info.InstantiateConfig)
}

func TestUpdateParams(t *testing.T) {
	wasmApp := app.Setup(t)
	ctx := wasmApp.BaseApp.NewContext(false, tmproto.Header{})

	var (
		myAddress              sdk.AccAddress = make([]byte, types.ContractAddrLen)
		oneAddressAccessConfig                = types.AccessTypeOnlyAddress.With(myAddress)
		govAuthority                          = wasmApp.AccountKeeper.GetModuleAddress(govtypes.ModuleName).String()
	)

	specs := map[string]struct {
		src                types.MsgUpdateParams
		expUploadConfig    types.AccessConfig
		expInstantiateType types.AccessType
	}{
		"update upload permission param": {
			src: types.MsgUpdateParams{
				Authority: govAuthority,
				Params: types.Params{
					CodeUploadAccess:             types.AllowNobody,
					InstantiateDefaultPermission: types.AccessTypeEverybody,
				},
			},
			expUploadConfig:    types.AllowNobody,
			expInstantiateType: types.AccessTypeEverybody,
		},
		"update upload permission with same as current value": {
			src: types.MsgUpdateParams{
				Authority: govAuthority,
				Params: types.Params{
					CodeUploadAccess:             types.AllowEverybody,
					InstantiateDefaultPermission: types.AccessTypeEverybody,
				},
			},
			expUploadConfig:    types.AllowEverybody,
			expInstantiateType: types.AccessTypeEverybody,
		},
		"update upload permission param with address": {
			src: types.MsgUpdateParams{
				Authority: govAuthority,
				Params: types.Params{
					CodeUploadAccess:             oneAddressAccessConfig,
					InstantiateDefaultPermission: types.AccessTypeEverybody,
				},
			},
			expUploadConfig:    oneAddressAccessConfig,
			expInstantiateType: types.AccessTypeEverybody,
		},
		"update instantiate param": {
			src: types.MsgUpdateParams{
				Authority: govAuthority,
				Params: types.Params{
					CodeUploadAccess:             types.AllowEverybody,
					InstantiateDefaultPermission: types.AccessTypeNobody,
				},
			},
			expUploadConfig:    types.AllowEverybody,
			expInstantiateType: types.AccessTypeNobody,
		},
		"update instantiate param as default": {
			src: types.MsgUpdateParams{
				Authority: govAuthority,
				Params: types.Params{
					CodeUploadAccess:             types.AllowEverybody,
					InstantiateDefaultPermission: types.AccessTypeEverybody,
				},
			},
			expUploadConfig:    types.AllowEverybody,
			expInstantiateType: types.AccessTypeEverybody,
		},
	}
	for msg, spec := range specs {
		t.Run(msg, func(t *testing.T) {
			err := wasmApp.WasmKeeper.SetParams(ctx, types.DefaultParams())
			require.NoError(t, err)

			// when
			rsp, err := wasmApp.MsgServiceRouter().Handler(&spec.src)(ctx, &spec.src)
			require.NoError(t, err)
			var result types.MsgUpdateParamsResponse
			require.NoError(t, wasmApp.AppCodec().Unmarshal(rsp.Data, &result))

			// then
			assert.True(t, spec.expUploadConfig.Equals(wasmApp.WasmKeeper.GetParams(ctx).CodeUploadAccess),
				"got %#v not %#v", wasmApp.WasmKeeper.GetParams(ctx).CodeUploadAccess, spec.expUploadConfig)
			assert.Equal(t, spec.expInstantiateType, wasmApp.WasmKeeper.GetParams(ctx).InstantiateDefaultPermission)
		})
	}
}
