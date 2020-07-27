package lcdtest

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/rest"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
)

func TestWasmStoreCode(t *testing.T) {
	kb, err := newKeybase()
	require.NoError(t, err)
	addr, _, err := CreateAddr(name1, kb)
	require.NoError(t, err)
	cleanup, _, _, port, err := InitializeLCD(1, []types.AccAddress{addr}, true)
	require.NoError(t, err)
	defer cleanup()

	wasmCode, err := ioutil.ReadFile("../x/wasm/internal/keeper/testdata/contract.wasm")
	require.NoError(t, err)

	var (
		chainID       = viper.GetString(flags.FlagChainID)
		from          = addr.String()
		acc           = getAccount(t, port, addr)
		accnum        = acc.GetAccountNumber()
		sequence      = acc.GetSequence()
		gas           = "1200000"
		simulate      = false
		gasAdjustment = 1.0
	)

	baseReq := rest.NewBaseReq(
		from, memo, chainID, gas, fmt.Sprintf("%f", gasAdjustment), accnum, sequence, fees, nil, simulate,
	)
	storeCodeReq := struct {
		BaseReq   rest.BaseReq `json:"base_req" yaml:"base_req"`
		WasmBytes []byte       `json:"wasm_bytes"`
	}{
		BaseReq:   baseReq,
		WasmBytes: wasmCode,
	}

	req, err := cdc.MarshalJSON(storeCodeReq)
	require.NoError(t, err)

	// generate tx
	resp, body := Request(t, port, "POST", "/wasm/code", req)
	require.Equal(t, http.StatusOK, resp.StatusCode, body)

	// sign and broadcast
	resp, body = signAndBroadcastGenTx(t, port, name1, body, acc, gasAdjustment, simulate, kb)
	require.Equal(t, http.StatusOK, resp.StatusCode, body)
	var payload map[string]json.RawMessage
	require.NoError(t, json.Unmarshal([]byte(body), &payload))
	require.Nil(t, payload["code"], body)

	// then check list view
	resp, body = Request(t, port, "GET", "/wasm/code", nil)
	require.Equal(t, http.StatusOK, resp.StatusCode, body)
	var listPayload struct {
		Height string
		Result []map[string]interface{}
	}
	require.NoError(t, json.Unmarshal([]byte(body), &listPayload), body)
	require.Len(t, listPayload.Result, 1)

	// and check detail view
	codeID := "1"
	resp, body = Request(t, port, "GET", fmt.Sprintf("/wasm/code/%s", codeID), nil)
	require.Equal(t, http.StatusOK, resp.StatusCode, body)
}
