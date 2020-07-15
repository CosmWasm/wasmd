package integration

/**
This file is full of test helper functions, taken from simapp
**/

import (
	"fmt"
	"os"
	"testing"

	wasmd "github.com/CosmWasm/wasmd/app"
	"github.com/CosmWasm/wasmd/x/wasm"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth"
	authexported "github.com/cosmos/cosmos-sdk/x/auth/exported"
	"github.com/stretchr/testify/require"

	abci "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/crypto"
	"github.com/tendermint/tendermint/libs/log"
	dbm "github.com/tendermint/tm-db"
)

// SimAppChainID hardcoded chainID for simulation
const (
	DefaultGenTxGas = 1000000
	SimAppChainID   = "wasmd-app"
)

// Setup initializes a new wasmd.WasmApp. A Nop logger is set in WasmApp.
func Setup(isCheckTx bool) *wasmd.WasmApp {
	db := dbm.NewMemDB()
	app := wasmd.NewWasmApp(log.NewNopLogger(), db, nil, true, 0, wasm.EnableAllProposals, nil)
	// app := wasmd.NewWasmApp(log.NewNopLogger(), db, nil, true, map[int64]bool{}, 0)
	if !isCheckTx {
		// init chain must be called to stop deliverState from being nil
		genesisState := wasmd.NewDefaultGenesisState()
		stateBytes, err := codec.MarshalJSONIndent(app.Codec(), genesisState)
		if err != nil {
			panic(err)
		}

		// Initialize the chain
		app.InitChain(
			abci.RequestInitChain{
				Validators:    []abci.ValidatorUpdate{},
				AppStateBytes: stateBytes,
			},
		)
	}

	return app
}

// SetupWithGenesisAccounts initializes a new wasmd.WasmApp with the passed in
// genesis accounts.
func SetupWithGenesisAccounts(genAccs []authexported.GenesisAccount) *wasmd.WasmApp {
	db := dbm.NewMemDB()
	app := wasmd.NewWasmApp(log.NewTMLogger(log.NewSyncWriter(os.Stdout)), db, nil, true, 0, wasm.EnableAllProposals, nil)
	// app := wasmd.NewWasmApp(log.NewTMLogger(log.NewSyncWriter(os.Stdout)), db, nil, true, map[int64]bool{}, 0)

	// initialize the chain with the passed in genesis accounts
	genesisState := wasmd.NewDefaultGenesisState()

	authGenesis := auth.NewGenesisState(auth.DefaultParams(), genAccs)
	genesisStateBz := app.Codec().MustMarshalJSON(authGenesis)
	genesisState[auth.ModuleName] = genesisStateBz

	stateBytes, err := codec.MarshalJSONIndent(app.Codec(), genesisState)
	if err != nil {
		panic(err)
	}
	fmt.Println(string(stateBytes))

	// Initialize the chain
	app.InitChain(
		abci.RequestInitChain{
			Validators:    []abci.ValidatorUpdate{},
			AppStateBytes: stateBytes,
		},
	)

	app.Commit()
	app.BeginBlock(abci.RequestBeginBlock{Header: abci.Header{Height: app.LastBlockHeight() + 1, ChainID: SimAppChainID}})

	return app
}

// SignAndDeliver checks a generated signed transaction and simulates a
// block commitment with the given transaction. A test assertion is made using
// the parameter 'expPass' against the result. A corresponding result is
// returned.
func SignAndDeliver(
	t *testing.T, app *wasmd.WasmApp, msgs []sdk.Msg,
	accNums, seq []uint64, expPass bool, priv ...crypto.PrivKey,
) (sdk.GasInfo, *sdk.Result, error) {

	tx := GenTx(
		msgs,
		sdk.Coins{sdk.NewInt64Coin(sdk.DefaultBondDenom, 0)},
		DefaultGenTxGas,
		SimAppChainID,
		accNums,
		seq,
		priv...,
	)

	// Simulate a sending a transaction and committing a block
	app.BeginBlock(abci.RequestBeginBlock{Header: abci.Header{Height: app.LastBlockHeight() + 1, ChainID: SimAppChainID}})

	gasInfo, res, err := app.Deliver(tx)
	if expPass {
		require.NoError(t, err)
		require.NotNil(t, res)
	} else {
		require.Error(t, err)
		require.Nil(t, res)
	}

	app.EndBlock(abci.RequestEndBlock{})
	app.Commit()

	return gasInfo, res, err
}

// GenTx generates a signed mock transaction.
func GenTx(msgs []sdk.Msg, feeAmt sdk.Coins, gas uint64, chainID string, accnums []uint64, seq []uint64, priv ...crypto.PrivKey) auth.StdTx {
	fee := auth.StdFee{
		Amount: feeAmt,
		Gas:    gas,
	}

	sigs := make([]auth.StdSignature, len(priv))

	memo := "Test tx"
	for i, p := range priv {
		// use a empty chainID for ease of testing
		sig, err := p.Sign(auth.StdSignBytes(chainID, accnums[i], seq[i], fee, msgs, memo))
		if err != nil {
			panic(err)
		}

		sigs[i] = auth.StdSignature{
			PubKey:    p.PubKey(),
			Signature: sig,
		}
	}

	return auth.NewStdTx(msgs, fee, sigs, memo)
}
