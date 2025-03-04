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

const ibcV2PortIDPrefix = "wasm2."

func IbcV2PortIDForContract(addr sdk.AccAddress) string {
	return ibcV2PortIDPrefix + addr.String()
}

func ContractFromPortID(portID string) (sdk.AccAddress, error) {
	if strings.HasPrefix(portID, ibcV2PortIDPrefix) {
		return sdk.AccAddressFromBech32(portID[len(ibcV2PortIDPrefix):])
	} else if strings.HasPrefix(portID, portIDPrefix) {
		return sdk.AccAddressFromBech32(portID[len(portIDPrefix):])
	}

	return nil, errorsmod.Wrapf(types.ErrInvalid, "without prefix")
}
