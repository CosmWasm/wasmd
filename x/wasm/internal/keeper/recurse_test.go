package keeper

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"testing"

	"github.com/CosmWasm/wasmd/x/wasm/internal/types"
	abci "github.com/tendermint/tendermint/abci/types"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

type Recurse struct {
	Depth    uint32         `json:"depth"`
	Work     uint32         `json:"work"`
	Contract sdk.AccAddress `json:"contract"`
}

type recurseWrapper struct {
	Recurse Recurse `json:"recurse"`
}

func buildQuery(t *testing.T, msg Recurse) []byte {
	wrapper := recurseWrapper{Recurse: msg}
	bz, err := json.Marshal(wrapper)
	require.NoError(t, err)
	return bz
}

type recurseResponse struct {
	Hashed []byte `json:"hashed"`
}

func TestGasCostOnQuery(t *testing.T) {
	const (
		// after moving to instance cost from params we got a +5081 in gas. I append the value to highlight it for review
		GasNoWork uint64 = types.DefaultInstanceCost + 2_756 + 5081
		// Note: about 100 SDK gas (10k wasmer gas) for each round of sha256
		GasWork50 uint64 = types.DefaultInstanceCost + 8_464 + 5081 // this is a little shy of 50k gas - to keep an eye on the limit

		GasReturnUnhashed uint64 = 647
		GasReturnHashed   uint64 = 597
	)

	cases := map[string]struct {
		gasLimit    uint64
		msg         Recurse
		expectedGas uint64
	}{
		"no recursion, no work": {
			gasLimit:    400_000,
			msg:         Recurse{},
			expectedGas: GasNoWork,
		},
		"no recursion, some work": {
			gasLimit: 400_000,
			msg: Recurse{
				Work: 50, // 50 rounds of sha256 inside the contract
			},
			expectedGas: GasWork50,
		},
		"recursion 1, no work": {
			gasLimit: 400_000,
			msg: Recurse{
				Depth: 1,
			},
			expectedGas: 2*GasNoWork + GasReturnUnhashed,
		},
		"recursion 1, some work": {
			gasLimit: 400_000,
			msg: Recurse{
				Depth: 1,
				Work:  50,
			},
			expectedGas: 2*GasWork50 + GasReturnHashed,
		},
		"recursion 4, some work": {
			gasLimit: 400_000,
			msg: Recurse{
				Depth: 4,
				Work:  50,
			},
			// this is (currently) 244_708 gas
			expectedGas: 5*GasWork50 + 4*GasReturnHashed,
		},
	}

	// we do one basic setup before all test cases (which are read-only and don't change state)
	tempDir, err := ioutil.TempDir("", "wasm")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	ctx, keepers := CreateTestInput(t, false, tempDir, SupportedFeatures, nil, nil)
	accKeeper, keeper := keepers.AccountKeeper, keepers.WasmKeeper
	deposit := sdk.NewCoins(sdk.NewInt64Coin("denom", 100000))
	creator := createFakeFundedAccount(ctx, accKeeper, deposit.Add(deposit...))

	// store the code
	wasmCode, err := ioutil.ReadFile("./testdata/contract.wasm")
	require.NoError(t, err)
	codeID, err := keeper.Create(ctx, creator, wasmCode, "", "", nil)
	require.NoError(t, err)

	// instantiate the contract
	_, _, bob := keyPubAddr()
	_, _, fred := keyPubAddr()
	initMsg := InitMsg{
		Verifier:    fred,
		Beneficiary: bob,
	}
	initMsgBz, err := json.Marshal(initMsg)
	require.NoError(t, err)
	contractAddr, err := keeper.Instantiate(ctx, codeID, creator, nil, initMsgBz, "recursive contract", deposit)
	require.NoError(t, err)

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			// external limit has no effect (we get a panic if this is enforced)
			keeper.queryGasLimit = 1000

			// make sure we set a limit before calling
			ctx = ctx.WithGasMeter(sdk.NewGasMeter(tc.gasLimit))
			require.Equal(t, uint64(0), ctx.GasMeter().GasConsumed())

			// do the query
			recurse := tc.msg
			recurse.Contract = contractAddr
			msg := buildQuery(t, recurse)
			data, err := keeper.QuerySmart(ctx, contractAddr, msg)
			require.NoError(t, err)

			// check the gas is what we expected
			assert.Equal(t, tc.expectedGas, ctx.GasMeter().GasConsumed())

			// assert result is 32 byte sha256 hash (if hashed), or contractAddr if not
			var resp recurseResponse
			err = json.Unmarshal(data, &resp)
			require.NoError(t, err)
			if recurse.Work == 0 {
				assert.Equal(t, len(resp.Hashed), len(creator.String()))
			} else {
				assert.Equal(t, len(resp.Hashed), 32)
			}
		})
	}
}

