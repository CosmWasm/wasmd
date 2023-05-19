// Code generated by protoc-gen-gogo. DO NOT EDIT.
// source: terra/wasm/v1beta1/wasm.proto

package legacy

import (
	bytes "bytes"
	encoding_json "encoding/json"
	fmt "fmt"
	_ "github.com/cosmos/gogoproto/gogoproto"
	proto "github.com/gogo/protobuf/proto"
	io "io"
	math "math"
	math_bits "math/bits"
)

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.GoGoProtoPackageIsVersion3 // please upgrade the proto package

// CodeInfo is data for the uploaded contract WASM code
type CodeInfo struct {
	// CodeID is the sequentially increasing unique identifier
	CodeID uint64 `protobuf:"varint,1,opt,name=code_id,json=codeId,proto3" json:"code_id,omitempty" yaml:"code_id"`
	// CodeHash is the unique identifier created by wasmvm
	CodeHash []byte `protobuf:"bytes,2,opt,name=code_hash,json=codeHash,proto3" json:"code_hash,omitempty" yaml:"code_hash"`
	// Creator address who initially stored the code
	Creator string `protobuf:"bytes,3,opt,name=creator,proto3" json:"creator,omitempty" yaml:"creator"`
}

func (m *CodeInfo) Reset()         { *m = CodeInfo{} }
func (m *CodeInfo) String() string { return proto.CompactTextString(m) }
func (*CodeInfo) ProtoMessage()    {}
func (*CodeInfo) Descriptor() ([]byte, []int) {
	return fileDescriptor_2bd5d0123068c880, []int{0}
}
func (m *CodeInfo) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *CodeInfo) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_CodeInfo.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *CodeInfo) XXX_Merge(src proto.Message) {
	xxx_messageInfo_CodeInfo.Merge(m, src)
}
func (m *CodeInfo) XXX_Size() int {
	return m.Size()
}
func (m *CodeInfo) XXX_DiscardUnknown() {
	xxx_messageInfo_CodeInfo.DiscardUnknown(m)
}

var xxx_messageInfo_CodeInfo proto.InternalMessageInfo

func (m *CodeInfo) GetCodeID() uint64 {
	if m != nil {
		return m.CodeID
	}
	return 0
}

func (m *CodeInfo) GetCodeHash() []byte {
	if m != nil {
		return m.CodeHash
	}
	return nil
}

func (m *CodeInfo) GetCreator() string {
	if m != nil {
		return m.Creator
	}
	return ""
}

// ContractInfo stores a WASM contract instance
type ContractInfo struct {
	// Address is the address of the contract
	Address string `protobuf:"bytes,1,opt,name=address,proto3" json:"address,omitempty" yaml:"address"`
	// Creator is the contract creator address
	Creator string `protobuf:"bytes,2,opt,name=creator,proto3" json:"creator,omitempty" yaml:"creator"`
	// Admin is who can execute the contract migration
	Admin string `protobuf:"bytes,3,opt,name=admin,proto3" json:"admin,omitempty" yaml:"admin"`
	// CodeID is the reference to the stored Wasm code
	CodeID uint64 `protobuf:"varint,4,opt,name=code_id,json=codeId,proto3" json:"code_id,omitempty" yaml:"code_id"`
	// InitMsg is the raw message used when instantiating a contract
	InitMsg encoding_json.RawMessage `protobuf:"bytes,5,opt,name=init_msg,json=initMsg,proto3,casttype=encoding/json.RawMessage" json:"init_msg,omitempty" yaml:"init_msg"`
}

func (m *ContractInfo) Reset()         { *m = ContractInfo{} }
func (m *ContractInfo) String() string { return proto.CompactTextString(m) }
func (*ContractInfo) ProtoMessage()    {}
func (*ContractInfo) Descriptor() ([]byte, []int) {
	return fileDescriptor_2bd5d0123068c880, []int{1}
}
func (m *ContractInfo) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *ContractInfo) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_ContractInfo.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *ContractInfo) XXX_Merge(src proto.Message) {
	xxx_messageInfo_ContractInfo.Merge(m, src)
}
func (m *ContractInfo) XXX_Size() int {
	return m.Size()
}
func (m *ContractInfo) XXX_DiscardUnknown() {
	xxx_messageInfo_ContractInfo.DiscardUnknown(m)
}

var xxx_messageInfo_ContractInfo proto.InternalMessageInfo

func (m *ContractInfo) GetAddress() string {
	if m != nil {
		return m.Address
	}
	return ""
}

func (m *ContractInfo) GetCreator() string {
	if m != nil {
		return m.Creator
	}
	return ""
}

func (m *ContractInfo) GetAdmin() string {
	if m != nil {
		return m.Admin
	}
	return ""
}

