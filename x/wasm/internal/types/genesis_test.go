package types

import (
	"crypto/sha256"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tendermint/tendermint/libs/rand"
)

func TestValidateGenesisState(t *testing.T) {
	specs := map[string]struct {
		srcMutator func(state GenesisState)
		expError   bool
	}{
		"all good": {
			srcMutator: func(s GenesisState) {},
		},
		"codeinfo invalid": {
			srcMutator: func(s GenesisState) {
				s.Codes[0].CodeInfo.CodeHash = nil
			},
			expError: true,
		},
		"contract invalid": {
			srcMutator: func(s GenesisState) {
				s.Contracts[0].ContractAddress = nil
			},
			expError: true,
		},
		"sequence invalid": {
			srcMutator: func(s GenesisState) {
				s.Sequences[0].IDKey = nil
			},
			expError: true,
		},
	}
	for msg, spec := range specs {
		t.Run(msg, func(t *testing.T) {
			state := genesisFixture(spec.srcMutator)
			got := state.ValidateBasic()
			if spec.expError {
				require.Error(t, got)
				return
			}
			require.NoError(t, got)
		})
	}

}

func genesisFixture(mutators ...func(state GenesisState)) GenesisState {
	const (
		numCodes     = 2
		numContracts = 2
		numSequences = 2
	)

	fixture := GenesisState{
		Codes:     make([]Code, numCodes),
		Contracts: make([]Contract, numContracts),
		Sequences: make([]Sequence, numSequences),
	}
	for i := 0; i < numCodes; i++ {
		fixture.Codes[i] = codeFixture()
	}
	for i := 0; i < numContracts; i++ {
		fixture.Contracts[i] = contractFixture()
	}
	for i := 0; i < numSequences; i++ {
		fixture.Sequences[i] = Sequence{
			IDKey: rand.Bytes(5),
			Value: uint64(i),
		}
	}
	for _, m := range mutators {
		m(fixture)
	}
	return fixture
}

func codeFixture() Code {
	wasmCode := rand.Bytes(100)
	codeHash := sha256.Sum256(wasmCode)
	anyAddress := make([]byte, 20)

	return Code{
		CodeInfo: CodeInfo{
			CodeHash: codeHash[:],
			Creator:  anyAddress,
		},
		CodesBytes: wasmCode,
	}
}

func contractFixture() Contract {
	anyAddress := make([]byte, 20)
	return Contract{
		ContractAddress: anyAddress,
		ContractInfo: ContractInfo{
			CodeID:  1,
			Creator: anyAddress,
			Label:   "any",
			Created: &AbsoluteTxPosition{BlockHeight: 1, TxIndex: 1},
		},
	}
}
