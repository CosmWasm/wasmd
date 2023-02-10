package keeper

import (
	"crypto/sha256"
	"errors"
	"os"
	"testing"
	"time"

	fuzz "github.com/google/gofuzz"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	abci "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/proto/tendermint/crypto"
	tmproto "github.com/tendermint/tendermint/proto/tendermint/types"
	dbm "github.com/tendermint/tm-db"

	"github.com/line/lbm-sdk/store"
	sdk "github.com/line/lbm-sdk/types"
	authkeeper "github.com/line/lbm-sdk/x/auth/keeper"
	authtypes "github.com/line/lbm-sdk/x/auth/types"
	banktypes "github.com/line/lbm-sdk/x/bank/types"
	bankpluskeeper "github.com/line/lbm-sdk/x/bankplus/keeper"
	distributionkeeper "github.com/line/lbm-sdk/x/distribution/keeper"
	govtypes "github.com/line/lbm-sdk/x/gov/types"
	paramskeeper "github.com/line/lbm-sdk/x/params/keeper"
	paramstypes "github.com/line/lbm-sdk/x/params/types"
	stakingkeeper "github.com/line/lbm-sdk/x/staking/keeper"
	"github.com/line/ostracon/libs/log"

	"github.com/line/wasmd/x/wasm/keeper"
	wasmkeeper "github.com/line/wasmd/x/wasm/keeper"
	wasmTypes "github.com/line/wasmd/x/wasm/types"
	"github.com/line/wasmd/x/wasmplus/types"
)

const (
	firstCodeID  = 1
	humanAddress = "link1hcttwju93d5m39467gjcq63p5kc4fdcn30dgd8"

	AvailableCapabilities = "iterator,staking,stargate,cosmwasm_1_1"
)

func TestGenesisExportImport(t *testing.T) {
	wasmKeeper, srcCtx, _ := setupKeeper(t)
	contractKeeper := NewPermissionedKeeper(*wasmkeeper.NewGovPermissionKeeper(wasmKeeper), wasmKeeper)

	wasmCode, err := os.ReadFile("../../wasm/keeper/testdata/hackatom.wasm")
	require.NoError(t, err)

	// store some test data
	f := fuzz.New().Funcs(wasmkeeper.ModelFuzzers...)

	wasmKeeper.SetParams(srcCtx, wasmTypes.DefaultParams())

	for i := 0; i < 5; i++ {
		var (
			codeInfo          wasmTypes.CodeInfo
			contract          wasmTypes.ContractInfo
			stateModels       []wasmTypes.Model
			history           []wasmTypes.ContractCodeHistoryEntry
			pinned            bool
			contractExtension bool
			verifier          sdk.AccAddress
			beneficiary       sdk.AccAddress
		)
		f.Fuzz(&codeInfo)
		f.Fuzz(&contract)
		f.Fuzz(&stateModels)
		f.NilChance(0).Fuzz(&history)
		f.Fuzz(&pinned)
		f.Fuzz(&contractExtension)
		f.Fuzz(&verifier)
		f.Fuzz(&beneficiary)

		creatorAddr, err := sdk.AccAddressFromBech32(codeInfo.Creator)
		require.NoError(t, err)
		codeID, _, err := contractKeeper.Create(srcCtx, creatorAddr, wasmCode, &codeInfo.InstantiateConfig)
		require.NoError(t, err)
		if pinned {
			err = contractKeeper.PinCode(srcCtx, codeID)
			require.NoError(t, err)
		}
		if contractExtension {
			anyTime := time.Now().UTC()
			var nestedType govtypes.TextProposal
			f.NilChance(0).Fuzz(&nestedType)
			myExtension, err := govtypes.NewProposal(&nestedType, 1, anyTime, anyTime)
			require.NoError(t, err)
			err = contract.SetExtension(&myExtension)
			require.NoError(t, err)
		}

		initMsgBz := HackatomExampleInitMsg{
			Verifier:    verifier,
			Beneficiary: beneficiary,
		}.GetBytes(t)

		_, _, err = contractKeeper.Instantiate(srcCtx, codeID, creatorAddr, creatorAddr, initMsgBz, "test", nil)
		require.NoError(t, err)
	}
	var wasmParams wasmTypes.Params
	f.NilChance(0).Fuzz(&wasmParams)
	wasmKeeper.SetParams(srcCtx, wasmParams)

	// add inactiveContractAddr
	var inactiveContractAddr []sdk.AccAddress
	wasmKeeper.IterateContractInfo(srcCtx, func(address sdk.AccAddress, info wasmTypes.ContractInfo) bool {
		err = contractKeeper.DeactivateContract(srcCtx, address)
		require.NoError(t, err)
		inactiveContractAddr = append(inactiveContractAddr, address)
		return false
	})

	// export
	exportedState := ExportGenesis(srcCtx, wasmKeeper)
	exportedGenesis, err := wasmKeeper.cdc.MarshalJSON(exportedState)
	require.NoError(t, err)

	// setup new instances
	dstKeeper, dstCtx, _ := setupKeeper(t)

	// re-import
	var importState types.GenesisState
	err = dstKeeper.cdc.UnmarshalJSON(exportedGenesis, &importState)
	require.NoError(t, err)
	_, err = InitGenesis(dstCtx, dstKeeper, importState, &StakingKeeperMock{}, TestHandler(contractKeeper))
	require.NoError(t, err)

	// compare
	dstParams := dstKeeper.GetParams(dstCtx)
	require.Equal(t, wasmParams, dstParams)

	var destInactiveContractAddr []sdk.AccAddress
	dstKeeper.IterateInactiveContracts(dstCtx, func(contractAddress sdk.AccAddress) (stop bool) {
		destInactiveContractAddr = append(destInactiveContractAddr, contractAddress)
		return false
	})
	require.Equal(t, inactiveContractAddr, destInactiveContractAddr)
}

