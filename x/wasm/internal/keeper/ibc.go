package keeper

import (
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"strings"

	"github.com/CosmWasm/wasmd/x/wasm/internal/types"
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
	if k.portKeeper == nil {
		return nil
	}
	cap := k.portKeeper.BindPort(ctx, portID)
	return k.ClaimCapability(ctx, cap, host.PortPath(portID))
}

// ensureIbcPort is like registerIbcPort, but it checks if we already hold the port
// before calling register, so this is safe to call multiple times.
// Returns success if we already registered or just registered and error if we cannot
// (lack of permissions or someone else has it)
func (k Keeper) ensureIbcPort(ctx sdk.Context, codeID, instanceID uint64) (string, error) {
	// TODO: always set up IBC in tests, so we don't need to disable this
	if k.portKeeper == nil {
		return PortIDForContract(codeID, instanceID), nil
	}

	portID := PortIDForContract(codeID, instanceID)
	if _, ok := k.scopedKeeper.GetCapability(ctx, host.PortPath(portID)); ok {
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
	return k.scopedKeeper.ClaimCapability(ctx, cap, name)
}

func (k Keeper) OnRecvPacket(ctx sdk.Context, contractAddr sdk.AccAddress, data types.WasmIBCContractPacketData) error {
	codeInfo, prefixStore, err := k.contractInstance(ctx, contractAddr)
	if err != nil {
		return err
	}
	params := types.NewEnv(ctx, data.Sender, nil, contractAddr)

	// todo send data to contract
	_, _, _ = codeInfo, prefixStore, params
	return nil
}

func (k Keeper) IBCCallContract(
	ctx sdk.Context,
	sourcePort,
	sourceChannel string,
	sender sdk.AccAddress,
	receiver sdk.AccAddress,
	timeoutHeight,
	timeoutTimestamp uint64,
	msg json.RawMessage,
) error {
	sourceChannelEnd, found := k.channelKeeper.GetChannel(ctx, sourcePort, sourceChannel)
	if !found {
		return sdkerrors.Wrap(channeltypes.ErrChannelNotFound, sourceChannel)
	}

	destinationPort := sourceChannelEnd.GetCounterparty().GetPortID()
	destinationChannel := sourceChannelEnd.GetCounterparty().GetChannelID()

	// get the next sequence
	sequence, found := k.channelKeeper.GetNextSequenceSend(ctx, sourcePort, sourceChannel)
	if !found {
		return sdkerrors.Wrapf(
			channeltypes.ErrSequenceSendNotFound,
			"source port: %s, source channel: %s", sourcePort, sourceChannel,
		)
	}
	channelCap, ok := k.scopedKeeper.GetCapability(ctx, host.ChannelCapabilityPath(sourcePort, sourceChannel))
	if !ok {
		return sdkerrors.Wrap(channeltypes.ErrChannelCapabilityNotFound, "module does not own channel capability")
	}

	packetData := types.WasmIBCContractPacketData{
		Sender:           sender,
		DestContractAddr: receiver,
		Msg:              msg,
	}
	// TODO: using json as payload as in ibc-transfer
	payload := sdk.MustSortJSON(k.cdc.MustMarshalJSON(packetData))

	packet := channeltypes.NewPacket(
		payload,
		sequence,
		sourcePort,
		sourceChannel,
		destinationPort,
		destinationChannel,
		timeoutHeight,
		timeoutTimestamp,
	)

	return k.channelKeeper.SendPacket(ctx, channelCap, packet)
}
