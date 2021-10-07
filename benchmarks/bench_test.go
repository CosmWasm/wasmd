package benchmarks

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/require"
	abci "github.com/tendermint/tendermint/abci/types"
	tmproto "github.com/tendermint/tendermint/proto/tendermint/types"

	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	"github.com/cosmos/cosmos-sdk/simapp"
	"github.com/cosmos/cosmos-sdk/simapp/helpers"
	simappparams "github.com/cosmos/cosmos-sdk/simapp/params"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"

	wasmapp "github.com/CosmWasm/wasmd/app"
	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
)

var moduleAccAddr = authtypes.NewModuleAddress(stakingtypes.BondedPoolName)

var (
	priv1 = secp256k1.GenPrivKey()
	addr1 = sdk.AccAddress(priv1.PubKey().Address())
	priv2 = secp256k1.GenPrivKey()
	addr2 = sdk.AccAddress(priv2.PubKey().Address())
)

type cw20InitMsg struct {
	Name            string    `json:"name"`
	Symbol          string    `json:"symbol"`
	Decimals        uint8     `json:"decimals"`
	InitialBalances []balance `json:"initial_balances"`
}

type balance struct {
	Address string `json:"address"`
	Amount  uint64 `json:"amount,string"`
}

type cw20ExecMsg struct {
	Transfer *transferMsg `json:"transfer,omitempty"`
}

type transferMsg struct {
	Recipient string `json:"recipient"`
	Amount    uint64 `json:"amount,string"`
}

func BenchmarkNCw20SendTxPerBlock(b *testing.B) {
	// Add an account at genesis
	acc := authtypes.BaseAccount{
		Address: addr1.String(),
	}

	// construct genesis state
	genAccs := []authtypes.GenesisAccount{&acc}
	benchmarkApp := wasmapp.SetupWithGenesisAccounts(genAccs, banktypes.Balance{
		Address: addr1.String(),
		Coins:   sdk.NewCoins(sdk.NewInt64Coin("foocoin", 100000000000)),
	})
	txGen := simappparams.MakeTestEncodingConfig().TxConfig

	// wasm setup
	height := int64(2)
	benchmarkApp.BeginBlock(abci.RequestBeginBlock{Header: tmproto.Header{Height: height}})

	// upload the code
	cw20Code, err := ioutil.ReadFile("./testdata/cw20_base.wasm")
	require.NoError(b, err)
	storeMsg := wasmtypes.MsgStoreCode{
		Sender:       addr1.String(),
		WASMByteCode: cw20Code,
	}
	storeTx, err := helpers.GenTx(txGen, []sdk.Msg{&storeMsg}, nil, 55123123, "", []uint64{0}, []uint64{0}, priv1)
	require.NoError(b, err)
	_, res, err := benchmarkApp.Deliver(txGen.TxEncoder(), storeTx)
	require.NoError(b, err)
	fmt.Printf("Data: %X\n", res.Data)
	codeID := uint64(1)

	// instantiate the contract
	init := cw20InitMsg{
		Name:     "Cash Money",
		Symbol:   "CASH",
		Decimals: 2,
		InitialBalances: []balance{{
			Address: addr1.String(),
			Amount:  100000000000,
		}},
	}
	initBz, err := json.Marshal(init)
	require.NoError(b, err)
	initMsg := wasmtypes.MsgInstantiateContract{
		Sender: addr1.String(),
		Admin:  addr1.String(),
		CodeID: codeID,
		Label:  "Demo contract",
		Msg:    initBz,
	}
	initTx, err := helpers.GenTx(txGen, []sdk.Msg{&initMsg}, nil, 500000, "", []uint64{0}, []uint64{1}, priv1)
	require.NoError(b, err)
	_, res, err = benchmarkApp.Deliver(txGen.TxEncoder(), initTx)
	require.NoError(b, err)
	// TODO: parse contract address
	fmt.Printf("Data: %X\n", res.Data)
	contractAddr := "wasm123456789"

	benchmarkApp.EndBlock(abci.RequestEndBlock{Height: height})
	benchmarkApp.Commit()
	height++

	// Precompute all txs
	transfer := cw20ExecMsg{Transfer: &transferMsg{
		Recipient: addr2.String(),
		Amount:    765,
	}}
	transferBz, err := json.Marshal(transfer)
	sendMsg1 := wasmtypes.MsgExecuteContract{
		Sender:   addr1.String(),
		Contract: contractAddr,
		Msg:      transferBz,
	}
	txs, err := simapp.GenSequenceOfTxs(txGen, []sdk.Msg{&sendMsg1}, []uint64{0}, []uint64{uint64(2)}, b.N, priv1)
	require.NoError(b, err)
	b.ResetTimer()

	// number of Tx per block for the benchmarks
	blockSize := 20

	// Run this with a profiler, so its easy to distinguish what time comes from
	// Committing, and what time comes from Check/Deliver Tx.
	for i := 0; i < b.N/blockSize; i++ {
		benchmarkApp.BeginBlock(abci.RequestBeginBlock{Header: tmproto.Header{Height: height}})

		for j := 0; j < blockSize; j++ {
			idx := i*blockSize + j

			_, _, err := benchmarkApp.Check(txGen.TxEncoder(), txs[idx])
			if err != nil {
				panic("something is broken in checking transaction")
			}
			_, _, err = benchmarkApp.Deliver(txGen.TxEncoder(), txs[idx])
			require.NoError(b, err)
		}

		benchmarkApp.EndBlock(abci.RequestEndBlock{Height: height})
		benchmarkApp.Commit()
		height++
	}
}

func BenchmarkNBankSendTxsPerBlock(b *testing.B) {
	// Add an account at genesis
	acc := authtypes.BaseAccount{
		Address: addr1.String(),
	}

	// construct genesis state
	genAccs := []authtypes.GenesisAccount{&acc}
	benchmarkApp := wasmapp.SetupWithGenesisAccounts(genAccs, banktypes.Balance{
		Address: addr1.String(),
		Coins:   sdk.NewCoins(sdk.NewInt64Coin("foocoin", 100000000000)),
	})
	txGen := simappparams.MakeTestEncodingConfig().TxConfig

	// Precompute all txs
	coins := sdk.Coins{sdk.NewInt64Coin("foocoin", 10)}
	sendMsg1 := banktypes.NewMsgSend(addr1, addr2, coins)
	txs, err := simapp.GenSequenceOfTxs(txGen, []sdk.Msg{sendMsg1}, []uint64{0}, []uint64{uint64(0)}, b.N, priv1)
	require.NoError(b, err)
	b.ResetTimer()

	height := int64(2)
	// number of Tx per block for the benchmarks
	blockSize := 20

	// Run this with a profiler, so its easy to distinguish what time comes from
	// Committing, and what time comes from Check/Deliver Tx.
	for i := 0; i < b.N/blockSize; i++ {
		benchmarkApp.BeginBlock(abci.RequestBeginBlock{Header: tmproto.Header{Height: height}})

		for j := 0; j < blockSize; j++ {
			idx := i*blockSize + j

			_, _, err := benchmarkApp.Check(txGen.TxEncoder(), txs[idx])
			if err != nil {
				panic("something is broken in checking transaction")
			}
			_, _, err = benchmarkApp.Deliver(txGen.TxEncoder(), txs[idx])
			require.NoError(b, err)
		}

		benchmarkApp.EndBlock(abci.RequestEndBlock{Height: height})
		benchmarkApp.Commit()
		height++
	}
}
