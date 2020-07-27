package keeper

import (
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"strings"

	wasm "github.com/CosmWasm/go-cosmwasm"
	"github.com/CosmWasm/wasmd/x/wasm/internal/keeper/cosmwasm"
	"github.com/CosmWasm/wasmd/x/wasm/internal/types"
	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	capabilitytypes "github.com/cosmos/cosmos-sdk/x/capability/types"
	channeltypes "github.com/cosmos/cosmos-sdk/x/ibc/04-channel/types"
	host "github.com/cosmos/cosmos-sdk/x/ibc/24-host"
)

// bindIbcPort will reserve the port.
// returns a string name of the port or error if we cannot bind it.
// this will fail if call twice.
func (k Keeper) bindIbcPort(ctx sdk.Context, portID string) error {
	// TODO: always set up IBC in tests, so we don't need to disable this
	if k.PortKeeper == nil {
		return nil
	}
	cap := k.PortKeeper.BindPort(ctx, portID)
	return k.ClaimCapability(ctx, cap, host.PortPath(portID))
}

// ensureIbcPort is like registerIbcPort, but it checks if we already hold the port
// before calling register, so this is safe to call multiple times.
// Returns success if we already registered or just registered and error if we cannot
// (lack of permissions or someone else has it)
func (k Keeper) ensureIbcPort(ctx sdk.Context, codeID, instanceID uint64) (string, error) {
	// TODO: always set up IBC in tests, so we don't need to disable this
	if k.PortKeeper == nil {
		return PortIDForContract(codeID, instanceID), nil
	}

	portID := PortIDForContract(codeID, instanceID)
	if _, ok := k.ScopedKeeper.GetCapability(ctx, host.PortPath(portID)); ok {
		return portID, nil
	}
	return portID, k.bindIbcPort(ctx, portID)
}

const portIDPrefix = "wasm"

func PortIDForContract(codeID, instanceID uint64) string {
	data := make([]byte, binary.MaxVarintLen64)
	contractID := codeID<<32 + instanceID // as in contractAddress
	size := binary.PutUvarint(data, contractID)
	// max total length = 4 + 16
	return portIDPrefix + base64.StdEncoding.WithPadding(base64.NoPadding).EncodeToString(data[0:size]) // encoded to make it readable
}

func ContractFromPortID(portID string) (sdk.AccAddress, error) {
	if !strings.HasPrefix(portID, portIDPrefix) {
		return nil, sdkerrors.Wrapf(types.ErrInvalid, "without prefix")
	}
	data, err := base64.StdEncoding.WithPadding(base64.NoPadding).DecodeString(portID[len(portIDPrefix):])
	if err != nil {
		return nil, sdkerrors.Wrapf(err, "decoding payload data")
	}
	contractID, n := binary.Uvarint(data)
	if n == 0 && n <= 0 {
		return nil, sdkerrors.Wrapf(types.ErrInvalid, "decoding contract id")
	}
	codeID := contractID >> 32
	instanceID := contractID & 0xffffffff
	return contractAddress(codeID, instanceID), nil
}

// ClaimCapability allows the transfer module to claim a capability
//that IBC module passes to it
// TODO: make private and inline??
func (k Keeper) ClaimCapability(ctx sdk.Context, cap *capabilitytypes.Capability, name string) error {
	return k.ScopedKeeper.ClaimCapability(ctx, cap, name)
}

type IBCCallbacks interface {
	// IBC packet lifecycle
	OnReceive(hash []byte, params cosmwasm.Env, msg []byte, store prefix.Store, api wasm.GoAPI, querier QueryHandler, meter sdk.GasMeter, gas uint64) (*cosmwasm.OnReceiveIBCResponse, uint64, error)
	OnAcknowledgement(hash []byte, params cosmwasm.Env, originalData []byte, acknowledgement []byte, store prefix.Store, api wasm.GoAPI, querier QueryHandler, meter sdk.GasMeter, gas uint64) (*cosmwasm.OnAcknowledgeIBCResponse, uint64, error)
	OnTimeout(hash []byte, params cosmwasm.Env, msg []byte, store prefix.Store, api wasm.GoAPI, querier QueryHandler, meter sdk.GasMeter, gas uint64) (*cosmwasm.OnTimeoutIBCResponse, uint64, error)
	// IBC channel livecycle
	AcceptChannel(hash []byte, params cosmwasm.Env, order channeltypes.Order, version string, connectionHops []string, store prefix.Store, api wasm.GoAPI, querier QueryHandler, meter sdk.GasMeter, gas uint64) (*cosmwasm.AcceptChannelResponse, uint64, error)
	//OnConnect(hash []byte, params cosmwasm.Env, msg []byte, store prefix.Store, api wasm.GoAPI, querier QueryHandler, meter sdk.GasMeter, gas uint64) (*cosmwasm.OnTimeoutIBCResponse, uint64, error)
	//OnClose(hash []byte, params cosmwasm.Env, msg []byte, store prefix.Store, api wasm.GoAPI, querier QueryHandler, meter sdk.GasMeter, gas uint64) (*cosmwasm.OnTimeoutIBCResponse, uint64, error)
}

