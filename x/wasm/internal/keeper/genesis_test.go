package keeper

import (
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
	srcKeeper, srcCtx := setupKeeper(t)
	wasmCode, err := ioutil.ReadFile("./testdata/contract.wasm")
	require.NoError(t, err)

	f := fuzz.New().Funcs(FuzzAddr, FuzzAbsoluteTxPosition, FuzzContractInfo, FuzzStateModel)
	for i := 0; i < 20; i++ {
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
		srcKeeper.setContractState(srcCtx, contractAddr, stateModels)
	}

	// export
	genesisState := ExportGenesis(srcCtx, srcKeeper)

	// re-import
	dstKeeper, dstCtx := setupKeeper(t)
	InitGenesis(dstCtx, dstKeeper, genesisState)

	// compare whole DB
	srcIT := srcCtx.KVStore(srcKeeper.storeKey).Iterator(nil, nil)
	dstIT := dstCtx.KVStore(dstKeeper.storeKey).Iterator(nil, nil)

	for i := 0; srcIT.Valid(); i++ {
		require.True(t, dstIT.Valid(), "destination DB has less elements than source. Missing: %q", srcIT.Key())
		require.Equal(t, srcIT.Key(), dstIT.Key(), i)
		require.Equal(t, srcIT.Value(), dstIT.Value(), i)
		srcIT.Next()
		dstIT.Next()
	}
	require.False(t, dstIT.Valid())
}

func setupKeeper(t *testing.T) (Keeper, sdk.Context) {
	tempDir, err := ioutil.TempDir("", "wasm")
	require.NoError(t, err)
	t.Cleanup(func() { os.RemoveAll(tempDir) })

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
	return srcKeeper, ctx
}
