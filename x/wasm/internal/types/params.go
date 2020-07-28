package types

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/cosmos-sdk/x/params"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

const (
	// DefaultParamspace for params keeper
	DefaultParamspace = ModuleName
)

type AccessType string

const (
	Undefined   AccessType = "Undefined"
	Nobody      AccessType = "Nobody"
	OnlyAddress AccessType = "OnlyAddress"
	Everybody   AccessType = "Everybody"
)

var AllAccessTypes = map[AccessType]struct{}{
	Nobody:      {},
	OnlyAddress: {},
	Everybody:   {},
}

func (a AccessType) With(addr sdk.AccAddress) AccessConfig {
	switch a {
	case Nobody:
		return AllowNobody
	case OnlyAddress:
		if err := sdk.VerifyAddressFormat(addr); err != nil {
			panic(err)
		}
		return AccessConfig{Type: OnlyAddress, Address: addr}
	case Everybody:
		return AllowEverybody
	}
	panic("unsupported access type")
}

func (a *AccessType) UnmarshalText(text []byte) error {
	s := AccessType(text)
	if _, ok := AllAccessTypes[s]; ok {
		*a = s
		return nil
	}
	*a = Undefined
	return nil
}

func (a AccessType) MarshalText() ([]byte, error) {
	if _, ok := AllAccessTypes[a]; ok {
		return []byte(a), nil
	}
	return []byte(Undefined), nil
}

var (
	AllowEverybody = AccessConfig{Type: Everybody}
	AllowNobody    = AccessConfig{Type: Nobody}
)

type AccessConfig struct {
	Type    AccessType     `json:"permission" yaml:"permission"`
	Address sdk.AccAddress `json:"address,omitempty" yaml:"address"`
}

func (a AccessConfig) Equals(o AccessConfig) bool {
	return a.Type == o.Type && a.Address.Equals(o.Address)
}

var (
	ParamStoreKeyUploadAccess      = []byte("uploadAccess")
	ParamStoreKeyInstantiateAccess = []byte("instantiateAccess")
	ParamStoreKeyGasMultiplier     = []byte("gasMultiplier")
	ParamStoreKeyInstanceCost      = []byte("instanceCost")
	ParamStoreKeyCompileCost       = []byte("compileCost")
)

// Params defines the set of wasm parameters.
type Params struct {
	// UploadAccess defines the permissions required for code upload
	UploadAccess AccessConfig `json:"code_upload_access" yaml:"code_upload_access"`

	// DefaultInstantiatePermission defines the permission to use for contract instantiation when the code uploader has not set any.
	DefaultInstantiatePermission AccessType `json:"instantiate_default_permission" yaml:"instantiate_default_permission"`

	// GasMultiplier is how many cosmwasm gas points = 1 sdk gas point
	// SDK reference costs can be found here: https://github.com/cosmos/cosmos-sdk/blob/02c6c9fafd58da88550ab4d7d494724a477c8a68/store/types/gas.go#L153-L164
	// A write at ~3000 gas and ~200us = 10 gas per us (microsecond) cpu/io
	// Rough timing have 88k gas at 90us, which is equal to 1k sdk gas... (one read)
	//
	// Please not that all gas prices returned to the wasmer engine should have this multiplied
	GasMultiplier uint64 `json:"gas_multiplier" yaml:"gas_multiplier"`

	// InstanceCost is how much SDK gas we charge each time we load a WASM instance.
	// Creating a new instance is costly, and this helps put a recursion limit to contracts calling contracts.
	InstanceCost uint64 `json:"instance_cost" yaml:"instance_cost"`

	// CompileCost is how much SDK gas we charge *per byte* for compiling WASM code.
	CompileCost uint64 `json:"compile_cost" yaml:"compile_cost"`
}

// ParamKeyTable returns the parameter key table.
func ParamKeyTable() params.KeyTable {
	return params.NewKeyTable().RegisterParamSet(&Params{})
}

const (
	DefaultGasMultiplier uint64 = 100
	DefaultInstanceCost  uint64 = 40_000
	DefaultCompileCost   uint64 = 2
)

// DefaultParams returns default wasm parameters
func DefaultParams() Params {
	return Params{
		UploadAccess:                 AllowEverybody,
		DefaultInstantiatePermission: Everybody,
		GasMultiplier:                DefaultGasMultiplier,
		InstanceCost:                 DefaultInstanceCost,
		CompileCost:                  DefaultCompileCost,
	}
}

func (p Params) String() string {
	out, _ := yaml.Marshal(p)
	return string(out)
}

// ParamSetPairs returns the parameter set pairs.
func (p *Params) ParamSetPairs() params.ParamSetPairs {
	return params.ParamSetPairs{
		params.NewParamSetPair(ParamStoreKeyUploadAccess, &p.UploadAccess, validateAccessConfig),
		params.NewParamSetPair(ParamStoreKeyInstantiateAccess, &p.DefaultInstantiatePermission, validateAccessType),
		params.NewParamSetPair(ParamStoreKeyGasMultiplier, &p.GasMultiplier, validateUint64Type),
		params.NewParamSetPair(ParamStoreKeyInstanceCost, &p.InstanceCost, validateUint64Type),
		params.NewParamSetPair(ParamStoreKeyCompileCost, &p.CompileCost, validateUint64Type),
	}
}

// ValidateBasic performs basic validation on wasm parameters
func (p Params) ValidateBasic() error {
	if err := validateAccessType(p.DefaultInstantiatePermission); err != nil {
		return errors.Wrap(err, "instantiate default permission")
	}
	if err := validateAccessConfig(p.UploadAccess); err != nil {
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
	v, ok := i.(AccessType)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}
	if v == Undefined {
		return sdkerrors.Wrap(ErrEmpty, "type")
	}
	if _, ok := AllAccessTypes[v]; !ok {
		return sdkerrors.Wrapf(ErrInvalid, "unknown type: %q", v)
	}
	return nil
}

func validateUint64Type(i interface{}) error {
	if _, ok := i.(uint64); !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}
	return nil
}

func (v AccessConfig) ValidateBasic() error {
	switch v.Type {
	case Undefined, "":
		return sdkerrors.Wrap(ErrEmpty, "type")
	case Nobody, Everybody:
		if len(v.Address) != 0 {
			return sdkerrors.Wrap(ErrInvalid, "address not allowed for this type")
		}
		return nil
	case OnlyAddress:
		return sdk.VerifyAddressFormat(v.Address)
	}
	return sdkerrors.Wrapf(ErrInvalid, "unknown type: %q", v.Type)
}

func (v AccessConfig) Allowed(actor sdk.AccAddress) bool {
	switch v.Type {
	case Nobody:
		return false
	case Everybody:
		return true
	case OnlyAddress:
		return v.Address.Equals(actor)
	default:
		panic("unknown type")
	}
}
