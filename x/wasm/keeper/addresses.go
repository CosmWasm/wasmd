package keeper

import (
	"encoding/binary"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/address"

	"github.com/CosmWasm/wasmd/x/wasm/types"
)

// AddressGenerator abstract address generator to be used for  a single contract address
type AddressGenerator func(ctx sdk.Context, codeID uint64, checksum []byte) sdk.AccAddress

// ClassicAddressGenerator generates a contract address from codeID + instanceID sequence
func (k Keeper) ClassicAddressGenerator() AddressGenerator {
	return func(ctx sdk.Context, codeID uint64, _ []byte) sdk.AccAddress {
		instanceID := k.autoIncrementID(ctx, types.KeyLastInstanceID)
		return BuildContractAddressClassic(codeID, instanceID)
	}
}

// PredicableAddressGenerator generates a predictable contract address
func PredicableAddressGenerator(creator sdk.AccAddress, salt []byte, initMsg []byte, includeInitMsg bool) AddressGenerator {
	return func(ctx sdk.Context, _ uint64, checksum []byte) sdk.AccAddress {
		if includeInitMsg {
			return BuildContractAddressPredictable2(checksum, creator, salt, initMsg)
		}
		return BuildContractAddressPredictable1(checksum, creator, salt)
	}
}

// BuildContractAddressClassic builds an sdk account address for a contract.
func BuildContractAddressClassic(codeID, instanceID uint64) sdk.AccAddress {
	contractID := make([]byte, 16)
	binary.BigEndian.PutUint64(contractID[:8], codeID)
	binary.BigEndian.PutUint64(contractID[8:], instanceID)
	return address.Module(types.ModuleName, contractID)[:types.ContractAddrLen]
}

// BuildContractAddressPredictable1 generates a contract address for the wasm module with len = types.ContractAddrLen using the
// Cosmos SDK address.Module function.
// Internally a key is built containing (len(checksum) | checksum | len(sender_address) | sender_address | len(salt) | salt).
// All method parameter values must be valid and not be empty or nil.
func BuildContractAddressPredictable1(checksum []byte, creator sdk.AccAddress, salt []byte) sdk.AccAddress {
	checksum = address.MustLengthPrefix(checksum)
	creator = address.MustLengthPrefix(creator)
	salt = address.MustLengthPrefix(salt)
	key := make([]byte, len(checksum)+len(creator)+len(salt))
	copy(key[0:], checksum)
	copy(key[len(checksum):], creator)
	copy(key[len(checksum)+len(creator):], salt)
	return address.Module(types.ModuleName, key)[:types.ContractAddrLen]
}

// BuildContractAddressPredictable2 generates a contract address for the wasm module with len = types.ContractAddrLen using the
// Cosmos SDK address.Module function. Similar to BuildContractAddressPredictable1 but including the initMsg.
// Internally a key is built containing (len(checksum) | checksum | len(sender_address) | sender_address | len(salt) | salt| len(initMsg)|initMsg).
// All method parameter values must be valid and not be empty or nil.
func BuildContractAddressPredictable2(checksum []byte, creator sdk.AccAddress, salt, initMsg []byte) sdk.AccAddress {
	checksum = address.MustLengthPrefix(checksum)
	creator = address.MustLengthPrefix(creator)
	salt = address.MustLengthPrefix(salt)
	initMsg = address.MustLengthPrefix(initMsg)
	key := make([]byte, len(checksum)+len(creator)+len(salt)+len(initMsg))
	copy(key[0:], checksum)
	copy(key[len(checksum):], creator)
	copy(key[len(checksum)+len(creator):], salt)
	copy(key[len(checksum)+len(creator)+len(salt):], initMsg)
	return address.Module(types.ModuleName, key)[:types.ContractAddrLen]
}
