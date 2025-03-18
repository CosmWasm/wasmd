package e2e_test

import (
	"encoding/json"
	"testing"

	ibctransfertypes "github.com/cosmos/ibc-go/v10/modules/apps/transfer/types"
	channeltypes "github.com/cosmos/ibc-go/v10/modules/core/04-channel/types"
	ibctesting "github.com/cosmos/ibc-go/v10/testing"
	mockv2 "github.com/cosmos/ibc-go/v10/testing/mock/v2"
	"github.com/stretchr/testify/require"

	wasmibctesting "github.com/CosmWasm/wasmd/tests/wasmibctesting"
	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
)

// QueryMsg is used to encode query messages to ibc2 contract
type QueryMsg struct {
	QueryState struct{} `json:"query_state"`
}

// ibc2 contract response type
type State struct {
	IBC2PacketReceiveCounter uint32 `json:"ibc2_packet_receive_counter"`
	LastChannelID            string `json:"last_channel_id"`
	LastPacketSeq            uint64 `json:"last_packet_seq"`
}

// Message sent to the ibc2 contract over IBCv2 channel
type IbcPayload struct {
	ResponseWithoutAck     bool `json:"response_without_ack"`
	SendAsyncAckForPrevMsg bool `json:"send_async_ack_for_prev_msg"`
}

func TestIBC2ReceiveEntrypoint(t *testing.T) {
	coord := wasmibctesting.NewCoordinator(t, 2)
	chainA := wasmibctesting.NewWasmTestChain(coord.GetChain(ibctesting.GetChainID(1)))
	chainB := wasmibctesting.NewWasmTestChain(coord.GetChain(ibctesting.GetChainID(2)))

	contractCodeA := chainA.StoreCodeFile("./testdata/ibc2.wasm").CodeID
	contractAddrA := chainA.InstantiateContract(contractCodeA, []byte(`{}`))
	contractPortA := wasmkeeper.PortIDForContractV2(contractAddrA)

	contractCodeB := chainB.StoreCodeFile("./testdata/ibc2.wasm").CodeID
	contractAddrB := chainB.InstantiateContract(contractCodeB, []byte(`{}`))
	contractPortB := wasmkeeper.PortIDForContractV2(contractAddrB)
	require.NotEmpty(t, contractAddrA)

	path := wasmibctesting.NewWasmPath(chainA, chainB)
	path.EndpointA.ChannelConfig = &ibctesting.ChannelConfig{
		PortID:  contractPortA,
		Version: ibctransfertypes.V1,
		Order:   channeltypes.UNORDERED,
	}
	path.EndpointB.ChannelConfig = &ibctesting.ChannelConfig{
		PortID:  contractPortB,
		Version: ibctransfertypes.V1,
		Order:   channeltypes.UNORDERED,
	}

	path.Path.SetupV2()

	var err error
	timeoutTimestamp := chainA.GetTimeoutTimestampSecs()
	payload := mockv2.NewMockPayload(contractPortB, contractPortA)
	payload.Value, err = json.Marshal(IbcPayload{ResponseWithoutAck: true})
	require.NoError(t, err)
	packet, err := path.EndpointB.MsgSendPacket(timeoutTimestamp, payload)
	require.NoError(t, err)
	err = path.EndpointA.MsgRecvPacket(packet)
	require.NoError(t, err)

	var response State
	err = chainA.SmartQuery(contractAddrA.String(), QueryMsg{QueryState: struct{}{}}, &response)
	require.NoError(t, err)
	require.Equal(t, uint32(1), response.IBC2PacketReceiveCounter)

	timeoutTimestamp = chainA.GetTimeoutTimestampSecs()
	payload = mockv2.NewMockPayload(contractPortB, contractPortA)
	payload.Value, err = json.Marshal(IbcPayload{SendAsyncAckForPrevMsg: true})
	require.NoError(t, err)
	packet, err = path.EndpointB.MsgSendPacket(timeoutTimestamp, payload)
	require.NoError(t, err)
	err = path.EndpointA.MsgRecvPacket(packet)
	require.NoError(t, err)

	// TODO tkulik: We need https://github.com/CosmWasm/wasmd/issues/2171 in order to properly test receiving async ACKs
}
