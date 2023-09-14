//go:build system_test

package system

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func TestGrantStoreCode(t *testing.T) {
	sut.ResetChain(t)
	cli := NewWasmdCLI(t, sut, verbose)

	// add genesis account with some tokens
	account1Addr := cli.AddKey("account1")

	// add genesis account with some tokens
	account2Addr := cli.AddKey("account2")

	sut.ModifyGenesisCLI(t,
		[]string{"genesis", "add-genesis-account", account1Addr, "100000000stake"},
	)
	sut.ModifyGenesisCLI(t,
		[]string{"genesis", "add-genesis-account", account2Addr, "100000000stake"},
	)

	//set params
	sut.ModifyGenesisJSON(t, SetCodeUploadPermission(t, "AnyOfAddresses", []string{account1Addr}))
	sut.StartChain(t)

	// query validator address to delegate tokens
	rsp := cli.CustomQuery("q", "wasm", "params")
	permission := gjson.Get(rsp, "code_upload_access.permission").String()
	addresses := gjson.Get(rsp, "code_upload_access.addresses").Array()

	assert.Equal(t, permission, "AnyOfAddresses")
	assert.Equal(t, []string{account1Addr}, addresses)

	// grant upload permission to address2
	rsp = cli.CustomCommand("tx", "wasm", "grant-store-code", account2Addr, "*:*", "--from="+account1Addr)
	RequireTxSuccess(t, rsp)

	// address2 store code fails
	rsp = cli.CustomCommand("tx", "wasm", "store", "./testdata/hackatom.wasm.gzip", "--from="+account2Addr, "--gas=1500000", "--fees=2stake")
	RequireTxFailure(t, rsp)

	args := cli.withTXFlags("tx", "wasm", "store", "./testdata/hackatom.wasm.gzip", "--from="+account2Addr, "--generate-only")
	tx, ok := cli.run(args)
	require.True(t, ok)

	pathToTx := filepath.Join(t.TempDir(), "tx.json")
	err := os.WriteFile(pathToTx, []byte(tx), os.FileMode(0o744))
	require.NoError(t, err)

	// address2 authz exec  store code should succeed
	rsp = cli.CustomCommand("tx", "authz", "exec", pathToTx, "--from="+account2Addr, "--gas=1500000", "--fees=2stake")
	RequireTxSuccess(t, rsp)
}
