package system

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"golang.org/x/exp/slices"

	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"

	"github.com/cosmos/cosmos-sdk/client/rpc"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"

	"github.com/CosmWasm/wasmd/app"
)

type (
	// blocks until next block is minted
	awaitNextBlock func(t *testing.T, timeout ...time.Duration) int64
	// RunErrorAssert is custom type that is satisfies by testify matchers as well
	RunErrorAssert func(t require.TestingT, err error, msgAndArgs ...interface{})
)

// WasmdCli wraps the command line interface
type WasmdCli struct {
	t              *testing.T
	nodeAddress    string
	chainID        string
	homeDir        string
	fees           string
	Debug          bool
	amino          *codec.LegacyAmino
	assertErrorFn  RunErrorAssert
	awaitNextBlock awaitNextBlock
	expTXCommitted bool
}

// NewWasmdCLI constructor
func NewWasmdCLI(t *testing.T, sut *SystemUnderTest, verbose bool) *WasmdCli {
	return NewWasmdCLIx(t, sut.rpcAddr, sut.chainID, sut.AwaitNextBlock, filepath.Join(workDir, sut.outputDir), "1"+sdk.DefaultBondDenom, verbose)
}

// NewWasmdCLIx extended constructor
func NewWasmdCLIx(
	t *testing.T,
	nodeAddress string,
	chainID string,
	awaiter awaitNextBlock,
	homeDir string,
	fees string,
	debug bool,
) *WasmdCli {
	return &WasmdCli{
		t:              t,
		nodeAddress:    nodeAddress,
		chainID:        chainID,
		homeDir:        homeDir,
		Debug:          debug,
		amino:          app.MakeEncodingConfig().Amino,
		assertErrorFn:  require.NoError,
		awaitNextBlock: awaiter,
		fees:           fees,
		expTXCommitted: true,
	}
}

// WithRunErrorsIgnored does not fail on any error
func (c WasmdCli) WithRunErrorsIgnored() WasmdCli {
	return c.WithRunErrorMatcher(func(t require.TestingT, err error, msgAndArgs ...interface{}) {})
}

// WithRunErrorMatcher assert function to ensure run command error value
func (c WasmdCli) WithRunErrorMatcher(f RunErrorAssert) WasmdCli {
	return WasmdCli{
		t:              c.t,
		nodeAddress:    c.nodeAddress,
		chainID:        c.chainID,
		homeDir:        c.homeDir,
		Debug:          c.Debug,
		amino:          c.amino,
		assertErrorFn:  f,
		awaitNextBlock: c.awaitNextBlock,
		fees:           c.fees,
		expTXCommitted: c.expTXCommitted,
	}
}

func (c WasmdCli) WithNodeAddress(addr string) WasmdCli {
	return WasmdCli{
		t:              c.t,
		nodeAddress:    addr,
		chainID:        c.chainID,
		homeDir:        c.homeDir,
		Debug:          c.Debug,
		amino:          c.amino,
		assertErrorFn:  c.assertErrorFn,
		awaitNextBlock: c.awaitNextBlock,
		fees:           c.fees,
		expTXCommitted: c.expTXCommitted,
	}
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
	rsp, committed := c.awaitTxCommitted(c.run(args), defaultWaitTime)
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
	return c.run(args)
}

// CustomQuery main entrypoint for wasmd CLI queries
func (c WasmdCli) CustomQuery(args ...string) string {
	args = c.withQueryFlags(args...)
	return c.run(args)
}

