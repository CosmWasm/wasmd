package types

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"math/rand"

	sdk "github.com/line/lbm-sdk/types"
)

func GenesisFixture(mutators ...func(*GenesisState)) GenesisState {
	const (
		numCodes     = 2
		numContracts = 2
		numSequences = 2
		numMsg       = 3
	)

	fixture := GenesisState{
		Params:    DefaultParams(),
		Codes:     make([]Code, numCodes),
		Contracts: make([]Contract, numContracts),
		Sequences: make([]Sequence, numSequences),
	}
	for i := 0; i < numCodes; i++ {
		fixture.Codes[i] = CodeFixture()
	}
	for i := 0; i < numContracts; i++ {
		fixture.Contracts[i] = ContractFixture()
	}
	for i := 0; i < numSequences; i++ {
		fixture.Sequences[i] = Sequence{
			IDKey: randBytes(5),
			Value: uint64(i),
		}
	}
	fixture.GenMsgs = []GenesisState_GenMsgs{
		{Sum: &GenesisState_GenMsgs_StoreCode{StoreCode: MsgStoreCodeFixture()}},
		{Sum: &GenesisState_GenMsgs_InstantiateContract{InstantiateContract: MsgInstantiateContractFixture()}},
		{Sum: &GenesisState_GenMsgs_ExecuteContract{ExecuteContract: MsgExecuteContractFixture()}},
	}
	for _, m := range mutators {
		m(&fixture)
	}
	return fixture
}

func randBytes(n int) []byte {
	r := make([]byte, n)
	rand.Read(r)
	return r
}

func CodeFixture(mutators ...func(*Code)) Code {
	wasmCode := randBytes(100)

	fixture := Code{
		CodeID:    1,
		CodeInfo:  CodeInfoFixture(WithSHA256CodeHash(wasmCode)),
		CodeBytes: wasmCode,
	}

	for _, m := range mutators {
		m(&fixture)
	}
	return fixture
}

func CodeInfoFixture(mutators ...func(*CodeInfo)) CodeInfo {
	wasmCode := bytes.Repeat([]byte{0x1}, 10)
	codeHash := sha256.Sum256(wasmCode)
	const anyAddress = "link1qyqszqgpqyqszqgpqyqszqgpqyqszqgpqyqszqgpqyqszqgpqyqsh9tp23"
	fixture := CodeInfo{
		CodeHash:          codeHash[:],
		Creator:           anyAddress,
		InstantiateConfig: AllowEverybody,
	}
	for _, m := range mutators {
		m(&fixture)
	}
	return fixture
}

func ContractFixture(mutators ...func(*Contract)) Contract {
	const anyAddress = "link1qyqszqgpqyqszqgpqyqszqgpqyqszqgpqyqszqgpqyqszqgpqyqsh9tp23"

	fixture := Contract{
		ContractAddress: anyAddress,
		ContractInfo:    ContractInfoFixture(OnlyGenesisFields),
		ContractState:   []Model{{Key: []byte("anyKey"), Value: []byte("anyValue")}},
	}

	for _, m := range mutators {
		m(&fixture)
	}
	return fixture
}

func OnlyGenesisFields(info *ContractInfo) {
	info.Created = nil
}

func ContractInfoFixture(mutators ...func(*ContractInfo)) ContractInfo {
	const anyAddress = "link1qyqszqgpqyqszqgpqyqszqgpqyqszqgpqyqszqgpqyqszqgpqyqsh9tp23"

	fixture := ContractInfo{
		CodeID:  1,
		Creator: anyAddress,
		Label:   "any",
		Created: &AbsoluteTxPosition{BlockHeight: 1, TxIndex: 1},
	}

	for _, m := range mutators {
		m(&fixture)
	}
	return fixture
}

func WithSHA256CodeHash(wasmCode []byte) func(info *CodeInfo) {
	return func(info *CodeInfo) {
		codeHash := sha256.Sum256(wasmCode)
		info.CodeHash = codeHash[:]
	}
}

func MsgStoreCodeFixture(mutators ...func(*MsgStoreCode)) *MsgStoreCode {
	var wasmIdent = []byte("\x00\x61\x73\x6D")
	const anyAddress = "link1qyqszqgpqyqszqgpqyqszqgpqyqszqgpqyqszqgpqyqszqgpqyqsh9tp23"
	r := &MsgStoreCode{
		Sender:                anyAddress,
		WASMByteCode:          wasmIdent,
		InstantiatePermission: &AllowEverybody,
	}
	for _, m := range mutators {
		m(r)
	}
	return r
}

func MsgInstantiateContractFixture(mutators ...func(*MsgInstantiateContract)) *MsgInstantiateContract {
	const anyAddress = "link1qyqszqgpqyqszqgpqyqszqgpqyqszqgpqyqszqgpqyqszqgpqyqsh9tp23"
	r := &MsgInstantiateContract{
		Sender: anyAddress,
		Admin:  anyAddress,
		CodeID: 1,
		Label:  "testing",
		Msg:    []byte(`{"foo":"bar"}`),
		Funds: sdk.Coins{{
			Denom:  "stake",
			Amount: sdk.NewInt(1),
		}},
	}
	for _, m := range mutators {
		m(r)
	}
	return r
}

