package keeper

import (
	"crypto/sha256"
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/CosmWasm/wasmd/x/wasm/internal/types"
	wasmTypes "github.com/CosmWasm/wasmd/x/wasm/internal/types"
	"github.com/cosmos/cosmos-sdk/store"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth"
	"github.com/cosmos/cosmos-sdk/x/staking"
	fuzz "github.com/google/gofuzz"
	"github.com/stretchr/testify/require"
	abci "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/libs/log"
	dbm "github.com/tendermint/tm-db"
)

func TestGenesisExportImport(t *testing.T) {
	srcKeeper, srcCtx, srcCleanup := setupKeeper(t)
	defer srcCleanup()
	wasmCode, err := ioutil.ReadFile("./testdata/contract.wasm")
	require.NoError(t, err)

	// store some test data
	f := fuzz.New().Funcs(FuzzAddr, FuzzAbsoluteTxPosition, FuzzContractInfo, FuzzStateModel)
	for i := 0; i < 25; i++ {
		var (
			codeInfo    types.CodeInfo
			contract    types.ContractInfo
			stateModels []types.Model
		)
		f.Fuzz(&codeInfo)
		f.Fuzz(&contract)
		f.Fuzz(&stateModels)

		codeID, err := srcKeeper.Create(srcCtx, codeInfo.Creator, wasmCode, codeInfo.Source, codeInfo.Builder)
		require.NoError(t, err)
		contract.CodeID = codeID
		contractAddr := srcKeeper.generateContractAddress(srcCtx, codeID)
		srcKeeper.setContractInfo(srcCtx, contractAddr, &contract)
		srcKeeper.importContractState(srcCtx, contractAddr, stateModels)
	}

	// export
	genesisState := ExportGenesis(srcCtx, srcKeeper)

	// re-import
	dstKeeper, dstCtx, dstCleanup := setupKeeper(t)
	defer dstCleanup()
	InitGenesis(dstCtx, dstKeeper, genesisState)

	// compare whole DB
	srcIT := srcCtx.KVStore(srcKeeper.storeKey).Iterator(nil, nil)
	dstIT := dstCtx.KVStore(dstKeeper.storeKey).Iterator(nil, nil)

	for i := 0; srcIT.Valid(); i++ {
		require.True(t, dstIT.Valid(), "destination DB has less elements than source. Missing: %q", srcIT.Key())
		require.Equal(t, srcIT.Key(), dstIT.Key(), i)
		require.Equal(t, srcIT.Value(), dstIT.Value(), "element (%d): %s", i, srcIT.Key())
		srcIT.Next()
		dstIT.Next()
	}
	require.False(t, dstIT.Valid())
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
					CodeInfo: wasmTypes.CodeInfo{
						CodeHash: codeHash[:],
						Creator:  anyAddress,
					},
					CodesBytes: wasmCode,
				}},
				Contracts: nil,
			},
			expSuccess: true,
		},
		"prevent code hash mismatch": {src: types.GenesisState{
			Codes: []types.Code{{
				CodeInfo: wasmTypes.CodeInfo{
					CodeHash: make([]byte, len(codeHash)),
					Creator:  anyAddress,
				},
				CodesBytes: wasmCode,
			}},
			Contracts: nil,
		}},
		"happy path: code info and contract do match": {
			src: types.GenesisState{
				Codes: []types.Code{{
					CodeInfo: wasmTypes.CodeInfo{
						CodeHash: codeHash[:],
						Creator:  anyAddress,
					},
					CodesBytes: wasmCode,
				}},
				Contracts: []types.Contract{
					{
						ContractAddress: addrFromUint64(1<<32 + 1),
						ContractInfo: wasmTypes.ContractInfo{
							CodeID:  1,
							Creator: anyAddress,
							Label:   "any",
							Created: &types.AbsoluteTxPosition{BlockHeight: 1, TxIndex: 1},
						},
					},
				},
			},
			expSuccess: true,
		},
		"happy path: code info with two contracts": {
			src: types.GenesisState{
				Codes: []types.Code{{
					CodeInfo: wasmTypes.CodeInfo{
						CodeHash: codeHash[:],
						Creator:  anyAddress,
					},
					CodesBytes: wasmCode,
				}},
				Contracts: []types.Contract{
					{
						ContractAddress: addrFromUint64(1<<32 + 1),
						ContractInfo: wasmTypes.ContractInfo{
							CodeID:  1,
							Creator: anyAddress,
							Label:   "any",
							Created: &types.AbsoluteTxPosition{BlockHeight: 1, TxIndex: 1},
						},
					}, {
						ContractAddress: addrFromUint64(2<<32 + 1),
						ContractInfo: wasmTypes.ContractInfo{
							CodeID:  1,
							Creator: anyAddress,
							Label:   "any",
							Created: &types.AbsoluteTxPosition{BlockHeight: 1, TxIndex: 1},
						},
					},
				},
			},
			expSuccess: true,
		},
		"prevent contracts that points to non existing codeID": {
			src: types.GenesisState{
				Contracts: []types.Contract{
					{
						ContractAddress: contractAddress(1, 1),
						ContractInfo: wasmTypes.ContractInfo{
							CodeID:  1,
							Creator: anyAddress,
							Label:   "any",
							Created: &types.AbsoluteTxPosition{BlockHeight: 1, TxIndex: 1},
						},
					},
				},
			},
		},
		"prevent duplicate contracts": {
			src: types.GenesisState{
				Codes: []types.Code{{
					CodeInfo: wasmTypes.CodeInfo{
						CodeHash: codeHash[:],
						Creator:  anyAddress,
					},
					CodesBytes: wasmCode,
				}},
				Contracts: []types.Contract{
					{
						ContractAddress: contractAddress(1, 1),
						ContractInfo: wasmTypes.ContractInfo{
							CodeID:  1,
							Creator: anyAddress,
							Label:   "any",
							Created: &types.AbsoluteTxPosition{BlockHeight: 1, TxIndex: 1},
						},
					}, {
						ContractAddress: contractAddress(1, 1),
						ContractInfo: wasmTypes.ContractInfo{
							CodeID:  1,
							Creator: anyAddress,
							Label:   "any",
							Created: &types.AbsoluteTxPosition{BlockHeight: 1, TxIndex: 1},
						},
					},
				},
			},
		},
		"prevent duplicate contract model": {
			src: types.GenesisState{
				Codes: []types.Code{{
					CodeInfo: wasmTypes.CodeInfo{
						CodeHash: codeHash[:],
						Creator:  anyAddress,
					},
					CodesBytes: wasmCode,
				}},
				Contracts: []types.Contract{
					{
						ContractAddress: addrFromUint64(1<<32 + 1),
						ContractInfo: wasmTypes.ContractInfo{
							CodeID:  1,
							Creator: anyAddress,
							Label:   "any",
							Created: &types.AbsoluteTxPosition{BlockHeight: 1, TxIndex: 1},
						},
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
			},
		},
		"prevent duplicate sequences": {
			src: types.GenesisState{
				Sequences: []types.Sequence{
					{IDKey: []byte("foo"), Value: 1},
					{IDKey: []byte("foo"), Value: 9999},
				},
			},
		},
	}

	for msg, spec := range specs {
		t.Run(msg, func(t *testing.T) {
			keeper, ctx, cleanup := setupKeeper(t)
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

func setupKeeper(t *testing.T) (Keeper, sdk.Context, func()) {
	tempDir, err := ioutil.TempDir("", "wasm")
	require.NoError(t, err)
	cleanup := func() { os.RemoveAll(tempDir) }
	//t.Cleanup(cleanup) todo: add with Go 1.14

	keyContract := sdk.NewKVStoreKey(wasmTypes.StoreKey)
	db := dbm.NewMemDB()
	ms := store.NewCommitMultiStore(db)
	ms.MountStoreWithDB(keyContract, sdk.StoreTypeIAVL, db)
	require.NoError(t, ms.LoadLatestVersion())

	ctx := sdk.NewContext(ms, abci.Header{
		Height: 1234567,
		Time:   time.Date(2020, time.April, 22, 12, 0, 0, 0, time.UTC),
	}, false, log.NewNopLogger())

	cdc := MakeTestCodec()
	wasmConfig := wasmTypes.DefaultWasmConfig()

	srcKeeper := NewKeeper(cdc, keyContract, auth.AccountKeeper{}, nil, staking.Keeper{}, nil, tempDir, wasmConfig, "", nil, nil)
	return srcKeeper, ctx, cleanup
}
