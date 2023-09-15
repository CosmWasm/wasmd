//go:build system_test

package system

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func TestGrantStoreCodePermissionedChain(t *testing.T) {
	sut.ResetChain(t)
	cli := NewWasmdCLI(t, sut, verbose)

	chainAuthorizedAccount := cli.AddKey("chain_authorized_account")
	devAccount := cli.AddKey("dev_account")

	//set params
	sut.ModifyGenesisJSON(t, SetCodeUploadPermission(t, "AnyOfAddresses", chainAuthorizedAccount))

	sut.ModifyGenesisCLI(t,
		[]string{"genesis", "add-genesis-account", chainAuthorizedAccount, "100000000stake"},
	)
	sut.ModifyGenesisCLI(t,
		[]string{"genesis", "add-genesis-account", devAccount, "100000000stake"},
	)

	sut.StartChain(t)

	// query params
	rsp := cli.CustomQuery("q", "wasm", "params")
	permission := gjson.Get(rsp, "code_upload_access.permission").String()
	addrRes := gjson.Get(rsp, "code_upload_access.addresses").Array()
	require.Equal(t, 1, len(addrRes))

	require.Equal(t, permission, "AnyOfAddresses")
	require.Equal(t, chainAuthorizedAccount, addrRes[0].Str)

	// chain_authorized_account grant upload permission to dev_account
	rsp = cli.CustomCommand("tx", "wasm", "grant", devAccount, "store-code", "*:*", "--from="+chainAuthorizedAccount)
	RequireTxSuccess(t, rsp)

	// dev_account store code fails as the address is not in the code-upload accept-list
	rsp = cli.CustomCommand("tx", "wasm", "store", "./testdata/hackatom.wasm.gzip", "--from="+devAccount, "--gas=1500000", "--fees=2stake")
	RequireTxFailure(t, rsp)

	// create tx should work for addresses in the accept-list
	args := cli.withTXFlags("tx", "wasm", "store", "./testdata/hackatom.wasm.gzip", "--from="+chainAuthorizedAccount, "--generate-only")
	tx, ok := cli.run(args)
	require.True(t, ok)

	pathToTx := filepath.Join(t.TempDir(), "tx.json")
	err := os.WriteFile(pathToTx, []byte(tx), os.FileMode(0o744))
	require.NoError(t, err)

	// store code via authz execution uses the given grant and should succeed
	rsp = cli.CustomCommand("tx", "authz", "exec", pathToTx, "--from="+devAccount, "--gas=1500000", "--fees=2stake")
	RequireTxSuccess(t, rsp)
}
