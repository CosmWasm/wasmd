//go:build system_test

package system

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func TestBasicWasmTest(t *testing.T) {
	// Scenario:
	// upload code
	// instantiate contract
	// watch for an event
	sut.ResetChain(t)
	sut.StartChain(t)

	cli := NewWasmdCLI(t, sut, verbose)
	t.Log("List keys")
	t.Log("keys", cli.Keys("keys", "list"))

	t.Log("Upload wasm code")
	txResult := cli.CustomCommand("tx", "wasm", "store", "./testdata/hackatom.wasm.gzip", "--from=node0", "--gas=1500000", "--fees=2stake")
	RequireTxSuccess(t, txResult)

	t.Log("Waiting for block")
	sut.AwaitNextBlock(t)

	t.Log("Query wasm code list")
	qResult := cli.CustomQuery("q", "wasm", "list-code")
	codes := gjson.Get(qResult, "code_infos.#.code_id").Array()
	t.Log("got query result", qResult)

	require.Equal(t, int64(1), codes[0].Int())
	codeID := 1

	l := sut.NewEventListener(t)
	c, done := CaptureAllEventsConsumer(t)
	expContractAddr := ContractBech32Address(1, 1)
	query := fmt.Sprintf(`tm.event='Tx' AND wasm._contract_address='%s'`, expContractAddr)
	t.Logf("Subscribe to events: %s", query)
	cleanupFn := l.Subscribe(query, c)
	t.Cleanup(cleanupFn)

	t.Log("Instantiate wasm code")
	initMsg := fmt.Sprintf(`{"verifier":%q, "beneficiary":%q}`, randomBech32Addr(), randomBech32Addr())
	newContractAddr := cli.WasmInstantiate(codeID, initMsg)
	assert.Equal(t, expContractAddr, newContractAddr)
	assert.Len(t, done(), 1)
}
