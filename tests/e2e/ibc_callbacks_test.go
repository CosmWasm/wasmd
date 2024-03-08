package e2e

import (
	"encoding/json"
	"testing"

	wasmvmtypes "github.com/CosmWasm/wasmvm/v2/types"
	ibcfee "github.com/cosmos/ibc-go/v8/modules/apps/29-fee/types"
	ibctransfertypes "github.com/cosmos/ibc-go/v8/modules/apps/transfer/types"
	channeltypes "github.com/cosmos/ibc-go/v8/modules/core/04-channel/types"
	ibctesting "github.com/cosmos/ibc-go/v8/testing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	sdkmath "cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/CosmWasm/wasmd/app"
	wasmibctesting "github.com/CosmWasm/wasmd/x/wasm/ibctesting"
	"github.com/CosmWasm/wasmd/x/wasm/types"
)

func TestIBCCallbacks(t *testing.T) {
	// scenario:
	// given two chains
	//   with an ics-20 channel established
	//   and an ibc-callbacks contract deployed on chain A
	// when the ibc-callbacks contract sends an IBCMsg::Transfer to chain B
	// then the contract should receive a callback with the result
	marshaler := app.MakeEncodingConfig(t).Codec
	coord := wasmibctesting.NewCoordinator(t, 2)
	chainA := coord.GetChain(wasmibctesting.GetChainID(1))
	chainB := coord.GetChain(wasmibctesting.GetChainID(2))

	actorChainA := sdk.AccAddress(chainA.SenderPrivKey.PubKey().Address())
	actorChainB := sdk.AccAddress(chainB.SenderPrivKey.PubKey().Address())
	oneToken := sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, sdkmath.NewInt(1)))

	path := wasmibctesting.NewPath(chainA, chainB)
	path.EndpointA.ChannelConfig = &ibctesting.ChannelConfig{
		PortID:  ibctransfertypes.PortID,
		Version: string(marshaler.MustMarshalJSON(&ibcfee.Metadata{FeeVersion: ibcfee.Version, AppVersion: ibctransfertypes.Version})),
		Order:   channeltypes.UNORDERED,
	}
	path.EndpointB.ChannelConfig = &ibctesting.ChannelConfig{
		PortID:  ibctransfertypes.PortID,
		Version: string(marshaler.MustMarshalJSON(&ibcfee.Metadata{FeeVersion: ibcfee.Version, AppVersion: ibctransfertypes.Version})),
		Order:   channeltypes.UNORDERED,
	}
	// with an ics-20 transfer channel setup between both chains
	coord.Setup(path)

	// with an ibc-callbacks contract deployed on chain A
	codeID := chainA.StoreCodeFile("./testdata/ibc_callbacks.wasm").CodeID
	contractAddr := chainA.InstantiateContract(codeID, []byte(`{}`))
	require.NotEmpty(t, contractAddr)

	// ExecuteMsg is the ibc-callbacks contract's execute msg
	type ExecuteMsg struct {
		ToAddress      string             `json:"to_address"`
		ChannelID      string             `json:"channel_id"`
		TimeoutSeconds wasmvmtypes.Uint64 `json:"timeout_seconds"`
	}
	contractMsg := ExecuteMsg{
		ToAddress:      actorChainB.String(),
		ChannelID:      path.EndpointA.ChannelID,
		TimeoutSeconds: 100,
	}
	contractMsgBz, err := json.Marshal(contractMsg)
	require.NoError(t, err)

	// when the contract sends an IBCMsg::Transfer to chain B
	execMsg := types.MsgExecuteContract{
		Sender:   actorChainA.String(),
		Contract: contractAddr.String(),
		Msg:      contractMsgBz,
		Funds:    oneToken,
	}
	_, err = chainA.SendMsgs(&execMsg)
	require.NoError(t, err)

	// and the packet is relayed
	require.NoError(t, coord.RelayAndAckPendingPackets(path))

	// then the contract should receive a callback with the ack result
	type QueryMsg struct {
		CallbackStats struct{} `json:"callback_stats"`
	}
	type QueryResp struct {
		IBCAckCallbacks     []wasmvmtypes.IBCPacketAckMsg     `json:"ibc_ack_callbacks"`
		IBCTimeoutCallbacks []wasmvmtypes.IBCPacketTimeoutMsg `json:"ibc_timeout_callbacks"`
		// TODO: receive callback
	}
	var response QueryResp
	chainA.SmartQuery(contractAddr.String(), QueryMsg{CallbackStats: struct{}{}}, &response)
	assert.Len(t, response.IBCAckCallbacks, 1)
	assert.Empty(t, response.IBCTimeoutCallbacks)

	// and the ack result should be the ics20 success ack
	assert.Equal(t, []byte(`{"result":"AQ=="}`), response.IBCAckCallbacks[0].Acknowledgement.Data)

	// now the same, but with a timeout:
	contractMsg.TimeoutSeconds = 1
	contractMsgBz, err = json.Marshal(contractMsg)
	require.NoError(t, err)

	// when the contract sends an IBCMsg::Transfer to chain B
	execMsg.Msg = contractMsgBz
	_, err = chainA.SendMsgs(&execMsg)
	require.NoError(t, err)

	// and the packet times out
	require.NoError(t, coord.TimeoutPendingPackets(path))

	// then the contract should receive a callback with the timeout result
	chainA.SmartQuery(contractAddr.String(), QueryMsg{CallbackStats: struct{}{}}, &response)
	assert.Len(t, response.IBCAckCallbacks, 1)
	assert.Len(t, response.IBCTimeoutCallbacks, 1)
}
