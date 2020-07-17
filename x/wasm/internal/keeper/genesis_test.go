package keeper

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"testing"
	"time"

	"github.com/CosmWasm/wasmd/x/wasm/internal/types"
	wasmTypes "github.com/CosmWasm/wasmd/x/wasm/internal/types"
	"github.com/cosmos/cosmos-sdk/store"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth"
	"github.com/cosmos/cosmos-sdk/x/params"
	"github.com/cosmos/cosmos-sdk/x/staking"
	fuzz "github.com/google/gofuzz"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	abci "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/libs/log"
	dbm "github.com/tendermint/tm-db"
)

func TestGenesisExportImport(t *testing.T) {
	srcKeeper, srcCtx, srcStoreKeys, srcCleanup := setupKeeper(t)
	defer srcCleanup()
	wasmCode, err := ioutil.ReadFile("./testdata/contract.wasm")
	require.NoError(t, err)

	// store some test data
	f := fuzz.New().Funcs(ModelFuzzers...)
	for i := 0; i < 25; i++ {
		var (
			codeInfo    types.CodeInfo
			contract    types.ContractInfo
			stateModels []types.Model
		)
		f.Fuzz(&codeInfo)
		f.Fuzz(&contract)
		f.Fuzz(&stateModels)
		codeID, err := srcKeeper.Create(srcCtx, codeInfo.Creator, wasmCode, codeInfo.Source, codeInfo.Builder, &codeInfo.InstantiateConfig)
		require.NoError(t, err)
		contract.CodeID = codeID
		contractAddr := srcKeeper.generateContractAddress(srcCtx, codeID)
		srcKeeper.setContractInfo(srcCtx, contractAddr, &contract)
		srcKeeper.importContractState(srcCtx, contractAddr, stateModels)
	}
	var wasmParams types.Params
	f.Fuzz(&wasmParams)
	srcKeeper.setParams(srcCtx, wasmParams)

	// export
	exportedState := ExportGenesis(srcCtx, srcKeeper)
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
	exportedGenesis, err := json.Marshal(exportedState)
	require.NoError(t, err)

	// reset contract history in source DB for comparision with dest DB
	srcKeeper.IterateContractInfo(srcCtx, func(address sdk.AccAddress, info wasmTypes.ContractInfo) bool {
		info.ResetFromGenesis(srcCtx)
		srcKeeper.setContractInfo(srcCtx, address, &info)
		return false
	})

	// re-import
	dstKeeper, dstCtx, dstStoreKeys, dstCleanup := setupKeeper(t)
	defer dstCleanup()

	var importState wasmTypes.GenesisState
	err = json.Unmarshal(exportedGenesis, &importState)
	require.NoError(t, err)
	InitGenesis(dstCtx, dstKeeper, importState)

	// compare whole DB
	for j := range srcStoreKeys {
		srcIT := srcCtx.KVStore(srcStoreKeys[j]).Iterator(nil, nil)
		dstIT := dstCtx.KVStore(dstStoreKeys[j]).Iterator(nil, nil)

		for i := 0; srcIT.Valid(); i++ {
			require.True(t, dstIT.Valid(), "[%s] destination DB has less elements than source. Missing: %s", srcStoreKeys[j].Name(), srcIT.Key())
			require.Equal(t, srcIT.Key(), dstIT.Key(), i)
			require.Equal(t, srcIT.Value(), dstIT.Value(), "[%s] element (%d): %s", srcStoreKeys[j].Name(), i, srcIT.Key())
			srcIT.Next()
			dstIT.Next()
		}
		require.False(t, dstIT.Valid())
	}
}

