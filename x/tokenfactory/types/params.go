package types

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
)

// Parameter store keys.
var (
	KeyDenomCreationFee        = []byte("DenomCreationFee")
	KeyDenomCreationGasConsume = []byte("DenomCreationGasConsume")
)

// ParamTable for gamm module.
func ParamKeyTable() paramtypes.KeyTable {
	return paramtypes.NewKeyTable().RegisterParamSet(&Params{})
}

func NewParams(denomCreationFee sdk.Coins, denomCreationGasFee uint64) Params {
	return Params{
		DenomCreationFee:        denomCreationFee,
		DenomCreationGasConsume: denomCreationGasFee,
	}
}

func DefaultParams() Params {
	return Params{
		DenomCreationFee:        sdk.NewCoins(sdk.NewInt64Coin(sdk.DefaultBondDenom, 10_000_000)), // 10 TOKEN
		DenomCreationGasConsume: 2_000_000,
	}
}

// validate params.
func (p Params) Validate() error {
	err := validateDenomCreationFee(p.DenomCreationFee)

	return err
}

// Implements params.ParamSet.
func (p *Params) ParamSetPairs() paramtypes.ParamSetPairs {
	return paramtypes.ParamSetPairs{
		paramtypes.NewParamSetPair(KeyDenomCreationFee, &p.DenomCreationFee, validateDenomCreationFee),
		paramtypes.NewParamSetPair(KeyDenomCreationGasConsume, &p.DenomCreationGasConsume, validateDenomCreationFeeGasConsume),
	}
}

func validateDenomCreationFee(i interface{}) error {
	v, ok := i.(sdk.Coins)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}

	if v.Validate() != nil {
		return fmt.Errorf("invalid denom creation fee: %+v", i)
	}

	return nil
}

func validateDenomCreationFeeGasConsume(i interface{}) error {
	_, ok := i.(uint64)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}

	return nil
}
