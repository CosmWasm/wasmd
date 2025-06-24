//go:build system_test

package system

import (
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"

	sdkmath "cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func TestUnsafeResetAll(t *testing.T) {
	// scenario:
	// 	given a non-empty wasm dir exists in the node home
	//  when `unsafe-reset-all` is executed
	// 	then the dir and all files in it are removed

	wasmDir := filepath.Join(WorkDir, sut.nodePath(0), "wasm")
	require.NoError(t, os.MkdirAll(wasmDir, os.ModePerm))

	_, err := os.CreateTemp(wasmDir, "testing")
	require.NoError(t, err)

	// when
	sut.ForEachNodeExecAndWait(t, []string{"comet", "unsafe-reset-all"})

	// then
	sut.withEachNodeHome(func(i int, home string) {
		if _, err := os.Stat(wasmDir); !os.IsNotExist(err) {
			t.Fatal("expected wasm dir to be removed")
		}
	})
}

func TestVestingAccounts(t *testing.T) {
	// Scenario:
	//   given: a genesis file
	//   when: add-genesis-account with vesting flags is executed
	//   then: the vesting account data is added to the genesis
	sut.ResetChain(t)
	cli := NewWasmdCLI(t, sut, verbose)
	vest1Addr := cli.AddKey("vesting1")
	vest2Addr := cli.AddKey("vesting2")
	vest3Addr := cli.AddKey("vesting3")
	myStartTimestamp := time.Now().Add(time.Minute).Unix()
	myEndTimestamp := time.Now().Add(time.Hour).Unix()
	sut.ModifyGenesisCLI(t,
		// delayed vesting no cash
		[]string{"genesis", "add-genesis-account", vest1Addr, "100000000stake", "--vesting-amount=100000000stake", fmt.Sprintf("--vesting-end-time=%d", myEndTimestamp)},
		// continuous vesting no cash
		[]string{"genesis", "add-genesis-account", vest2Addr, "100000001stake", "--vesting-amount=100000001stake", fmt.Sprintf("--vesting-start-time=%d", myStartTimestamp), fmt.Sprintf("--vesting-end-time=%d", myEndTimestamp)},
		// continuous vesting with some cash
		[]string{"genesis", "add-genesis-account", vest3Addr, "200000002stake", "--vesting-amount=100000002stake", fmt.Sprintf("--vesting-start-time=%d", myStartTimestamp), fmt.Sprintf("--vesting-end-time=%d", myEndTimestamp)},
	)
	raw := sut.ReadGenesisJSON(t)
	// delayed vesting: without a start time
	accounts := gjson.GetBytes([]byte(raw), `app_state.auth.accounts.#[@type=="/cosmos.vesting.v1beta1.DelayedVestingAccount"]#`).Array()
	require.Len(t, accounts, 1)
	gotAddr := accounts[0].Get("base_vesting_account.base_account.address").String()
	assert.Equal(t, vest1Addr, gotAddr)
	amounts := accounts[0].Get("base_vesting_account.original_vesting").Array()
	require.Len(t, amounts, 1)
	assert.Equal(t, "stake", amounts[0].Get("denom").String())
	assert.Equal(t, "100000000", amounts[0].Get("amount").String())
	assert.Equal(t, myEndTimestamp, accounts[0].Get("base_vesting_account.end_time").Int())
	assert.Equal(t, int64(0), accounts[0].Get("start_time").Int())

	// continuous vesting: start time
	accounts = gjson.GetBytes([]byte(raw), `app_state.auth.accounts.#[@type=="/cosmos.vesting.v1beta1.ContinuousVestingAccount"]#`).Array()
	require.Len(t, accounts, 2)
	gotAddr = accounts[0].Get("base_vesting_account.base_account.address").String()
	assert.Equal(t, vest2Addr, gotAddr)
	amounts = accounts[0].Get("base_vesting_account.original_vesting").Array()
	require.Len(t, amounts, 1)
	assert.Equal(t, "stake", amounts[0].Get("denom").String())
	assert.Equal(t, "100000001", amounts[0].Get("amount").String())
	assert.Equal(t, myEndTimestamp, accounts[0].Get("base_vesting_account.end_time").Int())
	assert.Equal(t, myStartTimestamp, accounts[0].Get("start_time").Int())
	// with some cash
	gotAddr = accounts[1].Get("base_vesting_account.base_account.address").String()
	assert.Equal(t, vest3Addr, gotAddr)
	amounts = accounts[1].Get("base_vesting_account.original_vesting").Array()
	require.Len(t, amounts, 1)
	assert.Equal(t, "stake", amounts[0].Get("denom").String())
	assert.Equal(t, "100000002", amounts[0].Get("amount").String())
	assert.Equal(t, myEndTimestamp, accounts[0].Get("base_vesting_account.end_time").Int())
	assert.Equal(t, myStartTimestamp, accounts[0].Get("start_time").Int())

	// check accounts have some balances
	assert.Equal(t, sdk.NewCoins(sdk.NewCoin("stake", sdkmath.NewInt(100000000))), GetGenesisBalance([]byte(raw), vest1Addr))
	assert.Equal(t, sdk.NewCoins(sdk.NewCoin("stake", sdkmath.NewInt(100000001))), GetGenesisBalance([]byte(raw), vest2Addr))
	assert.Equal(t, sdk.NewCoins(sdk.NewCoin("stake", sdkmath.NewInt(200000002))), GetGenesisBalance([]byte(raw), vest3Addr))
}

func TestParseMsgExecuteContractResponse(t *testing.T) {
	data, err := hex.DecodeString("12FA010A2C2F636F736D7761736D2E7761736D2E76312E4D736745786563757465436F6E7472616374526573706F6E736512C9010AC6017B22626C6F636B223A7B22686569676874223A31352C2274696D65223A2231373438393437343035313834313738303030222C22636861696E5F6964223A2274657374696E67227D2C227472616E73616374696F6E223A7B22696E646578223A302C2268617368223A22227D2C22636F6E7472616374223A7B2261646472657373223A227761736D3134686A32746176713866706573647778786375343472747933686839307668756A7276636D73746C347A723374786D6676773973307068673464227D7D")
	require.NoError(t, err)
	contractResponse, err := ParseMsgExecuteContractResponse(data)
	require.NoError(t, err)
	assert.JSONEq(t, `{"block":{"height":15,"time":"1748947405184178000","chain_id":"testing"},"transaction":{"index":0,"hash":""},"contract":{"address":"wasm14hj2tavq8fpesdwxxcu44rty3hh90vhujrvcmstl4zr3txmfvw9s0phg4d"}}`, string(contractResponse))
}