func MsgExecuteContractFixture(mutators ...func(*MsgExecuteContract)) *MsgExecuteContract {
	const (
		anyAddress           = "link1qyqszqgpqyqszqgpqyqszqgpqyqszqgpqyqszqgpqyqszqgpqyqsh9tp23"
		firstContractAddress = "link1ghekyjucln7y67ntx7cf27m9dpuxxemnqk82wt"
	)
	r := &MsgExecuteContract{
		Sender:   anyAddress,
		Contract: firstContractAddress,
		Msg:      []byte(`{"do":"something"}`),
		Funds: sdk.Coins{{
			Denom:  "stake",
			Amount: sdk.NewInt(1),
		}},
	}
	for _, m := range mutators {
		m(r)
	}
	return r
}

func StoreCodeProposalFixture(mutators ...func(*StoreCodeProposal)) *StoreCodeProposal {
	const anyAddress = "link1qyqszqgpqyqszqgpqyqszqgpqyqszqgpqyqszqgpqyqszqgpqyqsh9tp23"
	p := &StoreCodeProposal{
		Title:        "Foo",
		Description:  "Bar",
		RunAs:        anyAddress,
		WASMByteCode: []byte{0x0},
	}
	for _, m := range mutators {
		m(p)
	}
	return p
}

func InstantiateContractProposalFixture(mutators ...func(p *InstantiateContractProposal)) *InstantiateContractProposal {
	var (
		anyValidAddress sdk.AccAddress = bytes.Repeat([]byte{0x1}, ContractAddrLen)

		initMsg = struct {
			Verifier    sdk.AccAddress `json:"verifier"`
			Beneficiary sdk.AccAddress `json:"beneficiary"`
		}{
			Verifier:    anyValidAddress,
			Beneficiary: anyValidAddress,
		}
	)
	const anyAddress = "link1qyqszqgpqyqszqgpqyqszqgpqyqszqgpqyqszqgpqyqszqgpqyqsh9tp23"

	initMsgBz, err := json.Marshal(initMsg)
	if err != nil {
		panic(err)
	}
	p := &InstantiateContractProposal{
		Title:       "Foo",
		Description: "Bar",
		RunAs:       anyAddress,
		Admin:       anyAddress,
		CodeID:      1,
		Label:       "testing",
		Msg:         initMsgBz,
		Funds:       nil,
	}

	for _, m := range mutators {
		m(p)
	}
	return p
}

func MigrateContractProposalFixture(mutators ...func(p *MigrateContractProposal)) *MigrateContractProposal {
	var (
		anyValidAddress sdk.AccAddress = bytes.Repeat([]byte{0x1}, ContractAddrLen)

		migMsg = struct {
			Verifier sdk.AccAddress `json:"verifier"`
		}{Verifier: anyValidAddress}
	)

	migMsgBz, err := json.Marshal(migMsg)
	if err != nil {
		panic(err)
	}
	const (
		contractAddr = "link1hcttwju93d5m39467gjcq63p5kc4fdcn30dgd8"
		anyAddress   = "link1qyqszqgpqyqszqgpqyqszqgpqyqszqgpqyqszqgpqyqszqgpqyqsh9tp23"
	)
	p := &MigrateContractProposal{
		Title:       "Foo",
		Description: "Bar",
		Contract:    contractAddr,
		CodeID:      1,
		Msg:         migMsgBz,
	}

	for _, m := range mutators {
		m(p)
	}
	return p
}

func SudoContractProposalFixture(mutators ...func(p *SudoContractProposal)) *SudoContractProposal {
	const (
		contractAddr = "link1hcttwju93d5m39467gjcq63p5kc4fdcn30dgd8"
	)

	p := &SudoContractProposal{
		Title:       "Foo",
		Description: "Bar",
		Contract:    contractAddr,
		Msg:         []byte(`{"do":"something"}`),
	}

	for _, m := range mutators {
		m(p)
	}
	return p
}

func ExecuteContractProposalFixture(mutators ...func(p *ExecuteContractProposal)) *ExecuteContractProposal {
	const (
		contractAddr = "link1hcttwju93d5m39467gjcq63p5kc4fdcn30dgd8"
		anyAddress   = "link1qyqszqgpqyqszqgpqyqszqgpqyqszqgpqyqszqgpqyqszqgpqyqsh9tp23"
	)

	p := &ExecuteContractProposal{
		Title:       "Foo",
		Description: "Bar",
		Contract:    contractAddr,
		RunAs:       anyAddress,
		Msg:         []byte(`{"do":"something"}`),
		Funds: sdk.Coins{{
			Denom:  "stake",
			Amount: sdk.NewInt(1),
		}},
	}

	for _, m := range mutators {
		m(p)
	}
	return p
}

func UpdateAdminProposalFixture(mutators ...func(p *UpdateAdminProposal)) *UpdateAdminProposal {
	const (
		contractAddr = "link1hcttwju93d5m39467gjcq63p5kc4fdcn30dgd8"
		anyAddress   = "link1qyqszqgpqyqszqgpqyqszqgpqyqszqgpqyqszqgpqyqszqgpqyqsh9tp23"
	)

	p := &UpdateAdminProposal{
		Title:       "Foo",
		Description: "Bar",
		NewAdmin:    anyAddress,
		Contract:    contractAddr,
	}
	for _, m := range mutators {
		m(p)
	}
	return p
}

func ClearAdminProposalFixture(mutators ...func(p *ClearAdminProposal)) *ClearAdminProposal {
	const contractAddr = "link1hcttwju93d5m39467gjcq63p5kc4fdcn30dgd8"
	p := &ClearAdminProposal{
		Title:       "Foo",
		Description: "Bar",
		Contract:    contractAddr,
	}
	for _, m := range mutators {
		m(p)
	}
	return p
}
