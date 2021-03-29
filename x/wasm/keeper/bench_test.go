package keeper

import (
	"github.com/CosmWasm/wasmd/x/wasm/types"
	"github.com/stretchr/testify/require"
	"github.com/syndtr/goleveldb/leveldb/opt"
	dbm "github.com/tendermint/tm-db"
	"testing"
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
