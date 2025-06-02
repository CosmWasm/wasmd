package keeper

import (
	"time"

	wasmvmtypes "github.com/CosmWasm/wasmvm/v3/types"
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
	contractAddr, err := ContractFromPortID2(payload.SourcePort)
	if err != nil {
		panic(errorsmod.Wrapf(err, "Invalid contract port id"))
	}

	msg := wasmvmtypes.IBC2PacketSendMsg{
		Payload:           newIBC2Payload(payload),
		SourceClient:      sourceClient,
		DestinationClient: destinationClient,
		PacketSequence:    sequence,
		Signer:            signer.String(),
	}

	err = module.keeper.OnSendIBC2Packet(ctx, contractAddr, msg)
	if err != nil {
		return errorsmod.Wrap(err, "on ibc2 send")
	}
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
	msg := wasmvmtypes.IBC2PacketReceiveMsg{Payload: newIBC2Payload(payload), Relayer: relayer.String(), SourceClient: sourceClient, PacketSequence: sequence}

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
	contractAddr, err := ContractFromPortID2(payload.SourcePort)
	if err != nil {
		return errorsmod.Wrapf(err, "contract port id")
	}
	msg := wasmvmtypes.IBC2PacketTimeoutMsg{
		Payload:           newIBC2Payload(payload),
		SourceClient:      sourceClient,
		DestinationClient: destinationClient,
		PacketSequence:    sequence,
		Relayer:           relayer.String(),
	}
	err = module.keeper.OnTimeoutIBC2Packet(ctx, contractAddr, msg)
	if err != nil {
		return errorsmod.Wrap(err, "on timeout")
	}
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
	contractAddr, err := ContractFromPortID2(payload.SourcePort)
	if err != nil {
		return errorsmod.Wrapf(err, "contract port id")
	}
	msg := wasmvmtypes.IBC2AcknowledgeMsg{
		SourceClient:      sourceClient,
		DestinationClient: destinationClient,
		Data:              newIBC2Payload(payload),
		Acknowledgement:   acknowledgement,
		Relayer:           relayer.String(),
	}
	err = module.keeper.OnAckIBC2Packet(ctx, contractAddr, msg)
	if err != nil {
		return errorsmod.Wrap(err, "on ack")
	}
	return nil
}

func (k Keeper) OnAckIBC2Packet(
	ctx sdk.Context,
	contractAddr sdk.AccAddress,
	msg wasmvmtypes.IBC2AcknowledgeMsg,
) error {
	defer telemetry.MeasureSince(time.Now(), "wasm", "contract", "ibc2-ack-packet")

	contractInfo, codeInfo, prefixStore, err := k.contractInstance(ctx, contractAddr)
	if err != nil {
		return err
	}

	env := types.NewEnv(ctx, k.txHash, contractAddr)
	querier := k.newQueryHandler(ctx, contractAddr)

	gasLeft := k.runtimeGasForContract(ctx)
	res, gasUsed, execErr := k.wasmVM.IBC2PacketAck(codeInfo.CodeHash, env, msg, prefixStore, cosmwasmAPI, querier, ctx.GasMeter(), gasLeft, costJSONDeserialization)
	k.consumeRuntimeGas(ctx, gasUsed)
	if execErr != nil {
		return errorsmod.Wrap(types.ErrExecuteFailed, execErr.Error())
	}
	if res == nil {
		// If this gets executed, that's a bug in wasmvm
		return errorsmod.Wrap(types.ErrVMError, "internal wasmvm error")
	}
	if res.Err != "" {
		return types.MarkErrorDeterministic(errorsmod.Wrap(types.ErrExecuteFailed, res.Err))
	}

	return k.handleIBCBasicContractResponse(ctx, contractAddr, contractInfo.IBC2PortID, res.Ok)
}

