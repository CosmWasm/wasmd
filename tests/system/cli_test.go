//go:build system_test

package system

import (
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
