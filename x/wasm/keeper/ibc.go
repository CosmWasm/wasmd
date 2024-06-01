package keeper

import (
	"context"
	"encoding/hex"
	"strings"

	capabilitytypes "github.com/cosmos/ibc-go/modules/capability/types"
	host "github.com/cosmos/ibc-go/v8/modules/core/24-host"

	errorsmod "cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/CosmWasm/wasmd/x/wasm/types"
)

// bindIbcPort will reserve the port.
// returns a string name of the port or error if we cannot bind it.
// this will fail if call twice.
func (k Keeper) bindIbcPort(ctx sdk.Context, portID string) error {
	portCap := k.portKeeper.BindPort(ctx, portID)
	return k.ClaimCapability(ctx, portCap, host.PortPath(portID))
}

// ensureIBCPort is like registerIbcPort, but it checks if we already hold the port
// before calling register, so this is safe to call multiple times.
// Returns success if we already registered or just registered and error if we cannot
// (lack of permissions or someone else has it)
func (k Keeper) ensureIBCPort(ctx sdk.Context, contractAddr sdk.AccAddress) (string, error) {
	portID := k.ibcPortNameGenerator.PortIDForContract(ctx, contractAddr)
	if _, ok := k.capabilityKeeper.GetCapability(ctx, host.PortPath(portID)); ok {
		return portID, nil
	}
	return portID, k.bindIbcPort(ctx, portID)
}

type IBCPortNameGenerator interface {
	// PortIDForContract converts an address into an ibc port-id.
	PortIDForContract(ctx context.Context, addr sdk.AccAddress) string
	// ContractFromPortID returns the contract address for given port-id. The method does not check if the contract exists
	ContractFromPortID(ctx context.Context, portID string) (sdk.AccAddress, error)
}

const portIDPrefix = "wasm."

// DefaultIBCPortNameGenerator uses Bech32 address string in port-id
type DefaultIBCPortNameGenerator struct{}

// PortIDForContract coverts contract into port-id in the format "wasm.<bech32-address>"
func (DefaultIBCPortNameGenerator) PortIDForContract(ctx context.Context, addr sdk.AccAddress) string {
	return portIDPrefix + addr.String()
}

// ContractFromPortID reads the contract address from bech32 address in the port-id.
func (DefaultIBCPortNameGenerator) ContractFromPortID(ctx context.Context, portID string) (sdk.AccAddress, error) {
	if !strings.HasPrefix(portID, portIDPrefix) {
		return nil, errorsmod.Wrapf(types.ErrInvalid, "without prefix")
	}
	return sdk.AccAddressFromBech32(portID[len(portIDPrefix):])
}

// HexIBCPortNameGenerator uses Hex address string
type HexIBCPortNameGenerator struct{}

// PortIDForContract coverts contract into port-id in the format "wasm.<hex-address>"
func (HexIBCPortNameGenerator) PortIDForContract(ctx context.Context, addr sdk.AccAddress) string {
	return portIDPrefix + hex.EncodeToString(addr)
}

// ContractFromPortID reads the contract address from hex address in the port-id.
func (HexIBCPortNameGenerator) ContractFromPortID(ctx context.Context, portID string) (sdk.AccAddress, error) {
	if !strings.HasPrefix(portID, portIDPrefix) {
		return nil, errorsmod.Wrapf(types.ErrInvalid, "without prefix")
	}
	return sdk.AccAddressFromHexUnsafe(portID[len(portIDPrefix):])
}

// AuthenticateCapability wraps the scopedKeeper's AuthenticateCapability function
func (k Keeper) AuthenticateCapability(ctx sdk.Context, cap *capabilitytypes.Capability, name string) bool {
	return k.capabilityKeeper.AuthenticateCapability(ctx, cap, name)
}

// ClaimCapability allows the transfer module to claim a capability
// that IBC module passes to it
func (k Keeper) ClaimCapability(ctx sdk.Context, cap *capabilitytypes.Capability, name string) error {
	return k.capabilityKeeper.ClaimCapability(ctx, cap, name)
}
