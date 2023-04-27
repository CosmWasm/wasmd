package types

import (
	wasmvmtypes "github.com/CosmWasm/wasmvm/types"
	channeltypes "github.com/cosmos/ibc-go/v7/modules/core/04-channel/types"
)

func AsIBCPacket(packet channeltypes.Packet) wasmvmtypes.IBCPacket {
	timeout := wasmvmtypes.IBCTimeout{
		Timestamp: packet.TimeoutTimestamp,
	}
	if !packet.TimeoutHeight.IsZero() {
		timeout.Block = &wasmvmtypes.IBCTimeoutBlock{
			Height:   packet.TimeoutHeight.RevisionHeight,
			Revision: packet.TimeoutHeight.RevisionNumber,
		}
	}

	return wasmvmtypes.IBCPacket{
		Data:     packet.Data,
		Src:      wasmvmtypes.IBCEndpoint{ChannelID: packet.SourceChannel, PortID: packet.SourcePort},
		Dest:     wasmvmtypes.IBCEndpoint{ChannelID: packet.DestinationChannel, PortID: packet.DestinationPort},
		Sequence: packet.Sequence,
		Timeout:  timeout,
	}
}
