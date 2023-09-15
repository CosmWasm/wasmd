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
	cli := NewWasmdCLI(t, sut, verbose)
	// set params to restrict chain
	const chainAuthorityAddress = "wasm1pvuujjdk0xt043ga0j9nrfh5u8pzj4rpplyqkm"
	sut.ModifyGenesisJSON(t, SetCodeUploadPermission(t, "AnyOfAddresses", chainAuthorityAddress))

	recoveredAddress := cli.AddKeyFromSeed("chain_authority", "aisle ship absurd wedding arch admit fringe foam cluster tide trim aisle salad shiver tackle palm glance wrist valley hamster couch crystal frozen chronic")
	require.Equal(t, chainAuthorityAddress, recoveredAddress)
	devAccount := cli.AddKey("dev_account")

	sut.ModifyGenesisCLI(t,
		[]string{"genesis", "add-genesis-account", chainAuthorityAddress, "100000000stake"},
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
	require.Equal(t, chainAuthorityAddress, addrRes[0].Str)

	// chain_authority grant upload permission to dev_account
	rsp = cli.CustomCommand("tx", "wasm", "grant", "store-code", devAccount, "*:*", "--from="+chainAuthorityAddress)
	RequireTxSuccess(t, rsp)

	// dev_account store code fails as the address is not in the code-upload accept-list
	rsp = cli.CustomCommand("tx", "wasm", "store", "./testdata/hackatom.wasm.gzip", "--from="+devAccount, "--gas=1500000", "--fees=2stake")
	RequireTxFailure(t, rsp)

	// create tx should work for addresses in the accept-list
	args := cli.withTXFlags("tx", "wasm", "store", "./testdata/hackatom.wasm.gzip", "--from="+chainAuthorityAddress, "--generate-only")
	tx, ok := cli.run(args)
	require.True(t, ok)

	pathToTx := filepath.Join(t.TempDir(), "tx.json")
	err := os.WriteFile(pathToTx, []byte(tx), os.FileMode(0o744))
	require.NoError(t, err)

	// store code via authz execution uses the given grant and should succeed
	rsp = cli.CustomCommand("tx", "authz", "exec", pathToTx, "--from="+devAccount, "--gas=1500000", "--fees=2stake")
	RequireTxSuccess(t, rsp)
}
