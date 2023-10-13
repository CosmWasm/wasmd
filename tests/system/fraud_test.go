//go:build system_test

package system

import (
	"fmt"
	"math"
	"testing"

	"github.com/stretchr/testify/require"

	sdkmath "cosmossdk.io/math"
)

func TestRecursiveMsgsExternalTrigger(t *testing.T) {
	sut.ResetChain(t)
	const maxBlockGas = 2_000_000
	sut.ModifyGenesisJSON(t, SetConsensusMaxGas(t, maxBlockGas))
	sut.StartChain(t)
	cli := NewWasmdCLI(t, sut, verbose)

	codeID := cli.WasmStore("./testdata/hackatom.wasm.gzip", "--from=node0", "--gas=1500000", "--fees=2stake")
	initMsg := fmt.Sprintf(`{"verifier":%q, "beneficiary":%q}`, randomBech32Addr(), randomBech32Addr())
	contractAddr := cli.WasmInstantiate(codeID, initMsg)

	specs := map[string]struct {
		gas           string
		expErrMatcher RunErrorAssert
	}{
		"simulation": {
			gas:           "auto",
			expErrMatcher: ErrOutOfGasMatcher,
		},
	}
	for name, spec := range specs {
		t.Run(name, func(t *testing.T) {
			cli := NewWasmdCLI(t, sut, verbose)
			execMsg := `{"message_loop":{}}`
			fees := "1stake"
			gas := spec.gas
			if gas != "auto" {
				fees = calcMinFeeRequired(t, gas)
			}
			for _, n := range sut.AllNodes(t) {
				clix := cli.
					WithRunErrorMatcher(spec.expErrMatcher).
					WithNodeAddress(n.RPCAddr()).
					WithAssertTXUncommitted()
				clix.WasmExecute(contractAddr, execMsg, defaultSrcAddr, "--gas="+gas, "--broadcast-mode=sync", "--fees="+fees)
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

// with default gas factor and token
func calcMinFeeRequired(t *testing.T, gas string) string {
	x, ok := sdkmath.NewIntFromString(gas)
	require.True(t, ok)
	const defaultTestnetFee = "0.000006"
	minFee, err := sdkmath.LegacyNewDecFromStr(defaultTestnetFee)
	require.NoError(t, err)
	return fmt.Sprintf("%sstake", minFee.Mul(sdkmath.LegacyNewDecFromInt(x)).RoundInt().String())
}
