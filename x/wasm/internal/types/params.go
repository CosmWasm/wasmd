package types

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/cosmos-sdk/x/params"
)

const (
	// DefaultParamspace for params keeper
	DefaultParamspace = ModuleName
)

var ParamStoreKeyUploadAccess = []byte("uploadAccess")
var ParamStoreKeyInstantiateAccess = []byte("instantiateAccess")

type AccessType uint8

const (
	Undefined   AccessType = 0
	Nobody      AccessType = 1
	OnlyAddress AccessType = 2
	Everybody   AccessType = 3
)

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

type AccessConfig struct {
	Type    AccessType     `json:"type"`
	Address sdk.AccAddress `json:"address"`
}

var (
	DefaultUploadAccess = AllowEverybody
	AllowEverybody      = AccessConfig{Type: Everybody}
	AllowNobody         = AccessConfig{Type: Nobody}
)

// ParamKeyTable type declaration for parameters
func ParamKeyTable() params.KeyTable {
	return params.NewKeyTable(
		params.NewParamSetPair(ParamStoreKeyUploadAccess, AllowEverybody, validateAccessConfig),
		params.NewParamSetPair(ParamStoreKeyInstantiateAccess, Everybody, validateAccessType),
	)
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
	// TODO: should we prevent Nobody here?
	return nil
}

func (v AccessConfig) ValidateBasic() error {
	switch v.Type {
	case Undefined:
		return sdkerrors.Wrap(ErrEmpty, "type")
	case Nobody, Everybody:
		if len(v.Address) != 0 {
			return sdkerrors.Wrap(ErrInvalid, "address not allowed for this type")
		}
	case OnlyAddress:
		return sdk.VerifyAddressFormat(v.Address)
	}
	return sdkerrors.Wrap(ErrInvalid, "unknown type")
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
