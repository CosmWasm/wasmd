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

const portIDPrefixV2 = "wasmV2"

func PortIDForContractV2(addr sdk.AccAddress) string {
	return portIDPrefixV2 + addr.String()
}

func ContractFromPortID2(portID string) (sdk.AccAddress, error) {
	if !strings.HasPrefix(portID, portIDPrefixV2) {
		return nil, errorsmod.Wrapf(types.ErrInvalid, "without prefix")
	}
	return sdk.AccAddressFromBech32(portID[len(portIDPrefixV2):])
}
