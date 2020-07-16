package keeper

import (
	"crypto/sha256"
	"encoding/json"
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
	for i := 0; i < 1; i++ {
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
			require.True(t, dstIT.Valid(), "destination DB has less elements than source. Missing: %q", srcIT.Key())
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
	codeHash := sha256.Sum256(wasmCode)
	anyAddress := make([]byte, 20)

	specs := map[string]struct {
		src        types.GenesisState
		expSuccess bool
	}{
		"happy path: code info correct": {
			src: types.GenesisState{
				Codes: []types.Code{{
					CodeID: 1,
					CodeInfo: wasmTypes.CodeInfo{
						CodeHash: codeHash[:],
						Creator:  anyAddress,
					},
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
					CodeID: 1,
					CodeInfo: wasmTypes.CodeInfo{
						CodeHash: codeHash[:],
						Creator:  anyAddress,
					},
					CodesBytes: wasmCode,
				}, {
					CodeID: 3,
					CodeInfo: wasmTypes.CodeInfo{
						CodeHash: codeHash[:],
						Creator:  anyAddress,
					},
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
					CodeID: 2,
					CodeInfo: wasmTypes.CodeInfo{
						CodeHash: codeHash[:],
						Creator:  anyAddress,
					},
					CodesBytes: wasmCode,
				}, {
					CodeID: 1,
					CodeInfo: wasmTypes.CodeInfo{
						CodeHash: codeHash[:],
						Creator:  anyAddress,
					},
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
				CodeID: 1,
				CodeInfo: wasmTypes.CodeInfo{
					CodeHash: make([]byte, len(codeHash)),
					Creator:  anyAddress,
				},
				CodesBytes: wasmCode,
			}},
			Params: types.DefaultParams(),
		}},
		"prevent duplicate codeIDs": {src: types.GenesisState{
			Codes: []types.Code{
				{
					CodeID: 1,
					CodeInfo: wasmTypes.CodeInfo{
						CodeHash: codeHash[:],
						Creator:  anyAddress,
					},
					CodesBytes: wasmCode,
				},
				{
					CodeID: 1,
					CodeInfo: wasmTypes.CodeInfo{
						CodeHash: codeHash[:],
						Creator:  anyAddress,
					},
					CodesBytes: wasmCode,
				},
			},
			Params: types.DefaultParams(),
		}},
		"happy path: code id in info and contract do match": {
			src: types.GenesisState{
				Codes: []types.Code{{
					CodeID: 1,
					CodeInfo: wasmTypes.CodeInfo{
						CodeHash: codeHash[:],
						Creator:  anyAddress,
					},
					CodesBytes: wasmCode,
				}},
				Contracts: []types.Contract{
					{
						ContractAddress: contractAddress(1, 1),
						ContractInfo:    types.ContractInfoFixture(func(c *wasmTypes.ContractInfo) { c.CodeID = 1 }),
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
					CodeID: 1,
					CodeInfo: wasmTypes.CodeInfo{
						CodeHash: codeHash[:],
						Creator:  anyAddress,
					},
					CodesBytes: wasmCode,
				}},
				Contracts: []types.Contract{
					{
						ContractAddress: contractAddress(1, 1),
						ContractInfo:    types.ContractInfoFixture(func(c *wasmTypes.ContractInfo) { c.CodeID = 1 }),
					}, {
						ContractAddress: contractAddress(1, 2),
						ContractInfo:    types.ContractInfoFixture(func(c *wasmTypes.ContractInfo) { c.CodeID = 1 }),
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
						ContractInfo:    types.ContractInfoFixture(func(c *wasmTypes.ContractInfo) { c.CodeID = 1 }),
					},
				},
				Params: types.DefaultParams(),
			},
		},
		"prevent duplicate contract address": {
			src: types.GenesisState{
				Codes: []types.Code{{
					CodeID: 1,
					CodeInfo: wasmTypes.CodeInfo{
						CodeHash: codeHash[:],
						Creator:  anyAddress,
					},
					CodesBytes: wasmCode,
				}},
				Contracts: []types.Contract{
					{
						ContractAddress: contractAddress(1, 1),
						ContractInfo:    types.ContractInfoFixture(func(c *wasmTypes.ContractInfo) { c.CodeID = 1 }),
					}, {
						ContractAddress: contractAddress(1, 1),
						ContractInfo:    types.ContractInfoFixture(func(c *wasmTypes.ContractInfo) { c.CodeID = 1 }),
					},
				},
				Params: types.DefaultParams(),
			},
		},
		"prevent duplicate contract model keys": {
			src: types.GenesisState{
				Codes: []types.Code{{
					CodeID: 1,
					CodeInfo: wasmTypes.CodeInfo{
						CodeHash: codeHash[:],
						Creator:  anyAddress,
					},
					CodesBytes: wasmCode,
				}},
				Contracts: []types.Contract{
					{
						ContractAddress: contractAddress(1, 1),
						ContractInfo:    types.ContractInfoFixture(func(c *wasmTypes.ContractInfo) { c.CodeID = 1 }),
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
					CodeID: 2,
					CodeInfo: wasmTypes.CodeInfo{
						CodeHash: codeHash[:],
						Creator:  anyAddress,
					},
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
					CodeID: 1,
					CodeInfo: wasmTypes.CodeInfo{
						CodeHash: codeHash[:],
						Creator:  anyAddress,
					},
					CodesBytes: wasmCode,
				}},
				Contracts: []types.Contract{
					{
						ContractAddress: contractAddress(1, 1),
						ContractInfo:    types.ContractInfoFixture(func(c *wasmTypes.ContractInfo) { c.CodeID = 1 }),
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
