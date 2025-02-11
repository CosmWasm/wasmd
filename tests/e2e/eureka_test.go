package e2e_test

import (
	"testing"

	sdkmath "cosmossdk.io/math"
	ibcfee "github.com/cosmos/ibc-go/v10/modules/apps/29-fee/types"
	ibctransfertypes "github.com/cosmos/ibc-go/v10/modules/apps/transfer/types"
	channeltypes "github.com/cosmos/ibc-go/v10/modules/core/04-channel/types"
	ibctesting "github.com/cosmos/ibc-go/v10/testing"
	"github.com/stretchr/testify/require"

	"github.com/CosmWasm/wasmd/app"
	"github.com/CosmWasm/wasmd/tests/e2e"
	wasmibctesting "github.com/CosmWasm/wasmd/tests/wasmibctesting"
	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
	mockv2 "github.com/cosmos/ibc-go/v10/testing/mock/v2"
)

func TestEurekaReceiveEntrypoint(t *testing.T) {
	coord := wasmibctesting.NewCoordinator(t, 2)
	chainA := wasmibctesting.NewWasmTestChain(coord.GetChain(ibctesting.GetChainID(1)))
	chainB := wasmibctesting.NewWasmTestChain(coord.GetChain(ibctesting.GetChainID(2)))
	contractAddr := e2e.InstantiateStargateReflectContract(t, chainA)
	chainA.Fund(contractAddr, sdkmath.NewIntFromUint64(1_000_000_000))
	marshaler := app.MakeEncodingConfig(t).Codec

	contractCode := chainA.StoreCodeFile("./testdata/eureka.wasm").CodeID
	contractAddrA := chainA.InstantiateContract(contractCode, []byte(`{}`))
	contractPortA := wasmkeeper.PortIDForContract(contractAddrA)
	contractPortB := "wasm.ChainBContractAddr"

	require.NotEmpty(t, contractAddrA)

	path := wasmibctesting.NewWasmPath(chainA, chainB)
	path.EndpointA.ChannelConfig = &ibctesting.ChannelConfig{
		PortID:  contractPortA,
		Version: string(marshaler.MustMarshalJSON(&ibcfee.Metadata{FeeVersion: ibcfee.Version, AppVersion: ibctransfertypes.V1})),
		Order:   channeltypes.UNORDERED,
	}
	path.EndpointB.ChannelConfig = &ibctesting.ChannelConfig{
		PortID:  contractPortB,
		Version: string(marshaler.MustMarshalJSON(&ibcfee.Metadata{FeeVersion: ibcfee.Version, AppVersion: ibctransfertypes.V1})),
		Order:   channeltypes.UNORDERED,
	}

	path.Path.SetupV2()

	timeoutTimestamp := chainA.GetTimeoutTimestampSecs()

	// TODO tkulik: Not sure about if endpoint A or B should be source
	_, err := path.EndpointA.MsgSendPacket(timeoutTimestamp, mockv2.NewMockPayload(contractPortA, contractPortB))
	require.NoError(t, err)

	// eurekaMsg := &wasmvmtypes.EurekaMsg{
	// 	SendPacket: &wasmvmtypes.EurekaSendPacketMsg{
	// 		Payloads: []wasmvmtypes.EurekaPayload{{
	// 			DestinationPort: "port-1",
	// 			Version:         "v1",
	// 			Encoding:        icatypes.EncodingProto3JSON,
	// 			Value:           []byte{},
	// 		}},
	// 		ChannelID: "channel-1",
	// 		Timeout:   100,
	// 	},
	// }

	// _, err = chain.SendMsgs(&eurekaMsg)
}