func TestFailFastImport(t *testing.T) {
	wasmCode, err := ioutil.ReadFile("./testdata/contract.wasm")
	require.NoError(t, err)

	myCodeInfo := wasmTypes.CodeInfoFixture(wasmTypes.WithSHA256CodeHash(wasmCode))
	specs := map[string]struct {
		src        types.GenesisState
		expSuccess bool
	}{
		"happy path: code info correct": {
			src: types.GenesisState{
				Codes: []types.Code{{
					CodeID:     1,
					CodeInfo:   myCodeInfo,
					CodesBytes: wasmCode,
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
					CodeID:     1,
					CodeInfo:   myCodeInfo,
					CodesBytes: wasmCode,
				}, {
					CodeID:     3,
					CodeInfo:   myCodeInfo,
					CodesBytes: wasmCode,
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
					CodeID:     2,
					CodeInfo:   myCodeInfo,
					CodesBytes: wasmCode,
				}, {
					CodeID:     1,
					CodeInfo:   myCodeInfo,
					CodesBytes: wasmCode,
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
				CodeID:     1,
				CodeInfo:   wasmTypes.CodeInfoFixture(func(i *wasmTypes.CodeInfo) { i.CodeHash = make([]byte, sha256.Size) }),
				CodesBytes: wasmCode,
			}},
			Params: types.DefaultParams(),
		}},
		"prevent duplicate codeIDs": {src: types.GenesisState{
			Codes: []types.Code{
				{
					CodeID:     1,
					CodeInfo:   myCodeInfo,
					CodesBytes: wasmCode,
				},
				{
					CodeID:     1,
					CodeInfo:   myCodeInfo,
					CodesBytes: wasmCode,
				},
			},
			Params: types.DefaultParams(),
		}},
		"happy path: code id in info and contract do match": {
			src: types.GenesisState{
				Codes: []types.Code{{
					CodeID:     1,
					CodeInfo:   myCodeInfo,
					CodesBytes: wasmCode,
				}},
				Contracts: []types.Contract{
					{
						ContractAddress: contractAddress(1, 1),
						ContractInfo:    types.ContractInfoFixture(func(c *wasmTypes.ContractInfo) { c.CodeID = 1 }, types.OnlyGenesisFields),
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
					CodeID:     1,
					CodeInfo:   myCodeInfo,
					CodesBytes: wasmCode,
				}},
				Contracts: []types.Contract{
					{
						ContractAddress: contractAddress(1, 1),
						ContractInfo:    types.ContractInfoFixture(func(c *wasmTypes.ContractInfo) { c.CodeID = 1 }, types.OnlyGenesisFields),
					}, {
						ContractAddress: contractAddress(1, 2),
						ContractInfo:    types.ContractInfoFixture(func(c *wasmTypes.ContractInfo) { c.CodeID = 1 }, types.OnlyGenesisFields),
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
						ContractAddress: contractAddress(1, 1),
						ContractInfo:    types.ContractInfoFixture(func(c *wasmTypes.ContractInfo) { c.CodeID = 1 }, types.OnlyGenesisFields),
					},
				},
				Params: types.DefaultParams(),
			},
		},
		"prevent duplicate contract address": {
			src: types.GenesisState{
				Codes: []types.Code{{
					CodeID:     1,
					CodeInfo:   myCodeInfo,
					CodesBytes: wasmCode,
				}},
				Contracts: []types.Contract{
					{
						ContractAddress: contractAddress(1, 1),
						ContractInfo:    types.ContractInfoFixture(func(c *wasmTypes.ContractInfo) { c.CodeID = 1 }, types.OnlyGenesisFields),
					}, {
						ContractAddress: contractAddress(1, 1),
						ContractInfo:    types.ContractInfoFixture(func(c *wasmTypes.ContractInfo) { c.CodeID = 1 }, types.OnlyGenesisFields),
					},
				},
				Params: types.DefaultParams(),
			},
		},
		"prevent duplicate contract model keys": {
			src: types.GenesisState{
				Codes: []types.Code{{
					CodeID:     1,
					CodeInfo:   myCodeInfo,
					CodesBytes: wasmCode,
				}},
				Contracts: []types.Contract{
					{
						ContractAddress: contractAddress(1, 1),
						ContractInfo:    types.ContractInfoFixture(func(c *wasmTypes.ContractInfo) { c.CodeID = 1 }, types.OnlyGenesisFields),
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
					CodeID:     2,
					CodeInfo:   myCodeInfo,
					CodesBytes: wasmCode,
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
					CodeID:     1,
					CodeInfo:   myCodeInfo,
					CodesBytes: wasmCode,
				}},
				Contracts: []types.Contract{
					{
						ContractAddress: contractAddress(1, 1),
						ContractInfo:    types.ContractInfoFixture(func(c *wasmTypes.ContractInfo) { c.CodeID = 1 }, types.OnlyGenesisFields),
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
			keeper, ctx, _, cleanup := setupKeeper(t)
			defer cleanup()

			require.NoError(t, types.ValidateGenesis(spec.src))
			got := InitGenesis(ctx, keeper, spec.src)
			if spec.expSuccess {
				require.NoError(t, got)
				return
			}
			require.Error(t, got)
		})
	}
}

func TestExportShouldNotContainContractCodeHistory(t *testing.T) {
	tempDir, err := ioutil.TempDir("", "wasm")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)
	ctx, keepers := CreateTestInput(t, false, tempDir, SupportedFeatures, nil, nil)
	accKeeper, keeper := keepers.AccountKeeper, keepers.WasmKeeper

	wasmCode, err := ioutil.ReadFile("./testdata/contract.wasm")
	require.NoError(t, err)
	var (
		deposit = sdk.NewCoins(sdk.NewInt64Coin("denom", 100000))
		creator = createFakeFundedAccount(ctx, accKeeper, deposit)
		anyAddr = make([]byte, sdk.AddrLen)
	)

	firstCodeID, err := keeper.Create(ctx, creator, wasmCode, "https://github.com/CosmWasm/wasmd/blob/master/x/wasm/testdata/escrow.wasm", "", &types.AllowEverybody)
	require.NoError(t, err)
	secondCodeID, err := keeper.Create(ctx, creator, wasmCode, "https://github.com/CosmWasm/wasmd/blob/master/x/wasm/testdata/escrow.wasm", "", &types.AllowEverybody)
	require.NoError(t, err)
	initMsg := InitMsg{
		Verifier:    anyAddr,
		Beneficiary: anyAddr,
	}
	initMsgBz, err := json.Marshal(initMsg)
	require.NoError(t, err)

	// create instance
	ctx = ctx.WithBlockHeight(ctx.BlockHeight() + 1)
	contractAddr, err := keeper.Instantiate(ctx, firstCodeID, creator, creator, initMsgBz, "demo contract 1", nil)
	require.NoError(t, err)

	// and migrate to second code id
	ctx = ctx.WithBlockHeight(ctx.BlockHeight() + 1)
	_, err = keeper.Migrate(ctx, contractAddr, creator, secondCodeID, initMsgBz)
	require.NoError(t, err)
	// and contract contains 2 history elements
	contractInfo := keeper.GetContractInfo(ctx, contractAddr)
	require.NotNil(t, contractInfo)
	require.Len(t, contractInfo.ContractCodeHistory, 2)
	// when exported
	state := ExportGenesis(ctx, keeper)
	require.NoError(t, state.ValidateBasic())
	require.Len(t, state.Contracts, 1)
	assert.Len(t, state.Contracts[0].ContractInfo.ContractCodeHistory, 0)
	assert.Nil(t, state.Contracts[0].ContractInfo.Created)
}

func TestImportContractWithCodeHistoryReset(t *testing.T) {
	genesis := `
{
	"params":{
		"code_upload_access": {
			"permission": "Everybody"
		},
		"instantiate_default_permission": "Everybody"
	},
  "codes": [
    {
      "code_id": 1,
      "code_info": {
        "code_hash": "mrFpzGE5s+Qfn9Xe7OikXc0q169m7sMm4ZuV6pVA8Mc=",
        "creator": "cosmos1qtu5n0cnhfkjj6l2rq97hmky9fd89gwca9yarx",
        "source": "https://example.com",
        "builder": "foo/bar:tag",
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
      "contract_address": "cosmos18vd8fpwxzck93qlwghaj6arh4p7c5n89uzcee5",
      "contract_info": {
        "code_id": 1,
        "creator": "cosmos13x849jzd03vne42ynpj25hn8npjecxqrjghd8x",
        "admin": "cosmos1h5t8zxmjr30e9dqghtlpl40f2zz5cgey6esxtn",
        "label": "ȀĴnZV芢毤"
      }
    }
  ]
}`
	wasmCode, err := ioutil.ReadFile("./testdata/contract.wasm")
	require.NoError(t, err)

	keeper, ctx, _, dstCleanup := setupKeeper(t)
	defer dstCleanup()

	var importState wasmTypes.GenesisState
	err = json.Unmarshal([]byte(fmt.Sprintf(genesis, base64.StdEncoding.EncodeToString(wasmCode))), &importState)
	require.NoError(t, err)
	require.NoError(t, importState.ValidateBasic())
	InitGenesis(ctx, keeper, importState)

	contractAddr, err := sdk.AccAddressFromBech32("cosmos18vd8fpwxzck93qlwghaj6arh4p7c5n89uzcee5")
	require.NoError(t, err)
	contractInfo := keeper.GetContractInfo(ctx, contractAddr)
	require.NotNil(t, contractInfo)
	require.Len(t, contractInfo.ContractCodeHistory, 1)
	exp := []types.ContractCodeHistoryEntry{{
		Operation: types.GenesisContractCodeHistoryType,
		CodeID:    1,
		Updated:   types.NewAbsoluteTxPosition(ctx),
	},
	}
	assert.Equal(t, exp, contractInfo.ContractCodeHistory)
}

func setupKeeper(t *testing.T) (Keeper, sdk.Context, []sdk.StoreKey, func()) {
	t.Helper()
	tempDir, err := ioutil.TempDir("", "wasm")
	require.NoError(t, err)
	cleanup := func() { os.RemoveAll(tempDir) }
	//t.Cleanup(cleanup) todo: add with Go 1.14
	var (
		keyParams  = sdk.NewKVStoreKey(params.StoreKey)
		tkeyParams = sdk.NewTransientStoreKey(params.TStoreKey)
		keyWasm    = sdk.NewKVStoreKey(wasmTypes.StoreKey)
	)

	db := dbm.NewMemDB()
	ms := store.NewCommitMultiStore(db)
	ms.MountStoreWithDB(keyWasm, sdk.StoreTypeIAVL, db)
	ms.MountStoreWithDB(keyParams, sdk.StoreTypeIAVL, db)
	ms.MountStoreWithDB(tkeyParams, sdk.StoreTypeTransient, db)
	require.NoError(t, ms.LoadLatestVersion())

	ctx := sdk.NewContext(ms, abci.Header{
		Height: 1234567,
		Time:   time.Date(2020, time.April, 22, 12, 0, 0, 0, time.UTC),
	}, false, log.NewNopLogger())
	cdc := MakeTestCodec()
	pk := params.NewKeeper(cdc, keyParams, tkeyParams)
	wasmConfig := wasmTypes.DefaultWasmConfig()
	srcKeeper := NewKeeper(cdc, keyWasm, pk.Subspace(wasmTypes.DefaultParamspace), auth.AccountKeeper{}, nil, staking.Keeper{}, nil, tempDir, wasmConfig, "", nil, nil)
	srcKeeper.setParams(ctx, wasmTypes.DefaultParams())

	return srcKeeper, ctx, []sdk.StoreKey{keyWasm, keyParams}, cleanup
}