func TestGasOnExternalQuery(t *testing.T) {
	const (
		GasWork50       uint64 = types.DefaultInstanceCost + 8_464
		GasReturnHashed uint64 = 597
	)

	cases := map[string]struct {
		gasLimit    uint64
		msg         Recurse
		expectPanic bool
	}{
		"no recursion, plenty gas": {
			gasLimit: 400_000,
			msg: Recurse{
				Work: 50, // 50 rounds of sha256 inside the contract
			},
		},
		"recursion 4, plenty gas": {
			// this uses 244708 gas
			gasLimit: 400_000,
			msg: Recurse{
				Depth: 4,
				Work:  50,
			},
		},
		"no recursion, external gas limit": {
			gasLimit: 5000, // this is not enough
			msg: Recurse{
				Work: 50,
			},
			expectPanic: true,
		},
		"recursion 4, external gas limit": {
			// this uses 244708 gas but give less
			gasLimit: 4 * GasWork50,
			msg: Recurse{
				Depth: 4,
				Work:  50,
			},
			expectPanic: true,
		},
	}

	// we do one basic setup before all test cases (which are read-only and don't change state)
	tempDir, err := ioutil.TempDir("", "wasm")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	ctx, keepers := CreateTestInput(t, false, tempDir, SupportedFeatures, nil, nil)
	accKeeper, keeper := keepers.AccountKeeper, keepers.WasmKeeper
	deposit := sdk.NewCoins(sdk.NewInt64Coin("denom", 100000))
	creator := createFakeFundedAccount(ctx, accKeeper, deposit.Add(deposit...))

	// store the code
	wasmCode, err := ioutil.ReadFile("./testdata/contract.wasm")
	require.NoError(t, err)
	codeID, err := keeper.Create(ctx, creator, wasmCode, "", "", nil)
	require.NoError(t, err)

	// instantiate the contract
	_, _, bob := keyPubAddr()
	_, _, fred := keyPubAddr()
	initMsg := InitMsg{
		Verifier:    fred,
		Beneficiary: bob,
	}
	initMsgBz, err := json.Marshal(initMsg)
	require.NoError(t, err)
	contractAddr, err := keeper.Instantiate(ctx, codeID, creator, nil, initMsgBz, "recursive contract", deposit)
	require.NoError(t, err)

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			// set the external gas limit (normally from config file)
			keeper.queryGasLimit = tc.gasLimit

			recurse := tc.msg
			recurse.Contract = contractAddr
			msg := buildQuery(t, recurse)

			// do the query
			path := []string{QueryGetContractState, contractAddr.String(), QueryMethodContractStateSmart}
			req := abci.RequestQuery{Data: msg}
			if tc.expectPanic {
				require.Panics(t, func() {
					// this should run out of gas
					_, _ = NewQuerier(keeper)(ctx, path, req)
				})
			} else {
				// otherwise, make sure we get a good success
				_, err := NewQuerier(keeper)(ctx, path, req)
				require.NoError(t, err)
			}
		})
	}
}
