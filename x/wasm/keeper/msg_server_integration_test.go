package keeper_test

import (
	"crypto/sha256"
	_ "embed"
	"encoding/json"
	"testing"
	"time"

	"github.com/cosmos/cosmos-sdk/testutil/testdata"
	sdk "github.com/cosmos/cosmos-sdk/types"

	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/CosmWasm/wasmd/app"
	"github.com/CosmWasm/wasmd/x/wasm/keeper"
	"github.com/CosmWasm/wasmd/x/wasm/types"
)

//go:embed testdata/reflect.wasm
var wasmContract []byte

//go:embed testdata/hackatom.wasm
var hackatomContract []byte

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
		oneAddressAccessConfig                = types.AccessTypeAnyOfAddresses.With(myAddress)
		govAuthority                          = wasmApp.WasmKeeper.GetAuthority()
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
	for name, spec := range specs {
		t.Run(name, func(t *testing.T) {
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

func TestPinCodes(t *testing.T) {
	wasmApp := app.Setup(t)
	ctx := wasmApp.BaseApp.NewContext(false, tmproto.Header{})

	var (
		myAddress sdk.AccAddress = make([]byte, types.ContractAddrLen)
		authority                = wasmApp.WasmKeeper.GetAuthority()
	)

	specs := map[string]struct {
		addr   string
		expErr bool
	}{
		"authority can pin codes": {
			addr:   authority,
			expErr: false,
		},
		"other address cannot pin codes": {
			addr:   myAddress.String(),
			expErr: true,
		},
	}
	for name, spec := range specs {
		t.Run(name, func(t *testing.T) {
			// setup
			_, _, sender := testdata.KeyTestPubAddr()
			msg := types.MsgStoreCodeFixture(func(m *types.MsgStoreCode) {
				m.WASMByteCode = wasmContract
				m.Sender = sender.String()
			})

			// store code
			rsp, err := wasmApp.MsgServiceRouter().Handler(msg)(ctx, msg)
			require.NoError(t, err)
			var result types.MsgStoreCodeResponse
			require.NoError(t, wasmApp.AppCodec().Unmarshal(rsp.Data, &result))
			require.False(t, wasmApp.WasmKeeper.IsPinnedCode(ctx, result.CodeID))

			// when
			msgPinCodes := &types.MsgPinCodes{
				Authority: spec.addr,
				CodeIDs:   []uint64{result.CodeID},
			}
			_, err = wasmApp.MsgServiceRouter().Handler(msgPinCodes)(ctx, msgPinCodes)

			// then
			if spec.expErr {
				require.Error(t, err)
				assert.False(t, wasmApp.WasmKeeper.IsPinnedCode(ctx, result.CodeID))
			} else {
				require.NoError(t, err)
				assert.True(t, wasmApp.WasmKeeper.IsPinnedCode(ctx, result.CodeID))
			}
		})
	}
}

func TestUnpinCodes(t *testing.T) {
	wasmApp := app.Setup(t)
	ctx := wasmApp.BaseApp.NewContext(false, tmproto.Header{})

	var (
		myAddress sdk.AccAddress = make([]byte, types.ContractAddrLen)
		authority                = wasmApp.WasmKeeper.GetAuthority()
	)

	specs := map[string]struct {
		addr   string
		expErr bool
	}{
		"authority can unpin codes": {
			addr:   authority,
			expErr: false,
		},
		"other address cannot unpin codes": {
			addr:   myAddress.String(),
			expErr: true,
		},
	}
	for name, spec := range specs {
		t.Run(name, func(t *testing.T) {
			// setup
			_, _, sender := testdata.KeyTestPubAddr()
			msg := types.MsgStoreCodeFixture(func(m *types.MsgStoreCode) {
				m.WASMByteCode = wasmContract
				m.Sender = sender.String()
			})

			// store code
			rsp, err := wasmApp.MsgServiceRouter().Handler(msg)(ctx, msg)
			require.NoError(t, err)
			var result types.MsgStoreCodeResponse
			require.NoError(t, wasmApp.AppCodec().Unmarshal(rsp.Data, &result))

			// pin code
			msgPin := &types.MsgPinCodes{
				Authority: authority,
				CodeIDs:   []uint64{result.CodeID},
			}
			_, err = wasmApp.MsgServiceRouter().Handler(msgPin)(ctx, msgPin)
			require.NoError(t, err)
			assert.True(t, wasmApp.WasmKeeper.IsPinnedCode(ctx, result.CodeID))

			// when
			msgUnpinCodes := &types.MsgUnpinCodes{
				Authority: spec.addr,
				CodeIDs:   []uint64{result.CodeID},
			}
			_, err = wasmApp.MsgServiceRouter().Handler(msgUnpinCodes)(ctx, msgUnpinCodes)

			// then
			if spec.expErr {
				require.Error(t, err)
				assert.True(t, wasmApp.WasmKeeper.IsPinnedCode(ctx, result.CodeID))
			} else {
				require.NoError(t, err)
				assert.False(t, wasmApp.WasmKeeper.IsPinnedCode(ctx, result.CodeID))
			}
		})
	}
}

func TestSudoContract(t *testing.T) {
	wasmApp := app.Setup(t)
	ctx := wasmApp.BaseApp.NewContext(false, tmproto.Header{Time: time.Now()})

	var (
		myAddress sdk.AccAddress = make([]byte, types.ContractAddrLen)
		authority                = wasmApp.WasmKeeper.GetAuthority()
	)

	type StealMsg struct {
		Recipient string     `json:"recipient"`
		Amount    []sdk.Coin `json:"amount"`
	}

	stealMsg := struct {
		Steal StealMsg `json:"steal_funds"`
	}{Steal: StealMsg{
		Recipient: myAddress.String(),
		Amount:    []sdk.Coin{},
	}}

	stealMsgBz, err := json.Marshal(stealMsg)
	require.NoError(t, err)

	specs := map[string]struct {
		addr   string
		expErr bool
	}{
		"authority can call sudo on a contract": {
			addr:   authority,
			expErr: false,
		},
		"other address cannot call sudo on a contract": {
			addr:   myAddress.String(),
			expErr: true,
		},
	}
	for name, spec := range specs {
		t.Run(name, func(t *testing.T) {
			// setup
			_, _, sender := testdata.KeyTestPubAddr()
			msg := types.MsgStoreCodeFixture(func(m *types.MsgStoreCode) {
				m.WASMByteCode = hackatomContract
				m.Sender = sender.String()
			})

			// store code
			rsp, err := wasmApp.MsgServiceRouter().Handler(msg)(ctx, msg)
			require.NoError(t, err)
			var storeCodeResponse types.MsgStoreCodeResponse
			require.NoError(t, wasmApp.AppCodec().Unmarshal(rsp.Data, &storeCodeResponse))

			// instantiate contract
			initMsg := keeper.HackatomExampleInitMsg{
				Verifier:    sender,
				Beneficiary: myAddress,
			}
			initMsgBz, err := json.Marshal(initMsg)
			require.NoError(t, err)

			msgInstantiate := &types.MsgInstantiateContract{
				Sender: sender.String(),
				Admin:  sender.String(),
				CodeID: storeCodeResponse.CodeID,
				Label:  "test",
				Msg:    initMsgBz,
				Funds:  sdk.Coins{},
			}
			rsp, err = wasmApp.MsgServiceRouter().Handler(msgInstantiate)(ctx, msgInstantiate)
			require.NoError(t, err)
			var instantiateResponse types.MsgInstantiateContractResponse
			require.NoError(t, wasmApp.AppCodec().Unmarshal(rsp.Data, &instantiateResponse))

			// when
			msgSudoContract := &types.MsgSudoContract{
				Authority: spec.addr,
				Msg:       stealMsgBz,
				Contract:  instantiateResponse.Address,
			}
			_, err = wasmApp.MsgServiceRouter().Handler(msgSudoContract)(ctx, msgSudoContract)

			// then
			if spec.expErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestStoreAndInstantiateContract(t *testing.T) {
	wasmApp := app.Setup(t)
	ctx := wasmApp.BaseApp.NewContext(false, tmproto.Header{Time: time.Now()})

	var (
		myAddress sdk.AccAddress = make([]byte, types.ContractAddrLen)
		authority                = wasmApp.WasmKeeper.GetAuthority()
	)

	specs := map[string]struct {
		addr       string
		permission *types.AccessConfig
		expErr     bool
	}{
		"authority can store and instantiate a contract when permission is nobody": {
			addr:       authority,
			permission: &types.AllowNobody,
			expErr:     false,
		},
		"other address cannot store and instantiate a contract when permission is nobody": {
			addr:       myAddress.String(),
			permission: &types.AllowNobody,
			expErr:     true,
		},
		"authority can store and instantiate a contract when permission is everybody": {
			addr:       authority,
			permission: &types.AllowEverybody,
			expErr:     false,
		},
		"other address can store and instantiate a contract when permission is everybody": {
			addr:       myAddress.String(),
			permission: &types.AllowEverybody,
			expErr:     false,
		},
	}
	for name, spec := range specs {
		t.Run(name, func(t *testing.T) {
			// when
			msg := &types.MsgStoreAndInstantiateContract{
				Authority:             spec.addr,
				WASMByteCode:          wasmContract,
				InstantiatePermission: spec.permission,
				Admin:                 myAddress.String(),
				UnpinCode:             false,
				Label:                 "test",
				Msg:                   []byte(`{}`),
				Funds:                 sdk.Coins{},
			}
			_, err := wasmApp.MsgServiceRouter().Handler(msg)(ctx, msg)

			// then
			if spec.expErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestUpdateAdmin(t *testing.T) {
	wasmApp := app.Setup(t)
	ctx := wasmApp.BaseApp.NewContext(false, tmproto.Header{Time: time.Now()})

	var (
		myAddress       sdk.AccAddress = make([]byte, types.ContractAddrLen)
		authority                      = wasmApp.WasmKeeper.GetAuthority()
		_, _, otherAddr                = testdata.KeyTestPubAddr()
	)

	specs := map[string]struct {
		addr   string
		expErr bool
	}{
		"authority can update admin": {
			addr:   authority,
			expErr: false,
		},
		"admin can update admin": {
			addr:   myAddress.String(),
			expErr: false,
		},
		"other address cannot update admin": {
			addr:   otherAddr.String(),
			expErr: true,
		},
	}
	for name, spec := range specs {
		t.Run(name, func(t *testing.T) {
			_, _, newAdmin := testdata.KeyTestPubAddr()

			// setup
			msg := &types.MsgStoreAndInstantiateContract{
				Authority:             spec.addr,
				WASMByteCode:          wasmContract,
				InstantiatePermission: &types.AllowEverybody,
				Admin:                 myAddress.String(),
				UnpinCode:             false,
				Label:                 "test",
				Msg:                   []byte(`{}`),
				Funds:                 sdk.Coins{},
			}
			rsp, err := wasmApp.MsgServiceRouter().Handler(msg)(ctx, msg)
			require.NoError(t, err)
			var storeAndInstantiateResponse types.MsgStoreAndInstantiateContractResponse
			require.NoError(t, wasmApp.AppCodec().Unmarshal(rsp.Data, &storeAndInstantiateResponse))

			// when
			msgUpdateAdmin := &types.MsgUpdateAdmin{
				Sender:   spec.addr,
				NewAdmin: newAdmin.String(),
				Contract: storeAndInstantiateResponse.Address,
			}
			_, err = wasmApp.MsgServiceRouter().Handler(msgUpdateAdmin)(ctx, msgUpdateAdmin)

			// then
			if spec.expErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestClearAdmin(t *testing.T) {
	wasmApp := app.Setup(t)
	ctx := wasmApp.BaseApp.NewContext(false, tmproto.Header{Time: time.Now()})

	var (
		myAddress       sdk.AccAddress = make([]byte, types.ContractAddrLen)
		authority                      = wasmApp.WasmKeeper.GetAuthority()
		_, _, otherAddr                = testdata.KeyTestPubAddr()
	)

	specs := map[string]struct {
		addr   string
		expErr bool
	}{
		"authority can clear admin": {
			addr:   authority,
			expErr: false,
		},
		"admin can clear admin": {
			addr:   myAddress.String(),
			expErr: false,
		},
		"other address cannot clear admin": {
			addr:   otherAddr.String(),
			expErr: true,
		},
	}
	for name, spec := range specs {
		t.Run(name, func(t *testing.T) {
			// setup
			msg := &types.MsgStoreAndInstantiateContract{
				Authority:             spec.addr,
				WASMByteCode:          wasmContract,
				InstantiatePermission: &types.AllowEverybody,
				Admin:                 myAddress.String(),
				UnpinCode:             false,
				Label:                 "test",
				Msg:                   []byte(`{}`),
				Funds:                 sdk.Coins{},
			}
			rsp, err := wasmApp.MsgServiceRouter().Handler(msg)(ctx, msg)
			require.NoError(t, err)
			var storeAndInstantiateResponse types.MsgStoreAndInstantiateContractResponse
			require.NoError(t, wasmApp.AppCodec().Unmarshal(rsp.Data, &storeAndInstantiateResponse))

			// when
			msgClearAdmin := &types.MsgClearAdmin{
				Sender:   spec.addr,
				Contract: storeAndInstantiateResponse.Address,
			}
			_, err = wasmApp.MsgServiceRouter().Handler(msgClearAdmin)(ctx, msgClearAdmin)

			// then
			if spec.expErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestMigrateContract(t *testing.T) {
	wasmApp := app.Setup(t)
	ctx := wasmApp.BaseApp.NewContext(false, tmproto.Header{Time: time.Now()})

	var (
		myAddress       sdk.AccAddress = make([]byte, types.ContractAddrLen)
		authority                      = wasmApp.WasmKeeper.GetAuthority()
		_, _, otherAddr                = testdata.KeyTestPubAddr()
	)

	specs := map[string]struct {
		addr   string
		expErr bool
	}{
		"authority can migrate a contract": {
			addr:   authority,
			expErr: false,
		},
		"admin can migrate a contract": {
			addr:   myAddress.String(),
			expErr: false,
		},
		"other address cannot migrate a contract": {
			addr:   otherAddr.String(),
			expErr: true,
		},
	}
	for name, spec := range specs {
		t.Run(name, func(t *testing.T) {
			// setup
			_, _, sender := testdata.KeyTestPubAddr()
			msg := types.MsgStoreCodeFixture(func(m *types.MsgStoreCode) {
				m.WASMByteCode = hackatomContract
				m.Sender = sender.String()
			})

			// store code
			rsp, err := wasmApp.MsgServiceRouter().Handler(msg)(ctx, msg)
			require.NoError(t, err)
			var storeCodeResponse types.MsgStoreCodeResponse
			require.NoError(t, wasmApp.AppCodec().Unmarshal(rsp.Data, &storeCodeResponse))

			// instantiate contract
			initMsg := keeper.HackatomExampleInitMsg{
				Verifier:    sender,
				Beneficiary: myAddress,
			}
			initMsgBz, err := json.Marshal(initMsg)
			require.NoError(t, err)

			msgInstantiate := &types.MsgInstantiateContract{
				Sender: sender.String(),
				Admin:  myAddress.String(),
				CodeID: storeCodeResponse.CodeID,
				Label:  "test",
				Msg:    initMsgBz,
				Funds:  sdk.Coins{},
			}
			rsp, err = wasmApp.MsgServiceRouter().Handler(msgInstantiate)(ctx, msgInstantiate)
			require.NoError(t, err)
			var instantiateResponse types.MsgInstantiateContractResponse
			require.NoError(t, wasmApp.AppCodec().Unmarshal(rsp.Data, &instantiateResponse))

			// when
			migMsg := struct {
				Verifier sdk.AccAddress `json:"verifier"`
			}{Verifier: myAddress}
			migMsgBz, err := json.Marshal(migMsg)
			require.NoError(t, err)
			msgMigrateContract := &types.MsgMigrateContract{
				Sender:   spec.addr,
				Msg:      migMsgBz,
				Contract: instantiateResponse.Address,
				CodeID:   storeCodeResponse.CodeID,
			}
			_, err = wasmApp.MsgServiceRouter().Handler(msgMigrateContract)(ctx, msgMigrateContract)

			// then
			if spec.expErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestInstantiateContract(t *testing.T) {
	wasmApp := app.Setup(t)
	ctx := wasmApp.BaseApp.NewContext(false, tmproto.Header{Time: time.Now()})

	var (
		myAddress sdk.AccAddress = make([]byte, types.ContractAddrLen)
		authority                = wasmApp.WasmKeeper.GetAuthority()
	)

	specs := map[string]struct {
		addr       string
		permission *types.AccessConfig
		expErr     bool
	}{
		"authority can instantiate a contract when permission is nobody": {
			addr:       authority,
			permission: &types.AllowNobody,
			expErr:     false,
		},
		"other address cannot instantiate a contract when permission is nobody": {
			addr:       myAddress.String(),
			permission: &types.AllowNobody,
			expErr:     true,
		},
		"authority can instantiate a contract when permission is everybody": {
			addr:       authority,
			permission: &types.AllowEverybody,
			expErr:     false,
		},
		"other address can  instantiate a contract when permission is everybody": {
			addr:       myAddress.String(),
			permission: &types.AllowEverybody,
			expErr:     false,
		},
	}
	for name, spec := range specs {
		t.Run(name, func(t *testing.T) {
			// setup
			_, _, sender := testdata.KeyTestPubAddr()
			msg := types.MsgStoreCodeFixture(func(m *types.MsgStoreCode) {
				m.WASMByteCode = wasmContract
				m.Sender = sender.String()
				m.InstantiatePermission = spec.permission
			})

			// store code
			rsp, err := wasmApp.MsgServiceRouter().Handler(msg)(ctx, msg)
			require.NoError(t, err)
			var result types.MsgStoreCodeResponse
			require.NoError(t, wasmApp.AppCodec().Unmarshal(rsp.Data, &result))

			// when
			msgInstantiate := &types.MsgInstantiateContract{
				Sender: spec.addr,
				Admin:  myAddress.String(),
				CodeID: result.CodeID,
				Label:  "test",
				Msg:    []byte(`{}`),
				Funds:  sdk.Coins{},
			}
			_, err = wasmApp.MsgServiceRouter().Handler(msgInstantiate)(ctx, msgInstantiate)

			// then
			if spec.expErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestInstantiateContract2(t *testing.T) {
	wasmApp := app.Setup(t)
	ctx := wasmApp.BaseApp.NewContext(false, tmproto.Header{Time: time.Now()})

	var (
		myAddress sdk.AccAddress = make([]byte, types.ContractAddrLen)
		authority                = wasmApp.WasmKeeper.GetAuthority()
	)

	specs := map[string]struct {
		addr       string
		salt       string
		permission *types.AccessConfig
		expErr     bool
	}{
		"authority can instantiate a contract when permission is nobody": {
			addr:       authority,
			permission: &types.AllowNobody,
			salt:       "salt1",
			expErr:     false,
		},
		"other address cannot instantiate a contract when permission is nobody": {
			addr:       myAddress.String(),
			permission: &types.AllowNobody,
			salt:       "salt2",
			expErr:     true,
		},
		"authority can instantiate a contract when permission is everybody": {
			addr:       authority,
			permission: &types.AllowEverybody,
			salt:       "salt3",
			expErr:     false,
		},
		"other address can  instantiate a contract when permission is everybody": {
			addr:       myAddress.String(),
			permission: &types.AllowEverybody,
			salt:       "salt4",
			expErr:     false,
		},
	}
	for name, spec := range specs {
		t.Run(name, func(t *testing.T) {
			// setup
			_, _, sender := testdata.KeyTestPubAddr()
			msg := types.MsgStoreCodeFixture(func(m *types.MsgStoreCode) {
				m.WASMByteCode = wasmContract
				m.Sender = sender.String()
				m.InstantiatePermission = spec.permission
			})

			// store code
			rsp, err := wasmApp.MsgServiceRouter().Handler(msg)(ctx, msg)
			require.NoError(t, err)
			var result types.MsgStoreCodeResponse
			require.NoError(t, wasmApp.AppCodec().Unmarshal(rsp.Data, &result))

			// when
			msgInstantiate := &types.MsgInstantiateContract2{
				Sender: spec.addr,
				Admin:  myAddress.String(),
				CodeID: result.CodeID,
				Label:  "label",
				Msg:    []byte(`{}`),
				Funds:  sdk.Coins{},
				Salt:   []byte(spec.salt),
				FixMsg: true,
			}
			_, err = wasmApp.MsgServiceRouter().Handler(msgInstantiate)(ctx, msgInstantiate)

			// then
			if spec.expErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestUpdateInstantiateConfig(t *testing.T) {
	wasmApp := app.Setup(t)
	ctx := wasmApp.BaseApp.NewContext(false, tmproto.Header{Time: time.Now()})

	var (
		creator   sdk.AccAddress = make([]byte, types.ContractAddrLen)
		authority                = wasmApp.WasmKeeper.GetAuthority()
	)

	specs := map[string]struct {
		addr       string
		permission *types.AccessConfig
		expErr     bool
	}{
		"authority can update instantiate config when permission is subset": {
			addr:       authority,
			permission: &types.AllowNobody,
			expErr:     false,
		},
		"creator can update instantiate config when permission is subset": {
			addr:       creator.String(),
			permission: &types.AllowNobody,
			expErr:     false,
		},
		"authority can update instantiate config when permission is not subset": {
			addr:       authority,
			permission: &types.AllowEverybody,
			expErr:     false,
		},
		"creator cannot  update instantiate config when permission is not subset": {
			addr:       creator.String(),
			permission: &types.AllowEverybody,
			expErr:     true,
		},
	}
	for name, spec := range specs {
		t.Run(name, func(t *testing.T) {
			// setup
			err := wasmApp.WasmKeeper.SetParams(ctx, types.Params{
				CodeUploadAccess:             types.AllowEverybody,
				InstantiateDefaultPermission: types.AccessTypeNobody,
			})
			require.NoError(t, err)

			msg := types.MsgStoreCodeFixture(func(m *types.MsgStoreCode) {
				m.WASMByteCode = wasmContract
				m.Sender = creator.String()
				m.InstantiatePermission = &types.AllowNobody
			})

			// store code
			rsp, err := wasmApp.MsgServiceRouter().Handler(msg)(ctx, msg)
			require.NoError(t, err)
			var result types.MsgStoreCodeResponse
			require.NoError(t, wasmApp.AppCodec().Unmarshal(rsp.Data, &result))

			// when
			msgUpdateInstantiateConfig := &types.MsgUpdateInstantiateConfig{
				Sender:                   spec.addr,
				CodeID:                   result.CodeID,
				NewInstantiatePermission: spec.permission,
			}
			_, err = wasmApp.MsgServiceRouter().Handler(msgUpdateInstantiateConfig)(ctx, msgUpdateInstantiateConfig)

			// then
			if spec.expErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestPruneWasmCodesAuthority(t *testing.T) {
	wasmApp := app.Setup(t)
	ctx := wasmApp.BaseApp.NewContext(false, tmproto.Header{})

	var (
		myAddress sdk.AccAddress = make([]byte, types.ContractAddrLen)
		authority                = wasmApp.WasmKeeper.GetAuthority()
	)

	specs := map[string]struct {
		addr      string
		pinCode   bool
		expPruned bool
		expErr    bool
	}{
		"authority can prune unpinned code": {
			addr:      authority,
			pinCode:   false,
			expPruned: true,
		},
		"authority cannot prune pinned code": {
			addr:      authority,
			pinCode:   true,
			expPruned: false,
		},
		"other address cannot prune unpinned code": {
			addr:      myAddress.String(),
			pinCode:   false,
			expPruned: false,
			expErr:    true,
		},
		"other address cannot prune pinned code": {
			addr:      myAddress.String(),
			pinCode:   true,
			expPruned: false,
			expErr:    true,
		},
	}
	for name, spec := range specs {
		t.Run(name, func(t *testing.T) {
			codeID, _ := setupWasmCode(t, authority, ctx, wasmApp, wasmContract, spec.pinCode)

			// when
			msgPruneCodes := &types.MsgPruneWasmCodes{
				Authority: spec.addr,
				MaxCodeID: 5,
			}
			_, err := wasmApp.MsgServiceRouter().Handler(msgPruneCodes)(ctx, msgPruneCodes)

			// then
			if spec.expErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			if spec.expPruned {
				assert.Nil(t, wasmApp.WasmKeeper.GetCodeInfo(ctx, codeID))
			} else {
				assert.NotNil(t, wasmApp.WasmKeeper.GetCodeInfo(ctx, codeID))
			}
		})
	}
}

func TestPruneWasmCodes(t *testing.T) {
	specs := map[string]struct {
		wasmCode1          []byte
		wasmCode2          []byte
		pinCode1           bool
		pinCode2           bool
		expPrunedCodeInfo1 bool
		expPrunedCodeInfo2 bool
		expPrunedWasmCode1 bool
		expPrunedWasmCode2 bool
		maxCodeID          uint64
	}{
		"duplicate codes - both pinned - max code id 2": {
			wasmCode1:          wasmContract,
			wasmCode2:          wasmContract,
			pinCode1:           true,
			pinCode2:           true,
			expPrunedCodeInfo1: false,
			expPrunedCodeInfo2: false,
			expPrunedWasmCode1: false,
			expPrunedWasmCode2: false,
			maxCodeID:          2,
		},
		"duplicate codes - only one pinned - max code id 2": {
			wasmCode1:          wasmContract,
			wasmCode2:          wasmContract,
			pinCode1:           true,
			pinCode2:           false,
			expPrunedCodeInfo1: false,
			expPrunedCodeInfo2: true,
			expPrunedWasmCode1: false,
			expPrunedWasmCode2: false,
			maxCodeID:          2,
		},
		"duplicate codes - both unpinned - max code id 2": {
			wasmCode1:          wasmContract,
			wasmCode2:          wasmContract,
			pinCode1:           false,
			pinCode2:           false,
			expPrunedCodeInfo1: true,
			expPrunedCodeInfo2: true,
			expPrunedWasmCode1: true,
			expPrunedWasmCode2: true,
			maxCodeID:          2,
		},
		"different codes - both pinned - max code id 2": {
			wasmCode1:          wasmContract,
			wasmCode2:          hackatomContract,
			pinCode1:           true,
			pinCode2:           true,
			expPrunedCodeInfo1: false,
			expPrunedCodeInfo2: false,
			expPrunedWasmCode1: false,
			expPrunedWasmCode2: false,
			maxCodeID:          2,
		},
		"different codes - only one pinned - max code id 2": {
			wasmCode1:          wasmContract,
			wasmCode2:          hackatomContract,
			pinCode1:           true,
			pinCode2:           false,
			expPrunedCodeInfo1: false,
			expPrunedCodeInfo2: true,
			expPrunedWasmCode1: false,
			expPrunedWasmCode2: true,
			maxCodeID:          2,
		},
		"different codes - both unpinned - max code id 2": {
			wasmCode1:          wasmContract,
			wasmCode2:          hackatomContract,
			pinCode1:           false,
			pinCode2:           false,
			expPrunedCodeInfo1: true,
			expPrunedCodeInfo2: true,
			expPrunedWasmCode1: true,
			expPrunedWasmCode2: true,
			maxCodeID:          2,
		},
		"duplicate codes - both pinned - max code id 1": {
			wasmCode1:          wasmContract,
			wasmCode2:          wasmContract,
			pinCode1:           true,
			pinCode2:           true,
			expPrunedCodeInfo1: false,
			expPrunedCodeInfo2: false,
			expPrunedWasmCode1: false,
			expPrunedWasmCode2: false,
			maxCodeID:          1,
		},
		"duplicate codes - only one pinned - max code id 1": {
			wasmCode1:          wasmContract,
			wasmCode2:          wasmContract,
			pinCode1:           true,
			pinCode2:           false,
			expPrunedCodeInfo1: false,
			expPrunedCodeInfo2: false,
			expPrunedWasmCode1: false,
			expPrunedWasmCode2: false,
			maxCodeID:          1,
		},
		"duplicate codes - both unpinned - max code id 1": {
			wasmCode1:          wasmContract,
			wasmCode2:          wasmContract,
			pinCode1:           false,
			pinCode2:           false,
			expPrunedCodeInfo1: true,
			expPrunedCodeInfo2: false,
			expPrunedWasmCode1: false,
			expPrunedWasmCode2: false,
			maxCodeID:          1,
		},
		"different codes - both pinned - max code id 1": {
			wasmCode1:          wasmContract,
			wasmCode2:          hackatomContract,
			pinCode1:           true,
			pinCode2:           true,
			expPrunedCodeInfo1: false,
			expPrunedCodeInfo2: false,
			expPrunedWasmCode1: false,
			expPrunedWasmCode2: false,
			maxCodeID:          1,
		},
		"different codes - only one pinned - max code id 1": {
			wasmCode1:          wasmContract,
			wasmCode2:          hackatomContract,
			pinCode1:           true,
			pinCode2:           false,
			expPrunedCodeInfo1: false,
			expPrunedCodeInfo2: false,
			expPrunedWasmCode1: false,
			expPrunedWasmCode2: false,
			maxCodeID:          1,
		},
		"different codes - both unpinned - max code id 1": {
			wasmCode1:          wasmContract,
			wasmCode2:          hackatomContract,
			pinCode1:           false,
			pinCode2:           false,
			expPrunedCodeInfo1: true,
			expPrunedCodeInfo2: false,
			expPrunedWasmCode1: true,
			expPrunedWasmCode2: false,
			maxCodeID:          1,
		},
	}
	for name, spec := range specs {
		t.Run(name, func(t *testing.T) {
			wasmApp := app.Setup(t)
			ctx := wasmApp.BaseApp.NewContext(false, tmproto.Header{})
			authority := wasmApp.WasmKeeper.GetAuthority()

			code1, checksum1 := setupWasmCode(t, authority, ctx, wasmApp, spec.wasmCode1, spec.pinCode1)
			code2, checksum2 := setupWasmCode(t, authority, ctx, wasmApp, spec.wasmCode2, spec.pinCode2)

			// when
			msgPruneCodes := &types.MsgPruneWasmCodes{
				Authority: authority,
				MaxCodeID: spec.maxCodeID,
			}
			_, err := wasmApp.MsgServiceRouter().Handler(msgPruneCodes)(ctx, msgPruneCodes)

			// then
			require.NoError(t, err)

			assertCodeInfoPruning(t, ctx, wasmApp, code1, spec.expPrunedCodeInfo1)
			assertWasmCodePruning(t, ctx, wasmApp, checksum1, spec.expPrunedWasmCode1)

			assertCodeInfoPruning(t, ctx, wasmApp, code2, spec.expPrunedCodeInfo2)
			assertWasmCodePruning(t, ctx, wasmApp, checksum2, spec.expPrunedWasmCode2)
		})
	}
}

func setupWasmCode(t *testing.T, authority string, ctx sdk.Context, wasmApp *app.WasmApp, wasmCode []byte, pin bool) (uint64, []byte) {
	// setup
	_, _, sender := testdata.KeyTestPubAddr()
	msg := types.MsgStoreCodeFixture(func(m *types.MsgStoreCode) {
		m.WASMByteCode = wasmCode
		m.Sender = sender.String()
	})

	// store code
	rsp, err := wasmApp.MsgServiceRouter().Handler(msg)(ctx, msg)
	require.NoError(t, err)
	var result types.MsgStoreCodeResponse
	require.NoError(t, wasmApp.AppCodec().Unmarshal(rsp.Data, &result))

	if pin {
		// pin code
		msgPin := &types.MsgPinCodes{
			Authority: authority,
			CodeIDs:   []uint64{result.CodeID},
		}
		_, err = wasmApp.MsgServiceRouter().Handler(msgPin)(ctx, msgPin)
		require.NoError(t, err)
		assert.True(t, wasmApp.WasmKeeper.IsPinnedCode(ctx, result.CodeID))
	}
	return result.CodeID, result.Checksum
}

func assertCodeInfoPruning(t *testing.T, ctx sdk.Context, wasmApp *app.WasmApp, codeID uint64, pruned bool) {
	if pruned {
		assert.Nil(t, wasmApp.WasmKeeper.GetCodeInfo(ctx, codeID))
	} else {
		assert.NotNil(t, wasmApp.WasmKeeper.GetCodeInfo(ctx, codeID))
	}
}

func assertWasmCodePruning(t *testing.T, ctx sdk.Context, wasmApp *app.WasmApp, checksum []byte, pruned bool) {
	wasmCode, err := wasmApp.WasmKeeper.GetByteCodeByChecksum(ctx, checksum)
	if pruned {
		assert.Nil(t, wasmCode)
		assert.Error(t, err)
	} else {
		assert.NotNil(t, wasmCode)
		assert.NoError(t, err)
	}
}