// execute shell command
func (c WasmdCli) run(args []string) string {
	if c.Debug {
		c.t.Logf("+++ running `wasmd %s`", strings.Join(args, " "))
	}
	gotOut, gotErr := func() (out []byte, err error) {
		defer func() {
			if r := recover(); r != nil {
				err = fmt.Errorf("recovered from panic: %v", r)
			}
		}()
		cmd := exec.Command(locateExecutable("wasmd"), args...) //nolint:gosec
		cmd.Dir = workDir
		return cmd.CombinedOutput()
	}()
	c.assertErrorFn(c.t, gotErr, string(gotOut))
	return string(gotOut)
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
	r := append(args, //nolint:gocritic
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
	out := c.run(cmd)
	addr := gjson.Get(out, "address").String()
	require.NotEmpty(c.t, addr, "got %q", out)
	return addr
}

// GetKeyAddr returns address
func (c WasmdCli) GetKeyAddr(name string) string {
	cmd := c.withKeyringFlags("keys", "show", name, "-a")
	out := c.run(cmd)
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
		args = []string{"--from=" + defaultSrcAddr, "--gas=2500000"}
	}
	cmd := append([]string{"tx", "wasm", "store", file}, args...)
	rsp := c.CustomCommand(cmd...)

	RequireTxSuccess(c.t, rsp)
	codeID := gjson.Get(rsp, "logs.#.events.#.attributes.#(key=code_id).value").Array()[0].Array()[0].Int()
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
	addr := gjson.Get(rsp, "logs.#.events.#.attributes.#(key=_contract_address).value").Array()[0].Array()[0].String()
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
	raw := c.CustomQuery("q", "bank", "balances", addr, "--denom="+denom)
	require.Contains(c.t, raw, "amount", raw)
	return gjson.Get(raw, "amount").Int()
}

// QueryTotalSupply returns total amount of tokens for a given denom.
// 0 when not found
func (c WasmdCli) QueryTotalSupply(denom string) int64 {
	raw := c.CustomQuery("q", "bank", "total", "--denom="+denom)
	require.Contains(c.t, raw, "amount", raw)
	return gjson.Get(raw, "amount").Int()
}

func (c WasmdCli) GetTendermintValidatorSet() rpc.ResultValidatorsOutput {
	args := []string{"q", "tendermint-validator-set"}
	got := c.CustomQuery(args...)

	var res rpc.ResultValidatorsOutput
	require.NoError(c.t, c.amino.UnmarshalJSON([]byte(got), &res), got)
	return res
}

// IsInTendermintValset returns true when the giben pub key is in the current active tendermint validator set
func (c WasmdCli) IsInTendermintValset(valPubKey cryptotypes.PubKey) (rpc.ResultValidatorsOutput, bool) {
	valResult := c.GetTendermintValidatorSet()
	var found bool
	for _, v := range valResult.Validators {
		if v.PubKey.Equals(valPubKey) {
			found = true
			break
		}
	}
	return valResult, found
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
	require.True(t, code.Exists(), got)

	details := got
	if log := gjson.Get(got, "raw_log"); log.Exists() {
		details = log.String()
	}
	return code.Int(), details
}

var (
	// ErrOutOfGasMatcher requires error with "out of gas" message
	ErrOutOfGasMatcher RunErrorAssert = func(t require.TestingT, err error, args ...interface{}) {
		const oogMsg = "out of gas"
		expErrWithMsg(t, err, args, oogMsg)
	}
	// ErrTimeoutMatcher requires time out message
	ErrTimeoutMatcher RunErrorAssert = func(t require.TestingT, err error, args ...interface{}) {
		const expMsg = "timed out waiting for tx to be included in a block"
		expErrWithMsg(t, err, args, expMsg)
	}
	// ErrPostFailedMatcher requires post failed
	ErrPostFailedMatcher RunErrorAssert = func(t require.TestingT, err error, args ...interface{}) {
		const expMsg = "post failed"
		expErrWithMsg(t, err, args, expMsg)
	}
	// ErrInvalidQuery requires smart query request failed
	ErrInvalidQuery RunErrorAssert = func(t require.TestingT, err error, args ...interface{}) {
		const expMsg = "query wasm contract failed"
		expErrWithMsg(t, err, args, expMsg)
	}
)

func expErrWithMsg(t require.TestingT, err error, args []interface{}, expMsg string) {
	require.Error(t, err, args)
	var found bool
	for _, v := range args {
		if strings.Contains(fmt.Sprintf("%s", v), expMsg) {
			found = true
			break
		}
	}
	require.True(t, found, "expected %q but got: %s", expMsg, args)
}