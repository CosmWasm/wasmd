package v4

import (
	"github.com/cosmos/gogoproto/proto"
)

// LegacyContractInfo represents the ContractInfo structure from wasmd v0.61.2
// where ibc2_port_id was at field 7 and extension was at field 8.
// This is the INCORRECT field order that was fixed in v0.61.5+ (commit 6cbaaae4).
//
// IMPORTANT: We use *LegacyAny instead of *codectypes.Any for the Extension field
// because codectypes.Any has an unexported cachedValue field without a protobuf tag.
// Since LegacyContractInfo is a hand-written type (no generated XXX_Unmarshal),
// gogoproto falls back to its table-driven unmarshaler which panics on
// "protobuf tag not enough fields in Any.cachedValue".
type LegacyContractInfo struct {
	CodeID     uint64              `protobuf:"varint,1,opt,name=code_id,json=codeId,proto3" json:"code_id,omitempty"`
	Creator    string              `protobuf:"bytes,2,opt,name=creator,proto3" json:"creator,omitempty"`
	Admin      string              `protobuf:"bytes,3,opt,name=admin,proto3" json:"admin,omitempty"`
	Label      string              `protobuf:"bytes,4,opt,name=label,proto3" json:"label,omitempty"`
	Created    *AbsoluteTxPosition `protobuf:"bytes,5,opt,name=created,proto3" json:"created,omitempty"`
	IBCPortID  string              `protobuf:"bytes,6,opt,name=ibc_port_id,json=ibcPortId,proto3" json:"ibc_port_id,omitempty"`
	IBC2PortID string              `protobuf:"bytes,7,opt,name=ibc2_port_id,json=ibc2PortId,proto3" json:"ibc2_port_id,omitempty"` // OLD: field 7
	Extension  *LegacyAny          `protobuf:"bytes,8,opt,name=extension,proto3" json:"extension,omitempty"`                       // OLD: field 8
}

// LegacyAny is a minimal replacement for codectypes.Any that only contains
// protobuf-tagged fields. This avoids the gogoproto table-driven unmarshaler
// panic caused by codectypes.Any's unexported cachedValue field.
type LegacyAny struct {
	TypeUrl string `protobuf:"bytes,1,opt,name=type_url,json=typeUrl,proto3" json:"type_url,omitempty"`
	Value   []byte `protobuf:"bytes,2,opt,name=value,proto3" json:"value,omitempty"`
}

// AbsoluteTxPosition is a unique transaction position that allows for global ordering of transactions.
type AbsoluteTxPosition struct {
	BlockHeight uint64 `protobuf:"varint,1,opt,name=block_height,json=blockHeight,proto3" json:"block_height,omitempty"`
	TxIndex     uint64 `protobuf:"varint,2,opt,name=tx_index,json=txIndex,proto3" json:"tx_index,omitempty"`
}

func (m *LegacyContractInfo) Reset()         { *m = LegacyContractInfo{} }
func (m *LegacyContractInfo) String() string { return proto.CompactTextString(m) }
func (*LegacyContractInfo) ProtoMessage()    {}

func (m *LegacyAny) Reset()         { *m = LegacyAny{} }
func (m *LegacyAny) String() string { return proto.CompactTextString(m) }
func (*LegacyAny) ProtoMessage()    {}

func (m *AbsoluteTxPosition) Reset()         { *m = AbsoluteTxPosition{} }
func (m *AbsoluteTxPosition) String() string { return proto.CompactTextString(m) }
func (*AbsoluteTxPosition) ProtoMessage()    {}
