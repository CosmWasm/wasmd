package types

import (
	"encoding/json"
	"fmt"

	"github.com/gogo/protobuf/jsonpb"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"

	sdk "github.com/line/lbm-sdk/types"
	sdkerrors "github.com/line/lbm-sdk/types/errors"
	paramtypes "github.com/line/lbm-sdk/x/params/types"
)

const (
	// DefaultGasMultiplier is how many CosmWasm gas points = 1 Cosmos SDK gas point.
	//
	// CosmWasm gas strategy is documented in https://https://github.com/line/cosmwasm/blob/v1.0.0-0.6.0/docs/GAS.md.
	// LBM SDK reference costs can be found here: https://github.com/line/lbm-sdk/blob/main/store/types/gas.go#L199-L209.
	//
	// The original multiplier of 100 up to CosmWasm 0.16 was based on
	//     "A write at ~3000 gas and ~200us = 10 gas per us (microsecond) cpu/io
	//     Rough timing have 88k gas at 90us, which is equal to 1k sdk gas... (one read)"
	// as well as manual Wasmer benchmarks from 2019. This was then multiplied by 150_000
	// in the 0.16 -> 1.0 upgrade (https://github.com/CosmWasm/cosmwasm/pull/1120).
	//
	// The multiplier deserves more reproducible benchmarking and a strategy that allows easy adjustments.
	// This is tracked in https://github.com/CosmWasm/wasmd/issues/566 and https://github.com/CosmWasm/wasmd/issues/631.
	// Gas adjustments are consensus breaking but may happen in any release marked as consensus breaking.
	// Do not make assumptions on how much gas an operation will consume in places that are hard to adjust,
	// such as hardcoding them in contracts.
	//
	// Please note that all gas prices returned to wasmvm should have this multiplied.
	DefaultGasMultiplier uint64 = 140_000_000
	// DefaultInstanceCost is how much SDK gas we charge each time we load a WASM instance.
	// Creating a new instance is costly, and this helps put a recursion limit to contract calling contracts.
	DefaultInstanceCost = 60_000
	// DefaultCompileCost is how much SDK gas we charge *per byte* for compiling WASM code.
	DefaultCompileCost = 3
)

var ParamStoreKeyUploadAccess = []byte("uploadAccess")
var ParamStoreKeyInstantiateAccess = []byte("instantiateAccess")

// TODO: detach below params because these are  lbm-sdk custom params

var ParamStoreKeyGasMultiplier = []byte("gasMultiplier")
var ParamStoreKeyInstanceCost = []byte("instanceCost")
var ParamStoreKeyCompileCost = []byte("compileCost")

var AllAccessTypes = []AccessType{
	AccessTypeNobody,
	AccessTypeOnlyAddress,
	AccessTypeEverybody,
}

func (a AccessType) With(addr sdk.AccAddress) AccessConfig {
	switch a {
	case AccessTypeNobody:
		return AllowNobody
	case AccessTypeOnlyAddress:
		if err := sdk.VerifyAddressFormat(addr); err != nil {
			panic(err)
		}
		return AccessConfig{Permission: AccessTypeOnlyAddress, Address: addr.String()}
	case AccessTypeEverybody:
		return AllowEverybody
	}
	panic("unsupported access type")
}

func (a AccessType) String() string {
	switch a {
	case AccessTypeNobody:
		return "Nobody"
	case AccessTypeOnlyAddress:
		return "OnlyAddress"
	case AccessTypeEverybody:
		return "Everybody"
	}
	return "Unspecified"
}

func (a *AccessType) UnmarshalText(text []byte) error {
	for _, v := range AllAccessTypes {
		if v.String() == string(text) {
			*a = v
			return nil
		}
	}
	*a = AccessTypeUnspecified
	return nil
}

func (a AccessType) MarshalText() ([]byte, error) {
	return []byte(a.String()), nil
}

func (a *AccessType) MarshalJSONPB(_ *jsonpb.Marshaler) ([]byte, error) {
	return json.Marshal(a)
}

func (a *AccessType) UnmarshalJSONPB(_ *jsonpb.Unmarshaler, data []byte) error {
	return json.Unmarshal(data, a)
}

func (a AccessConfig) Equals(o AccessConfig) bool {
	return a.Permission == o.Permission && a.Address == o.Address
}

var (
	DefaultUploadAccess = AllowEverybody
	AllowEverybody      = AccessConfig{Permission: AccessTypeEverybody}
	AllowNobody         = AccessConfig{Permission: AccessTypeNobody}
)

