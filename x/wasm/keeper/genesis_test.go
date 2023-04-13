package keeper

import (
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"math/rand"
	"os"
	"testing"
	"time"

	"github.com/cosmos/cosmos-sdk/store"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	distributionkeeper "github.com/cosmos/cosmos-sdk/x/distribution/keeper"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	paramskeeper "github.com/cosmos/cosmos-sdk/x/params/keeper"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	fuzz "github.com/google/gofuzz"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tendermint/tendermint/libs/log"
	tmproto "github.com/tendermint/tendermint/proto/tendermint/types"
	dbm "github.com/tendermint/tm-db"

	"github.com/CosmWasm/wasmd/x/wasm/types"
	wasmTypes "github.com/CosmWasm/wasmd/x/wasm/types"
)

const firstCodeID = 1

func TestGenesisExportImport(t *testing.T) {
	wasmKeeper, srcCtx, srcStoreKeys := setupKeeper(t)
	contractKeeper := NewGovPermissionKeeper(wasmKeeper)

	wasmCode, err := os.ReadFile("./testdata/hackatom.wasm")
	require.NoError(t, err)

	// store some test data
	f := fuzz.New().Funcs(ModelFuzzers...)

	wasmKeeper.SetParams(srcCtx, types.DefaultParams())

	for i := 0; i < 25; i++ {
		var (
			codeInfo          types.CodeInfo
			contract          types.ContractInfo
			stateModels       []types.Model
			history           []types.ContractCodeHistoryEntry
			pinned            bool
			contractExtension bool
		)
		f.Fuzz(&codeInfo)
		f.Fuzz(&contract)
		f.Fuzz(&stateModels)
		f.NilChance(0).Fuzz(&history)
		f.Fuzz(&pinned)
		f.Fuzz(&contractExtension)

		creatorAddr, err := sdk.AccAddressFromBech32(codeInfo.Creator)
		require.NoError(t, err)
		codeID, _, err := contractKeeper.Create(srcCtx, creatorAddr, wasmCode, &codeInfo.InstantiateConfig)
		require.NoError(t, err)
		if pinned {
			contractKeeper.PinCode(srcCtx, codeID)
		}
		if contractExtension {
			anyTime := time.Now().UTC()
			var nestedType govtypes.TextProposal
			f.NilChance(0).Fuzz(&nestedType)
			myExtension, err := govtypes.NewProposal(&nestedType, 1, anyTime, anyTime)
			require.NoError(t, err)
			contract.SetExtension(&myExtension)
		}

		contract.CodeID = codeID
		contractAddr := wasmKeeper.ClassicAddressGenerator()(srcCtx, codeID, nil)
		wasmKeeper.storeContractInfo(srcCtx, contractAddr, &contract)
		wasmKeeper.appendToContractHistory(srcCtx, contractAddr, history...)
		wasmKeeper.importContractState(srcCtx, contractAddr, stateModels)
	}
	var wasmParams types.Params
	f.NilChance(0).Fuzz(&wasmParams)
	wasmKeeper.SetParams(srcCtx, wasmParams)

	// export
	exportedState := ExportGenesis(srcCtx, wasmKeeper)
	// order should not matter
	rand.Shuffle(len(exportedState.Codes), func(i, j int) {
		exportedState.Codes[i], exportedState.Codes[j] = exportedState.Codes[j], exportedState.Codes[i]
	})
	rand.Shuffle(len(exportedState.Contracts), func(i, j int) {
		exportedState.Contracts[i], exportedState.Contracts[j] = exportedState.Contracts[j], exportedState.Contracts[i]
	})
	rand.Shuffle(len(exportedState.Sequences), func(i, j int) {
		exportedState.Sequences[i], exportedState.Sequences[j] = exportedState.Sequences[j], exportedState.Sequences[i]
	})
	exportedGenesis, err := wasmKeeper.cdc.MarshalJSON(exportedState)
	require.NoError(t, err)

	// setup new instances
	dstKeeper, dstCtx, dstStoreKeys := setupKeeper(t)

	// reset contract code index in source DB for comparison with dest DB
	wasmKeeper.IterateContractInfo(srcCtx, func(address sdk.AccAddress, info wasmTypes.ContractInfo) bool {
		creatorAddress := sdk.MustAccAddressFromBech32(info.Creator)
		history := wasmKeeper.GetContractHistory(srcCtx, address)

		wasmKeeper.addToContractCodeSecondaryIndex(srcCtx, address, history[len(history)-1])
		wasmKeeper.addToContractCreatorSecondaryIndex(srcCtx, creatorAddress, history[0].Updated, address)
		return false
	})

	// re-import
	var importState wasmTypes.GenesisState
	err = dstKeeper.cdc.UnmarshalJSON(exportedGenesis, &importState)
	require.NoError(t, err)
	InitGenesis(dstCtx, dstKeeper, importState)

	// compare whole DB
	for j := range srcStoreKeys {
		srcIT := srcCtx.KVStore(srcStoreKeys[j]).Iterator(nil, nil)
		dstIT := dstCtx.KVStore(dstStoreKeys[j]).Iterator(nil, nil)

		for i := 0; srcIT.Valid(); i++ {
			require.True(t, dstIT.Valid(), "[%s] destination DB has less elements than source. Missing: %x", srcStoreKeys[j].Name(), srcIT.Key())
			require.Equal(t, srcIT.Key(), dstIT.Key(), i)
			require.Equal(t, srcIT.Value(), dstIT.Value(), "[%s] element (%d): %X", srcStoreKeys[j].Name(), i, srcIT.Key())
			dstIT.Next()
			srcIT.Next()
		}
		if !assert.False(t, dstIT.Valid()) {
			t.Fatalf("dest Iterator still has key :%X", dstIT.Key())
		}
		srcIT.Close()
		dstIT.Close()
	}
}

