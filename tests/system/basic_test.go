//go:build system_test

package system

import (
	"encoding/base64"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func TestBasicWasm(t *testing.T) {
	// Scenario:
	// upload code
	// instantiate contract
	// watch for an event
	// update instantiate contract
	// set contract admin
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
	verifierAddr := randomBech32Addr()
	initMsg := fmt.Sprintf(`{"verifier":%q, "beneficiary":%q}`, verifierAddr, randomBech32Addr())
	newContractAddr := cli.WasmInstantiate(codeID, initMsg, "--admin="+defaultSrcAddr, "--label=label1", "--from="+defaultSrcAddr)
	require.Equal(t, expContractAddr, newContractAddr)
	require.Len(t, done(), 1)
	gotRsp := cli.QuerySmart(newContractAddr, `{"verifier":{}}`)
	require.Equal(t, fmt.Sprintf(`{"data":{"verifier":"%s"}}`, verifierAddr), gotRsp)

	t.Log("Update Instantiate Config")
	qResult = cli.CustomQuery("q", "wasm", "code-info", fmt.Sprint(codeID))
	assert.Equal(t, "Everybody", gjson.Get(qResult, "instantiate_permission.permission").String())

	rsp := cli.CustomCommand("tx", "wasm", "update-instantiate-config", fmt.Sprint(codeID), "--instantiate-anyof-addresses="+cli.GetKeyAddr(defaultSrcAddr), "--from="+defaultSrcAddr)
	RequireTxSuccess(t, rsp)

	qResult = cli.CustomQuery("q", "wasm", "code-info", fmt.Sprint(codeID))
	t.Log(qResult)
	assert.Equal(t, "AnyOfAddresses", gjson.Get(qResult, "instantiate_permission.permission").String())
	assert.Equal(t, cli.GetKeyAddr(defaultSrcAddr), gjson.Get(qResult, "instantiate_permission.addresses").Array()[0].String())

	t.Log("Set contract admin")
	newAdmin := randomBech32Addr()
	rsp = cli.CustomCommand("tx", "wasm", "set-contract-admin", newContractAddr, newAdmin, "--from="+defaultSrcAddr)
	RequireTxSuccess(t, rsp)

	qResult = cli.CustomQuery("q", "wasm", "contract", newContractAddr)
	actualAdmin := gjson.Get(qResult, "contract_info.admin").String()
	assert.Equal(t, newAdmin, actualAdmin)
}

func TestMultiContract(t *testing.T) {
	// Scenario:
	// upload reflect code
	// upload hackatom escrow code
	// creator instantiates a contract and gives it tokens
	// reflect a message through the reflect to call the escrow
	sut.ResetChain(t)
	sut.StartChain(t)

	cli := NewWasmdCLI(t, sut, verbose)

	bobAddr := randomBech32Addr()

	t.Log("Upload reflect code")
	reflectID := cli.WasmStore("./testdata/reflect.wasm.gzip", "--from=node0", "--gas=1900000", "--fees=2stake")

	t.Log("Upload hackatom code")
	hackatomID := cli.WasmStore("./testdata/hackatom.wasm.gzip", "--from=node0", "--gas=1900000", "--fees=2stake")

	t.Log("Instantiate reflect code")
	reflectContractAddr := cli.WasmInstantiate(reflectID, "{}", "--admin="+defaultSrcAddr, "--label=reflect_contract", "--from="+defaultSrcAddr, "--amount=100stake")

	t.Log("Instantiate hackatom code")
	initMsg := fmt.Sprintf(`{"verifier":%q, "beneficiary":%q}`, reflectContractAddr, bobAddr)
	hackatomContractAddr := cli.WasmInstantiate(hackatomID, initMsg, "--admin="+defaultSrcAddr, "--label=hackatom_contract", "--from="+defaultSrcAddr, "--amount=50stake")

	// check balances
	assert.Equal(t, int64(100), cli.QueryBalance(reflectContractAddr, "stake"))
	assert.Equal(t, int64(50), cli.QueryBalance(hackatomContractAddr, "stake"))
	assert.Equal(t, int64(0), cli.QueryBalance(bobAddr, "stake"))

	// now for the trick.... we reflect a message through the reflect to call the escrow
	// we also send an additional 20stake tokens there.
	// this should reduce the reflect balance by 20stake (to 80stake)
	// this 20stake is added to the escrow, then the entire balance is sent to bob (total: 70stake)
	approveMsg := []byte(`{"release":{}}`)
	reflectSendMsg := fmt.Sprintf(`{"reflect_msg":{"msgs":[{"wasm":{"execute":{"contract_addr":%q,"msg":%q,"funds":[{"denom":"stake","amount":"20"}]}}}]}}`, hackatomContractAddr, base64.StdEncoding.EncodeToString(approveMsg))
	t.Log(reflectSendMsg)
	rsp := cli.WasmExecute(reflectContractAddr, reflectSendMsg, defaultSrcAddr, "--gas=2500000", "--fees=4stake")
	RequireTxSuccess(t, rsp)

	assert.Equal(t, int64(80), cli.QueryBalance(reflectContractAddr, "stake"))
	assert.Equal(t, int64(0), cli.QueryBalance(hackatomContractAddr, "stake"))
	assert.Equal(t, int64(70), cli.QueryBalance(bobAddr, "stake"))
}
