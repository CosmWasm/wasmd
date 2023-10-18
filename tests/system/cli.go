package system

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
	"golang.org/x/exp/slices"

	"github.com/cosmos/cosmos-sdk/client/grpc/cmtservice"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	"github.com/cosmos/cosmos-sdk/std"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

type (
	// blocks until next block is minted
	awaitNextBlock func(t *testing.T, timeout ...time.Duration) int64
	// RunErrorAssert is custom type that is satisfies by testify matchers as well
	RunErrorAssert func(t assert.TestingT, err error, msgAndArgs ...interface{}) (ok bool)
)

// WasmdCli wraps the command line interface
type WasmdCli struct {
	t              *testing.T
	nodeAddress    string
	chainID        string
	homeDir        string
	fees           string
	Debug          bool
	assertErrorFn  RunErrorAssert
	awaitNextBlock awaitNextBlock
	expTXCommitted bool
	execBinary     string
	nodesCount     int
}

// NewWasmdCLI constructor
func NewWasmdCLI(t *testing.T, sut *SystemUnderTest, verbose bool) *WasmdCli {
	return NewWasmdCLIx(
		t,
		sut.ExecBinary,
		sut.rpcAddr,
		sut.chainID,
		sut.AwaitNextBlock,
		sut.nodesCount,
		filepath.Join(WorkDir, sut.outputDir),
		"1"+sdk.DefaultBondDenom,
		verbose,
		assert.NoError,
		true,
	)
}

// NewWasmdCLIx extended constructor
func NewWasmdCLIx(
	t *testing.T,
	execBinary string,
	nodeAddress string,
	chainID string,
	awaiter awaitNextBlock,
	nodesCount int,
	homeDir string,
	fees string,
	debug bool,
	assertErrorFn RunErrorAssert,
	expTXCommitted bool,
) *WasmdCli {
	if strings.TrimSpace(execBinary) == "" {
		panic("executable binary name must not be empty")
	}
	return &WasmdCli{
		t:              t,
		execBinary:     execBinary,
		nodeAddress:    nodeAddress,
		chainID:        chainID,
		homeDir:        homeDir,
		Debug:          debug,
		awaitNextBlock: awaiter,
		nodesCount:     nodesCount,
		fees:           fees,
		assertErrorFn:  assertErrorFn,
		expTXCommitted: expTXCommitted,
	}
}

// WithRunErrorsIgnored does not fail on any error
func (c WasmdCli) WithRunErrorsIgnored() WasmdCli {
	return c.WithRunErrorMatcher(func(t assert.TestingT, err error, msgAndArgs ...interface{}) bool {
		return true
	})
}

// WithRunErrorMatcher assert function to ensure run command error value
func (c WasmdCli) WithRunErrorMatcher(f RunErrorAssert) WasmdCli {
	return *NewWasmdCLIx(
		c.t,
		c.execBinary,
		c.nodeAddress,
		c.chainID,
		c.awaitNextBlock,
		c.nodesCount,
		c.homeDir,
		c.fees,
		c.Debug,
		f,
		c.expTXCommitted,
	)
}

func (c WasmdCli) WithNodeAddress(nodeAddr string) WasmdCli {
	return *NewWasmdCLIx(
		c.t,
		c.execBinary,
		nodeAddr,
		c.chainID,
		c.awaitNextBlock,
		c.nodesCount,
		c.homeDir,
		c.fees,
		c.Debug,
		c.assertErrorFn,
		c.expTXCommitted,
	)
}

func (c WasmdCli) WithAssertTXUncommitted() WasmdCli {
	return *NewWasmdCLIx(
		c.t,
		c.execBinary,
		c.nodeAddress,
		c.chainID,
		c.awaitNextBlock,
		c.nodesCount,
		c.homeDir,
		c.fees,
		c.Debug,
		c.assertErrorFn,
		false,
	)
}

// CustomCommand main entry for executing wasmd cli commands.
// When configured, method blocks until tx is committed.
func (c WasmdCli) CustomCommand(args ...string) string {
	if c.fees != "" && !slices.ContainsFunc(args, func(s string) bool {
		return strings.HasPrefix(s, "--fees")
	}) {
		args = append(args, "--fees="+c.fees) // add default fee
	}
	args = c.withTXFlags(args...)
	execOutput, ok := c.run(args)
	if !ok {
		return execOutput
	}
	rsp, committed := c.awaitTxCommitted(execOutput, DefaultWaitTime)
	c.t.Logf("tx committed: %v", committed)
	require.Equal(c.t, c.expTXCommitted, committed, "expected tx committed: %v", c.expTXCommitted)
	return rsp
}