func TestGenesisInit(t *testing.T) {
	wasmCode, err := os.ReadFile("./testdata/hackatom.wasm")
	require.NoError(t, err)

	myCodeInfo := wasmTypes.CodeInfoFixture(wasmTypes.WithSHA256CodeHash(wasmCode))
	specs := map[string]struct {
		src        types.GenesisState
		expSuccess bool
	}{
		"happy path: code info correct": {
			src: types.GenesisState{
				Codes: []types.Code{{
					CodeID:    firstCodeID,
					CodeInfo:  myCodeInfo,
					CodeBytes: wasmCode,
				}},
				Sequences: []types.Sequence{
					{IDKey: types.KeyLastCodeID, Value: 2},
					{IDKey: types.KeyLastInstanceID, Value: 1},
				},
				Params: types.DefaultParams(),
			},
			expSuccess: true,
		},
		"happy path: code ids can contain gaps": {
			src: types.GenesisState{
				Codes: []types.Code{{
					CodeID:    firstCodeID,
					CodeInfo:  myCodeInfo,
					CodeBytes: wasmCode,
				}, {
					CodeID:    3,
					CodeInfo:  myCodeInfo,
					CodeBytes: wasmCode,
				}},
				Sequences: []types.Sequence{
					{IDKey: types.KeyLastCodeID, Value: 10},
					{IDKey: types.KeyLastInstanceID, Value: 1},
				},
				Params: types.DefaultParams(),
			},
			expSuccess: true,
		},
		"happy path: code order does not matter": {
			src: types.GenesisState{
				Codes: []types.Code{{
					CodeID:    2,
					CodeInfo:  myCodeInfo,
					CodeBytes: wasmCode,
				}, {
					CodeID:    firstCodeID,
					CodeInfo:  myCodeInfo,
					CodeBytes: wasmCode,
				}},
				Contracts: nil,
				Sequences: []types.Sequence{
					{IDKey: types.KeyLastCodeID, Value: 3},
					{IDKey: types.KeyLastInstanceID, Value: 1},
				},
				Params: types.DefaultParams(),
			},
			expSuccess: true,
		},
		"prevent code hash mismatch": {src: types.GenesisState{
			Codes: []types.Code{{
				CodeID:    firstCodeID,
				CodeInfo:  wasmTypes.CodeInfoFixture(func(i *wasmTypes.CodeInfo) { i.CodeHash = make([]byte, sha256.Size) }),
				CodeBytes: wasmCode,
			}},
			Params: types.DefaultParams(),
		}},
		"prevent duplicate codeIDs": {src: types.GenesisState{
			Codes: []types.Code{
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
			Params: types.DefaultParams(),
		}},
		"codes with same checksum can be pinned": {
			src: types.GenesisState{
				Codes: []types.Code{
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
				Params: types.DefaultParams(),
			},
		},
		"happy path: code id in info and contract do match": {
			src: types.GenesisState{
				Codes: []types.Code{{
					CodeID:    firstCodeID,
					CodeInfo:  myCodeInfo,
					CodeBytes: wasmCode,
				}},
				Contracts: []types.Contract{
					{
						ContractAddress: BuildContractAddressClassic(1, 1).String(),
						ContractInfo:    types.ContractInfoFixture(func(c *wasmTypes.ContractInfo) { c.CodeID = 1 }, types.RandCreatedFields),
						ContractCodeHistory: []types.ContractCodeHistoryEntry{
							{
								Operation: types.ContractCodeHistoryOperationTypeMigrate,
								CodeID:    1,
								Updated:   &types.AbsoluteTxPosition{BlockHeight: rand.Uint64(), TxIndex: rand.Uint64()},
								Msg:       []byte(`{}`),
							},
						},
					},
				},
				Sequences: []types.Sequence{
					{IDKey: types.KeyLastCodeID, Value: 2},
					{IDKey: types.KeyLastInstanceID, Value: 2},
				},
				Params: types.DefaultParams(),
			},
			expSuccess: true,
		},
		"happy path: code info with two contracts": {
			src: types.GenesisState{
				Codes: []types.Code{{
					CodeID:    firstCodeID,
					CodeInfo:  myCodeInfo,
					CodeBytes: wasmCode,
				}},
				Contracts: []types.Contract{
					{
						ContractAddress: BuildContractAddressClassic(1, 1).String(),
						ContractInfo:    types.ContractInfoFixture(func(c *wasmTypes.ContractInfo) { c.CodeID = 1 }, types.RandCreatedFields),
						ContractCodeHistory: []types.ContractCodeHistoryEntry{
							{
								Operation: types.ContractCodeHistoryOperationTypeMigrate,
								CodeID:    1,
								Updated:   &types.AbsoluteTxPosition{BlockHeight: rand.Uint64(), TxIndex: rand.Uint64()},
								Msg:       []byte(`{}`),
							},
						},
					}, {
						ContractAddress: BuildContractAddressClassic(1, 2).String(),
						ContractInfo:    types.ContractInfoFixture(func(c *wasmTypes.ContractInfo) { c.CodeID = 1 }, types.RandCreatedFields),
						ContractCodeHistory: []types.ContractCodeHistoryEntry{
							{
								Operation: types.ContractCodeHistoryOperationTypeMigrate,
								CodeID:    1,
								Updated:   &types.AbsoluteTxPosition{BlockHeight: rand.Uint64(), TxIndex: rand.Uint64()},
								Msg:       []byte(`{"foo":"bar"}`),
							},
						},
					},
				},
				Sequences: []types.Sequence{
					{IDKey: types.KeyLastCodeID, Value: 2},
					{IDKey: types.KeyLastInstanceID, Value: 3},
				},
				Params: types.DefaultParams(),
			},
			expSuccess: true,
		},
		"prevent contracts that points to non existing codeID": {
			src: types.GenesisState{
				Contracts: []types.Contract{
					{
						ContractAddress: BuildContractAddressClassic(1, 1).String(),
						ContractInfo:    types.ContractInfoFixture(func(c *wasmTypes.ContractInfo) { c.CodeID = 1 }, types.RandCreatedFields),
						ContractCodeHistory: []types.ContractCodeHistoryEntry{
							{
								Operation: types.ContractCodeHistoryOperationTypeMigrate,
								CodeID:    1,
								Updated:   &types.AbsoluteTxPosition{BlockHeight: rand.Uint64(), TxIndex: rand.Uint64()},
								Msg:       []byte(`{"foo":"bar"}`),
							},
						},
					},
				},
				Params: types.DefaultParams(),
			},
		},
		"prevent duplicate contract address": {
			src: types.GenesisState{
				Codes: []types.Code{{
					CodeID:    firstCodeID,
					CodeInfo:  myCodeInfo,
					CodeBytes: wasmCode,
				}},
				Contracts: []types.Contract{
					{
						ContractAddress: BuildContractAddressClassic(1, 1).String(),
						ContractInfo:    types.ContractInfoFixture(func(c *wasmTypes.ContractInfo) { c.CodeID = 1 }, types.RandCreatedFields),
						ContractCodeHistory: []types.ContractCodeHistoryEntry{
							{
								Operation: types.ContractCodeHistoryOperationTypeMigrate,
								CodeID:    1,
								Updated:   &types.AbsoluteTxPosition{BlockHeight: rand.Uint64(), TxIndex: rand.Uint64()},
								Msg:       []byte(`{"foo":"bar"}`),
							},
						},
					}, {
						ContractAddress: BuildContractAddressClassic(1, 1).String(),
						ContractInfo:    types.ContractInfoFixture(func(c *wasmTypes.ContractInfo) { c.CodeID = 1 }, types.RandCreatedFields),
						ContractCodeHistory: []types.ContractCodeHistoryEntry{
							{
								Operation: types.ContractCodeHistoryOperationTypeMigrate,
								CodeID:    1,
								Updated:   &types.AbsoluteTxPosition{BlockHeight: rand.Uint64(), TxIndex: rand.Uint64()},
								Msg:       []byte(`{"other":"value"}`),
							},
						},
					},
				},
				Params: types.DefaultParams(),
			},
		},
		"prevent duplicate contract model keys": {
			src: types.GenesisState{
				Codes: []types.Code{{
					CodeID:    firstCodeID,
					CodeInfo:  myCodeInfo,
					CodeBytes: wasmCode,
				}},
				Contracts: []types.Contract{
					{
						ContractAddress: BuildContractAddressClassic(1, 1).String(),
						ContractInfo:    types.ContractInfoFixture(func(c *wasmTypes.ContractInfo) { c.CodeID = 1 }, types.RandCreatedFields),
						ContractState: []types.Model{
							{
								Key:   []byte{0x1},
								Value: []byte("foo"),
							},
							{
								Key:   []byte{0x1},
								Value: []byte("bar"),
							},
						},
						ContractCodeHistory: []types.ContractCodeHistoryEntry{
							{
								Operation: types.ContractCodeHistoryOperationTypeMigrate,
								CodeID:    1,
								Updated:   &types.AbsoluteTxPosition{BlockHeight: rand.Uint64(), TxIndex: rand.Uint64()},
								Msg:       []byte(`{"foo":"bar"}`),
							},
						},
					},
				},
				Params: types.DefaultParams(),
			},
		},
		"prevent duplicate sequences": {
			src: types.GenesisState{
				Sequences: []types.Sequence{
					{IDKey: []byte("foo"), Value: 1},
					{IDKey: []byte("foo"), Value: 9999},
				},
				Params: types.DefaultParams(),
			},
		},
		"prevent code id seq init value == max codeID used": {
			src: types.GenesisState{
				Codes: []types.Code{{
					CodeID:    2,
					CodeInfo:  myCodeInfo,
					CodeBytes: wasmCode,
				}},
				Sequences: []types.Sequence{
					{IDKey: types.KeyLastCodeID, Value: 1},
				},
				Params: types.DefaultParams(),
			},
		},
		"prevent contract id seq init value == count contracts": {
			src: types.GenesisState{
				Codes: []types.Code{{
					CodeID:    firstCodeID,
					CodeInfo:  myCodeInfo,
					CodeBytes: wasmCode,
				}},
				Contracts: []types.Contract{
					{
						ContractAddress: BuildContractAddressClassic(1, 1).String(),
						ContractInfo:    types.ContractInfoFixture(func(c *wasmTypes.ContractInfo) { c.CodeID = 1 }, types.RandCreatedFields),
						ContractCodeHistory: []types.ContractCodeHistoryEntry{
							{
								Operation: types.ContractCodeHistoryOperationTypeMigrate,
								CodeID:    1,
								Updated:   &types.AbsoluteTxPosition{BlockHeight: rand.Uint64(), TxIndex: rand.Uint64()},
								Msg:       []byte(`{}`),
							},
						},
					},
				},
				Sequences: []types.Sequence{
					{IDKey: types.KeyLastCodeID, Value: 2},
					{IDKey: types.KeyLastInstanceID, Value: 1},
				},
				Params: types.DefaultParams(),
			},
		},
	}
	for msg, spec := range specs {
		t.Run(msg, func(t *testing.T) {
			keeper, ctx, _ := setupKeeper(t)

			require.NoError(t, types.ValidateGenesis(spec.src))
			_, gotErr := InitGenesis(ctx, keeper, spec.src)
			if !spec.expSuccess {
				require.Error(t, gotErr)
				return
			}
			require.NoError(t, gotErr)

			for _, c := range spec.src.Codes {
				assert.Equal(t, c.Pinned, keeper.IsPinnedCode(ctx, c.CodeID))
			}
		})
	}
}

func TestImportContractWithCodeHistoryPreserved(t *testing.T) {
	genesisTemplate := `
{
	"params":{
		"code_upload_access": {
			"permission": "Everybody"
		},
		"instantiate_default_permission": "Everybody"
	},
  "codes": [
    {
      "code_id": "1",
      "code_info": {
        "code_hash": %q,
        "creator": "cosmos1qtu5n0cnhfkjj6l2rq97hmky9fd89gwca9yarx",
        "instantiate_config": {
          "permission": "OnlyAddress",
          "address": "cosmos1qtu5n0cnhfkjj6l2rq97hmky9fd89gwca9yarx"
        }
      },
      "code_bytes": %q
    }
  ],
  "contracts": [
    {
      "contract_address": "cosmos14hj2tavq8fpesdwxxcu44rty3hh90vhujrvcmstl4zr3txmfvw9s4hmalr",
      "contract_info": {
        "code_id": "1",
        "creator": "cosmos13x849jzd03vne42ynpj25hn8npjecxqrjghd8x",
        "admin": "cosmos1h5t8zxmjr30e9dqghtlpl40f2zz5cgey6esxtn",
        "label": "ȀĴnZV芢毤",
		"created": {
			"block_height" : "100",
			"tx_index" : "10"
		}
      },
	  "contract_code_history": [
		{
			"operation": "CONTRACT_CODE_HISTORY_OPERATION_TYPE_INIT",
			"code_id": "1",
			"updated": {
				"block_height" : "100",
				"tx_index" : "10"
			},
			"msg": {"foo": "bar"}
	  	},
		{
			"operation": "CONTRACT_CODE_HISTORY_OPERATION_TYPE_MIGRATE",
			"code_id": "1",
			"updated": {
				"block_height" : "200",
				"tx_index" : "10"
			},
			"msg": {"other": "msg"}
	  	}
		]
    }
  ],
  "sequences": [
  {"id_key": "BGxhc3RDb2RlSWQ=", "value": "2"},
  {"id_key": "BGxhc3RDb250cmFjdElk", "value": "3"}
  ]
}`
	keeper, ctx, _ := setupKeeper(t)

	wasmCode, err := os.ReadFile("./testdata/hackatom.wasm")
	require.NoError(t, err)

	wasmCodeHash := sha256.Sum256(wasmCode)
	enc64 := base64.StdEncoding.EncodeToString
	genesisStr := fmt.Sprintf(genesisTemplate, enc64(wasmCodeHash[:]), enc64(wasmCode))

	var importState wasmTypes.GenesisState
	err = keeper.cdc.UnmarshalJSON([]byte(genesisStr), &importState)
	require.NoError(t, err)
	require.NoError(t, importState.ValidateBasic(), genesisStr)

	ctx = ctx.WithBlockHeight(0).WithGasMeter(sdk.NewInfiniteGasMeter())

	// when
	_, err = InitGenesis(ctx, keeper, importState)
	require.NoError(t, err)

	// verify wasm code
	gotWasmCode, err := keeper.GetByteCode(ctx, 1)
	require.NoError(t, err)
	assert.Equal(t, wasmCode, gotWasmCode, "byte code does not match")

	// verify code info
	gotCodeInfo := keeper.GetCodeInfo(ctx, 1)
	require.NotNil(t, gotCodeInfo)
	codeCreatorAddr := "cosmos1qtu5n0cnhfkjj6l2rq97hmky9fd89gwca9yarx"
	expCodeInfo := types.CodeInfo{
		CodeHash: wasmCodeHash[:],
		Creator:  codeCreatorAddr,
		InstantiateConfig: wasmTypes.AccessConfig{
			Permission: types.AccessTypeOnlyAddress,
			Address:    codeCreatorAddr,
		},
	}
	assert.Equal(t, expCodeInfo, *gotCodeInfo)

	// verify contract
	contractAddr, _ := sdk.AccAddressFromBech32("cosmos14hj2tavq8fpesdwxxcu44rty3hh90vhujrvcmstl4zr3txmfvw9s4hmalr")
	gotContractInfo := keeper.GetContractInfo(ctx, contractAddr)
	require.NotNil(t, gotContractInfo)
	contractCreatorAddr := "cosmos13x849jzd03vne42ynpj25hn8npjecxqrjghd8x"
	adminAddr := "cosmos1h5t8zxmjr30e9dqghtlpl40f2zz5cgey6esxtn"

	expContractInfo := types.ContractInfo{
		CodeID:  firstCodeID,
		Creator: contractCreatorAddr,
		Admin:   adminAddr,
		Label:   "ȀĴnZV芢毤",
		Created: &types.AbsoluteTxPosition{BlockHeight: 100, TxIndex: 10},
	}
	assert.Equal(t, expContractInfo, *gotContractInfo)

	expHistory := []types.ContractCodeHistoryEntry{
		{
			Operation: types.ContractCodeHistoryOperationTypeInit,
			CodeID:    firstCodeID,
			Updated: &types.AbsoluteTxPosition{
				BlockHeight: 100,
				TxIndex:     10,
			},
			Msg: []byte(`{"foo": "bar"}`),
		},
		{
			Operation: types.ContractCodeHistoryOperationTypeMigrate,
			CodeID:    firstCodeID,
			Updated: &types.AbsoluteTxPosition{
				BlockHeight: 200,
				TxIndex:     10,
			},
			Msg: []byte(`{"other": "msg"}`),
		},
	}
	assert.Equal(t, expHistory, keeper.GetContractHistory(ctx, contractAddr))
	assert.Equal(t, uint64(2), keeper.PeekAutoIncrementID(ctx, types.KeyLastCodeID))
	assert.Equal(t, uint64(3), keeper.PeekAutoIncrementID(ctx, types.KeyLastInstanceID))
}

func setupKeeper(t *testing.T) (*Keeper, sdk.Context, []sdk.StoreKey) {
	t.Helper()
	tempDir, err := os.MkdirTemp("", "wasm")
	require.NoError(t, err)
	t.Cleanup(func() { os.RemoveAll(tempDir) })
	var (
		keyParams  = sdk.NewKVStoreKey(paramtypes.StoreKey)
		tkeyParams = sdk.NewTransientStoreKey(paramtypes.TStoreKey)
		keyWasm    = sdk.NewKVStoreKey(wasmTypes.StoreKey)
	)

	db := dbm.NewMemDB()
	ms := store.NewCommitMultiStore(db)
	ms.MountStoreWithDB(keyWasm, sdk.StoreTypeIAVL, db)
	ms.MountStoreWithDB(keyParams, sdk.StoreTypeIAVL, db)
	ms.MountStoreWithDB(tkeyParams, sdk.StoreTypeTransient, db)
	require.NoError(t, ms.LoadLatestVersion())

	ctx := sdk.NewContext(ms, tmproto.Header{
		Height: 1234567,
		Time:   time.Date(2020, time.April, 22, 12, 0, 0, 0, time.UTC),
	}, false, log.NewNopLogger())

	encodingConfig := MakeEncodingConfig(t)
	// register an example extension. must be protobuf
	encodingConfig.InterfaceRegistry.RegisterImplementations(
		(*types.ContractInfoExtension)(nil),
		&govtypes.Proposal{},
	)
	// also registering gov interfaces for nested Any type
	govtypes.RegisterInterfaces(encodingConfig.InterfaceRegistry)

	wasmConfig := wasmTypes.DefaultWasmConfig()
	pk := paramskeeper.NewKeeper(encodingConfig.Marshaler, encodingConfig.Amino, keyParams, tkeyParams)

	srcKeeper := NewKeeper(
		encodingConfig.Marshaler,
		keyWasm,
		pk.Subspace(wasmTypes.ModuleName),
		authkeeper.AccountKeeper{},
		&bankkeeper.BaseKeeper{},
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
	return &srcKeeper, ctx, []sdk.StoreKey{keyWasm, keyParams}
}
