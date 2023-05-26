//go:build system_test

package system

import (
	"fmt"
	"math"
	"strconv"
	"testing"

	sdkmath "cosmossdk.io/math"

	"github.com/stretchr/testify/require"
)

func TestRecursiveMsgsExternalTrigger(t *testing.T) {
	const maxBlockGas = 2_000_000
	sut.ModifyGenesisJSON(t, SetConsensusMaxGas(t, maxBlockGas))
	sut.StartChain(t)
	cli := NewWasmdCLI(t, sut, verbose)

	codeID := cli.WasmStore("./testdata/hackatom.wasm.gzip", "--from=node0", "--gas=1500000", "--fees=2stake")
	initMsg := fmt.Sprintf(`{"verifier":%q, "beneficiary":%q}`, randomBech32Addr(), randomBech32Addr())
	contractAddr := cli.WasmInstantiate(codeID, initMsg)

	specs := map[string]struct {
		gas           string
		expErrMatcher func(t require.TestingT, err error, msgAndArgs ...interface{})
	}{
		"simulation": {
			gas:           "auto",
			expErrMatcher: ErrOutOfGasMatcher,
		},
		"tx": { // tx will be rejected by CometBFT in post abci checkTX operation
			gas:           strconv.Itoa(maxBlockGas * 100),
			expErrMatcher: require.NoError,
		},
	}
	for name, spec := range specs {
		t.Run(name, func(t *testing.T) {
			cli := NewWasmdCLI(t, sut, verbose)
			execMsg := `{"message_loop":{}}`
			fees := "1stake"
			if spec.gas != "auto" {
				x, ok := sdkmath.NewIntFromString(spec.gas)
				require.True(t, ok)
				const defaultTestnetFee = "0.000006"
				minFee, err := sdkmath.LegacyNewDecFromStr(defaultTestnetFee)
				require.NoError(t, err)
				fees = fmt.Sprintf("%sstake", minFee.Mul(sdkmath.LegacyNewDecFromInt(x)).RoundInt().String())
			}
			for _, n := range sut.AllNodes(t) {
				clix := cli.WithRunErrorMatcher(spec.expErrMatcher).WithNodeAddress(n.RPCAddr())
				clix.expTXCommitted = false
				clix.WasmExecute(contractAddr, execMsg, defaultSrcAddr, "--gas="+spec.gas, "--broadcast-mode=sync", "--fees="+fees)
			}
			sut.AwaitNextBlock(t)
		})
	}
}

func TestRecursiveSmartQuery(t *testing.T) {
	sut.ResetDirtyChain(t)
	sut.StartChain(t)
	cli := NewWasmdCLI(t, sut, verbose)

	initMsg := fmt.Sprintf(`{"verifier":%q, "beneficiary":%q}`, randomBech32Addr(), randomBech32Addr())
	maliciousContractAddr := cli.WasmInstantiate(cli.WasmStore("./testdata/hackatom.wasm.gzip", "--from=node0", "--gas=1500000", "--fees=2stake"), initMsg)

	msg := fmt.Sprintf(`{"recurse":{"depth":%d, "work":0}}`, math.MaxUint32)

	// when
	for _, n := range sut.AllNodes(t) {
		cli.WithRunErrorMatcher(ErrInvalidQuery).WithNodeAddress(n.RPCAddr()).
			QuerySmart(maliciousContractAddr, msg)
	}
	sut.AwaitNextBlock(t)
}
