package keeper

import (
	sdk "github.com/line/lbm-sdk/types"
	wasmvm "github.com/line/wasmvm"
	wasmvmtypes "github.com/line/wasmvm/types"

	types "github.com/line/wasmd/x/wasm/types"
)

type cosmwasmAPIImpl struct {
	gasMultiplier GasMultiplier
}

const (
	// DefaultDeserializationCostPerByte The formular should be `len(data) * deserializationCostPerByte`
	DefaultDeserializationCostPerByte = 1
)

var (
	costJSONDeserialization = wasmvmtypes.UFraction{
		Numerator:   DefaultDeserializationCostPerByte * types.DefaultGasMultiplier,
		Denominator: 1,
	}
)

func (a cosmwasmAPIImpl) humanAddress(canon []byte) (string, uint64, error) {
	gas := a.gasMultiplier.FromWasmVMGas(5)
	if err := sdk.VerifyAddressFormat(canon); err != nil {
		return "", gas, err
	}

	return sdk.AccAddress(canon).String(), gas, nil
}

func (a cosmwasmAPIImpl) canonicalAddress(human string) ([]byte, uint64, error) {
	bz, err := sdk.AccAddressFromBech32(human)
	return bz, a.gasMultiplier.ToWasmVMGas(4), err
}

func (k Keeper) cosmwasmAPI(ctx sdk.Context) wasmvm.GoAPI {
	x := cosmwasmAPIImpl{
		gasMultiplier: k.getGasMultiplier(ctx),
	}
	return wasmvm.GoAPI{
		HumanAddress:     x.humanAddress,
		CanonicalAddress: x.canonicalAddress,
	}
}
