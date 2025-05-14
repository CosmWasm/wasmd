package keeper

import (
	"strings"

	errorsmod "cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/CosmWasm/wasmd/x/wasm/types"
)

const portIDPrefix = "wasm."

func PortIDForContract(addr sdk.AccAddress) string {
	return portIDPrefix + addr.String()
}

func ContractFromPortID(portID string) (sdk.AccAddress, error) {
	if !strings.HasPrefix(portID, portIDPrefix) {
		return nil, errorsmod.Wrapf(types.ErrInvalid, "without prefix")
	}

	return sdk.AccAddressFromBech32(portID[len(portIDPrefix):])
}

// The port prefix refers to "CosmWasm over IBC v2" and ensures packets are routed to the right entry points
const PortIDPrefixV2 = "wasm2"

func PortIDForContractV2(addr sdk.AccAddress) string {
	blockchainPrefix := sdk.GetConfig().GetBech32AccountAddrPrefix()
	return PortIDPrefixV2 + strings.TrimPrefix(addr.String(), blockchainPrefix)
}

func ContractFromPortID2(portID string) (sdk.AccAddress, error) {
	if !strings.HasPrefix(portID, PortIDPrefixV2) {
		return nil, errorsmod.Wrapf(types.ErrInvalid, "without prefix")
	}
	blockchainPrefix := sdk.GetConfig().GetBech32AccountAddrPrefix()
	return sdk.AccAddressFromBech32(blockchainPrefix + portID[len(PortIDPrefixV2):])
}