func (m *ContractInfo) GetCodeID() uint64 {
	if m != nil {
		return m.CodeID
	}
	return 0
}

func (m *ContractInfo) GetInitMsg() encoding_json.RawMessage {
	if m != nil {
		return m.InitMsg
	}
	return nil
}

func init() {
	proto.RegisterType((*CodeInfo)(nil), "terra.wasm.v1beta1.CodeInfo")
	proto.RegisterType((*ContractInfo)(nil), "terra.wasm.v1beta1.ContractInfo")
}

func init() { proto.RegisterFile("terra/wasm/v1beta1/wasm.proto", fileDescriptor_2bd5d0123068c880) }

var fileDescriptor_2bd5d0123068c880 = []byte{
	// 393 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x94, 0x92, 0xb1, 0xeb, 0xd3, 0x40,
	0x14, 0xc7, 0x7b, 0xb5, 0xbf, 0xa6, 0x3d, 0x8a, 0x96, 0xd0, 0x21, 0x88, 0x26, 0xe5, 0x06, 0xe9,
	0x50, 0x73, 0x14, 0x71, 0xe9, 0x98, 0x0a, 0x5a, 0xa1, 0x4b, 0x16, 0xc1, 0xa5, 0x5c, 0x73, 0xe7,
	0x25, 0xd2, 0xdc, 0x95, 0xdc, 0x69, 0xed, 0x7f, 0xe1, 0xee, 0x22, 0xfe, 0x35, 0x8e, 0x1d, 0x9d,
	0x82, 0xa4, 0x8b, 0x73, 0x46, 0x27, 0xc9, 0x25, 0x81, 0x38, 0x89, 0xdb, 0xcb, 0x7d, 0x3e, 0x79,
	0xef, 0xf1, 0xf8, 0xc2, 0xc7, 0x9a, 0x65, 0x19, 0xc1, 0x67, 0xa2, 0x52, 0xfc, 0x71, 0x75, 0x60,
	0x9a, 0xac, 0xcc, 0x87, 0x7f, 0xca, 0xa4, 0x96, 0xb6, 0x6d, 0xb0, 0x6f, 0x5e, 0x1a, 0xfc, 0x70,
	0xc6, 0x25, 0x97, 0x06, 0xe3, 0xaa, 0xaa, 0x4d, 0xf4, 0x0d, 0xc0, 0xd1, 0x46, 0x52, 0xb6, 0x15,
	0xef, 0xa4, 0xfd, 0x1c, 0x5a, 0x91, 0xa4, 0x6c, 0x9f, 0x50, 0x07, 0xcc, 0xc1, 0x62, 0x10, 0x3c,
	0x2a, 0x72, 0x6f, 0x68, 0xf0, 0x8b, 0x32, 0xf7, 0xee, 0x5f, 0x48, 0x7a, 0x5c, 0xa3, 0x46, 0x41,
	0xe1, 0xb0, 0xaa, 0xb6, 0xd4, 0x5e, 0xc1, 0xb1, 0x79, 0x8b, 0x89, 0x8a, 0x9d, 0xfe, 0x1c, 0x2c,
	0x26, 0xc1, 0xac, 0xcc, 0xbd, 0x69, 0x47, 0xaf, 0x10, 0x0a, 0x47, 0x55, 0xfd, 0x8a, 0xa8, 0xd8,
	0x5e, 0x42, 0x2b, 0xca, 0x18, 0xd1, 0x32, 0x73, 0xee, 0xcd, 0xc1, 0x62, 0x1c, 0xd8, 0x9d, 0xfe,
	0x35, 0x40, 0x61, 0xab, 0xa0, 0x2f, 0x7d, 0x38, 0xd9, 0x48, 0xa1, 0x33, 0x12, 0x69, 0xb3, 0xe8,
	0x12, 0x5a, 0x84, 0xd2, 0x8c, 0x29, 0x65, 0x16, 0xfd, 0xeb, 0xf7, 0x06, 0xa0, 0xb0, 0x55, 0xba,
	0xc3, 0xfa, 0xff, 0x1c, 0x66, 0x3f, 0x81, 0x77, 0x84, 0xa6, 0x89, 0x68, 0x16, 0x9b, 0x96, 0xb9,
	0x37, 0x69, 0x3b, 0xa7, 0x89, 0x40, 0x61, 0x8d, 0xbb, 0xc7, 0x1a, 0xfc, 0xc7, 0xb1, 0x5e, 0xc3,
	0x51, 0x22, 0x12, 0xbd, 0x4f, 0x15, 0x77, 0xee, 0xcc, 0xad, 0x70, 0x99, 0x7b, 0x0f, 0x6a, 0xbb,
	0x25, 0xe8, 0x77, 0xee, 0x39, 0x4c, 0x44, 0x92, 0x26, 0x82, 0xe3, 0xf7, 0x4a, 0x0a, 0x3f, 0x24,
	0xe7, 0x1d, 0x53, 0x8a, 0x70, 0x16, 0x5a, 0x95, 0xb6, 0x53, 0x7c, 0x3d, 0xf8, 0xf5, 0xd5, 0x03,
	0xc1, 0xcb, 0xef, 0x85, 0x0b, 0xae, 0x85, 0x0b, 0x7e, 0x16, 0x2e, 0xf8, 0x7c, 0x73, 0x7b, 0xd7,
	0x9b, 0xdb, 0xfb, 0x71, 0x73, 0x7b, 0x6f, 0x9f, 0xf2, 0x44, 0xc7, 0x1f, 0x0e, 0x7e, 0x24, 0x53,
	0xbc, 0x91, 0x2a, 0x7d, 0x53, 0xc5, 0xa5, 0x0a, 0x05, 0xc5, 0x9f, 0xea, 0xec, 0xe8, 0xcb, 0x89,
	0x29, 0x7c, 0x64, 0x9c, 0x44, 0x97, 0xc3, 0xd0, 0x44, 0xe2, 0xd9, 0x9f, 0x00, 0x00, 0x00, 0xff,
	0xff, 0xa6, 0xb4, 0x08, 0xcc, 0x5d, 0x02, 0x00, 0x00,
}

