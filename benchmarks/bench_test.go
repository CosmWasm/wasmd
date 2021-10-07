package benchmarks

import (
	"testing"

	"github.com/stretchr/testify/require"
	abci "github.com/tendermint/tendermint/abci/types"
	tmproto "github.com/tendermint/tendermint/proto/tendermint/types"

	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	"github.com/cosmos/cosmos-sdk/simapp"
	simappparams "github.com/cosmos/cosmos-sdk/simapp/params"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"

	wasmapp "github.com/CosmWasm/wasmd/app"
)

var moduleAccAddr = authtypes.NewModuleAddress(stakingtypes.BondedPoolName)

var (
	priv1 = secp256k1.GenPrivKey()
	addr1 = sdk.AccAddress(priv1.PubKey().Address())
	priv2 = secp256k1.GenPrivKey()
	addr2 = sdk.AccAddress(priv2.PubKey().Address())

	coins    = sdk.Coins{sdk.NewInt64Coin("foocoin", 10)}
	sendMsg1 = banktypes.NewMsgSend(addr1, addr2, coins)
)

func BenchmarkOneBankSendTxPerBlock(b *testing.B) {
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