// The method calls the contract to process the incoming IBC2 packet. The contract fully owns the data processing and
// returns the acknowledgement data for the chain level. This allows custom applications and protocols on top
// of IBCv2.
func (k Keeper) OnRecvIBC2Packet(
	ctx sdk.Context,
	contractAddr sdk.AccAddress,
	msg wasmvmtypes.IBC2PacketReceiveMsg,
) channeltypesv2.RecvPacketResult {
	defer telemetry.MeasureSince(time.Now(), "wasm", "contract", "ibc2-recv-packet")
	contractInfo, codeInfo, prefixStore, err := k.contractInstance(ctx, contractAddr)
	if err != nil {
		return channeltypesv2.RecvPacketResult{
			Status:          channeltypesv2.PacketStatus_Failure,
			Acknowledgement: []byte(err.Error()),
		}
	}

	env := types.NewEnv(ctx, k.txHash, contractAddr)
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
	data, err := k.handleContractResponse(ctx, contractAddr, contractInfo.IBC2PortID, res.Ok.Messages, res.Ok.Attributes, res.Ok.Acknowledgement, res.Ok.Events)
	if err != nil {
		// submessage errors result in error ACK with state reverted. Error message is redacted
		return channeltypesv2.RecvPacketResult{
			Status:          channeltypesv2.PacketStatus_Failure,
			Acknowledgement: []byte(err.Error()),
		}
	}

	if data == nil {
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

// OnTimeoutIBC2Packet calls the contract to let it know the packet was never received
// on the destination chain within the timeout boundaries.
// The contract should handle this on the application level and undo the original operation
func (k Keeper) OnTimeoutIBC2Packet(
	ctx sdk.Context,
	contractAddr sdk.AccAddress,
	msg wasmvmtypes.IBC2PacketTimeoutMsg,
) error {
	defer telemetry.MeasureSince(time.Now(), "wasm", "contract", "ibc2-timeout-packet")

	contractInfo, codeInfo, prefixStore, err := k.contractInstance(ctx, contractAddr)
	if err != nil {
		return err
	}

	env := types.NewEnv(ctx, k.txHash, contractAddr)
	querier := k.newQueryHandler(ctx, contractAddr)

	gasLeft := k.runtimeGasForContract(ctx)
	res, gasUsed, execErr := k.wasmVM.IBC2PacketTimeout(codeInfo.CodeHash, env, msg, prefixStore, cosmwasmAPI, querier, ctx.GasMeter(), gasLeft, costJSONDeserialization)
	k.consumeRuntimeGas(ctx, gasUsed)
	if execErr != nil {
		return errorsmod.Wrap(types.ErrExecuteFailed, execErr.Error())
	}
	if res == nil {
		// If this gets executed, that's a bug in wasmvm
		return errorsmod.Wrap(types.ErrVMError, "internal wasmvm error")
	}
	if res.Err != "" {
		return types.MarkErrorDeterministic(errorsmod.Wrap(types.ErrExecuteFailed, res.Err))
	}

	return k.handleIBCBasicContractResponse(ctx, contractAddr, contractInfo.IBC2PortID, res.Ok)
}

// OnSendIBC2Packet calls the contract to inform it that the packet was sent from
// the source port assigned to this contract. The contract should handle this at
// the application level and verify the message.
func (k Keeper) OnSendIBC2Packet(
	ctx sdk.Context,
	contractAddr sdk.AccAddress,
	msg wasmvmtypes.IBC2PacketSendMsg,
) error {
	defer telemetry.MeasureSince(time.Now(), "wasm", "contract", "ibc2-send-packet")

	contractInfo, codeInfo, prefixStore, err := k.contractInstance(ctx, contractAddr)
	if err != nil {
		return err
	}

	env := types.NewEnv(ctx, k.txHash, contractAddr)
	querier := k.newQueryHandler(ctx, contractAddr)

	gasLeft := k.runtimeGasForContract(ctx)
	res, gasUsed, execErr := k.wasmVM.IBC2PacketSend(codeInfo.CodeHash, env, msg, prefixStore, cosmwasmAPI, querier, ctx.GasMeter(), gasLeft, costJSONDeserialization)
	k.consumeRuntimeGas(ctx, gasUsed)
	if execErr != nil {
		return errorsmod.Wrap(types.ErrExecuteFailed, execErr.Error())
	}
	if res == nil {
		// If this gets executed, that's a bug in wasmvm
		return errorsmod.Wrap(types.ErrVMError, "internal wasmvm error")
	}
	if res.Err != "" {
		return types.MarkErrorDeterministic(errorsmod.Wrap(types.ErrExecuteFailed, res.Err))
	}

	return k.handleIBCBasicContractResponse(ctx, contractAddr, contractInfo.IBCPortID, res.Ok)
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