func (this *ContractInfo) Equal(that interface{}) bool {
	if that == nil {
		return this == nil
	}

	that1, ok := that.(*ContractInfo)
	if !ok {
		that2, ok := that.(ContractInfo)
		if ok {
			that1 = &that2
		} else {
			return false
		}
	}
	if that1 == nil {
		return this == nil
	} else if this == nil {
		return false
	}
	if this.Address != that1.Address {
		return false
	}
	if this.Creator != that1.Creator {
		return false
	}
	if this.Admin != that1.Admin {
		return false
	}
	if this.CodeID != that1.CodeID {
		return false
	}
	if !bytes.Equal(this.InitMsg, that1.InitMsg) {
		return false
	}
	return true
}
func (m *CodeInfo) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *CodeInfo) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *CodeInfo) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if len(m.Creator) > 0 {
		i -= len(m.Creator)
		copy(dAtA[i:], m.Creator)
		i = encodeVarintWasm(dAtA, i, uint64(len(m.Creator)))
		i--
		dAtA[i] = 0x1a
	}
	if len(m.CodeHash) > 0 {
		i -= len(m.CodeHash)
		copy(dAtA[i:], m.CodeHash)
		i = encodeVarintWasm(dAtA, i, uint64(len(m.CodeHash)))
		i--
		dAtA[i] = 0x12
	}
	if m.CodeID != 0 {
		i = encodeVarintWasm(dAtA, i, uint64(m.CodeID))
		i--
		dAtA[i] = 0x8
	}
	return len(dAtA) - i, nil
}

func (m *ContractInfo) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *ContractInfo) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *ContractInfo) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if len(m.InitMsg) > 0 {
		i -= len(m.InitMsg)
		copy(dAtA[i:], m.InitMsg)
		i = encodeVarintWasm(dAtA, i, uint64(len(m.InitMsg)))
		i--
		dAtA[i] = 0x2a
	}
	if m.CodeID != 0 {
		i = encodeVarintWasm(dAtA, i, uint64(m.CodeID))
		i--
		dAtA[i] = 0x20
	}
	if len(m.Admin) > 0 {
		i -= len(m.Admin)
		copy(dAtA[i:], m.Admin)
		i = encodeVarintWasm(dAtA, i, uint64(len(m.Admin)))
		i--
		dAtA[i] = 0x1a
	}
	if len(m.Creator) > 0 {
		i -= len(m.Creator)
		copy(dAtA[i:], m.Creator)
		i = encodeVarintWasm(dAtA, i, uint64(len(m.Creator)))
		i--
		dAtA[i] = 0x12
	}
	if len(m.Address) > 0 {
		i -= len(m.Address)
		copy(dAtA[i:], m.Address)
		i = encodeVarintWasm(dAtA, i, uint64(len(m.Address)))
		i--
		dAtA[i] = 0xa
	}
	return len(dAtA) - i, nil
}

func encodeVarintWasm(dAtA []byte, offset int, v uint64) int {
	offset -= sovWasm(v)
	base := offset
	for v >= 1<<7 {
		dAtA[offset] = uint8(v&0x7f | 0x80)
		v >>= 7
		offset++
	}
	dAtA[offset] = uint8(v)
	return base
}
func (m *CodeInfo) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	if m.CodeID != 0 {
		n += 1 + sovWasm(uint64(m.CodeID))
	}
	l = len(m.CodeHash)
	if l > 0 {
		n += 1 + l + sovWasm(uint64(l))
	}
	l = len(m.Creator)
	if l > 0 {
		n += 1 + l + sovWasm(uint64(l))
	}
	return n
}

