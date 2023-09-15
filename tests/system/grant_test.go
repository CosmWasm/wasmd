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

func TestGrantStoreCodePermissionedChain(t *testing.T) {
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

	// query params
	rsp := cli.CustomQuery("q", "wasm", "params")
	permission := gjson.Get(rsp, "code_upload_access.permission").String()
	addrRes := gjson.Get(rsp, "code_upload_access.addresses").Array()
	assert.Equal(t, 1, len(addrRes))

	assert.Equal(t, permission, "AnyOfAddresses")
	assert.Equal(t, account1Addr, addrRes[0].Str)

	// address1 grant upload permission to address2
	rsp = cli.CustomCommand("tx", "wasm", "grant-store-code", account2Addr, "*:*", "--from="+account1Addr)
	RequireTxSuccess(t, rsp)

	// address2 store code fails
	rsp = cli.CustomCommand("tx", "wasm", "store", "./testdata/hackatom.wasm.gzip", "--from="+account2Addr, "--gas=1500000", "--fees=2stake")
	RequireTxFailure(t, rsp)

	// create tx
	args := cli.withTXFlags("tx", "wasm", "store", "./testdata/hackatom.wasm.gzip", "--from="+account1Addr, "--generate-only")
	tx, ok := cli.run(args)
	require.True(t, ok)

	pathToTx := filepath.Join(t.TempDir(), "tx.json")
	err := os.WriteFile(pathToTx, []byte(tx), os.FileMode(0o744))
	require.NoError(t, err)

	// address2 authz exec  store code should succeed
	rsp = cli.CustomCommand("tx", "authz", "exec", pathToTx, "--from="+account2Addr, "--gas=1500000", "--fees=2stake")
	RequireTxSuccess(t, rsp)
}

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

	sut.StartChain(t)

	// address1 grant upload permission to address2
	rsp := cli.CustomCommand("tx", "wasm", "grant-store-code", account2Addr, "*:nobody", "--from="+account1Addr)
	RequireTxSuccess(t, rsp)

	// create tx - permission everybody
	args := cli.withTXFlags("tx", "wasm", "store", "./testdata/hackatom.wasm.gzip", "--instantiate-everybody=true", "--from="+account1Addr, "--generate-only")
	tx, ok := cli.run(args)
	require.True(t, ok)

	pathToTx := filepath.Join(t.TempDir(), "tx.json")
	err := os.WriteFile(pathToTx, []byte(tx), os.FileMode(0o744))
	require.NoError(t, err)

	// address2 authz exec fails because instantiate permissions do not match
	rsp = cli.CustomCommand("tx", "authz", "exec", pathToTx, "--from="+account2Addr, "--gas=1500000", "--fees=2stake")
	RequireTxFailure(t, rsp)

	// create tx - permission nobody
	args = cli.withTXFlags("tx", "wasm", "store", "./testdata/hackatom.wasm.gzip", "--instantiate-nobody=true", "--from="+account1Addr, "--generate-only")
	tx, ok = cli.run(args)
	require.True(t, ok)

	pathToTx = filepath.Join(t.TempDir(), "tx.json")
	err = os.WriteFile(pathToTx, []byte(tx), os.FileMode(0o744))
	require.NoError(t, err)

	// address2 authz exec succeeds
	rsp = cli.CustomCommand("tx", "authz", "exec", pathToTx, "--from="+account2Addr, "--gas=1500000", "--fees=2stake")
	RequireTxSuccess(t, rsp)
}
