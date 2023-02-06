package appplus

import (
	"encoding/json"

	wasmapp "github.com/line/wasmd/app"

	abci "github.com/line/ostracon/abci/types"
	"github.com/line/ostracon/libs/log"
	"github.com/line/wasmd/x/wasm"
	dbm "github.com/tendermint/tm-db"
)

// Setup initializes a new WasmApp with DefaultNodeHome for integration tests
func Setup(isCheckTx bool, opts ...wasm.Option) *WasmPlusApp {
	db := dbm.NewMemDB()
	app := NewWasmApp(log.NewNopLogger(), db, nil, true, map[int64]bool{}, DefaultNodeHome, 5, MakeEncodingConfig(), wasm.EnableAllProposals, EmptyBaseAppOptions{}, opts)

	if !isCheckTx {
		genesisState := NewDefaultGenesisState()
		stateBytes, err := json.MarshalIndent(genesisState, "", " ")
		if err != nil {
			panic(err)
		}

		app.InitChain(
			abci.RequestInitChain{
				Validators:      []abci.ValidatorUpdate{},
				ConsensusParams: wasmapp.DefaultConsensusParams,
				AppStateBytes:   stateBytes,
			},
		)
	}
	return app
}

// EmptyBaseAppOptions is a stub implementing AppOptions
type EmptyBaseAppOptions struct{}

// Get implements AppOptions
func (ao EmptyBaseAppOptions) Get(o string) interface{} {
	return nil
}
