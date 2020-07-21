package keeper

import (
	"strings"

	"github.com/CosmWasm/wasmd/x/wasm/internal/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	capabilitytypes "github.com/cosmos/cosmos-sdk/x/capability/types"
	host "github.com/cosmos/cosmos-sdk/x/ibc/24-host"
)

// bindIbcPort will reserve the port for the given contract.
// returns a string name of the port or error if we cannot bind it.
// this will fail if call twice.
func (k Keeper) bindIbcPort(ctx sdk.Context, contract sdk.AccAddress) (string, error) {
	portID := portIDForContract(contract)
	cap := k.portKeeper.BindPort(ctx, portID)
	err := k.ClaimCapability(ctx, cap, host.PortPath(portID))
	return portID, err
}

// ensureIbcPort is like registerIbcPort, but it checks if we already hold the port
// before calling register, so this is safe to call multiple times.
// Returns success if we already registered or just registered and error if we cannot
// (lack of permissions or someone else has it)
func (k Keeper) ensureIbcPort(ctx sdk.Context, contract sdk.AccAddress) (string, error) {
	portID := portIDForContract(contract)
	if _, ok := k.scopedKeeper.GetCapability(ctx, host.PortPath(portID)); ok {
		return portID, nil
	}
	return k.bindIbcPort(ctx, contract)
}

const portIDPrefix = "wasm:"

func portIDForContract(contract sdk.AccAddress) string {
	return portIDPrefix + contract.String()
}

func ContractFromPortID(portID string) (sdk.AccAddress, error) {
	if !strings.HasPrefix(portID, portIDPrefix) {
		return nil, sdkerrors.Wrapf(types.ErrInvalid, "without prefix")
	}
	return sdk.AccAddressFromBech32(portID[len(portIDPrefix):])
}

// ClaimCapability allows the transfer module to claim a capability
//that IBC module passes to it
// TODO: make private and inline??
func (k Keeper) ClaimCapability(ctx sdk.Context, cap *capabilitytypes.Capability, name string) error {
	return k.scopedKeeper.ClaimCapability(ctx, cap, name)
}

func (k Keeper) OnRecvPacket(ctx sdk.Context, contractAddr sdk.AccAddress, data types.WasmIBCContractPacketData) error {
	return nil
}