func TestGenesisInit(t *testing.T) {
	wasmCode, err := os.ReadFile("../../wasm/keeper/testdata/hackatom.wasm")
	require.NoError(t, err)

	myCodeInfo := wasmTypes.CodeInfoFixture(wasmTypes.WithSHA256CodeHash(wasmCode))
	specs := map[string]struct {
		src            types.GenesisState
		stakingMock    StakingKeeperMock
		msgHandlerMock MockMsgHandler
		expSuccess     bool
	}{
		"happy path: code info correct": {
			src: types.GenesisState{
				Codes: []wasmTypes.Code{{
					CodeID:    firstCodeID,
					CodeInfo:  myCodeInfo,
					CodeBytes: wasmCode,
				}},
				Sequences: []wasmTypes.Sequence{
					{IDKey: wasmTypes.KeyLastCodeID, Value: 2},
					{IDKey: wasmTypes.KeyLastInstanceID, Value: 1},
				},
				Params: wasmTypes.DefaultParams(),
			},
			expSuccess: true,
		},
		"happy path: code ids can contain gaps": {
			src: types.GenesisState{
				Codes: []wasmTypes.Code{{
					CodeID:    firstCodeID,
					CodeInfo:  myCodeInfo,
					CodeBytes: wasmCode,
				}, {
					CodeID:    3,
					CodeInfo:  myCodeInfo,
					CodeBytes: wasmCode,
				}},
				Sequences: []wasmTypes.Sequence{
					{IDKey: wasmTypes.KeyLastCodeID, Value: 10},
					{IDKey: wasmTypes.KeyLastInstanceID, Value: 1},
				},
				Params: wasmTypes.DefaultParams(),
			},
			expSuccess: true,
		},
		"happy path: code order does not matter": {
			src: types.GenesisState{
				Codes: []wasmTypes.Code{{
					CodeID:    2,
					CodeInfo:  myCodeInfo,
					CodeBytes: wasmCode,
				}, {
					CodeID:    firstCodeID,
					CodeInfo:  myCodeInfo,
					CodeBytes: wasmCode,
				}},
				Contracts: nil,
				Sequences: []wasmTypes.Sequence{
					{IDKey: wasmTypes.KeyLastCodeID, Value: 3},
					{IDKey: wasmTypes.KeyLastInstanceID, Value: 1},
				},
				Params: wasmTypes.DefaultParams(),
			},
			expSuccess: true,
		},
		"prevent code hash mismatch": {src: types.GenesisState{
			Codes: []wasmTypes.Code{{
				CodeID:    firstCodeID,
				CodeInfo:  wasmTypes.CodeInfoFixture(func(i *wasmTypes.CodeInfo) { i.CodeHash = make([]byte, sha256.Size) }),
				CodeBytes: wasmCode,
			}},
			Params: wasmTypes.DefaultParams(),
		}},
		"prevent duplicate codeIDs": {src: types.GenesisState{
			Codes: []wasmTypes.Code{
				{
					CodeID:    firstCodeID,
					CodeInfo:  myCodeInfo,
					CodeBytes: wasmCode,
				},
				{
					CodeID:    firstCodeID,
					CodeInfo:  myCodeInfo,
					CodeBytes: wasmCode,
				},
			},
			Params: wasmTypes.DefaultParams(),
		}},
		"codes with same checksum can be pinned": {
			src: types.GenesisState{
				Codes: []wasmTypes.Code{
					{
						CodeID:    firstCodeID,
						CodeInfo:  myCodeInfo,
						CodeBytes: wasmCode,
						Pinned:    true,
					},
					{
						CodeID:    2,
						CodeInfo:  myCodeInfo,
						CodeBytes: wasmCode,
						Pinned:    true,
					},
				},
				Params: wasmTypes.DefaultParams(),
			},
		},
		"happy path: code id in info and contract do match": {
			src: types.GenesisState{
				Codes: []wasmTypes.Code{{
					CodeID:    firstCodeID,
					CodeInfo:  myCodeInfo,
					CodeBytes: wasmCode,
				}},
				Contracts: []wasmTypes.Contract{
					{
						ContractAddress: keeper.BuildContractAddressClassic(1, 1).String(),
						ContractInfo:    wasmTypes.ContractInfoFixture(func(c *wasmTypes.ContractInfo) { c.CodeID = 1 }, wasmTypes.OnlyGenesisFields),
					},
				},
				Sequences: []wasmTypes.Sequence{
					{IDKey: wasmTypes.KeyLastCodeID, Value: 2},
					{IDKey: wasmTypes.KeyLastInstanceID, Value: 2},
				},
				Params: wasmTypes.DefaultParams(),
			},
			expSuccess: true,
		},
		"happy path: code info with two contracts": {
			src: types.GenesisState{
				Codes: []wasmTypes.Code{{
					CodeID:    firstCodeID,
					CodeInfo:  myCodeInfo,
					CodeBytes: wasmCode,
				}},
				Contracts: []wasmTypes.Contract{
					{
						ContractAddress: keeper.BuildContractAddressClassic(1, 1).String(),
						ContractInfo:    wasmTypes.ContractInfoFixture(func(c *wasmTypes.ContractInfo) { c.CodeID = 1 }, wasmTypes.OnlyGenesisFields),
					}, {
						ContractAddress: keeper.BuildContractAddressClassic(1, 2).String(),
						ContractInfo:    wasmTypes.ContractInfoFixture(func(c *wasmTypes.ContractInfo) { c.CodeID = 1 }, wasmTypes.OnlyGenesisFields),
					},
				},
				Sequences: []wasmTypes.Sequence{
					{IDKey: wasmTypes.KeyLastCodeID, Value: 2},
					{IDKey: wasmTypes.KeyLastInstanceID, Value: 3},
				},
				Params: wasmTypes.DefaultParams(),
			},
			expSuccess: true,
		},
		"prevent contracts that points to non existing codeID": {
			src: types.GenesisState{
				Contracts: []wasmTypes.Contract{
					{
						ContractAddress: keeper.BuildContractAddressClassic(1, 1).String(),
						ContractInfo:    wasmTypes.ContractInfoFixture(func(c *wasmTypes.ContractInfo) { c.CodeID = 1 }, wasmTypes.OnlyGenesisFields),
					},
				},
				Params: wasmTypes.DefaultParams(),
			},
		},
		"prevent duplicate contract address": {
			src: types.GenesisState{
				Codes: []wasmTypes.Code{{
					CodeID:    firstCodeID,
					CodeInfo:  myCodeInfo,
					CodeBytes: wasmCode,
				}},
				Contracts: []wasmTypes.Contract{
					{
						ContractAddress: keeper.BuildContractAddressClassic(1, 1).String(),
						ContractInfo:    wasmTypes.ContractInfoFixture(func(c *wasmTypes.ContractInfo) { c.CodeID = 1 }, wasmTypes.OnlyGenesisFields),
					}, {
						ContractAddress: keeper.BuildContractAddressClassic(1, 1).String(),
						ContractInfo:    wasmTypes.ContractInfoFixture(func(c *wasmTypes.ContractInfo) { c.CodeID = 1 }, wasmTypes.OnlyGenesisFields),
					},
				},
				Params: wasmTypes.DefaultParams(),
			},
		},
		"prevent duplicate contract model keys": {
			src: types.GenesisState{
				Codes: []wasmTypes.Code{{
					CodeID:    firstCodeID,
					CodeInfo:  myCodeInfo,
					CodeBytes: wasmCode,
				}},
				Contracts: []wasmTypes.Contract{
					{
						ContractAddress: keeper.BuildContractAddressClassic(1, 1).String(),
						ContractInfo:    wasmTypes.ContractInfoFixture(func(c *wasmTypes.ContractInfo) { c.CodeID = 1 }, wasmTypes.OnlyGenesisFields),
						ContractState: []wasmTypes.Model{
							{
								Key:   []byte{0x1},
								Value: []byte("foo"),
							},
							{
								Key:   []byte{0x1},
								Value: []byte("bar"),
							},
						},
					},
				},
				Params: wasmTypes.DefaultParams(),
			},
		},
		"prevent duplicate sequences": {
			src: types.GenesisState{
				Sequences: []wasmTypes.Sequence{
					{IDKey: []byte("foo"), Value: 1},
					{IDKey: []byte("foo"), Value: 9999},
				},
				Params: wasmTypes.DefaultParams(),
			},
		},
		"prevent code id seq init value == max codeID used": {
			src: types.GenesisState{
				Codes: []wasmTypes.Code{{
					CodeID:    2,
					CodeInfo:  myCodeInfo,
					CodeBytes: wasmCode,
				}},
				Sequences: []wasmTypes.Sequence{
					{IDKey: wasmTypes.KeyLastCodeID, Value: 1},
				},
				Params: wasmTypes.DefaultParams(),
			},
		},
		"prevent contract id seq init value == count contracts": {
			src: types.GenesisState{
				Codes: []wasmTypes.Code{{
					CodeID:    firstCodeID,
					CodeInfo:  myCodeInfo,
					CodeBytes: wasmCode,
				}},
				Contracts: []wasmTypes.Contract{
					{
						ContractAddress: keeper.BuildContractAddressClassic(1, 1).String(),
						ContractInfo:    wasmTypes.ContractInfoFixture(func(c *wasmTypes.ContractInfo) { c.CodeID = 1 }, wasmTypes.OnlyGenesisFields),
					},
				},
				Sequences: []wasmTypes.Sequence{
					{IDKey: wasmTypes.KeyLastCodeID, Value: 2},
					{IDKey: wasmTypes.KeyLastInstanceID, Value: 1},
				},
				Params: wasmTypes.DefaultParams(),
			},
		},
		"validator set update called for any genesis messages": {
			src: types.GenesisState{
				GenMsgs: []wasmTypes.GenesisState_GenMsgs{
					{Sum: &wasmTypes.GenesisState_GenMsgs_StoreCode{
						StoreCode: wasmTypes.MsgStoreCodeFixture(),
					}},
				},
				Params: wasmTypes.DefaultParams(),
			},
			stakingMock: StakingKeeperMock{expCalls: 1, validatorUpdate: []abci.ValidatorUpdate{
				{
					PubKey: crypto.PublicKey{Sum: &crypto.PublicKey_Ed25519{
						Ed25519: []byte("a valid key"),
					}},
					Power: 100,
				},
			}},
			msgHandlerMock: MockMsgHandler{expCalls: 1, expMsg: wasmTypes.MsgStoreCodeFixture()},
			expSuccess:     true,
		},
		"validator set update not called on genesis msg handler errors": {
			src: types.GenesisState{
				GenMsgs: []wasmTypes.GenesisState_GenMsgs{
					{Sum: &wasmTypes.GenesisState_GenMsgs_StoreCode{
						StoreCode: wasmTypes.MsgStoreCodeFixture(),
					}},
				},
				Params: wasmTypes.DefaultParams(),
			},
			msgHandlerMock: MockMsgHandler{expCalls: 1, err: errors.New("test error response")},
			stakingMock:    StakingKeeperMock{expCalls: 0},
			expSuccess:     false,
		},
		"happy path: inactiveContract": {
			src: types.GenesisState{
				Codes: []wasmTypes.Code{{
					CodeID:    firstCodeID,
					CodeInfo:  myCodeInfo,
					CodeBytes: wasmCode,
				}},
				Contracts: []wasmTypes.Contract{
					{
						ContractAddress: keeper.BuildContractAddressClassic(1, 1).String(),
						ContractInfo:    wasmTypes.ContractInfoFixture(func(c *wasmTypes.ContractInfo) { c.CodeID = 1 }, wasmTypes.OnlyGenesisFields),
					},
				},
				Sequences: []wasmTypes.Sequence{
					{IDKey: wasmTypes.KeyLastCodeID, Value: 2},
					{IDKey: wasmTypes.KeyLastInstanceID, Value: 3},
				},
				Params:                    wasmTypes.DefaultParams(),
				InactiveContractAddresses: []string{keeper.BuildContractAddressClassic(1, 1).String()},
			},
			expSuccess: true,
		},
		"invalid path: inactiveContract - human address": {
			src: types.GenesisState{
				Codes: []wasmTypes.Code{{
					CodeID:    firstCodeID,
					CodeInfo:  myCodeInfo,
					CodeBytes: wasmCode,
				}},
				Sequences: []wasmTypes.Sequence{
					{IDKey: wasmTypes.KeyLastCodeID, Value: 2},
					{IDKey: wasmTypes.KeyLastInstanceID, Value: 1},
				},
				Params:                    wasmTypes.DefaultParams(),
				InactiveContractAddresses: []string{humanAddress},
			},
		},
		"invalid path: inactiveContract - do not imported": {
			src: types.GenesisState{
				Codes: []wasmTypes.Code{{
					CodeID:    firstCodeID,
					CodeInfo:  myCodeInfo,
					CodeBytes: wasmCode,
				}},
				Sequences: []wasmTypes.Sequence{
					{IDKey: wasmTypes.KeyLastCodeID, Value: 2},
					{IDKey: wasmTypes.KeyLastInstanceID, Value: 1},
				},
				Params:                    wasmTypes.DefaultParams(),
				InactiveContractAddresses: []string{keeper.BuildContractAddressClassic(1, 1).String()},
			},
		},
	}
	for msg, spec := range specs {
		t.Run(msg, func(t *testing.T) {
			keeper, ctx, _ := setupKeeper(t)

			require.NoError(t, types.ValidateGenesis(spec.src))
			gotValidatorSet, gotErr := InitGenesis(ctx, keeper, spec.src, &spec.stakingMock, spec.msgHandlerMock.Handle)
			if !spec.expSuccess {
				require.Error(t, gotErr)
				return
			}
			require.NoError(t, gotErr)
			spec.msgHandlerMock.verifyCalls(t)
			spec.stakingMock.verifyCalls(t)
			assert.Equal(t, spec.stakingMock.validatorUpdate, gotValidatorSet)
			for _, c := range spec.src.Codes {
				assert.Equal(t, c.Pinned, keeper.IsPinnedCode(ctx, c.CodeID))
			}
		})
	}
}

