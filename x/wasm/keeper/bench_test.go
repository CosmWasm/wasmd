package keeper

import (
	"os"
	"testing"

	dbm "github.com/cometbft/cometbft-db"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	"github.com/syndtr/goleveldb/leveldb/opt"

	"github.com/CosmWasm/wasmd/x/wasm/keeper/wasmtesting"
	"github.com/CosmWasm/wasmd/x/wasm/types"
	wasmvm "github.com/CosmWasm/wasmvm"
)

// BenchmarkVerification benchmarks secp256k1 verification which is 1000 gas based on cpu time.
//
// Just this function is copied from
// https://github.com/cosmos/cosmos-sdk/blob/90e9370bd80d9a3d41f7203ddb71166865561569/crypto/keys/internal/benchmarking/bench.go#L48-L62
// And thus under the GO license (BSD style)
func BenchmarkGasNormalization(b *testing.B) {
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

// By comparing the timing for queries on pinned vs unpinned, the difference gives us the overhead of
// instantiating an unpinned contract. That value can be used to determine a reasonable gas price
// for the InstantiationCost
func BenchmarkInstantiationOverhead(b *testing.B) {
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
	}
	for name, spec := range specs {
		b.Run(name, func(b *testing.B) {
			wasmConfig := types.WasmConfig{MemoryCacheSize: 0}
			ctx, keepers := createTestInput(b, false, AvailableCapabilities, wasmConfig, spec.db())
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

// Calculate the time it takes to compile some wasm code the first time.
// This will help us adjust pricing for UploadCode
func BenchmarkCompilation(b *testing.B) {
	specs := map[string]struct {
		wasmFile string
	}{
		"hackatom": {
			wasmFile: "./testdata/hackatom.wasm",
		},
		"burner": {
			wasmFile: "./testdata/burner.wasm",
		},
		"ibc_reflect": {
			wasmFile: "./testdata/ibc_reflect.wasm",
		},
	}

	for name, spec := range specs {
		b.Run(name, func(b *testing.B) {
			wasmConfig := types.WasmConfig{MemoryCacheSize: 0}
			db := dbm.NewMemDB()
			ctx, keepers := createTestInput(b, false, AvailableCapabilities, wasmConfig, db)

			// print out code size for comparisons
			code, err := os.ReadFile(spec.wasmFile)
			require.NoError(b, err)
			b.Logf("\b\b\b\b\b\b\b\b\b\b\b\b\b\b\b\b(size: %d)  ", len(code))

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = StoreExampleContract(b, ctx, keepers, spec.wasmFile)
			}
		})
	}
}

// Calculate the time it takes to prune about 100k wasm codes
func BenchmarkPruningWasmCodes(b *testing.B) {
	specs := map[string]struct {
		db             func() dbm.DB
		setupContracts func(t testing.TB, ctx sdk.Context, keepers TestKeepers, mock types.WasmerEngine)
	}{
		"100k codes unpinned - memory db": {
			db: func() dbm.DB { return dbm.NewMemDB() },
			setupContracts: func(t testing.TB, ctx sdk.Context, keepers TestKeepers, mock types.WasmerEngine) {
				for i := 0; i < 100000; i++ {
					_ = StoreRandomContract(b, ctx, keepers, mock)
				}
			},
		},
		"100k codes unpinned - level db": {
			db: func() dbm.DB {
				levelDB, err := dbm.NewGoLevelDBWithOpts("testing", b.TempDir(), &opt.Options{BlockCacher: opt.NoCacher})
				require.NoError(b, err)
				return levelDB
			},
			setupContracts: func(t testing.TB, ctx sdk.Context, keepers TestKeepers, mock types.WasmerEngine) {
				for i := 0; i < 100000; i++ {
					_ = StoreRandomContract(b, ctx, keepers, mock)
				}
			},
		},
		"100k codes - 20% instantiated - memory db": {
			db: func() dbm.DB { return dbm.NewMemDB() },
			setupContracts: func(t testing.TB, ctx sdk.Context, keepers TestKeepers, mock types.WasmerEngine) {
				for i := 0; i < 100000; i++ {
					if i%5 == 0 {
						_ = SeedNewContractInstance(b, ctx, keepers, mock)
						continue
					}
					_ = StoreRandomContract(b, ctx, keepers, mock)
				}
			},
		},
		"100k codes - 20% instantiated - level db": {
			db: func() dbm.DB {
				levelDB, err := dbm.NewGoLevelDBWithOpts("testing", b.TempDir(), &opt.Options{BlockCacher: opt.NoCacher})
				require.NoError(b, err)
				return levelDB
			},
			setupContracts: func(t testing.TB, ctx sdk.Context, keepers TestKeepers, mock types.WasmerEngine) {
				for i := 0; i < 100000; i++ {
					if i%5 == 0 {
						_ = SeedNewContractInstance(b, ctx, keepers, mock)
						continue
					}
					_ = StoreRandomContract(b, ctx, keepers, mock)
				}
			},
		},
		"100k codes - 50% instantiated - memory db": {
			db: func() dbm.DB { return dbm.NewMemDB() },
			setupContracts: func(t testing.TB, ctx sdk.Context, keepers TestKeepers, mock types.WasmerEngine) {
				for i := 0; i < 100000; i++ {
					if i%2 == 0 {
						_ = StoreRandomContract(b, ctx, keepers, mock)
						continue
					}
					_ = SeedNewContractInstance(b, ctx, keepers, mock)
				}
			},
		},
		"100k codes - 50% instantiated - level db": {
			db: func() dbm.DB {
				levelDB, err := dbm.NewGoLevelDBWithOpts("testing", b.TempDir(), &opt.Options{BlockCacher: opt.NoCacher})
				require.NoError(b, err)
				return levelDB
			},
			setupContracts: func(t testing.TB, ctx sdk.Context, keepers TestKeepers, mock types.WasmerEngine) {
				for i := 0; i < 100000; i++ {
					if i%2 == 0 {
						_ = StoreRandomContract(b, ctx, keepers, mock)
						continue
					}
					_ = SeedNewContractInstance(b, ctx, keepers, mock)
				}
			},
		},
		"100k codes - 80% instantiated - memory db": {
			db: func() dbm.DB { return dbm.NewMemDB() },
			setupContracts: func(t testing.TB, ctx sdk.Context, keepers TestKeepers, mock types.WasmerEngine) {
				for i := 0; i < 100000; i++ {
					if i%5 == 0 {
						_ = StoreRandomContract(b, ctx, keepers, mock)
						continue
					}
					_ = SeedNewContractInstance(b, ctx, keepers, mock)
				}
			},
		},
		"100k codes - 80% instantiated - level db": {
			db: func() dbm.DB {
				levelDB, err := dbm.NewGoLevelDBWithOpts("testing", b.TempDir(), &opt.Options{BlockCacher: opt.NoCacher})
				require.NoError(b, err)
				return levelDB
			},
			setupContracts: func(t testing.TB, ctx sdk.Context, keepers TestKeepers, mock types.WasmerEngine) {
				for i := 0; i < 100000; i++ {
					if i%5 == 0 {
						_ = StoreRandomContract(b, ctx, keepers, mock)
						continue
					}
					_ = SeedNewContractInstance(b, ctx, keepers, mock)
				}
			},
		},
	}

	for name, spec := range specs {
		b.Run(name, func(b *testing.B) {
			mockWasmVM := wasmtesting.MockWasmer{
				RemoveCodeFn: func(checksum wasmvm.Checksum) error {
					return nil
				},
			}
			wasmtesting.MakeInstantiable(&mockWasmVM)

			wasmConfig := types.WasmConfig{MemoryCacheSize: 0}
			ctx, keepers := createTestInput(b, false, AvailableCapabilities, wasmConfig, spec.db(), WithWasmEngine(&mockWasmVM))

			spec.setupContracts(b, ctx, keepers, &mockWasmVM)

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = keepers.WasmKeeper.PruneWasmCodes(ctx, 100000)
			}
		})
	}
}
