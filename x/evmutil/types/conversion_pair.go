package types

import (
	bytes "bytes"
	"encoding/hex"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
)

// NewConversionPair returns a new ConversionPair.
func NewConversionPair(address InternalEVMAddress, denom string) ConversionPair {
	return ConversionPair{
		OraiERC20Address: address.Address.Bytes(),
		Denom:            denom,
	}
}

// GetAddress returns the InternalEVMAddress of the Kava ERC20 address.
func (pair ConversionPair) GetAddress() InternalEVMAddress {
	return NewInternalEVMAddress(common.BytesToAddress(pair.OraiERC20Address))
}

// Validate returns an error if the ConversionPair is invalid.
func (pair ConversionPair) Validate() error {
	if err := sdk.ValidateDenom(pair.Denom); err != nil {
		return fmt.Errorf("conversion pair denom invalid: %v", err)
	}

	if len(pair.OraiERC20Address) != common.AddressLength {
		return fmt.Errorf("address length is %v but expected %v", len(pair.OraiERC20Address), common.AddressLength)
	}

	if bytes.Equal(pair.OraiERC20Address, common.Address{}.Bytes()) {
		return fmt.Errorf("address cannot be zero value %v", hex.EncodeToString(pair.OraiERC20Address))
	}

	return nil
}

// ConversionPairs defines a slice of ConversionPair.
type ConversionPairs []ConversionPair

// NewConversionPairs returns ConversionPairs from the provided values.
func NewConversionPairs(pairs ...ConversionPair) ConversionPairs {
	return ConversionPairs(pairs)
}

func (pairs ConversionPairs) Validate() error {
	// Check for duplicates for both addrs and denoms
	addrs := map[string]bool{}
	denoms := map[string]bool{}

	for _, pair := range pairs {
		if addrs[hex.EncodeToString(pair.OraiERC20Address)] {
			return fmt.Errorf(
				"found duplicate enabled conversion pair internal ERC20 address %s",
				hex.EncodeToString(pair.OraiERC20Address),
			)
		}

		if denoms[pair.Denom] {
			return fmt.Errorf(
				"found duplicate enabled conversion pair denom %s",
				pair.Denom,
			)
		}

		if err := pair.Validate(); err != nil {
			return err
		}

		addrs[hex.EncodeToString(pair.OraiERC20Address)] = true
		denoms[pair.Denom] = true
	}

	return nil
}

// validateConversionPairs validates an interface as ConversionPairs
func validateConversionPairs(i interface{}) error {
	pairs, ok := i.(ConversionPairs)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}

	return pairs.Validate()
}
