//go:build system_test && linux

package system

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func TestChainUpgrade(t *testing.T) {
	// Scenario:
	// start a legacy chain with some state
	// when a chain upgrade proposal is executed
	// then the chain upgrades successfully

	legacyBinary := FetchExecutable(t, "v0.41.0")
	t.Logf("+++ legacy binary: %s\n", legacyBinary)
	currentBranchBinary := sut.ExecBinary
	sut.ExecBinary = legacyBinary
	sut.SetupChain()
	votingPeriod := 5 * time.Second // enough time to vote
	sut.ModifyGenesisJSON(t, SetGovVotingPeriod(t, votingPeriod))

	const (
		upgradeHeight int64 = 22
		upgradeName         = "v0.50"
	)

	sut.StartChain(t, fmt.Sprintf("--halt-height=%d", upgradeHeight))

	cli := NewWasmdCLI(t, sut, verbose)

	// set some state to ensure that migrations work
	verifierAddr := cli.AddKey("verifier")
	beneficiary := randomBech32Addr()
	cli.FundAddress(verifierAddr, "1000stake")

	t.Log("Launch hackatom contract")
	codeID := cli.WasmStore("./testdata/hackatom.wasm.gzip")
	initMsg := fmt.Sprintf(`{"verifier":%q, "beneficiary":%q}`, verifierAddr, beneficiary)
	contractAddr := cli.WasmInstantiate(codeID, initMsg, "--admin="+defaultSrcAddr, "--label=label1", "--from="+defaultSrcAddr, "--amount=1000000stake")

	gotRsp := cli.QuerySmart(contractAddr, `{"verifier":{}}`)
	require.Equal(t, fmt.Sprintf(`{"data":{"verifier":"%s"}}`, verifierAddr), gotRsp)

	// submit upgrade proposal
	proposal := fmt.Sprintf(`
{
 "messages": [
  {
   "@type": "/cosmos.upgrade.v1beta1.MsgSoftwareUpgrade",
   "authority": "wasm10d07y265gmmuvt4z0w9aw880jnsr700js7zslc",
   "plan": {
    "name": %q,
    "height": "%d"
   }
  }
 ],
 "metadata": "ipfs://CID",
 "deposit": "100000000stake",
 "title": "my upgrade",
 "summary": "testing"
}`, upgradeName, upgradeHeight)
	proposalID := cli.SubmitAndVoteGovProposal(proposal)
	t.Logf("current_height: %d\n", sut.currentHeight)
	raw := cli.CustomQuery("q", "gov", "proposal", proposalID)
	t.Log(raw)
	sut.AwaitBlockHeight(t, upgradeHeight-1)
	t.Logf("current_height: %d\n", sut.currentHeight)
	raw = cli.CustomQuery("q", "gov", "proposal", proposalID)
	proposalStatus := gjson.Get(raw, "status").String()
	require.Equal(t, "PROPOSAL_STATUS_PASSED", proposalStatus, raw)

	t.Log("waiting for upgrade info")
	sut.AwaitUpgradeInfo(t)
	sut.StopChain()

	t.Log("Upgrade height was reached. Upgrading chain")
	sut.ExecBinary = currentBranchBinary
	sut.StartChain(t)
	cli = NewWasmdCLI(t, sut, verbose)

	// ensure that state matches expectations
	gotRsp = cli.QuerySmart(contractAddr, `{"verifier":{}}`)
	require.Equal(t, fmt.Sprintf(`{"data":{"verifier":"%s"}}`, verifierAddr), gotRsp)
	// and contract execution works as expected
	RequireTxSuccess(t, cli.WasmExecute(contractAddr, `{"release":{}}`, verifierAddr))
	assert.Equal(t, int64(1_000_000), cli.QueryBalance(beneficiary, "stake"))
}

const cacheDir = "binaries"

// FetchExecutable to download and extract tar.gz for linux
func FetchExecutable(t *testing.T, version string) string {
	// use local cache
	cacheFolder := filepath.Join(WorkDir, cacheDir)
	err := os.MkdirAll(cacheFolder, 0o777)
	if err != nil && !os.IsExist(err) {
		panic(err)
	}

	cacheFile := filepath.Join(cacheFolder, fmt.Sprintf("%s_%s", execBinaryName, version))
	if _, err := os.Stat(cacheFile); err == nil {
		return cacheFile
	}
	t.Logf("+++ version not in cache, downloading from github")

	// then download from GH releases: only works with Linux currently as we are not publishing OSX binaries
	const releaseUrl = "https://github.com/CosmWasm/wasmd/releases/download/%s/wasmd-%s-linux-amd64.tar.gz"
	destDir := t.TempDir()
	rsp, err := http.Get(fmt.Sprintf(releaseUrl, version, version))
	require.NoError(t, err)
	defer rsp.Body.Close()
	gzr, err := gzip.NewReader(rsp.Body)
	require.NoError(t, err)
	defer gzr.Close()
	tr := tar.NewReader(gzr)

	var workFileName string
	for {
		header, err := tr.Next()
		switch {
		case err == io.EOF:
			require.NotEmpty(t, workFileName)
			require.NoError(t, os.Rename(workFileName, cacheFile))
			return cacheFile
		case err != nil:
			require.NoError(t, err)
		case header == nil:
			continue
		}
		workFileName = filepath.Join(destDir, header.Name)
		switch header.Typeflag {
		case tar.TypeDir:
			t.Fatalf("unexpected type")
		case tar.TypeReg:
			f, err := os.OpenFile(workFileName, os.O_CREATE|os.O_RDWR, os.FileMode(0o755))
			require.NoError(t, err)
			_, err = io.Copy(f, tr)
			require.NoError(t, err)
			_ = f.Close()
		}
	}
}
