package v4

import (
	"testing"

	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/gogoproto/proto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestLegacyContractInfoUnmarshalWithExtension verifies that unmarshaling a
// LegacyContractInfo with a non-nil Extension field does NOT panic.
//
// This was the root cause of the v28 upgrade failure:
// codectypes.Any has an unexported cachedValue field without a protobuf tag.
// When gogoproto's table-driven unmarshaler encounters it, it panics with
// "protobuf tag not enough fields in Any.cachedValue".
//
// The fix uses LegacyAny (which only has tagged fields) instead of codectypes.Any.
func TestLegacyContractInfoUnmarshalWithExtension(t *testing.T) {
	// Build a LegacyContractInfo with Extension populated (field 8 in legacy schema)
	original := &LegacyContractInfo{
		CodeID:     42,
		Creator:    "xion1creator",
		Admin:      "xion1admin",
		Label:      "my-contract",
		IBCPortID:  "wasm.xion1port",
		IBC2PortID: "wasm.xion1ibc2port",
		Created: &AbsoluteTxPosition{
			BlockHeight: 100,
			TxIndex:     5,
		},
		Extension: &LegacyAny{
			TypeUrl: "/xion.v1.ContractExtension",
			Value:   []byte{0x0a, 0x04, 0x74, 0x65, 0x73, 0x74}, // some protobuf bytes
		},
	}

	// Marshal it
	bz, err := proto.Marshal(original)
	require.NoError(t, err)

	// Unmarshal - this used to panic with codectypes.Any
	var decoded LegacyContractInfo
	assert.NotPanics(t, func() {
		err = proto.Unmarshal(bz, &decoded)
	}, "unmarshaling LegacyContractInfo with Extension should not panic")
	require.NoError(t, err)

	// Verify fields round-tripped correctly
	assert.Equal(t, original.CodeID, decoded.CodeID)
	assert.Equal(t, original.Creator, decoded.Creator)
	assert.Equal(t, original.Admin, decoded.Admin)
	assert.Equal(t, original.Label, decoded.Label)
	assert.Equal(t, original.IBCPortID, decoded.IBCPortID)
	assert.Equal(t, original.IBC2PortID, decoded.IBC2PortID)
	assert.Equal(t, original.Created.BlockHeight, decoded.Created.BlockHeight)
	assert.Equal(t, original.Created.TxIndex, decoded.Created.TxIndex)
	require.NotNil(t, decoded.Extension)
	assert.Equal(t, original.Extension.TypeUrl, decoded.Extension.TypeUrl)
	assert.Equal(t, original.Extension.Value, decoded.Extension.Value)
}

// TestLegacyAnyToCodecTypesAny verifies conversion from LegacyAny to codectypes.Any.
func TestLegacyAnyToCodecTypesAny(t *testing.T) {
	legacy := &LegacyAny{
		TypeUrl: "/xion.v1.SomeType",
		Value:   []byte{0x01, 0x02, 0x03},
	}

	converted := &codectypes.Any{
		TypeUrl: legacy.TypeUrl,
		Value:   legacy.Value,
	}

	assert.Equal(t, legacy.TypeUrl, converted.TypeUrl)
	assert.Equal(t, legacy.Value, converted.Value)
}

// TestLegacyContractInfoUnmarshalWithoutExtension verifies that contracts
// without an Extension field still unmarshal correctly.
func TestLegacyContractInfoUnmarshalWithoutExtension(t *testing.T) {
	original := &LegacyContractInfo{
		CodeID:     1,
		Creator:    "xion1creator",
		Label:      "simple-contract",
		IBC2PortID: "",
	}

	bz, err := proto.Marshal(original)
	require.NoError(t, err)

	var decoded LegacyContractInfo
	assert.NotPanics(t, func() {
		err = proto.Unmarshal(bz, &decoded)
	})
	require.NoError(t, err)

	assert.Equal(t, original.CodeID, decoded.CodeID)
	assert.Equal(t, original.Creator, decoded.Creator)
	assert.Equal(t, original.Label, decoded.Label)
	assert.Nil(t, decoded.Extension)
}
