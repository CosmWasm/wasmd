//go:build system_test

package system

import (
	"fmt"
	"math"
	"testing"
)

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
