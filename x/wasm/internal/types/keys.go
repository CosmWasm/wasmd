package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	// ModuleName is the name of the contract module
	ModuleName = "wasm"

	// StoreKey is the string store representation
	StoreKey = ModuleName

	// TStoreKey is the string transient store representation
	TStoreKey = "transient_" + ModuleName

	// QuerierRoute is the querier route for the staking module
	QuerierRoute = ModuleName

	// RouterKey is the msg router key for the staking module
	RouterKey = ModuleName
)

const ( // event attributes
	AttributeKeyContract = "contract_address"
	AttributeKeyCodeID   = "code_id"
	AttributeKeySigner   = "signer"
)

// nolint
var (
	CodeKeyPrefix                                  = []byte{0x01}
	ContractKeyPrefix                              = []byte{0x02}
	ContractStorePrefix                            = []byte{0x03}
	SequenceKeyPrefix                              = []byte{0x04}
	ContractHistoryStorePrefix                     = []byte{0x05}
	ContractByCodeIDAndCreatedSecondaryIndexPrefix = []byte{0x06}

	KeyLastCodeID     = append(SequenceKeyPrefix, []byte("lastCodeId")...)
	KeyLastInstanceID = append(SequenceKeyPrefix, []byte("lastContractId")...)
)

// GetCodeKey constructs the key for retreiving the ID for the WASM code
func GetCodeKey(codeID uint64) []byte {
	contractIDBz := sdk.Uint64ToBigEndian(codeID)
	return append(CodeKeyPrefix, contractIDBz...)
}

// GetContractAddressKey returns the key for the WASM contract instance
func GetContractAddressKey(addr sdk.AccAddress) []byte {
	return append(ContractKeyPrefix, addr...)
}

// GetContractStorePrefixKey returns the store prefix for the WASM contract instance
func GetContractStorePrefixKey(addr sdk.AccAddress) []byte {
	return append(ContractStorePrefix, addr...)
}

// GetContractByCreatedSecondaryIndexKey returns the key for the secondary index:
// `<prefix><codeID><created><contractAddr>`
func GetContractByCreatedSecondaryIndexKey(addr sdk.AccAddress, c *ContractInfo) []byte {
	created := c.Created.Bytes()
	prefix := GetContractByCodeIDSecondaryIndexPrefix(c.CodeID)
	prefixLen := len(prefix)
	r := make([]byte, prefixLen+AbsoluteTxPositionLen+sdk.AddrLen)
	copy(r[0:], prefix)
	copy(r[prefixLen:], created)
	copy(r[prefixLen+AbsoluteTxPositionLen:], addr)
	return r
}

func GetContractByCodeIDSecondaryIndexPrefix(codeID uint64) []byte {
	prefixLen := len(ContractByCodeIDAndCreatedSecondaryIndexPrefix)
	const codeIDLen = 8
	r := make([]byte, prefixLen+codeIDLen)
	copy(r[0:], ContractByCodeIDAndCreatedSecondaryIndexPrefix)
	copy(r[prefixLen:], sdk.Uint64ToBigEndian(codeID))
	return r
}
