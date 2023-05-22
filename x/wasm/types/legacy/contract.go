package legacy

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// NewContractInfo creates a new instance of a given WASM contract info
func NewContractInfo(codeID uint64, address, creator, admin sdk.AccAddress, initMsg []byte) ContractInfo {
	var adminAddr string
	if !admin.Empty() {
		adminAddr = admin.String()
	}

	return ContractInfo{
		Address: address.String(),
		CodeID:  codeID,
		Creator: creator.String(),
		Admin:   adminAddr,
		InitMsg: initMsg,
	}
}

// NewCodeInfo creates new code info from input
func NewCodeInfo(codeID uint64, creator sdk.AccAddress, hash []byte) CodeInfo {

	return CodeInfo{
		CodeID: codeID,
		CodeHash: hash,
		Creator: creator.String(),
	}

}