// ParamKeyTable returns the parameter key table.
func ParamKeyTable() paramtypes.KeyTable {
	return paramtypes.NewKeyTable().RegisterParamSet(&Params{})
}

// DefaultParams returns default wasm parameters
func DefaultParams() Params {
	return Params{
		CodeUploadAccess:             AllowEverybody,
		InstantiateDefaultPermission: AccessTypeEverybody,
		GasMultiplier:                DefaultGasMultiplier,
		InstanceCost:                 DefaultInstanceCost,
		CompileCost:                  DefaultCompileCost,
	}
}

func (p Params) String() string {
	out, err := yaml.Marshal(p)
	if err != nil {
		panic(err)
	}
	return string(out)
}

// ParamSetPairs returns the parameter set pairs.
func (p *Params) ParamSetPairs() paramtypes.ParamSetPairs {
	return paramtypes.ParamSetPairs{
		paramtypes.NewParamSetPair(ParamStoreKeyUploadAccess, &p.CodeUploadAccess, validateAccessConfig),
		paramtypes.NewParamSetPair(ParamStoreKeyInstantiateAccess, &p.InstantiateDefaultPermission, validateAccessType),
		paramtypes.NewParamSetPair(ParamStoreKeyGasMultiplier, &p.GasMultiplier, validateGasMultiplier),
		paramtypes.NewParamSetPair(ParamStoreKeyInstanceCost, &p.InstanceCost, validateInstanceCost),
		paramtypes.NewParamSetPair(ParamStoreKeyCompileCost, &p.CompileCost, validateCompileCost),
	}
}

// ValidateBasic performs basic validation on wasm parameters
func (p Params) ValidateBasic() error {
	if err := validateAccessType(p.InstantiateDefaultPermission); err != nil {
		return errors.Wrap(err, "instantiate default permission")
	}
	if err := validateAccessConfig(p.CodeUploadAccess); err != nil {
		return errors.Wrap(err, "upload access")
	}
	return nil
}

func validateAccessConfig(i interface{}) error {
	v, ok := i.(AccessConfig)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}
	return v.ValidateBasic()
}

func validateAccessType(i interface{}) error {
	a, ok := i.(AccessType)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}
	if a == AccessTypeUnspecified {
		return sdkerrors.Wrap(ErrEmpty, "type")
	}
	for _, v := range AllAccessTypes {
		if v == a {
			return nil
		}
	}
	return sdkerrors.Wrapf(ErrInvalid, "unknown type: %q", a)
}

func (a AccessConfig) ValidateBasic() error {
	switch a.Permission {
	case AccessTypeUnspecified:
		return sdkerrors.Wrap(ErrEmpty, "type")
	case AccessTypeNobody, AccessTypeEverybody:
		if len(a.Address) != 0 {
			return sdkerrors.Wrap(ErrInvalid, "address not allowed for this type")
		}
		return nil
	case AccessTypeOnlyAddress:
		_, err := sdk.AccAddressFromBech32(a.Address)
		return err
	}
	return sdkerrors.Wrapf(ErrInvalid, "unknown type: %q", a.Permission)
}

func validateGasMultiplier(i interface{}) error {
	a, ok := i.(uint64)
	if !ok {
		return sdkerrors.Wrapf(ErrInvalid, "type: %T", i)
	}
	if a == 0 {
		return sdkerrors.Wrap(ErrInvalid, "must be greater than 0")
	}
	return nil
}

func validateInstanceCost(i interface{}) error {
	a, ok := i.(uint64)
	if !ok {
		return sdkerrors.Wrapf(ErrInvalid, "type: %T", i)
	}
	if a == 0 {
		return sdkerrors.Wrap(ErrInvalid, "must be greater than 0")
	}
	return nil
}

func validateCompileCost(i interface{}) error {
	a, ok := i.(uint64)
	if !ok {
		return sdkerrors.Wrapf(ErrInvalid, "type: %T", i)
	}
	if a == 0 {
		return sdkerrors.Wrap(ErrInvalid, "must be greater than 0")
	}
	return nil
}

func (a AccessConfig) Allowed(actor sdk.AccAddress) bool {
	switch a.Permission {
	case AccessTypeNobody:
		return false
	case AccessTypeEverybody:
		return true
	case AccessTypeOnlyAddress:
		return a.Address == actor.String()
	default:
		panic("unknown type")
	}
}
