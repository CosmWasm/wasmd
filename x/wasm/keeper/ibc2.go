package keeper

import (
	"time"

	wasmvmtypes "github.com/CosmWasm/wasmvm/v2/types"
	channeltypesv2 "github.com/cosmos/ibc-go/v10/modules/core/04-channel/v2/types"
	ibcapi "github.com/cosmos/ibc-go/v10/modules/core/api"

	errorsmod "cosmossdk.io/errors"

	"github.com/cosmos/cosmos-sdk/telemetry"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/CosmWasm/wasmd/x/wasm/types"
)

var _ ibcapi.IBCModule = IBC2Handler{}

type IBC2Handler struct {
	keeper types.IBC2ContractKeeper
}

func NewIBC2Handler(keeper types.IBC2ContractKeeper) IBC2Handler {
	return IBC2Handler{
		keeper: keeper,
	}
}

func (module IBC2Handler) OnSendPacket(
	ctx sdk.Context,
	sourceClient string,
	destinationClient string,
	sequence uint64,
	payload channeltypesv2.Payload,
	signer sdk.AccAddress,
) error {
	return nil
}

func (module IBC2Handler) OnRecvPacket(
	ctx sdk.Context,
	sourceClient string,
	destinationClient string,
	sequence uint64,
	payload channeltypesv2.Payload,
	relayer sdk.AccAddress,
) channeltypesv2.RecvPacketResult {
	contractAddr, err := ContractFromPortID2(payload.DestinationPort)
	if err != nil {
		panic(errorsmod.Wrapf(err, "Invalid contract port id"))
	}

	em := sdk.NewEventManager()
	msg := wasmvmtypes.IBC2PacketReceiveMsg{Payload: newIBC2Payload(payload), Relayer: relayer.String(), SourceClient: sourceClient}

	ack := module.keeper.OnRecvIBC2Packet(ctx.WithEventManager(em), contractAddr, msg)

	if ack.Status == channeltypesv2.PacketStatus_Success {
		// emit all contract and submessage events on success
		ctx.EventManager().EmitEvents(em.Events())
	}
	types.EmitAcknowledgementIBC2Event(ctx, contractAddr, ack, err)

	return ack
}

func (module IBC2Handler) OnTimeoutPacket(
	ctx sdk.Context,
	sourceClient string,
	destinationClient string,
	sequence uint64,
	payload channeltypesv2.Payload,
	relayer sdk.AccAddress,
) error {
	return nil
}

func (module IBC2Handler) OnAcknowledgementPacket(
	ctx sdk.Context,
	sourceClient string,
	destinationClient string,
	sequence uint64,
	acknowledgement []byte,
	payload channeltypesv2.Payload,
	relayer sdk.AccAddress,
) error {
	return nil
}

// The method calls the contract to process the incoming IBC2 packet. The contract fully owns the data processing and
// returns the acknowledgement data for the chain level. This allows custom applications and protocols on top
// of IBC2.
func (k Keeper) OnRecvIBC2Packet(
	ctx sdk.Context,
	contractAddr sdk.AccAddress,
	msg wasmvmtypes.IBC2PacketReceiveMsg,
) channeltypesv2.RecvPacketResult {
	defer telemetry.MeasureSince(time.Now(), "wasm", "contract", "ibc-recv-packet")
	contractInfo, codeInfo, prefixStore, err := k.contractInstance(ctx, contractAddr)
	if err != nil {
		return channeltypesv2.RecvPacketResult{
			Status:          channeltypesv2.PacketStatus_Failure,
			Acknowledgement: []byte(err.Error()),
		}
	}

	env := types.NewEnv(ctx, contractAddr)
	querier := k.newQueryHandler(ctx, contractAddr)

	gasLeft := k.runtimeGasForContract(ctx)
	res, gasUsed, execErr := k.wasmVM.IBC2PacketReceive(codeInfo.CodeHash, env, msg, prefixStore, cosmwasmAPI, querier, ctx.GasMeter(), gasLeft, costJSONDeserialization)
	k.consumeRuntimeGas(ctx, gasUsed)
	if execErr != nil {
		panic(execErr) // let the contract fully abort an IBC packet receive.
		// Throwing a panic here instead of an error ack will revert
		// all state downstream and not persist any data in ibc-go.
		// This can be triggered by throwing a panic in the contract
	}
	if res == nil {
		// If this gets executed, that's a bug in wasmvm
		return channeltypesv2.RecvPacketResult{
			Status:          channeltypesv2.PacketStatus_Failure,
			Acknowledgement: []byte(errorsmod.Wrap(types.ErrVMError, "internal wasmvm error").Error()),
		}
	}
	if res.Err != "" {
		// return error ACK with non-redacted contract message, state will be reverted
		return channeltypesv2.RecvPacketResult{
			Status:          channeltypesv2.PacketStatus_Failure,
			Acknowledgement: []byte(res.Err),
		}
	}

	// note submessage reply results can overwrite the `Acknowledgement` data
	data, err := k.handleContractResponse(ctx, contractAddr, contractInfo.IBCPortID, res.Ok.Messages, res.Ok.Attributes, res.Ok.Acknowledgement, res.Ok.Events)
	if err != nil {
		// submessage errors result in error ACK with state reverted. Error message is redacted
		return channeltypesv2.RecvPacketResult{
			Status:          channeltypesv2.PacketStatus_Failure,
			Acknowledgement: []byte(err.Error()),
		}
	}

	if data == nil {
		// In case of lack of ack, we assume that the packet should
		// be handled asynchronously.
		// TODO: https://github.com/CosmWasm/wasmd/issues/2161
		// err = k.StoreAsyncAckPacket(ctx, convertPacket(msg.Payload))
		// if err != nil {
		// 	return channeltypesv2.RecvPacketResult{
		// 		Status:          channeltypesv2.PacketStatus_Failure,
		// 		Acknowledgement: []byte(err.Error()),
		// 	}
		// }
		return channeltypesv2.RecvPacketResult{
			Status: channeltypesv2.PacketStatus_Async,
		}
	}

	// success ACK, state will be committed
	return channeltypesv2.RecvPacketResult{
		Status:          channeltypesv2.PacketStatus_Success,
		Acknowledgement: data,
	}
}

func newIBC2Payload(payload channeltypesv2.Payload) wasmvmtypes.IBC2Payload {
	return wasmvmtypes.IBC2Payload{
		SourcePort:      payload.SourcePort,
		DestinationPort: payload.DestinationPort,
		Version:         payload.Version,
		Encoding:        payload.Encoding,
		Value:           payload.Value,
	}
}
