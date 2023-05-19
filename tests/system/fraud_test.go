//go:build system_test

package system

import (
	"fmt"
	"math"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRecursiveMsgsExternalTrigger(t *testing.T) {
	t.Skip()
	sut.ResetDirtyChain(t)
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
		"tx": { // tx will be rejected by Tendermint in post abci checkTX operation
			gas:           strconv.Itoa(math.MaxInt64),
			expErrMatcher: ErrTimeoutMatcher,
		},
	}
	for name, spec := range specs {
		t.Run(name, func(t *testing.T) {
			cli := NewWasmdCLI(t, sut, verbose)
			execMsg := `{"message_loop":{}}`
			for _, n := range sut.AllNodes(t) {
				cli.WithRunErrorMatcher(spec.expErrMatcher).WithNodeAddress(n.RPCAddr()).
					WasmExecute(contractAddr, execMsg, defaultSrcAddr, "--gas="+spec.gas, "--broadcast-mode=sync", "--fees=1stake")
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
