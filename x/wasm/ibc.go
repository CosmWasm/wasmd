package wasm

import (
	"github.com/CosmWasm/wasmd/x/wasm/internal/keeper/cosmwasm"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	capabilitytypes "github.com/cosmos/cosmos-sdk/x/capability/types"
	"github.com/cosmos/cosmos-sdk/x/ibc/03-connection/types"
	channeltypes "github.com/cosmos/cosmos-sdk/x/ibc/04-channel/types"
	host "github.com/cosmos/cosmos-sdk/x/ibc/24-host"
)

type IBCHandler struct {
	keeper Keeper
}

func NewIBCHandler(keeper Keeper) IBCHandler {
	return IBCHandler{keeper: keeper}
}

func (i IBCHandler) OnChanOpenInit(ctx sdk.Context, order channeltypes.Order, connectionHops []string, portID string, channelID string, channelCap *capabilitytypes.Capability, counterParty channeltypes.Counterparty, version string) error {
	// ensure port, version, capability

	contractAddr, err := ContractFromPortID(portID)
	if err != nil {
		return sdkerrors.Wrapf(err, "contract port id")
	}

	_, err = i.keeper.AcceptChannel(ctx, contractAddr, order, version, connectionHops, cosmwasm.IBCInfo{
		PortID:    portID,
		ChannelID: channelID,
	})
	if err != nil {
		return err
	}
	// Claim channel capability passed back by IBC module
	if err := i.keeper.ClaimCapability(ctx, channelCap, host.ChannelCapabilityPath(portID, channelID)); err != nil {
		return sdkerrors.Wrap(channeltypes.ErrChannelCapabilityNotFound, err.Error())
	}
	return nil
}

func (i IBCHandler) OnChanOpenTry(ctx sdk.Context, order channeltypes.Order, connectionHops []string, portID, channelID string, channelCap *capabilitytypes.Capability, counterParty channeltypes.Counterparty, version, counterpartyVersion string) error {
	// ensure port, version, capability
	contractAddr, err := ContractFromPortID(portID)
	if err != nil {
		return sdkerrors.Wrapf(err, "contract port id")
	}

	restrictCounterpartyVersions, err := i.keeper.AcceptChannel(ctx, contractAddr, order, version, connectionHops, cosmwasm.IBCInfo{
		PortID:    portID,
		ChannelID: channelID,
	})
	if err != nil {
		return err
	}
	if len(restrictCounterpartyVersions) != 0 {
		var found bool
		for _, accept := range restrictCounterpartyVersions {
			if accept == counterpartyVersion {
				found = true
				break
			}
		}
		if !found {
			return sdkerrors.Wrapf(types.ErrInvalidCounterparty, "not in supported versions: %q", restrictCounterpartyVersions)
		}
	}

	// Claim channel capability passed back by IBC module
	if err := i.keeper.ClaimCapability(ctx, channelCap, host.ChannelCapabilityPath(portID, channelID)); err != nil {
		return sdkerrors.Wrap(channeltypes.ErrChannelCapabilityNotFound, err.Error())
	}
	return nil
}

func (i IBCHandler) OnChanOpenAck(ctx sdk.Context, portID, channelID string, counterpartyVersion string) error {
	// anything to do? We are not opening channels from wasm contracts
	return nil
}

func (i IBCHandler) OnChanOpenConfirm(ctx sdk.Context, portID, channelID string) error {
	//contractAddr, err := ContractFromPortID(portID)
	//if err != nil {
	//	return sdkerrors.Wrapf(err, "contract port id")
	//}
	//return i.keeper.OnChannelOpen(ctx, contractAddr, cosmwasm.IBCInfo{PortID: portID, ChannelID: channelID})
	// any events to send?
	panic("not implemented")
}

func (i IBCHandler) OnChanCloseInit(ctx sdk.Context, portID, channelID string) error {
	// we can let contracts close channels so we can play back this to the contract
	panic("not implemented")
}

func (i IBCHandler) OnChanCloseConfirm(ctx sdk.Context, portID, channelID string) error {
	// counterparty has closed the channel

	//contractAddr, err := ContractFromPortID(portID)
	//if err != nil {
	//	return sdkerrors.Wrapf(err, "contract port id")
	//}
	//return i.keeper.OnChannelClose(ctx, contractAddr, cosmwasm.IBCInfo{PortID: portID, ChannelID: channelID})
	// any events to send?
	panic("not implemented")
}

