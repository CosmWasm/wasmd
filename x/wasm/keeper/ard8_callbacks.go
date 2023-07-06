package keeper

import (
	"time"

	errorsmod "cosmossdk.io/errors"
	wasmvmtypes "github.com/CosmWasm/wasmvm/types"
	"github.com/cosmos/cosmos-sdk/telemetry"
	sdk "github.com/cosmos/cosmos-sdk/types"
	channeltypes "github.com/cosmos/ibc-go/v7/modules/core/04-channel/types"

	"github.com/CosmWasm/wasmd/x/wasm/types"
	"github.com/CosmWasm/wasmd/x/xadr8"
	"github.com/CosmWasm/wasmd/x/xwasmvm"
)

// OnIBCPacketAcked forwards the callback to the contract
func (k Keeper) OnIBCPacketAcked(ctx sdk.Context, packet channeltypes.Packet, acknowledgement []byte, relayer sdk.AccAddress, contractAddr sdk.AccAddress, meta xadr8.CustomDataI) error {
	defer telemetry.MeasureSince(time.Now(), "wasm", "contract", "callback-ibc-ack-packet")
	contractInfo, codeInfo, prefixStore, err := k.contractInstance(ctx, contractAddr)
	if err != nil {
		return err
	}

	env := types.NewEnv(ctx, contractAddr)
	querier := k.newQueryHandler(ctx, contractAddr)

	gas := k.runtimeGasForContract(ctx)
	res, gasUsed, execErr := k.wasmVM.OnIBCPacketAcked(codeInfo.CodeHash, env, xwasmvm.IBCPacketAckedMsg{
		Acknowledgement: wasmvmtypes.IBCAcknowledgement{Data: acknowledgement},
		OriginalPacket:  types.AsIBCPacket(packet),
		Relayer:         relayer.String(),
	}, prefixStore, cosmwasmAPI, querier, ctx.GasMeter(), gas, costJSONDeserialization)
	k.consumeRuntimeGas(ctx, gasUsed)
	if execErr != nil {
		return errorsmod.Wrap(types.ErrExecuteFailed, execErr.Error())
	}
	return k.handleIBCBasicContractResponse(ctx, contractAddr, contractInfo.IBCPortID, res)
}

func (k Keeper) OnIBCPacketTimedOut(ctx sdk.Context, packet channeltypes.Packet, relayer sdk.AccAddress, contractAddr sdk.AccAddress, meta xadr8.CustomDataI) error {
	// TODO implement me
	panic("implement me")
}
