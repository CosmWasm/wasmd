package ibctesting

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/golang/protobuf/proto"
	"github.com/stretchr/testify/require"
	abci "github.com/tendermint/tendermint/abci/types"

	"github.com/CosmWasm/wasmd/x/wasm/types"
)

var wasmIdent = []byte("\x00\x61\x73\x6D")

// SeedNewContractInstance stores some wasm code and instantiates a new contract on this chain.
// This method can be called to prepare the store with some valid CodeInfo and ContractInfo. The returned
// Address is the contract address for this instance. Test should make use of this data and/or use NewIBCContractMockWasmer
// for using a contract mock in Go.
func (c *TestChain) SeedNewContractInstance() sdk.AccAddress {
	// No longer likes the randomly created wasm, so we just use a test file now
	pInstResp := c.StoreCodeFile("./testdata/test.wasm")
	codeID := pInstResp.CodeID
	anyAddressStr := c.SenderAccount.GetAddress().String()
	initMsg := []byte(fmt.Sprintf(`{"verifier": %q, "beneficiary": %q}`, anyAddressStr, anyAddressStr))
	return c.InstantiateContract(codeID, initMsg)
}

func (c *TestChain) StoreCodeFile(filename string) types.MsgStoreCodeResponse {
	wasmCode, err := ioutil.ReadFile(filename)
	require.NoError(c.t, err)
	if strings.HasSuffix(filename, "wasm") { // compress for gas limit
		var buf bytes.Buffer
		gz := gzip.NewWriter(&buf)
		_, err := gz.Write(wasmCode)
		require.NoError(c.t, err)
		err = gz.Close()
		require.NoError(c.t, err)
		wasmCode = buf.Bytes()
	}
	return c.StoreCode(wasmCode)
}

func (c *TestChain) StoreCode(byteCode []byte) types.MsgStoreCodeResponse {
	storeMsg := &types.MsgStoreCode{
		Sender:       c.SenderAccount.GetAddress().String(),
		WASMByteCode: byteCode,
	}
	r, err := c.SendMsgs(storeMsg)
	require.NoError(c.t, err)
	protoResult := c.parseSDKResultData(r)
	require.Len(c.t, protoResult.Data, 1)
	// unmarshal protobuf response from data
	var pInstResp types.MsgStoreCodeResponse
	require.NoError(c.t, pInstResp.Unmarshal(protoResult.Data[0].Data))
	require.NotEmpty(c.t, pInstResp.CodeID)
	return pInstResp
}

func (c *TestChain) InstantiateContract(codeID uint64, msg []byte) sdk.AccAddress {
	instantiateMsg := &types.MsgInstantiateContract{
		Sender: c.SenderAccount.GetAddress().String(),
		Admin:  c.SenderAccount.GetAddress().String(),
		CodeID: codeID,
		Label:  "ibc-test",
		Msg:    msg,
		Funds:  sdk.Coins{TestCoin},
	}

	r, err := c.SendMsgs(instantiateMsg)
	require.NoError(c.t, err)
	protoResult := c.parseSDKResultData(r)
	require.Len(c.t, protoResult.Data, 1)

	var pExecResp types.MsgInstantiateContractResponse
	require.NoError(c.t, pExecResp.Unmarshal(protoResult.Data[0].Data))
	a, err := sdk.AccAddressFromBech32(pExecResp.Address)
	require.NoError(c.t, err)
	return a
}

// This will serialize the query message and submit it to the contract.
// The response is parsed into the provided interface.
// Usage: SmartQuery(addr, QueryMsg{Foo: 1}, &response)
func (c *TestChain) SmartQuery(contractAddr string, queryMsg interface{}, response interface{}) error {
	msg, err := json.Marshal(queryMsg)
	if err != nil {
		return err
	}

	req := types.QuerySmartContractStateRequest{
		Address:   contractAddr,
		QueryData: msg,
	}
	reqBin, err := proto.Marshal(&req)
	if err != nil {
		return err
	}

	// TODO: what is the query?
	res := c.App.Query(abci.RequestQuery{
		Path: "/cosmwasm.wasm.v1.Query/SmartContractState",
		Data: reqBin,
	})

	if res.Code != 0 {
		return fmt.Errorf("Query failed: (%d) %s", res.Code, res.Log)
	}

	// unpack protobuf
	var resp types.QuerySmartContractStateResponse
	err = proto.Unmarshal(res.Value, &resp)
	if err != nil {
		return err
	}
	// unpack json content
	return json.Unmarshal(resp.Data, response)
}

func (c *TestChain) parseSDKResultData(r *sdk.Result) sdk.TxMsgData {
	var protoResult sdk.TxMsgData
	require.NoError(c.t, proto.Unmarshal(r.Data, &protoResult))
	return protoResult
}
