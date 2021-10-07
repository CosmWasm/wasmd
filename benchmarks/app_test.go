package benchmarks

import (
	"encoding/json"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	abci "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/libs/log"
	tmproto "github.com/tendermint/tendermint/proto/tendermint/types"
	dbm "github.com/tendermint/tm-db"

	"github.com/CosmWasm/wasmd/app"
	"github.com/CosmWasm/wasmd/x/wasm"
)

func setup(withGenesis bool, invCheckPeriod uint, opts ...wasm.Option) (*app.WasmApp, app.GenesisState) {
	db := dbm.NewMemDB()
	wasmApp := app.NewWasmApp(log.NewNopLogger(), db, nil, true, map[int64]bool{}, app.DefaultNodeHome, invCheckPeriod, wasm.EnableAllProposals, app.EmptyBaseAppOptions{}, opts)
	if withGenesis {
		return wasmApp, app.NewDefaultGenesisState()
	}
	return wasmApp, app.GenesisState{}
}

// SetupWithGenesisAccounts initializes a new WasmApp with the provided genesis
// accounts and possible balances.
func SetupWithGenesisAccounts(genAccs []authtypes.GenesisAccount, balances ...banktypes.Balance) *app.WasmApp {
	wasmApp, genesisState := setup(true, 0)
	authGenesis := authtypes.NewGenesisState(authtypes.DefaultParams(), genAccs)
	encodingConfig := app.MakeEncodingConfig()
	appCodec := encodingConfig.Marshaler

	genesisState[authtypes.ModuleName] = appCodec.MustMarshalJSON(authGenesis)

	totalSupply := sdk.NewCoins()
	for _, b := range balances {
		totalSupply = totalSupply.Add(b.Coins...)
	}

	bankGenesis := banktypes.NewGenesisState(banktypes.DefaultGenesisState().Params, balances, totalSupply, []banktypes.Metadata{})
	genesisState[banktypes.ModuleName] = appCodec.MustMarshalJSON(bankGenesis)

	stateBytes, err := json.MarshalIndent(genesisState, "", " ")
	if err != nil {
		panic(err)
	}

	wasmApp.InitChain(
		abci.RequestInitChain{
			Validators:      []abci.ValidatorUpdate{},
			ConsensusParams: app.DefaultConsensusParams,
			AppStateBytes:   stateBytes,
		},
	)

	wasmApp.Commit()
	wasmApp.BeginBlock(abci.RequestBeginBlock{Header: tmproto.Header{Height: wasmApp.LastBlockHeight() + 1}})

	return wasmApp
}
