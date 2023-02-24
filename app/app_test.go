package app

import (
	"os"
	"testing"

	dbm "github.com/cometbft/cometbft-db"
	"github.com/cometbft/cometbft/libs/log"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/CosmWasm/wasmd/x/wasm"
)

var emptyWasmOpts []wasm.Option

func TestWasmdExport(t *testing.T) {
	db := dbm.NewMemDB()
	gapp := NewWasmAppWithCustomOptions(t, false, SetupOptions{
		Logger:  log.NewTMLogger(log.NewSyncWriter(os.Stdout)),
		DB:      db,
		AppOpts: simtestutil.NewAppOptionsWithFlagHome(t.TempDir()),
	})
	gapp.Commit()

	// Making a new app object with the db, so that initchain hasn't been called
	newGapp := NewWasmApp(log.NewTMLogger(log.NewSyncWriter(os.Stdout)), db, nil, true, wasm.EnableAllProposals, simtestutil.NewAppOptionsWithFlagHome(t.TempDir()), emptyWasmOpts)
	_, err := newGapp.ExportAppStateAndValidators(false, []string{}, nil)
	require.NoError(t, err, "ExportAppStateAndValidators should not have an error")
}

// ensure that blocked addresses are properly set in bank keeper
func TestBlockedAddrs(t *testing.T) {
	gapp := Setup(t)

	for acc := range BlockedAddresses() {
		t.Run(acc, func(t *testing.T) {
			var addr sdk.AccAddress
			if modAddr, err := sdk.AccAddressFromBech32(acc); err == nil {
				addr = modAddr
			} else {
				addr = gapp.AccountKeeper.GetModuleAddress(acc)
			}
			require.True(t, gapp.BankKeeper.BlockedAddr(addr), "ensure that blocked addresses are properly set in bank keeper")
		})
	}
}

func TestGetMaccPerms(t *testing.T) {
	dup := GetMaccPerms()
	require.Equal(t, maccPerms, dup, "duplicated module account permissions differed from actual module account permissions")
}

func TestGetEnabledProposals(t *testing.T) {
	cases := map[string]struct {
		proposalsEnabled string
		specificEnabled  string
		expected         []wasm.ProposalType
	}{
		"all disabled": {
			proposalsEnabled: "false",
			expected:         wasm.DisableAllProposals,
		},
		"all enabled": {
			proposalsEnabled: "true",
			expected:         wasm.EnableAllProposals,
		},
		"some enabled": {
			proposalsEnabled: "okay",
			specificEnabled:  "StoreCode,InstantiateContract",
			expected:         []wasm.ProposalType{wasm.ProposalTypeStoreCode, wasm.ProposalTypeInstantiateContract},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			ProposalsEnabled = tc.proposalsEnabled
			EnableSpecificProposals = tc.specificEnabled
			proposals := GetEnabledProposals()
			assert.Equal(t, tc.expected, proposals)
		})
	}
}
