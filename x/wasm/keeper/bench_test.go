package keeper

import (
	"testing"

	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	"github.com/stretchr/testify/require"
	"github.com/syndtr/goleveldb/leveldb/opt"
	dbm "github.com/tendermint/tm-db"

	"github.com/CosmWasm/wasmd/x/wasm/types"
)

func BenchmarkExecution(b *testing.B) {

	specs := map[string]struct {
		pinned bool
		db     func() dbm.DB
	}{
		"unpinned, memory db": {
			db: func() dbm.DB { return dbm.NewMemDB() },
		},
		"pinned, memory db": {
			db:     func() dbm.DB { return dbm.NewMemDB() },
			pinned: true,
		},
		"unpinned, level db": {
			db: func() dbm.DB {
				levelDB, err := dbm.NewGoLevelDBWithOpts("testing", b.TempDir(), &opt.Options{BlockCacher: opt.NoCacher})
				require.NoError(b, err)
				return levelDB
			},
		},
		"pinned, level db": {
			db: func() dbm.DB {
				levelDB, err := dbm.NewGoLevelDBWithOpts("testing", b.TempDir(), &opt.Options{BlockCacher: opt.NoCacher})
				require.NoError(b, err)
				return levelDB
			},
			pinned: true,
		},
	}
	for name, spec := range specs {
		b.Run(name, func(b *testing.B) {
			wasmConfig := types.WasmConfig{MemoryCacheSize: 0}
			ctx, keepers := createTestInput(b, false, SupportedFeatures, wasmConfig, spec.db())
			example := InstantiateHackatomExampleContract(b, ctx, keepers)
			if spec.pinned {
				require.NoError(b, keepers.ContractKeeper.PinCode(ctx, example.CodeID))
			}
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, err := keepers.WasmKeeper.QuerySmart(ctx, example.Contract, []byte(`{"verifier":{}}`))
				require.NoError(b, err)
			}
		})
	}
}

// BenchmarkVerification benchmarks the given verification algorithm using
// the provided privkey on a constant message.
// Copied from https://github.com/cosmos/cosmos-sdk/blob/90e9370bd80d9a3d41f7203ddb71166865561569/crypto/keys/internal/benchmarking/bench.go#L48-L62
// And thus under the GO license (BSD style)
func BenchmarkSecpVerification(b *testing.B) {
	priv := secp256k1.GenPrivKey()
	pub := priv.PubKey()

	// use a short message, so this time doesn't get dominated by hashing.
	message := []byte("Hello, world!")
	signature, err := priv.Sign(message)
	if err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pub.VerifySignature(message, signature)
	}
}
