package integration

/**
This file is full of test helper functions, taken from simapp
**/

import (
	"encoding/json"
	"math/rand"
	"os"
	"testing"
	"time"

	wasmd "github.com/CosmWasm/wasmd/app"
	"github.com/CosmWasm/wasmd/x/wasm"
	"github.com/cosmos/cosmos-sdk/client"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/simulation"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	authsign "github.com/cosmos/cosmos-sdk/x/auth/signing"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/stretchr/testify/require"
	abci "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/crypto"
	"github.com/tendermint/tendermint/libs/log"
	tmproto "github.com/tendermint/tendermint/proto/tendermint/types"
	tmtypes "github.com/tendermint/tendermint/types"
	dbm "github.com/tendermint/tm-db"
)

// SimAppChainID hardcoded chainID for simulation
const (
	DefaultGenTxGas = 1000000
	SimAppChainID   = "wasmd-app"
)

// Setup initializes a new wasmd.WasmApp. A Nop logger is set in WasmApp.
func Setup(isCheckTx bool, homeDir string) *wasmd.WasmApp {
	db := dbm.NewMemDB()
	app := wasmd.NewWasmApp(log.NewTMLogger(log.NewSyncWriter(os.Stdout)), db, nil, true, map[int64]bool{}, homeDir, 0, wasm.EnableAllProposals)
	// app := wasmd.NewWasmApp(log.NewNopLogger(), db, nil, true, map[int64]bool{}, 0)
	if !isCheckTx {
		// init chain must be called to stop deliverState from being nil
		genesisState := wasmd.NewDefaultGenesisState()
		stateBytes, err := json.Marshal(genesisState)
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

type GenesisState map[string]json.RawMessage

// NewDefaultGenesisState generates the default state for the application.
func NewDefaultGenesisState() GenesisState {
	encCfg := wasmd.MakeEncodingConfig()
	return wasmd.ModuleBasics.DefaultGenesis(encCfg.Marshaler)
}

func SetupWithGenesisValSet(t *testing.T, homeDir string, valSet *tmtypes.ValidatorSet, genAccs []authtypes.GenesisAccount, balances ...banktypes.Balance) *wasmd.WasmApp {
	db := dbm.NewMemDB()
	app := wasmd.NewWasmApp(log.NewNopLogger(), db, nil, true, map[int64]bool{}, homeDir, 5, wasm.EnableAllProposals)

	genesisState := NewDefaultGenesisState()

	// set genesis accounts
	authGenesis := authtypes.NewGenesisState(authtypes.DefaultParams(), genAccs)
	genesisState[authtypes.ModuleName] = app.AppCodec().MustMarshalJSON(authGenesis)

	validators := make([]stakingtypes.Validator, 0, len(valSet.Validators))
	delegations := make([]stakingtypes.Delegation, 0, len(valSet.Validators))

	bondAmt := sdk.NewInt(1000000)

	for _, val := range valSet.Validators {
		validator := stakingtypes.Validator{
			OperatorAddress:   val.Address.Bytes(),
			ConsensusPubkey:   sdk.MustBech32ifyPubKey(sdk.Bech32PubKeyTypeConsPub, val.PubKey),
			Jailed:            false,
			Status:            sdk.Bonded,
			Tokens:            bondAmt,
			DelegatorShares:   sdk.OneDec(),
			Description:       stakingtypes.Description{},
			UnbondingHeight:   int64(0),
			UnbondingTime:     time.Unix(0, 0).UTC(),
			Commission:        stakingtypes.NewCommission(sdk.ZeroDec(), sdk.ZeroDec(), sdk.ZeroDec()),
			MinSelfDelegation: sdk.ZeroInt(),
		}
		validators = append(validators, validator)
		delegations = append(delegations, stakingtypes.NewDelegation(genAccs[0].GetAddress(), val.Address.Bytes(), sdk.OneDec()))

	}

	// set validators and delegations
	stakingGenesis := stakingtypes.NewGenesisState(stakingtypes.DefaultParams(), validators, delegations)
	genesisState[stakingtypes.ModuleName] = app.AppCodec().MustMarshalJSON(stakingGenesis)

	totalSupply := sdk.NewCoins()
	for _, b := range balances {
		// add genesis acc tokens and delegated tokens to total supply
		totalSupply = totalSupply.Add(b.Coins.Add(sdk.NewCoin(sdk.DefaultBondDenom, bondAmt))...)
	}

	// update total supply
	bankGenesis := banktypes.NewGenesisState(banktypes.DefaultGenesisState().Params, balances, totalSupply, []banktypes.Metadata{})
	genesisState[banktypes.ModuleName] = app.AppCodec().MustMarshalJSON(bankGenesis)

	stateBytes, err := json.MarshalIndent(genesisState, "", " ")
	require.NoError(t, err)

	// init chain will set the validator set and initialize the genesis accounts
	app.InitChain(
		abci.RequestInitChain{
			Validators:      []abci.ValidatorUpdate{},
			ConsensusParams: DefaultConsensusParams,
			AppStateBytes:   stateBytes,
		},
	)

	// commit genesis changes
	app.Commit()
	app.BeginBlock(abci.RequestBeginBlock{Header: tmproto.Header{
		Height:             app.LastBlockHeight() + 1,
		AppHash:            app.LastCommitID().Hash,
		ValidatorsHash:     valSet.Hash(),
		NextValidatorsHash: valSet.Hash(),
	}})

	return app
}

var DefaultConsensusParams = &abci.ConsensusParams{
	Block: &abci.BlockParams{
		MaxBytes: 200000,
		MaxGas:   2000000,
	},
	Evidence: &tmproto.EvidenceParams{
		MaxAgeNumBlocks: 302400,
		MaxAgeDuration:  1814400,
	},
	Validator: &tmproto.ValidatorParams{
		PubKeyTypes: []string{
			tmtypes.ABCIPubKeyTypeEd25519,
		},
	},
}

// SignAndDeliver checks a generated signed transaction and simulates a
// block commitment with the given transaction. A test assertion is made using
// the parameter 'expPass' against the result. A corresponding result is
// returned.
func SignAndDeliver(
	t *testing.T, app *wasmd.WasmApp, msgs []sdk.Msg,
	accNums, seq []uint64, expPass bool, priv ...crypto.PrivKey,
) (sdk.GasInfo, *sdk.Result, error) {
	t.Helper()
	txGen := wasmd.MakeEncodingConfig().TxConfig
	tx, err := GenTx(
		txGen,
		msgs,
		sdk.Coins{sdk.NewInt64Coin(sdk.DefaultBondDenom, 0)},
		DefaultGenTxGas,
		SimAppChainID,
		accNums,
		seq,
		priv...,
	)
	require.NoError(t, err)
	txBytes, err := txGen.TxEncoder()(tx)
	require.Nil(t, err)

	// Must simulate now as CheckTx doesn't run Msgs anymore
	_, res, err := app.Simulate(txBytes, tx)

	expSimPass := true
	if expSimPass {
		require.NoError(t, err)
		require.NotNil(t, res)
	} else {
		require.Error(t, err)
		require.Nil(t, res)
	}

	// Simulate a sending a transaction and committing a block
	app.BeginBlock(abci.RequestBeginBlock{Header: tmproto.Header{Height: app.LastBlockHeight() + 1, ChainID: SimAppChainID}})

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
func GenTx(gen client.TxConfig, msgs []sdk.Msg, feeAmt sdk.Coins, gas uint64, chainID string, accnums []uint64, seq []uint64, priv ...crypto.PrivKey) (sdk.Tx, error) {
	sigs := make([]signing.SignatureV2, len(priv))

	// create a random length memo
	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	memo := simulation.RandStringOfLength(r, simulation.RandIntBetween(r, 0, 100))

	signMode := gen.SignModeHandler().DefaultMode()

	for i, p := range priv {
		sigs[i] = signing.SignatureV2{
			PubKey: p.PubKey(),
			Data: &signing.SingleSignatureData{
				SignMode: signMode,
			},
		}
	}

	tx := gen.NewTxBuilder()
	err := tx.SetMsgs(msgs...)
	if err != nil {
		return nil, err
	}
	err = tx.SetSignatures(sigs...)
	if err != nil {
		return nil, err
	}
	tx.SetMemo(memo)
	tx.SetFeeAmount(feeAmt)
	tx.SetGasLimit(gas)
	for i, p := range priv {
		// use a empty chainID for ease of testing
		signerData := authsign.SignerData{
			ChainID:         chainID,
			AccountNumber:   accnums[i],
			AccountSequence: seq[i],
		}
		signBytes, err := gen.SignModeHandler().GetSignBytes(signMode, signerData, tx.GetTx())
		if err != nil {
			panic(err)
		}
		sig, err := p.Sign(signBytes)
		if err != nil {
			panic(err)
		}
		sigs[i].Data.(*signing.SingleSignatureData).Signature = sig
		err = tx.SetSignatures(sigs...)
		if err != nil {
			panic(err)
		}
	}

	return tx.GetTx(), nil
}
