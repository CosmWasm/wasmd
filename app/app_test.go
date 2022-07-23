package app

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tendermint/tendermint/libs/log"
	db "github.com/tendermint/tm-db"
	dbm "github.com/tendermint/tm-db"

	abci "github.com/tendermint/tendermint/abci/types"

	"github.com/CosmWasm/wasmd/x/wasm"
)

var emptyWasmOpts []wasm.Option = nil
var enabeledProposals []wasm.ProposalType = nil

func TestSimAppExportAndBlockedAddrs(t *testing.T) {
	encCfg := MakeEncodingConfig()
	db := dbm.NewMemDB()
	logger := log.NewTMLogger(log.NewSyncWriter(os.Stdout))
	app := NewSimappWithCustomOptions(t, false, SetupOptions{
		Logger:             logger,
		DB:                 db,
		InvCheckPeriod:     0,
		EncConfig:          encCfg,
		EnabeledProposals:  enabeledProposals,
		HomePath:           DefaultNodeHome,
		SkipUpgradeHeights: map[int64]bool{},
		AppOpts:            EmptyBaseAppOptions{},
		WasmOpts:           emptyWasmOpts,
	})

	for acc := range maccPerms {
		require.True(
			t,
			app.BankKeeper.BlockedAddr(app.AccountKeeper.GetModuleAddress(acc)),
			"ensure that blocked addresses are properly set in bank keeper",
		)
	}

	app.Commit()

	logger2 := log.NewTMLogger(log.NewSyncWriter(os.Stdout))
	// Making a new app object with the db, so that initchain hasn't been called
	app2 := NewWasmApp(logger2, db, nil, true, map[int64]bool{}, DefaultNodeHome, 0, encCfg, wasm.EnableAllProposals, EmptyBaseAppOptions{}, emptyWasmOpts)
	_, err := app2.ExportAppStateAndValidators(false, []string{})
	require.NoError(t, err, "ExportAppStateAndValidators should not have an error")
}

// ensure that blocked addresses are properly set in bank keeper
func TestBlockedAddrs(t *testing.T) {
	db := db.NewMemDB()
	gapp := NewWasmApp(log.NewTMLogger(log.NewSyncWriter(os.Stdout)), db, nil, true, map[int64]bool{}, DefaultNodeHome, 0, MakeEncodingConfig(), wasm.EnableAllProposals, EmptyBaseAppOptions{}, emptyWasmOpts)

	for acc := range maccPerms {
		t.Run(acc, func(t *testing.T) {
			require.True(t, gapp.BankKeeper.BlockedAddr(gapp.AccountKeeper.GetModuleAddress(acc)),
				"ensure that blocked addresses are properly set in bank keeper",
			)
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

func setGenesis(gapp *WasmApp) error {
	genesisState := NewDefaultGenesisState()
	stateBytes, err := json.MarshalIndent(genesisState, "", " ")
	if err != nil {
		return err
	}

	// Initialize the chain
	gapp.InitChain(
		abci.RequestInitChain{
			Validators:    []abci.ValidatorUpdate{},
			AppStateBytes: stateBytes,
		},
	)

	gapp.Commit()
	return nil
}
