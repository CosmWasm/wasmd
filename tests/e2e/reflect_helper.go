package e2e

import (
	"encoding/json"
	"github.com/CosmWasm/wasmd/x/wasm/ibctesting"
	"github.com/CosmWasm/wasmd/x/wasm/keeper/testdata"
	types3 "github.com/CosmWasm/wasmd/x/wasm/types"
	types2 "github.com/CosmWasm/wasmvm/types"
	"github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	"testing"
)

// InstantiateReflectContract store and instantiate a reflect contract instance
func InstantiateReflectContract(t *testing.T, chain *ibctesting.TestChain) types.AccAddress {
	codeID := chain.StoreCodeFile("../../x/wasm/keeper/testdata/reflect_1_1.wasm").CodeID
	contractAddr := chain.InstantiateContract(codeID, []byte(`{}`))
	require.NotEmpty(t, contractAddr)
	return contractAddr
}

// MustExecViaReflectContract submit execute message to send payload to reflect contract
func MustExecViaReflectContract(t *testing.T, chain *ibctesting.TestChain, contractAddr types.AccAddress, msgs ...types2.CosmosMsg) {
	_, err := ExecViaReflectContract(t, chain, contractAddr, msgs)
	require.NoError(t, err)
}

// ExecViaReflectContract submit execute message to send payload to reflect contract
func ExecViaReflectContract(t *testing.T, chain *ibctesting.TestChain, contractAddr types.AccAddress, msgs []types2.CosmosMsg) (*types.Result, error) {
	reflectSend := testdata.ReflectHandleMsg{
		Reflect: &testdata.ReflectPayload{Msgs: msgs},
	}
	reflectSendBz, err := json.Marshal(reflectSend)
	require.NoError(t, err)
	execMsg := &types3.MsgExecuteContract{
		Sender:   chain.SenderAccount.GetAddress().String(),
		Contract: contractAddr.String(),
		Msg:      reflectSendBz,
	}
	return chain.SendMsgs(execMsg)
}