func (m *ContractInfo) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	l = len(m.Address)
	if l > 0 {
		n += 1 + l + sovWasm(uint64(l))
	}
	l = len(m.Creator)
	if l > 0 {
		n += 1 + l + sovWasm(uint64(l))
	}
	l = len(m.Admin)
	if l > 0 {
		n += 1 + l + sovWasm(uint64(l))
	}
	if m.CodeID != 0 {
		n += 1 + sovWasm(uint64(m.CodeID))
	}
	l = len(m.InitMsg)
	if l > 0 {
		n += 1 + l + sovWasm(uint64(l))
	}
	return n
}

func sovWasm(x uint64) (n int) {
	return (math_bits.Len64(x|1) + 6) / 7
}
func sozWasm(x uint64) (n int) {
	return sovWasm(uint64((x << 1) ^ uint64((int64(x) >> 63))))
}
func (m *CodeInfo) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowWasm
			}
			if iNdEx >= l {
				return io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= uint64(b&0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: CodeInfo: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: CodeInfo: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field CodeID", wireType)
			}
			m.CodeID = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowWasm
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.CodeID |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 2:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field CodeHash", wireType)
			}
			var byteLen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowWasm
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				byteLen |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if byteLen < 0 {
				return ErrInvalidLengthWasm
			}
			postIndex := iNdEx + byteLen
			if postIndex < 0 {
				return ErrInvalidLengthWasm
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.CodeHash = append(m.CodeHash[:0], dAtA[iNdEx:postIndex]...)
			if m.CodeHash == nil {
				m.CodeHash = []byte{}
			}
			iNdEx = postIndex
		case 3:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Creator", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowWasm
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthWasm
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthWasm
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Creator = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipWasm(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthWasm
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			iNdEx += skippy
		}
	}

	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}
func (m *ContractInfo) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowWasm
			}
			if iNdEx >= l {
				return io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= uint64(b&0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: ContractInfo: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: ContractInfo: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Address", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowWasm
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthWasm
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthWasm
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Address = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 2:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Creator", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowWasm
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthWasm
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthWasm
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Creator = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 3:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Admin", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowWasm
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthWasm
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthWasm
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Admin = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 4:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field CodeID", wireType)
			}
			m.CodeID = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowWasm
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.CodeID |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 5:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field InitMsg", wireType)
			}
			var byteLen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowWasm
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				byteLen |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if byteLen < 0 {
				return ErrInvalidLengthWasm
			}
			postIndex := iNdEx + byteLen
			if postIndex < 0 {
				return ErrInvalidLengthWasm
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.InitMsg = append(m.InitMsg[:0], dAtA[iNdEx:postIndex]...)
			if m.InitMsg == nil {
				m.InitMsg = []byte{}
			}
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipWasm(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthWasm
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			iNdEx += skippy
		}
	}

	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}
func skipWasm(dAtA []byte) (n int, err error) {
	l := len(dAtA)
	iNdEx := 0
	depth := 0
	for iNdEx < l {
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return 0, ErrIntOverflowWasm
			}
			if iNdEx >= l {
				return 0, io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= (uint64(b) & 0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		wireType := int(wire & 0x7)
		switch wireType {
		case 0:
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return 0, ErrIntOverflowWasm
				}
				if iNdEx >= l {
					return 0, io.ErrUnexpectedEOF
				}
				iNdEx++
				if dAtA[iNdEx-1] < 0x80 {
					break
				}
			}
		case 1:
			iNdEx += 8
		case 2:
			var length int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return 0, ErrIntOverflowWasm
				}
				if iNdEx >= l {
					return 0, io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				length |= (int(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if length < 0 {
				return 0, ErrInvalidLengthWasm
			}
			iNdEx += length
		case 3:
			depth++
		case 4:
			if depth == 0 {
				return 0, ErrUnexpectedEndOfGroupWasm
			}
			depth--
		case 5:
			iNdEx += 4
		default:
			return 0, fmt.Errorf("proto: illegal wireType %d", wireType)
		}
		if iNdEx < 0 {
			return 0, ErrInvalidLengthWasm
		}
		if depth == 0 {
			return iNdEx, nil
		}
	}
	return 0, io.ErrUnexpectedEOF
}

var (
	ErrInvalidLengthWasm        = fmt.Errorf("proto: negative length found during unmarshaling")
	ErrIntOverflowWasm          = fmt.Errorf("proto: integer overflow")
	ErrUnexpectedEndOfGroupWasm = fmt.Errorf("proto: unexpected end of group")
)