var MockContracts = make(map[string]IBCCallbacks, 0)

func (k Keeper) AcceptChannel(ctx sdk.Context, contractAddr sdk.AccAddress, order channeltypes.Order, version string, connectionHops []string, ibcInfo cosmwasm.IBCInfo) ([]string, error) {
	codeInfo, prefixStore, err := k.contractInstance(ctx, contractAddr)
	if err != nil {
		return nil, err
	}

	var sender sdk.AccAddress // we don't know the sender
	params := cosmwasm.NewEnv(ctx, sender, nil, contractAddr)
	params.IBC = &ibcInfo

	querier := QueryHandler{
		Ctx:     ctx,
		Plugins: k.queryPlugins,
	}

	gas := gasForContract(ctx)
	mock, ok := MockContracts[contractAddr.String()]
	if !ok { // hack for testing without wasmer
		panic("not supported")
	}
	res, gasUsed, execErr := mock.AcceptChannel(codeInfo.CodeHash, params, order, version, connectionHops, prefixStore, cosmwasmAPI, querier, ctx.GasMeter(), gas)
	consumeGas(ctx, gasUsed)
	if execErr != nil {
		return nil, sdkerrors.Wrap(types.ErrExecuteFailed, execErr.Error())
	}
	if !res.Result { // todo: would it make more sense to let the contract return an error instead?
		return nil, sdkerrors.Wrap(types.ErrInvalid, res.Reason)
	}
	return res.RestrictCounterpartyVersions, nil
}

func (k Keeper) OnRecvPacket(ctx sdk.Context, contractAddr sdk.AccAddress, payloadData []byte, ibcInfo cosmwasm.IBCInfo) ([]byte, error) {
	codeInfo, prefixStore, err := k.contractInstance(ctx, contractAddr)
	if err != nil {
		return nil, err
	}

	var sender sdk.AccAddress // we don't know the sender
	params := cosmwasm.NewEnv(ctx, sender, nil, contractAddr)
	params.IBC = &ibcInfo

	querier := QueryHandler{
		Ctx:     ctx,
		Plugins: k.queryPlugins,
	}

	gas := gasForContract(ctx)
	mock, ok := MockContracts[contractAddr.String()]
	if !ok { // hack for testing without wasmer
		panic("not supported")
	}
	res, gasUsed, execErr := mock.OnReceive(codeInfo.CodeHash, params, payloadData, prefixStore, cosmwasmAPI, querier, ctx.GasMeter(), gas)
	consumeGas(ctx, gasUsed)
	if execErr != nil {
		return nil, sdkerrors.Wrap(types.ErrExecuteFailed, execErr.Error())
	}

	// emit all events from this contract itself
	events := types.ParseEvents(res.Log, contractAddr)
	ctx.EventManager().EmitEvents(events)

	// hack: use sdk messages here for simplicity
	for _, m := range res.Messages {
		if err := k.messenger.handleSdkMessage(ctx, contractAddr, m); err != nil {
			return nil, err
		}
	}
	return res.Acknowledgement, nil
}

