package custom

import (
	"encoding/json"

	"github.com/CosmWasm/wasmd/x/xadr8"

	errorsmod "cosmossdk.io/errors"

	wasmvmtypes "github.com/CosmWasm/wasmvm/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
	"github.com/CosmWasm/wasmd/x/wasm/types"
)

// hack for fast prototyping. should not be here at all
type Adr8Keeper interface {
	RegisterPacketProcessedCallback(
		ctx sdk.Context,
		sender sdk.AccAddress,
		packetID xadr8.PacketId,
		limit uint64,
		meta xadr8.CustomDataI,
	) error
}

func XMessageHandler(k Adr8Keeper) wasmkeeper.MessengerFn {
	if k == nil {
		panic("keeper must not be nil")
	}
	return func(ctx sdk.Context, sender sdk.AccAddress, _ string, srcMsg wasmvmtypes.CosmosMsg) (events []sdk.Event, data [][]byte, err error) {
		if srcMsg.Custom == nil {
			return nil, nil, types.ErrUnknownMsg
		}
		var tmp map[string][]byte // unwrap from reflect msg container
		if err := json.Unmarshal(srcMsg.Custom, &tmp); err != nil {
			return nil, nil, err
		}
		if bz, ok := tmp["raw"]; ok {
			srcMsg.Custom = bz
		}
		println(string(srcMsg.Custom))

		// decode from transport format
		var msg CustomADR8Msg
		if err := json.Unmarshal(srcMsg.Custom, &msg); err != nil {
			return nil, nil, err
		}
		return handleMySpikedMsgs(ctx, sender, msg, k)
	}
}

func handleMySpikedMsgs(ctx sdk.Context, sender sdk.AccAddress, msg CustomADR8Msg, k Adr8Keeper) ([]sdk.Event, [][]byte, error) {
	switch {
	case msg.RegisterPacketProcessedCallback != nil:
		var noCustomMetadata xadr8.CustomDataI
		// todo: convert to ibc-go sdk message type for proper registration
		// I call the xibcgo keeper here as a shortcut only.
		err := k.RegisterPacketProcessedCallback(
			ctx,
			sender,
			xadr8.PacketId{
				PortId:    msg.RegisterPacketProcessedCallback.PortID,
				ChannelId: msg.RegisterPacketProcessedCallback.ChannelID,
				Sequence:  msg.RegisterPacketProcessedCallback.Sequence,
			},
			msg.RegisterPacketProcessedCallback.MaxCallbackGasLimit,
			noCustomMetadata,
		)
		if err != nil {
			return nil, nil, err
		}
		return nil, nil, nil // todo: hack called the keeper already. no msg returned here to end the processing
	default:
		return nil, nil, errorsmod.Wrap(types.ErrUnknownMsg, "unknown variant of IBC")
	}
}
