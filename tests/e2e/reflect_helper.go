package e2e

import (
	"encoding/json"
	"testing"

	wasmvmtypes "github.com/CosmWasm/wasmvm/types"
	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/cosmos/gogoproto/proto"
	"github.com/stretchr/testify/require"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/CosmWasm/wasmd/x/wasm/ibctesting"
	"github.com/CosmWasm/wasmd/x/wasm/keeper/testdata"
	"github.com/CosmWasm/wasmd/x/wasm/types"
)

// InstantiateReflectContract store and instantiate a reflect contract instance
func InstantiateReflectContract(t *testing.T, chain *ibctesting.TestChain) sdk.AccAddress {
	codeID := chain.StoreCodeFile("../../x/wasm/keeper/testdata/reflect_1_1.wasm").CodeID
	contractAddr := chain.InstantiateContract(codeID, []byte(`{}`))
	require.NotEmpty(t, contractAddr)
	return contractAddr
}

// MustExecViaReflectContract submit execute message to send payload to reflect contract
func MustExecViaReflectContract(t *testing.T, chain *ibctesting.TestChain, contractAddr sdk.AccAddress, msgs ...wasmvmtypes.CosmosMsg) *abci.ExecTxResult {
	rsp, err := ExecViaReflectContract(t, chain, contractAddr, msgs)
	require.NoError(t, err)
	return rsp
}

type sdkMessageType interface {
	proto.Message
	sdk.Msg
}

func MustExecViaStargateReflectContract[T sdkMessageType](t *testing.T, chain *ibctesting.TestChain, contractAddr sdk.AccAddress, msgs ...T) *abci.ExecTxResult {
	vmMsgs := make([]wasmvmtypes.CosmosMsg, len(msgs))
	for i, m := range msgs {
		bz, err := chain.Codec.Marshal(m)
		require.NoError(t, err)
		vmMsgs[i] = wasmvmtypes.CosmosMsg{
			Stargate: &wasmvmtypes.StargateMsg{
				TypeURL: sdk.MsgTypeURL(m),
				Value:   bz,
			},
		}
	}
	rsp, err := ExecViaReflectContract(t, chain, contractAddr, vmMsgs)
	require.NoError(t, err)
	return rsp
}

// ExecViaReflectContract submit execute message to send payload to reflect contract
func ExecViaReflectContract(t *testing.T, chain *ibctesting.TestChain, contractAddr sdk.AccAddress, msgs []wasmvmtypes.CosmosMsg) (*abci.ExecTxResult, error) {
	require.NotEmpty(t, msgs)
	reflectSend := testdata.ReflectHandleMsg{
		Reflect: &testdata.ReflectPayload{Msgs: msgs},
	}
	reflectSendBz, err := json.Marshal(reflectSend)
	require.NoError(t, err)
	execMsg := &types.MsgExecuteContract{
		Sender:   chain.SenderAccount.GetAddress().String(),
		Contract: contractAddr.String(),
		Msg:      reflectSendBz,
	}
	return chain.SendMsgs(execMsg)
}
