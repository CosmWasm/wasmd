package app

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	abci "github.com/tendermint/tendermint/abci/types"
	db "github.com/tendermint/tm-db"

	"github.com/line/ostracon/libs/log"

	"github.com/line/wasmd/x/wasm"
)

var emptyWasmOpts []wasm.Option = nil

func TestWasmdExport(t *testing.T) {
	db := db.NewMemDB()
	gapp := NewWasmApp(log.NewOCLogger(log.NewSyncWriter(os.Stdout)), db, nil, true, map[int64]bool{}, DefaultNodeHome, 0, MakeEncodingConfig(), wasm.EnableAllProposals, EmptyBaseAppOptions{}, emptyWasmOpts)

	genesisState := NewDefaultGenesisState()
	stateBytes, err := json.MarshalIndent(genesisState, "", "  ")
	require.NoError(t, err)

	// Initialize the chain
	gapp.InitChain(
		abci.RequestInitChain{
			Validators:    []abci.ValidatorUpdate{},
			AppStateBytes: stateBytes,
		},
	)
	gapp.Commit()

	// Making a new app object with the db, so that initchain hasn't been called
	newGapp := NewWasmApp(log.NewOCLogger(log.NewSyncWriter(os.Stdout)), db, nil, true, map[int64]bool{}, DefaultNodeHome, 0, MakeEncodingConfig(), wasm.EnableAllProposals, EmptyBaseAppOptions{}, emptyWasmOpts)
	_, err = newGapp.ExportAppStateAndValidators(false, []string{})
	require.NoError(t, err, "ExportAppStateAndValidators should not have an error")
}

// ensure that blocked addresses are properly set in bank keeper
func TestBlockedAddrs(t *testing.T) {
	db := db.NewMemDB()
	gapp := NewWasmApp(log.NewOCLogger(log.NewSyncWriter(os.Stdout)), db, nil, true, map[int64]bool{}, DefaultNodeHome, 0, MakeEncodingConfig(), wasm.EnableAllProposals, EmptyBaseAppOptions{}, emptyWasmOpts)
	blockedAddrs := gapp.BlockedAddrs()

	for acc := range maccPerms {
		//for acc := range gapp.BlockedAddrs() {
		t.Run(acc, func(t *testing.T) {
			addr := gapp.AccountKeeper.GetModuleAddress(acc)
			if blockedAddrs[addr.String()] {
				require.True(t, gapp.BankKeeper.BlockedAddr(addr),
					"ensure that blocked addresses are properly set in bank keeper",
				)
			}
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
