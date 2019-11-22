package keeper

import (
	"testing"

	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/store"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth"
	authexported "github.com/cosmos/cosmos-sdk/x/auth/exported"
	"github.com/cosmos/cosmos-sdk/x/bank"
	"github.com/cosmos/cosmos-sdk/x/params"
	"github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/stretchr/testify/require"
	abci "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/libs/log"
	dbm "github.com/tendermint/tm-db"
)

func MakeTestCodec() *codec.Codec {
	var cdc = codec.New()

	// Register AppAccount
	cdc.RegisterInterface((*authexported.Account)(nil), nil)
	cdc.RegisterConcrete(&auth.BaseAccount{}, "test/wasm/BaseAccount", nil)
	codec.RegisterCrypto(cdc)

	return cdc
}

func CreateTestInput(t *testing.T, isCheckTx bool, tempDir string) (sdk.Context, auth.AccountKeeper, Keeper) {
	keyContract := sdk.NewKVStoreKey(types.StoreKey)
	keyAcc := sdk.NewKVStoreKey(auth.StoreKey)
	keyParams := sdk.NewKVStoreKey(params.StoreKey)
	tkeyParams := sdk.NewTransientStoreKey(params.TStoreKey)

	db := dbm.NewMemDB()
	ms := store.NewCommitMultiStore(db)
	ms.MountStoreWithDB(keyContract, sdk.StoreTypeIAVL, db)
	ms.MountStoreWithDB(keyAcc, sdk.StoreTypeIAVL, db)
	ms.MountStoreWithDB(keyParams, sdk.StoreTypeIAVL, db)
	ms.MountStoreWithDB(tkeyParams, sdk.StoreTypeTransient, db)
	err := ms.LoadLatestVersion()
	require.Nil(t, err)

	ctx := sdk.NewContext(ms, abci.Header{}, isCheckTx, log.NewNopLogger())
	cdc := MakeTestCodec()

	pk := params.NewKeeper(cdc, keyParams, tkeyParams, params.DefaultCodespace)

	accountKeeper := auth.NewAccountKeeper(
		cdc,    // amino codec
		keyAcc, // target store
		pk.Subspace(auth.DefaultParamspace),
		auth.ProtoBaseAccount, // prototype
	)

	bk := bank.NewBaseKeeper(
		accountKeeper,
		pk.Subspace(bank.DefaultParamspace),
		bank.DefaultCodespace,
		nil,
	)
	bk.SetSendEnabled(ctx, true)

	// TODO: register more than bank.send
	router := baseapp.NewRouter()
	h := bank.NewHandler(bk)
	router.AddRoute(bank.RouterKey, h)

	keeper := NewKeeper(cdc, keyContract, accountKeeper, bk, router, tempDir)

	return ctx, accountKeeper, keeper
}
