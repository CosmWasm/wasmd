package ibc_testing

import (
	"fmt"
	"io/ioutil"
	"strconv"

	"github.com/CosmWasm/wasmd/x/wasm/internal/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/golang/protobuf/proto"
	"github.com/stretchr/testify/require"
)

func (c *TestChain) NewRandomContractInstance() sdk.AccAddress {
	wasmCode, err := ioutil.ReadFile("./internal/keeper/testdata/contract.wasm")
	require.NoError(c.t, err)

	storeMsg := &types.MsgStoreCode{
		Sender:       c.SenderAccount.GetAddress(),
		WASMByteCode: wasmCode,
	}
	r, err := c.SendMsgs(storeMsg)
	require.NoError(c.t, err)
	protoResult := c.parseSDKResultData(r)
	require.Len(c.t, protoResult.Data, 1)

	codeID, err := strconv.ParseUint(string(protoResult.Data[0].Data), 10, 64)
	require.NoError(c.t, err)

	anyAddressStr := c.SenderAccount.GetAddress().String()
	instantiateMsg := &types.MsgInstantiateContract{
		Sender:  c.SenderAccount.GetAddress(),
		Admin:   c.SenderAccount.GetAddress(),
		CodeID:  codeID,
		Label:   "ibc-test",
		InitMsg: []byte(fmt.Sprintf(`{"verifier": %q, "beneficiary": %q}`, anyAddressStr, anyAddressStr)),
	}

	r, err = c.SendMsgs(instantiateMsg)
	require.NoError(c.t, err)
	protoResult = c.parseSDKResultData(r)
	require.Len(c.t, protoResult.Data, 1)
	require.NoError(c.t, sdk.VerifyAddressFormat(protoResult.Data[0].Data))

	return protoResult.Data[0].Data
}

func (c *TestChain) parseSDKResultData(r *sdk.Result) sdk.TxMsgData {
	var protoResult sdk.TxMsgData
	require.NoError(c.t, proto.Unmarshal(r.Data, &protoResult))
	return protoResult
}

func (c *TestChain) ContractInfo(contractAddr sdk.AccAddress) *types.ContractInfo {
	return c.App.WasmKeeper.GetContractInfo(c.GetContext(), contractAddr)
}
