package keeper

import (
	"time"

	wasmvmtypes "github.com/CosmWasm/wasmvm/v2/types"
	channeltypes "github.com/cosmos/ibc-go/v10/modules/core/04-channel/types"
	ibcexported "github.com/cosmos/ibc-go/v10/modules/core/exported"

	errorsmod "cosmossdk.io/errors"

	"github.com/cosmos/cosmos-sdk/telemetry"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/CosmWasm/wasmd/x/wasm/types"
)

// The method calls the contract to process the incoming Eureka packet. The contract fully owns the data processing and
// returns the acknowledgement data for the chain level. This allows custom applications and protocols on top
// of IBC Eureka.
func (k Keeper) OnRecvEurekaPacket(
	ctx sdk.Context,
	contractAddr sdk.AccAddress,
	msg wasmvmtypes.EurekaPacketReceiveMsg,
) (ibcexported.Acknowledgement, error) {
	defer telemetry.MeasureSince(time.Now(), "wasm", "contract", "ibc-recv-packet")
	contractInfo, codeInfo, prefixStore, err := k.contractInstance(ctx, contractAddr)
	if err != nil {
		return nil, err
	}

	env := types.NewEnv(ctx, contractAddr)
	querier := k.newQueryHandler(ctx, contractAddr)

	gasLeft := k.runtimeGasForContract(ctx)
	res, gasUsed, execErr := k.wasmVM.EUPacketReceive(codeInfo.CodeHash, env, msg, prefixStore, cosmwasmAPI, querier, ctx.GasMeter(), gasLeft, costJSONDeserialization)
	k.consumeRuntimeGas(ctx, gasUsed)
	if execErr != nil {
		panic(execErr) // let the contract fully abort an IBC packet receive.
		// Throwing a panic here instead of an error ack will revert
		// all state downstream and not persist any data in ibc-go.
		// This can be triggered by throwing a panic in the contract
	}
	if res == nil {
		// If this gets executed, that's a bug in wasmvm
		return nil, errorsmod.Wrap(types.ErrVMError, "internal wasmvm error")
	}
	if res.Err != "" {
		// return error ACK with non-redacted contract message, state will be reverted
		return channeltypes.Acknowledgement{
			Response: &channeltypes.Acknowledgement_Error{Error: res.Err},
		}, nil
	}
	// note submessage reply results can overwrite the `Acknowledgement` data
	data, err := k.handleContractResponse(ctx, contractAddr, contractInfo.IBCPortID, res.Ok.Messages, res.Ok.Attributes, res.Ok.Acknowledgement, res.Ok.Events)
	if err != nil {
		// submessage errors result in error ACK with state reverted. Error message is redacted
		return nil, err
	}

	// TODO tkulik: What about this? Should we support async?
	// if data == nil {
	// 	// Protocol might never write acknowledgement or contract
	// 	// wants async acknowledgements, we don't know.
	// 	// So store the packet for later.
	// 	err = k.StoreAsyncAckPacket(ctx, convertPacket(msg.Packet))
	// 	if err != nil {
	// 		return nil, err
	// 	}
	// 	return nil, nil
	// }

	// success ACK, state will be committed
	return ContractConfirmStateAck(data), nil
}
