package v4

import (
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/gogoproto/proto"
)

// LegacyContractInfo represents the ContractInfo structure from wasmd v0.61.2
// where ibc2_port_id was at field 7 and extension was at field 8.
// This is the INCORRECT field order that was fixed in v0.61.5+ (commit 6cbaaae4).
type LegacyContractInfo struct {
	CodeID      uint64                 `protobuf:"varint,1,opt,name=code_id,json=codeId,proto3" json:"code_id,omitempty"`
	Creator     string                 `protobuf:"bytes,2,opt,name=creator,proto3" json:"creator,omitempty"`
	Admin       string                 `protobuf:"bytes,3,opt,name=admin,proto3" json:"admin,omitempty"`
	Label       string                 `protobuf:"bytes,4,opt,name=label,proto3" json:"label,omitempty"`
	Created     *AbsoluteTxPosition    `protobuf:"bytes,5,opt,name=created,proto3" json:"created,omitempty"`
	IBCPortID   string                 `protobuf:"bytes,6,opt,name=ibc_port_id,json=ibcPortId,proto3" json:"ibc_port_id,omitempty"`
	IBC2PortID  string                 `protobuf:"bytes,7,opt,name=ibc2_port_id,json=ibc2PortId,proto3" json:"ibc2_port_id,omitempty"` // OLD: field 7
	Extension   *codectypes.Any        `protobuf:"bytes,8,opt,name=extension,proto3" json:"extension,omitempty"`                        // OLD: field 8
}

// AbsoluteTxPosition is a unique transaction position that allows for global ordering of transactions.
type AbsoluteTxPosition struct {
	BlockHeight uint64 `protobuf:"varint,1,opt,name=block_height,json=blockHeight,proto3" json:"block_height,omitempty"`
	TxIndex     uint64 `protobuf:"varint,2,opt,name=tx_index,json=txIndex,proto3" json:"tx_index,omitempty"`
}

func (m *LegacyContractInfo) Reset()         { *m = LegacyContractInfo{} }
func (m *LegacyContractInfo) String() string { return proto.CompactTextString(m) }
func (*LegacyContractInfo) ProtoMessage()    {}

func (m *AbsoluteTxPosition) Reset()         { *m = AbsoluteTxPosition{} }
func (m *AbsoluteTxPosition) String() string { return proto.CompactTextString(m) }
func (*AbsoluteTxPosition) ProtoMessage()    {}
