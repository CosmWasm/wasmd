package types

import (
	"encoding/json"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
	"github.com/gogo/protobuf/jsonpb"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

var (
	ParamStoreKeyUploadAccess      = []byte("uploadAccess")
	ParamStoreKeyInstantiateAccess = []byte("instantiateAccess")
)

var AllAccessTypes = []AccessType{
	AccessTypeNobody,
	AccessTypeOnlyAddress,
	AccessTypeAnyOfAddresses,
	AccessTypeEverybody,
}

func (a AccessType) With(accessValue interface{}) AccessConfig {
	switch a {
	case AccessTypeNobody:
		return AllowNobody
	case AccessTypeOnlyAddress:
		addr, ok := accessValue.(sdk.AccAddress)
		if !ok {
			panic(fmt.Sprintf("expected address but got %v", accessValue))
		}
		if err := sdk.VerifyAddressFormat(addr); err != nil {
			panic(err)
		}
		return AccessConfig{Permission: AccessTypeOnlyAddress, Address: addr.String()}
	case AccessTypeEverybody:
		return AllowEverybody
	case AccessTypeAnyOfAddresses:
		addrs, ok := accessValue.([]sdk.AccAddress)
		if !ok {
			panic(fmt.Sprintf("expected addresses but got %v", accessValue))
		}
		bech32Addrs := make([]string, len(addrs))
		for i, v := range addrs {
			bech32Addrs[i] = v.String()
		}
		if err := assertValidAddresses(bech32Addrs); err != nil {
			panic(sdkerrors.Wrap(err, "addresses"))
		}
		return AccessConfig{Permission: AccessTypeAnyOfAddresses, Addresses: bech32Addrs}
	case AccessTypeAnyOfCodeIds:
		codeIds, ok := accessValue.([]uint64)
		if !ok {
			panic(fmt.Sprintf("expected codeIds but got %v", accessValue))
		}
		return AccessConfig{Permission: AccessTypeAnyOfAddresses, CodeIds: codeIds}
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
	case AccessTypeAnyOfAddresses:
		return "AnyOfAddresses"
	case AccessTypeAnyOfCodeIds:
		return "AnyOfCodeIds"
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

// ValidateBasic performs basic validation
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
		if len(a.Addresses) != 0 {
			return ErrInvalid.Wrap("addresses field set")
		}
		if len(a.CodeIds) != 0 {
			return ErrInvalid.Wrap("codeIds field set")
		}
		_, err := sdk.AccAddressFromBech32(a.Address)
		return err
	case AccessTypeAnyOfAddresses:
		if a.Address != "" {
			return ErrInvalid.Wrap("address field set")
		}
		if len(a.CodeIds) != 0 {
			return ErrInvalid.Wrap("codeIds field set")
		}
		return sdkerrors.Wrap(assertValidAddresses(a.Addresses), "addresses")
	case AccessTypeAnyOfCodeIds:
		if a.Address != "" {
			return ErrInvalid.Wrap("address field set")
		}
		if len(a.Addresses) != 0 {
			return ErrInvalid.Wrap("addresses field set")
		}
		return nil
	}
	return sdkerrors.Wrapf(ErrInvalid, "unknown type: %q", a.Permission)
}

func assertValidAddresses(addrs []string) error {
	if len(addrs) == 0 {
		return ErrEmpty
	}
	idx := make(map[string]struct{}, len(addrs))
	for _, a := range addrs {
		if _, err := sdk.AccAddressFromBech32(a); err != nil {
			return sdkerrors.Wrapf(err, "address: %s", a)
		}
		if _, exists := idx[a]; exists {
			return ErrDuplicate.Wrapf("address: %s", a)
		}
		idx[a] = struct{}{}
	}
	return nil
}

// Allowed returns if permission includes the actor.
// Actor address must be valid and not nil
func (a AccessConfig) Allowed(actor sdk.AccAddress, codeId uint64) bool {
	switch a.Permission {
	case AccessTypeNobody:
		return false
	case AccessTypeEverybody:
		return true
	case AccessTypeOnlyAddress:
		return a.Address == actor.String()
	case AccessTypeAnyOfAddresses:
		for _, v := range a.Addresses {
			if v == actor.String() {
				return true
			}
		}
		return false
	case AccessTypeAnyOfCodeIds:
		for _, v := range a.CodeIds {
			if v == codeId {
				return true
			}
		}
		return false
	default:
		panic("unknown type")
	}
}
