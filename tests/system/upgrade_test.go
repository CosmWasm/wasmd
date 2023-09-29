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

	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func TestChainUpgrade(t *testing.T) {
	// Scenario:
	// start a legacy chain with some state
	// when a chain upgrade proposal is executed
	// then the chain upgrades successfully

	// todo: this test works only with linux, currently
	legacyBinary := FetchExecutable(t, "v0.41.0")
	t.Logf("+++ legacy binary: %s\n", legacyBinary)
	currentBranchBinary := sut.ExecBinary
	sut.ExecBinary = legacyBinary
	sut.SetupChain()
	votingPeriod := 10 * time.Second // enough time to vote
	sut.ModifyGenesisJSON(t, SetGovVotingPeriod(t, votingPeriod))

	const upgradeHeight int64 = 20 //
	sut.StartChain(t, fmt.Sprintf("--halt-height=%d", upgradeHeight))

	cli := NewWasmdCLI(t, sut, verbose)

	// todo: set some state to ensure that migrations work

	// submit upgrade proposal
	// todo: all of this can be moved into the test_cli to make it more readable in the tests
	upgradeName := "v0.50.x"
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
	pathToProposal := filepath.Join(t.TempDir(), "proposal.json")
	err := os.WriteFile(pathToProposal, []byte(proposal), os.FileMode(0o744))
	require.NoError(t, err)
	t.Log("Submit upgrade proposal")
	rsp := cli.CustomCommand("tx", "gov", "submit-proposal", pathToProposal, "--from", cli.GetKeyAddr(defaultSrcAddr))
	RequireTxSuccess(t, rsp)
	raw := cli.CustomQuery("q", "gov", "proposals", "--depositor", cli.GetKeyAddr(defaultSrcAddr))
	proposals := gjson.Get(raw, "proposals.#.id").Array()
	require.NotEmpty(t, proposals, raw)
	ourProposalID := proposals[len(proposals)-1].String() // last is ours
	sut.withEachNodeHome(func(n int, _ string) {
		go func() { // do parallel
			t.Logf("Voting: validator %d\n", n)
			rsp = cli.CustomCommand("tx", "gov", "vote", ourProposalID, "yes", "--from", cli.GetKeyAddr(fmt.Sprintf("node%d", n)))
			RequireTxSuccess(t, rsp)
		}()
	})
	//
	t.Logf("current_height: %d\n", sut.currentHeight)
	raw = cli.CustomQuery("q", "gov", "proposal", ourProposalID)
	t.Log(raw)
	sut.AwaitBlockHeight(t, upgradeHeight-1)
	t.Logf("current_height: %d\n", sut.currentHeight)
	raw = cli.CustomQuery("q", "gov", "proposal", ourProposalID)
	t.Log(raw)

	t.Log("waiting for upgrade info")
	sut.AwaitUpgradeInfo(t)
	sut.StopChain() // just in case

	t.Log("Upgrade height was reached. Upgrading chain")
	sut.ExecBinary = currentBranchBinary
	sut.StartChain(t)
	// todo: ensure that state matches expectations
	t.Log("Done")
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