func (k Keeper) OnAckPacket(ctx sdk.Context, contractAddr sdk.AccAddress, payloadData []byte, acknowledgement []byte, ibcInfo cosmwasm.IBCInfo) error {
	codeInfo, prefixStore, err := k.contractInstance(ctx, contractAddr)
	if err != nil {
		return err
	}

	var sender sdk.AccAddress // we don't know the sender
	params := cosmwasm.NewEnv(ctx, sender, nil, contractAddr)
	params.IBC = &ibcInfo

	querier := QueryHandler{
		Ctx:     ctx,
		Plugins: k.queryPlugins,
	}

	gas := gasForContract(ctx)
	mock, ok := MockContracts[contractAddr.String()] // hack for testing without wasmer
	if !ok {
		panic("not supported")
	}
	res, gasUsed, execErr := mock.OnAcknowledgement(codeInfo.CodeHash, params, payloadData, acknowledgement, prefixStore, cosmwasmAPI, querier, ctx.GasMeter(), gas)
	consumeGas(ctx, gasUsed)
	if execErr != nil {
		return sdkerrors.Wrap(types.ErrExecuteFailed, execErr.Error())
	}

	// emit all events from this contract itself
	events := types.ParseEvents(res.Log, contractAddr)
	ctx.EventManager().EmitEvents(events)

	// hack: use sdk messages here for simplicity
	for _, m := range res.Messages {
		if err := k.messenger.handleSdkMessage(ctx, contractAddr, m); err != nil {
			return err
		}
	}
	return nil
}

func (k Keeper) OnTimeoutPacket(ctx sdk.Context, contractAddr sdk.AccAddress, payloadData []byte, ibcInfo cosmwasm.IBCInfo) error {
	codeInfo, prefixStore, err := k.contractInstance(ctx, contractAddr)
	if err != nil {
		return err
	}

	var sender sdk.AccAddress // we don't know the sender
	params := cosmwasm.NewEnv(ctx, sender, nil, contractAddr)
	params.IBC = &ibcInfo

	querier := QueryHandler{
		Ctx:     ctx,
		Plugins: k.queryPlugins,
	}

	gas := gasForContract(ctx)
	mock, ok := MockContracts[contractAddr.String()]
	if !ok { // hack for testing without wasmer
		panic("not supported")
	}
	res, gasUsed, execErr := mock.OnTimeout(codeInfo.CodeHash, params, payloadData, prefixStore, cosmwasmAPI, querier, ctx.GasMeter(), gas)
	consumeGas(ctx, gasUsed)
	if execErr != nil {
		return sdkerrors.Wrap(types.ErrExecuteFailed, execErr.Error())
	}

	// emit all events from this contract itself
	events := types.ParseEvents(res.Log, contractAddr)
	ctx.EventManager().EmitEvents(events)

	// hack: use sdk messages here for simplicity
	for _, m := range res.Messages {
		if err := k.messenger.handleSdkMessage(ctx, contractAddr, m); err != nil {
			return err
		}
	}
	return nil
}

func (k Keeper) IBCCallFromContract(ctx sdk.Context, sourcePort, sourceChannel string, sender sdk.AccAddress, timeoutHeight, timeoutTimestamp uint64, msg json.RawMessage) error {
	contractInfo := k.GetContractInfo(ctx, sender)
	if contractInfo == nil {
		return sdkerrors.Wrap(sdkerrors.ErrInvalidRequest, "unknown contract")
	}
	if sourcePort != contractInfo.IBCPortID {
		return sdkerrors.Wrap(sdkerrors.ErrUnauthorized, "not sender's port")
	}

	sourceChannelEnd, found := k.ChannelKeeper.GetChannel(ctx, sourcePort, sourceChannel)
	if !found {
		return sdkerrors.Wrap(channeltypes.ErrChannelNotFound, sourceChannel)
	}

	destinationPort := sourceChannelEnd.GetCounterparty().GetPortID()
	destinationChannel := sourceChannelEnd.GetCounterparty().GetChannelID()

	// get the next sequence
	sequence, found := k.ChannelKeeper.GetNextSequenceSend(ctx, sourcePort, sourceChannel)
	if !found {
		return sdkerrors.Wrapf(
			channeltypes.ErrSequenceSendNotFound,
			"source port: %s, source channel: %s", sourcePort, sourceChannel,
		)
	}
	channelCap, ok := k.ScopedKeeper.GetCapability(ctx, host.ChannelCapabilityPath(sourcePort, sourceChannel))
	if !ok {
		return sdkerrors.Wrap(channeltypes.ErrChannelCapabilityNotFound, "module does not own channel capability")
	}

	packet := channeltypes.NewPacket(
		msg,
		sequence,
		sourcePort,
		sourceChannel,
		destinationPort,
		destinationChannel,
		timeoutHeight,
		timeoutTimestamp,
	)
	return k.ChannelKeeper.SendPacket(ctx, channelCap, packet)
}