func (i IBCHandler) OnRecvPacket(ctx sdk.Context, packet channeltypes.Packet) (*sdk.Result, []byte, error) {
	contractAddr, err := ContractFromPortID(packet.DestinationPort)
	if err != nil {
		return nil, nil, sdkerrors.Wrapf(err, "contract port id")
	}
	msgBz, err := i.keeper.OnRecvPacket(ctx, contractAddr, packet.Data, ibcInfoFromPacket(packet))
	if err != nil {
		return nil, nil, err
	}

	// todo: send proper events
	//ctx.EventManager().EmitEvent(
	//	sdk.NewEvent(
	//		types.EventTypePacket,
	//		sdk.NewAttribute(sdk.AttributeKeyModule, types.ModuleName),
	//		sdk.NewAttribute(types.AttributeKeyReceiver, data.Receiver),
	//		sdk.NewAttribute(types.AttributeKeyValue, data.Amount.String()),
	//	),
	//)

	return &sdk.Result{
		Events: ctx.EventManager().Events().ToABCIEvents(),
	}, msgBz, nil
}

func (i IBCHandler) OnAcknowledgementPacket(ctx sdk.Context, packet channeltypes.Packet, acknowledgement []byte) (*sdk.Result, error) {
	contractAddr, err := ContractFromPortID(packet.DestinationPort)
	if err != nil {
		return nil, sdkerrors.Wrapf(err, "contract port id")
	}

	err = i.keeper.OnAckPacket(ctx, contractAddr, packet.Data, acknowledgement, ibcInfoFromPacket(packet))
	if err != nil {
		return nil, err
	}

	//ctx.EventManager().EmitEvent(
	//	sdk.NewEvent(
	//		types.EventTypePacket,
	//		sdk.NewAttribute(sdk.AttributeKeyModule, types.ModuleName),
	//		sdk.NewAttribute(types.AttributeKeyReceiver, data.Receiver),
	//		sdk.NewAttribute(types.AttributeKeyValue, data.Amount.String()),
	//		sdk.NewAttribute(types.AttributeKeyAckSuccess, fmt.Sprintf("%t", ack.Success)),
	//	),
	//)

	//if !ack.Success {
	//	ctx.EventManager().EmitEvent(
	//		sdk.NewEvent(
	//			types.EventTypePacket,
	//			sdk.NewAttribute(types.AttributeKeyAckError, ack.Error),
	//		),
	//	)
	//}

	return &sdk.Result{
		Events: ctx.EventManager().Events().ToABCIEvents(),
	}, nil

}

func (i IBCHandler) OnTimeoutPacket(ctx sdk.Context, packet channeltypes.Packet) (*sdk.Result, error) {
	contractAddr, err := ContractFromPortID(packet.DestinationPort)
	if err != nil {
		return nil, sdkerrors.Wrapf(err, "contract port id")
	}
	err = i.keeper.OnTimeoutPacket(ctx, contractAddr, packet.Data, ibcInfoFromPacket(packet))
	if err != nil {
		return nil, err
	}

	//ctx.EventManager().EmitEvent(
	//	sdk.NewEvent(
	//		types.EventTypeTimeout,
	//		sdk.NewAttribute(types.AttributeKeyRefundReceiver, data.Sender),
	//		sdk.NewAttribute(types.AttributeKeyRefundValue, data.Amount.String()),
	//		sdk.NewAttribute(sdk.AttributeKeyModule, types.ModuleName),
	//	),
	//)

	return &sdk.Result{
		Events: ctx.EventManager().Events().ToABCIEvents(),
	}, nil

}

func ibcInfoFromPacket(packet channeltypes.Packet) cosmwasm.IBCInfo {
	return cosmwasm.IBCInfo{
		PortID:    packet.DestinationPort,
		ChannelID: packet.DestinationChannel,
		Packet:    cosmwasm.NewIBCPacketInfo(packet.Sequence, packet.SourcePort, packet.SourceChannel, packet.TimeoutHeight, packet.TimeoutTimestamp),
	}
}