// wait for tx committed on chain
func (c WasmdCli) awaitTxCommitted(submitResp string, timeout ...time.Duration) (string, bool) {
	RequireTxSuccess(c.t, submitResp)
	txHash := gjson.Get(submitResp, "txhash")
	require.True(c.t, txHash.Exists())
	var txResult string
	for i := 0; i < 3; i++ { // max blocks to wait for a commit
		txResult = c.WithRunErrorsIgnored().CustomQuery("q", "tx", txHash.String())
		if code := gjson.Get(txResult, "code"); code.Exists() {
			if code.Int() != 0 { // 0 = success code
				c.t.Logf("+++ got error response code: %s\n", txResult)
			}
			return txResult, true
		}
		c.awaitNextBlock(c.t, timeout...)
	}
	return "", false
}

// Keys wasmd keys CLI command
func (c WasmdCli) Keys(args ...string) string {
	args = c.withKeyringFlags(args...)
	out, _ := c.run(args)
	return out
}

// CustomQuery main entrypoint for wasmd CLI queries
func (c WasmdCli) CustomQuery(args ...string) string {
	args = c.withQueryFlags(args...)
	out, _ := c.run(args)
	return out
}

// execute shell command
func (c WasmdCli) run(args []string) (output string, ok bool) {
	return c.runWithInput(args, nil)
}

func (c WasmdCli) runWithInput(args []string, input io.Reader) (output string, ok bool) {
	if c.Debug {
		c.t.Logf("+++ running `%s %s`", c.execBinary, strings.Join(args, " "))
	}
	gotOut, gotErr := func() (out []byte, err error) {
		defer func() {
			if r := recover(); r != nil {
				err = fmt.Errorf("recovered from panic: %v", r)
			}
		}()
		cmd := exec.Command(locateExecutable(c.execBinary), args...) //nolint:gosec
		cmd.Dir = WorkDir
		cmd.Stdin = input
		return cmd.CombinedOutput()
	}()
	ok = c.assertErrorFn(c.t, gotErr, string(gotOut))
	return strings.TrimSpace(string(gotOut)), ok
}

func (c WasmdCli) withQueryFlags(args ...string) []string {
	args = append(args, "--output", "json")
	return c.withChainFlags(args...)
}

func (c WasmdCli) withTXFlags(args ...string) []string {
	args = append(args,
		"--broadcast-mode", "sync",
		"--output", "json",
		"--yes",
		"--chain-id", c.chainID,
	)
	args = c.withKeyringFlags(args...)
	return c.withChainFlags(args...)
}

func (c WasmdCli) withKeyringFlags(args ...string) []string {
	r := append(args,
		"--home", c.homeDir,
		"--keyring-backend", "test",
	)
	for _, v := range args {
		if v == "-a" || v == "--address" { // show address only
			return r
		}
	}
	return append(r, "--output", "json")
}

func (c WasmdCli) withChainFlags(args ...string) []string {
	return append(args,
		"--node", c.nodeAddress,
	)
}

// WasmExecute send MsgExecute to a contract
func (c WasmdCli) WasmExecute(contractAddr, msg, from string, args ...string) string {
	cmd := append([]string{"tx", "wasm", "execute", contractAddr, msg, "--from", from}, args...)
	return c.CustomCommand(cmd...)
}

// AddKey add key to default keyring. Returns address
func (c WasmdCli) AddKey(name string) string {
	cmd := c.withKeyringFlags("keys", "add", name, "--no-backup")
	out, _ := c.run(cmd)
	addr := gjson.Get(out, "address").String()
	require.NotEmpty(c.t, addr, "got %q", out)
	return addr
}

// AddKeyFromSeed recovers the key from given seed and add it to default keyring. Returns address
func (c WasmdCli) AddKeyFromSeed(name, mnemoic string) string {
	cmd := c.withKeyringFlags("keys", "add", name, "--recover")
	out, _ := c.runWithInput(cmd, strings.NewReader(mnemoic))
	addr := gjson.Get(out, "address").String()
	require.NotEmpty(c.t, addr, "got %q", out)
	return addr
}