func setupKeeper(t *testing.T) (*Keeper, sdk.Context, []sdk.StoreKey) {
	t.Helper()
	tempDir, err := os.MkdirTemp("", "wasm")
	require.NoError(t, err)
	t.Cleanup(func() { os.RemoveAll(tempDir) })

	db := dbm.NewMemDB()
	ms := store.NewCommitMultiStore(db)

	ctx := sdk.NewContext(ms, tmproto.Header{
		Height: 1234567,
		Time:   time.Date(2020, time.April, 22, 12, 0, 0, 0, time.UTC),
	}, false, log.NewNopLogger())

	encodingConfig := MakeEncodingConfig(t)
	// register an example extension. must be protobuf
	encodingConfig.InterfaceRegistry.RegisterImplementations(
		(*wasmTypes.ContractInfoExtension)(nil),
		&govtypes.Proposal{},
	)
	// also registering gov interfaces for nested Any type
	govtypes.RegisterInterfaces(encodingConfig.InterfaceRegistry)

	wasmConfig := wasmTypes.DefaultWasmConfig()

	keys := sdk.NewKVStoreKeys(
		authtypes.StoreKey, banktypes.StoreKey, paramstypes.StoreKey, types.StoreKey,
	)
	for _, v := range keys {
		ms.MountStoreWithDB(v, sdk.StoreTypeIAVL, db)
	}
	tkeys := sdk.NewTransientStoreKeys(paramstypes.TStoreKey)
	for _, v := range tkeys {
		ms.MountStoreWithDB(v, sdk.StoreTypeTransient, db)
	}
	require.NoError(t, ms.LoadLatestVersion())
	appCodec, legacyAmino := encodingConfig.Marshaler, encodingConfig.Amino

	paramsKeeper := paramskeeper.NewKeeper(
		appCodec,
		legacyAmino,
		keys[paramstypes.StoreKey],
		tkeys[paramstypes.TStoreKey],
	)
	for _, m := range []string{
		authtypes.ModuleName,
		banktypes.ModuleName,
		types.ModuleName,
	} {
		paramsKeeper.Subspace(m)
	}
	subspace := func(m string) paramstypes.Subspace {
		r, ok := paramsKeeper.GetSubspace(m)
		require.True(t, ok)
		return r
	}
	maccPerms := map[string][]string{ // module account permissions
		types.ModuleName: {authtypes.Burner},
	}

	accountKeeper := authkeeper.NewAccountKeeper(
		appCodec,
		keys[authtypes.StoreKey], // target store
		subspace(authtypes.ModuleName),
		authtypes.ProtoBaseAccount, // prototype
		maccPerms,
	)

	bankKeeper := bankpluskeeper.NewBaseKeeper(
		appCodec,
		keys[banktypes.StoreKey],
		accountKeeper,
		subspace(banktypes.ModuleName),
		map[string]bool{},
		false,
	)

	srcKeeper := NewKeeper(
		appCodec,
		keys[types.StoreKey],
		subspace(types.ModuleName),
		accountKeeper,
		bankKeeper,
		stakingkeeper.Keeper{},
		distributionkeeper.Keeper{},
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		tempDir,
		wasmConfig,
		AvailableCapabilities,
	)
	return &srcKeeper, ctx, []sdk.StoreKey{keys[types.StoreKey], keys[paramstypes.StoreKey]}
}

type StakingKeeperMock struct {
	err             error
	validatorUpdate []abci.ValidatorUpdate
	expCalls        int
	gotCalls        int
}

func (s *StakingKeeperMock) ApplyAndReturnValidatorSetUpdates(_ sdk.Context) ([]abci.ValidatorUpdate, error) {
	s.gotCalls++
	return s.validatorUpdate, s.err
}

func (s *StakingKeeperMock) verifyCalls(t *testing.T) {
	assert.Equal(t, s.expCalls, s.gotCalls, "number calls")
}

type MockMsgHandler struct {
	result   *sdk.Result
	err      error
	expCalls int
	gotCalls int
	expMsg   sdk.Msg
	gotMsg   sdk.Msg
}

func (m *MockMsgHandler) Handle(ctx sdk.Context, msg sdk.Msg) (*sdk.Result, error) {
	m.gotCalls++
	m.gotMsg = msg
	return m.result, m.err
}

func (m *MockMsgHandler) verifyCalls(t *testing.T) {
	assert.Equal(t, m.expMsg, m.gotMsg, "message param")
	assert.Equal(t, m.expCalls, m.gotCalls, "number calls")
}
