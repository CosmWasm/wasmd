package keeper

import (
	"testing"

	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/std"
	"github.com/cosmos/cosmos-sdk/store"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth"
	"github.com/cosmos/cosmos-sdk/x/bank"
	"github.com/cosmos/cosmos-sdk/x/params"
	"github.com/cosmos/cosmos-sdk/x/staking/types"
	wasmTypes "github.com/cosmwasm/wasmd/x/wasm/types"
	"github.com/stretchr/testify/require"
	abci "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/libs/log"
	dbm "github.com/tendermint/tm-db"
)

const flagLRUCacheSize = "lru_size"
const flagQueryGasLimit = "query_gas_limit"

func MakeTestCodec() *codec.Codec {
	var cdc = codec.New()

	// Register AppAccount
	// cdc.RegisterInterface((*authexported.Account)(nil), nil)
	// cdc.RegisterConcrete(&auth.BaseAccount{}, "test/wasm/BaseAccount", nil)
	auth.AppModuleBasic{}.RegisterCodec(cdc)
	bank.AppModuleBasic{}.RegisterCodec(cdc)
	sdk.RegisterCodec(cdc)
	codec.RegisterCrypto(cdc)

	return cdc
}

func CreateTestInput(t *testing.T, tempDir string) (sdk.Context, auth.AccountKeeper, bank.Keeper, Keeper) {
	keyContract := sdk.NewKVStoreKey(types.StoreKey)
	keyAcc := sdk.NewKVStoreKey(auth.StoreKey)
	keyBank := sdk.NewKVStoreKey(bank.ModuleName)
	keyParams := sdk.NewKVStoreKey(params.StoreKey)
	tkeyParams := sdk.NewTransientStoreKey(params.TStoreKey)

	db := dbm.NewMemDB()
	ms := store.NewCommitMultiStore(db)
	ms.MountStoreWithDB(keyContract, sdk.StoreTypeIAVL, db)
	ms.MountStoreWithDB(keyBank, sdk.StoreTypeIAVL, db)
	ms.MountStoreWithDB(keyAcc, sdk.StoreTypeIAVL, db)
	ms.MountStoreWithDB(keyParams, sdk.StoreTypeIAVL, db)
	ms.MountStoreWithDB(tkeyParams, sdk.StoreTypeTransient, db)
	require.NoError(t, ms.LoadLatestVersion())

	isCheckTx := false
	ctx := sdk.NewContext(ms, abci.Header{}, isCheckTx, log.NewNopLogger())
	cdc := MakeTestCodec()
	interfaceRegistry := codectypes.NewInterfaceRegistry()
	sdk.RegisterInterfaces(interfaceRegistry)
	appCodec := std.NewAppCodec(cdc, interfaceRegistry)

	blacklistedAddrs := make(map[string]bool)
	maccPerms := make(map[string][]string)

	paramsKeeper := params.NewKeeper(appCodec, keyParams, tkeyParams)
	accountKeeper := auth.NewAccountKeeper(appCodec, keyAcc, paramsKeeper.Subspace(auth.DefaultParamspace), auth.ProtoBaseAccount, maccPerms)
	bankKeeper := bank.NewBaseKeeper(appCodec, keyBank, accountKeeper, paramsKeeper.Subspace(bank.DefaultParamspace), blacklistedAddrs)
	bankKeeper.SetSendEnabled(ctx, true)

	//accountKeeper := app.AccountKeeper
	//bankKeeper := app.BankKeeper
	bankKeeper.SetSendEnabled(ctx, true)

	// TODO: register more than bank.send
	router := baseapp.NewRouter()
	h := bank.NewHandler(bankKeeper)
	router.AddRoute(bank.RouterKey, h)

	// Load default wasm config
	wasmConfig := wasmTypes.DefaultWasmConfig()

	keeper := NewKeeper(appCodec, keyContract, accountKeeper, bankKeeper, router, tempDir, wasmConfig)

	return ctx, accountKeeper, bankKeeper, keeper
}