// GetKeyAddr returns address
func (c WasmdCli) GetKeyAddr(name string) string {
	cmd := c.withKeyringFlags("keys", "show", name, "-a")
	out, _ := c.run(cmd)
	addr := strings.Trim(out, "\n")
	require.NotEmpty(c.t, addr, "got %q", out)
	return addr
}

const defaultSrcAddr = "node0"

// FundAddress sends the token amount to the destination address
func (c WasmdCli) FundAddress(destAddr, amount string) string {
	require.NotEmpty(c.t, destAddr)
	require.NotEmpty(c.t, amount)
	cmd := []string{"tx", "bank", "send", defaultSrcAddr, destAddr, amount}
	rsp := c.CustomCommand(cmd...)
	RequireTxSuccess(c.t, rsp)
	return rsp
}

// WasmStore uploads a wasm contract to the chain. Returns code id
func (c WasmdCli) WasmStore(file string, args ...string) int {
	if len(args) == 0 {
		args = []string{"--from=" + defaultSrcAddr, "--gas=2500000", "--fees=3stake"}
	}
	cmd := append([]string{"tx", "wasm", "store", file}, args...)
	rsp := c.CustomCommand(cmd...)

	RequireTxSuccess(c.t, rsp)
	codeID := gjson.Get(rsp, "events.#.attributes.#(key=code_id).value").Array()[0].Array()[0].Int()
	require.NotEmpty(c.t, codeID)
	return int(codeID)
}

// WasmInstantiate create a new contract instance. returns contract address
func (c WasmdCli) WasmInstantiate(codeID int, initMsg string, args ...string) string {
	if len(args) == 0 {
		args = []string{"--label=testing", "--from=" + defaultSrcAddr, "--no-admin"}
	}
	cmd := append([]string{"tx", "wasm", "instantiate", strconv.Itoa(codeID), initMsg}, args...)
	rsp := c.CustomCommand(cmd...)
	RequireTxSuccess(c.t, rsp)
	addr := gjson.Get(rsp, "events.#.attributes.#(key=_contract_address).value").Array()[0].Array()[0].String()
	require.NotEmpty(c.t, addr)
	return addr
}

// QuerySmart run smart contract query
func (c WasmdCli) QuerySmart(contractAddr, msg string, args ...string) string {
	cmd := append([]string{"q", "wasm", "contract-state", "smart", contractAddr, msg}, args...)
	return c.CustomQuery(cmd...)
}

// QueryBalances queries all balances for an account. Returns json response
// Example:`{"balances":[{"denom":"node0token","amount":"1000000000"},{"denom":"stake","amount":"400000003"}],"pagination":{}}`
func (c WasmdCli) QueryBalances(addr string) string {
	return c.CustomQuery("q", "bank", "balances", addr)
}

// QueryBalance returns balance amount for given denom.
// 0 when not found
func (c WasmdCli) QueryBalance(addr, denom string) int64 {
	raw := c.CustomQuery("q", "bank", "balance", addr, denom)
	require.Contains(c.t, raw, "amount", raw)
	return gjson.Get(raw, "balance.amount").Int()
}

// QueryTotalSupply returns total amount of tokens for a given denom.
// 0 when not found
func (c WasmdCli) QueryTotalSupply(denom string) int64 {
	raw := c.CustomQuery("q", "bank", "total-supply")
	require.Contains(c.t, raw, "amount", raw)
	return gjson.Get(raw, fmt.Sprintf("supply.#(denom==%q).amount", denom)).Int()
}

func (c WasmdCli) GetCometBFTValidatorSet() cmtservice.GetLatestValidatorSetResponse {
	args := []string{"q", "comet-validator-set"}
	got := c.CustomQuery(args...)

	// still using amino here as the SDK
	amino := codec.NewLegacyAmino()
	std.RegisterLegacyAminoCodec(amino)
	std.RegisterInterfaces(codectypes.NewInterfaceRegistry())

	var res cmtservice.GetLatestValidatorSetResponse
	require.NoError(c.t, amino.UnmarshalJSON([]byte(got), &res), got)
	return res
}

// IsInCometBftValset returns true when the given pub key is in the current active tendermint validator set
func (c WasmdCli) IsInCometBftValset(valPubKey cryptotypes.PubKey) (cmtservice.GetLatestValidatorSetResponse, bool) {
	valResult := c.GetCometBFTValidatorSet()
	var found bool
	for _, v := range valResult.Validators {
		if v.PubKey.Equal(valPubKey) {
			found = true
			break
		}
	}
	return valResult, found
}

