package bindings_test

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/cometbft/cometbft/crypto"
	"github.com/cometbft/cometbft/crypto/ed25519"

	"cosmossdk.io/math"

	"cosmossdk.io/x/bank/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/CosmWasm/wasmd/app"
	"github.com/CosmWasm/wasmd/x/wasm/keeper"
)

func CreateTestInput(t *testing.T) (*app.WasmApp, sdk.Context) {
	osmosis := app.Setup(t)
	ctx := osmosis.BaseApp.NewContext(false)
	return osmosis, ctx
}

func FundAccount(t *testing.T, ctx sdk.Context, osmosis *app.WasmApp, acct sdk.AccAddress) {
	err := testutil.FundAccount(ctx, osmosis.BankKeeper, acct, sdk.NewCoins(
		sdk.NewCoin("uosmo", math.NewInt(10000000000)),
	))
	require.NoError(t, err)
}

// we need to make this deterministic (same every test run), as content might affect gas costs
func keyPubAddr() (crypto.PrivKey, crypto.PubKey, sdk.AccAddress) {
	key := ed25519.GenPrivKey()
	pub := key.PubKey()
	addr := sdk.AccAddress(pub.Address())
	return key, pub, addr
}

func RandomAccountAddress() sdk.AccAddress {
	_, _, addr := keyPubAddr()
	return addr
}

func RandomBech32AccountAddress() string {
	return RandomAccountAddress().String()
}

func storeReflectCode(t *testing.T, ctx sdk.Context, tokenz *app.WasmApp, addr sdk.AccAddress) uint64 {
	wasmCode, err := os.ReadFile("./testdata/token_reflect.wasm")
	require.NoError(t, err)

	contractKeeper := keeper.NewDefaultPermissionKeeper(tokenz.WasmKeeper)
	codeID, _, err := contractKeeper.Create(ctx, addr, wasmCode, nil)
	require.NoError(t, err)

	return codeID
}

func instantiateReflectContract(t *testing.T, ctx sdk.Context, tokenz *app.WasmApp, funder sdk.AccAddress) sdk.AccAddress {
	initMsgBz := []byte("{}")
	contractKeeper := keeper.NewDefaultPermissionKeeper(tokenz.WasmKeeper)
	codeID := uint64(1)
	addr, _, err := contractKeeper.Instantiate(ctx, codeID, funder, funder, initMsgBz, "demo contract", nil)
	require.NoError(t, err)

	return addr
}

func fundAccount(t *testing.T, ctx sdk.Context, tokenz *app.WasmApp, addr sdk.AccAddress, coins sdk.Coins) {
	err := testutil.FundAccount(
		ctx,
		tokenz.BankKeeper,
		addr,
		coins,
	)
	require.NoError(t, err)
}

func SetupCustomApp(t *testing.T, addr sdk.AccAddress) (*app.WasmApp, sdk.Context) {
	tokenz, ctx := CreateTestInput(t)
	wasmKeeper := tokenz.WasmKeeper

	storeReflectCode(t, ctx, tokenz, addr)

	cInfo := wasmKeeper.GetCodeInfo(ctx, 1)
	require.NotNil(t, cInfo)

	return tokenz, ctx
}
