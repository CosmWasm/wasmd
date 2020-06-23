package keeper

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
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

func portIDForContract(contract sdk.AccAddress) string {
	return fmt.Sprintf("wasm:%s", contract.String())
}

// ClaimCapability allows the transfer module to claim a capability
//that IBC module passes to it
// TODO: make private and inline??
func (k Keeper) ClaimCapability(ctx sdk.Context, cap *capabilitytypes.Capability, name string) error {
	return k.scopedKeeper.ClaimCapability(ctx, cap, name)
}
