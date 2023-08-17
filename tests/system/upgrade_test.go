//go:build system_test

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
	targetBinary := sut.ExecBinary
	sut.ExecBinary = legacyBinary
	sut.SetupChain()
	votingPeriod := 15 * time.Second
	sut.ModifyGenesisJSON(t, SetGovVotingPeriod(t, votingPeriod))

	const upgradeHeight int64 = 20 //
	sut.StartChain(t, fmt.Sprintf("--halt-height=%d", upgradeHeight-1))

	cli := NewWasmdCLI(t, sut, verbose)

	// todo: set some state to ensure that migrations work

	// submit upgrade proposal
	// todo: all of this can be moved into the test_cli to make it more readable in the tests
	upgradeName := "my chain upgrade"
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
		t.Logf("Voting: validator %d\n", n)
		rsp = cli.CustomCommand("tx", "gov", "vote", ourProposalID, "yes", "--from", cli.GetKeyAddr(fmt.Sprintf("node%d", n)))
		RequireTxSuccess(t, rsp)
	})
	t.Logf("current_height: %d\n", sut.currentHeight)
	raw = cli.CustomQuery("q", "gov", "proposal", ourProposalID)
	t.Log(raw)

	sut.AwaitChainStopped()
	t.Log("Upgrade height was reached. Upgrading chain")
	sut.ExecBinary = targetBinary
	sut.StartChain(t)
	// todo: ensure that state matches expectations
}

const cacheDir = "binaries"

// FetchExecutable to download and extract tar.gz for linux
func FetchExecutable(t *testing.T, version string) string {
	// use local cache
	cacheFile := filepath.Join(workDir, cacheDir, fmt.Sprintf("%s_%s", execBinaryName, version))
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