// SubmitGovProposal submit a gov v1 proposal
func (c WasmdCli) SubmitGovProposal(proposalJson string, args ...string) string {
	if len(args) == 0 {
		args = []string{"--from=" + defaultSrcAddr}
	}

	pathToProposal := filepath.Join(c.t.TempDir(), "proposal.json")
	err := os.WriteFile(pathToProposal, []byte(proposalJson), os.FileMode(0o744))
	require.NoError(c.t, err)
	c.t.Log("Submit upgrade proposal")
	return c.CustomCommand(append([]string{"tx", "gov", "submit-proposal", pathToProposal}, args...)...)
}

// SubmitAndVoteGovProposal submit proposal, let all validators vote yes and return proposal id
func (c WasmdCli) SubmitAndVoteGovProposal(proposalJson string, args ...string) string {
	rsp := c.SubmitGovProposal(proposalJson, args...)
	RequireTxSuccess(c.t, rsp)
	raw := c.CustomQuery("q", "gov", "proposals", "--depositor", c.GetKeyAddr(defaultSrcAddr))
	proposals := gjson.Get(raw, "proposals.#.id").Array()
	require.NotEmpty(c.t, proposals, raw)
	ourProposalID := proposals[len(proposals)-1].String() // last is ours
	for i := 0; i < c.nodesCount; i++ {
		go func(i int) { // do parallel
			c.t.Logf("Voting: validator %d\n", i)
			rsp = c.CustomCommand("tx", "gov", "vote", ourProposalID, "yes", "--from", c.GetKeyAddr(fmt.Sprintf("node%d", i)))
			RequireTxSuccess(c.t, rsp)
		}(i)
	}
	return ourProposalID
}

// Version returns the current version of the client binary
func (c WasmdCli) Version() string {
	v, ok := c.run([]string{"version"})
	require.True(c.t, ok)
	return v
}

// RequireTxSuccess require the received response to contain the success code
func RequireTxSuccess(t *testing.T, got string) {
	t.Helper()
	code, details := parseResultCode(t, got)
	require.Equal(t, int64(0), code, "non success tx code : %s", details)
}

// RequireTxFailure require the received response to contain any failure code and the passed msgsgs
func RequireTxFailure(t *testing.T, got string, containsMsgs ...string) {
	t.Helper()
	code, details := parseResultCode(t, got)
	require.NotEqual(t, int64(0), code, details)
	for _, msg := range containsMsgs {
		require.Contains(t, details, msg)
	}
}

func parseResultCode(t *testing.T, got string) (int64, string) {
	code := gjson.Get(got, "code")
	require.True(t, code.Exists(), "got response: %s", got)

	details := got
	if log := gjson.Get(got, "raw_log"); log.Exists() {
		details = log.String()
	}
	return code.Int(), details
}

var (
	// ErrOutOfGasMatcher requires error with "out of gas" message
	ErrOutOfGasMatcher RunErrorAssert = func(t assert.TestingT, err error, args ...interface{}) bool {
		const oogMsg = "out of gas"
		return expErrWithMsg(t, err, args, oogMsg)
	}
	// ErrTimeoutMatcher requires time out message
	ErrTimeoutMatcher RunErrorAssert = func(t assert.TestingT, err error, args ...interface{}) bool {
		const expMsg = "timed out waiting for tx to be included in a block"
		return expErrWithMsg(t, err, args, expMsg)
	}
	// ErrPostFailedMatcher requires post failed
	ErrPostFailedMatcher RunErrorAssert = func(t assert.TestingT, err error, args ...interface{}) bool {
		const expMsg = "post failed"
		return expErrWithMsg(t, err, args, expMsg)
	}
	// ErrInvalidQuery requires smart query request failed
	ErrInvalidQuery RunErrorAssert = func(t assert.TestingT, err error, args ...interface{}) bool {
		const expMsg = "query wasm contract failed"
		return expErrWithMsg(t, err, args, expMsg)
	}
)

func expErrWithMsg(t assert.TestingT, err error, args []interface{}, expMsg string) bool {
	if ok := assert.Error(t, err, args); !ok {
		return false
	}
	var found bool
	for _, v := range args {
		if strings.Contains(fmt.Sprintf("%s", v), expMsg) {
			found = true
			break
		}
	}
	assert.True(t, found, "expected %q but got: %s", expMsg, args)
	return false // always abort
}
