package app

import (
	"fmt"
	"testing"

	abci "github.com/cometbft/cometbft/api/cometbft/abci/v1"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/gogoproto/proto"
	"github.com/stretchr/testify/require"

	"cosmossdk.io/log"

	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/msgservice"

	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
)

var emptyWasmOpts []wasmkeeper.Option

func TestWasmdExport(t *testing.T) {
	db := dbm.NewMemDB()
	logger := log.NewTestLogger(t)
	gapp := NewWasmAppWithCustomOptions(t, false, SetupOptions{
		Logger:  logger.With("instance", "first"),
		DB:      db,
		AppOpts: simtestutil.NewAppOptionsWithFlagHome(t.TempDir()),
	})

	// finalize block so we have CheckTx state set
	_, err := gapp.FinalizeBlock(&abci.FinalizeBlockRequest{
		Height: 1,
	})
	require.NoError(t, err)

	_, err = gapp.Commit()
	require.NoError(t, err)

	// Making a new app object with the db, so that initchain hasn't been called
	newGapp := NewWasmApp(logger, db, nil, true, simtestutil.NewAppOptionsWithFlagHome(t.TempDir()), emptyWasmOpts)
	_, err = newGapp.ExportAppStateAndValidators(false, []string{}, nil)
	require.NoError(t, err, "ExportAppStateAndValidators should not have an error")
}

// ensure that blocked addresses are properly set in bank keeper
func TestBlockedAddrs(t *testing.T) {
	app := Setup(t)
	blockedAddrs, err := BlockedAddresses(app.interfaceRegistry.SigningContext().AddressCodec())
	require.NoError(t, err)
	for acc := range blockedAddrs {
		var addr sdk.AccAddress
		if modAddr, err := app.InterfaceRegistry().SigningContext().AddressCodec().StringToBytes(acc); err == nil {
			addr = modAddr
		} else {
			addr = app.AuthKeeper.GetModuleAddress(acc)
		}

		require.True(
			t,
			app.BankKeeper.BlockedAddr(addr),
			fmt.Sprintf("ensure that blocked addresses are properly set in bank keeper: %s should be blocked", acc),
		)
	}

}

func TestGetMaccPerms(t *testing.T) {
	dup := GetMaccPerms()
	require.Equal(t, maccPerms, dup, "duplicated module account permissions differed from actual module account permissions")
}

// TestMergedRegistry tests that fetching the gogo/protov2 merged registry
// doesn't fail after loading all file descriptors.
func TestMergedRegistry(t *testing.T) {
	r, err := proto.MergedRegistry()
	require.NoError(t, err)
	require.Greater(t, r.NumFiles(), 0)
}

func TestProtoAnnotations(t *testing.T) {
	r, err := proto.MergedRegistry()
	require.NoError(t, err)
	err = msgservice.ValidateProtoAnnotations(r)
	require.NoError(t, err)
}
