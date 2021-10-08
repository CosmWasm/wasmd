package benchmarks

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/syndtr/goleveldb/leveldb/opt"

	abci "github.com/tendermint/tendermint/abci/types"
	tmproto "github.com/tendermint/tendermint/proto/tendermint/types"
	dbm "github.com/tendermint/tm-db"

	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	"github.com/cosmos/cosmos-sdk/simapp"
	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
)

func bankSendTxs(b *testing.B, info *AppInfo) []sdk.Tx {
	// Precompute all txs
	rcpt := sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address())
	coins := sdk.Coins{sdk.NewInt64Coin(info.Denom, 100)}
	sendMsg := banktypes.NewMsgSend(info.MinterAddr, rcpt, coins)

	txs, err := simapp.GenSequenceOfTxs(info.TxConfig, []sdk.Msg{sendMsg}, []uint64{info.AccNum}, []uint64{info.SeqNum}, b.N, info.MinterKey)
	require.NoError(b, err)
	return txs
}

func cw20TransferTxs(b *testing.B, info *AppInfo) []sdk.Tx {
	rcpt := sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address())
	transfer := cw20ExecMsg{Transfer: &transferMsg{
		Recipient: rcpt.String(),
		Amount:    765,
	}}
	transferBz, err := json.Marshal(transfer)

	sendMsg := wasmtypes.MsgExecuteContract{
		Sender:   info.MinterAddr.String(),
		Contract: info.ContractAddr,
		Msg:      transferBz,
	}
	txs, err := simapp.GenSequenceOfTxs(info.TxConfig, []sdk.Msg{&sendMsg}, []uint64{info.AccNum}, []uint64{info.SeqNum}, b.N, info.MinterKey)
	require.NoError(b, err)
	return txs
}

func buildMemDB(b *testing.B) dbm.DB {
	return dbm.NewMemDB()
}

func buildLevelDB(b *testing.B) dbm.DB {
	levelDB, err := dbm.NewGoLevelDBWithOpts("testing", b.TempDir(), &opt.Options{BlockCacher: opt.NoCacher})
	require.NoError(b, err)
	return levelDB
}

func BenchmarkTxSending(b *testing.B) {
	cases := map[string]struct {
		db          func(*testing.B) dbm.DB
		txBuilder   func(*testing.B, *AppInfo) []sdk.Tx
		blockSize   int
		numAccounts int
	}{
		"basic send - memdb": {
			db:        buildMemDB,
			blockSize: 20,
			txBuilder: bankSendTxs,
		},
		"cw20 transfer - memdb": {
			db:        buildMemDB,
			blockSize: 20,
			txBuilder: cw20TransferTxs,
		},
		"basic send - leveldb": {
			db:        buildLevelDB,
			blockSize: 20,
			txBuilder: bankSendTxs,
		},
		"cw20 transfer - leveldb": {
			db:        buildLevelDB,
			blockSize: 20,
			txBuilder: cw20TransferTxs,
		},
	}

	for name, tc := range cases {
		b.Run(name, func(b *testing.B) {
			db := tc.db(b)
			appInfo := InitializeWasmApp(b, db, tc.numAccounts)
			txs := tc.txBuilder(b, &appInfo)

			// number of Tx per block for the benchmarks
			blockSize := tc.blockSize
			height := int64(3)
			txEncoder := appInfo.TxConfig.TxEncoder()

			b.ResetTimer()

			for i := 0; i < b.N/blockSize; i++ {
				appInfo.App.BeginBlock(abci.RequestBeginBlock{Header: tmproto.Header{Height: height, Time: time.Now()}})

				for j := 0; j < blockSize; j++ {
					idx := i*blockSize + j

					_, _, err := appInfo.App.Check(txEncoder, txs[idx])
					if err != nil {
						panic("something is broken in checking transaction")
					}
					_, _, err = appInfo.App.Deliver(txEncoder, txs[idx])
					require.NoError(b, err)
				}

				appInfo.App.EndBlock(abci.RequestEndBlock{Height: height})
				appInfo.App.Commit()
				height++
			}
		})
	}
}
