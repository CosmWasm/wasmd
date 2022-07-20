// Code generated by protoc-gen-gogo. DO NOT EDIT.
// source: cosmwasm/wasm/v1/stargate/mint/query.proto

package mint

import (
	fmt "fmt"
	io "io"
	math "math"
	math_bits "math/bits"

	github_com_cosmos_cosmos_sdk_types "github.com/cosmos/cosmos-sdk/types"
	types "github.com/cosmos/cosmos-sdk/x/mint/types"
	_ "github.com/gogo/protobuf/gogoproto"
	proto "github.com/gogo/protobuf/proto"
)

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal

var (
	_ = fmt.Errorf
	_ = math.Inf
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.GoGoProtoPackageIsVersion3 // please upgrade the proto package

// QueryParamsResponse is the response type for the Query/Params RPC method.
type QueryParamsResponse struct {
	// params defines the parameters of the module.
	Params types.Params `protobuf:"bytes,1,opt,name=params,proto3" json:"params"`
}

func (m *QueryParamsResponse) Reset()         { *m = QueryParamsResponse{} }
func (m *QueryParamsResponse) String() string { return proto.CompactTextString(m) }
func (*QueryParamsResponse) ProtoMessage()    {}
func (*QueryParamsResponse) Descriptor() ([]byte, []int) {
	return fileDescriptor_b8ebde63af3133dc, []int{0}
}

func (m *QueryParamsResponse) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}

func (m *QueryParamsResponse) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_QueryParamsResponse.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}

func (m *QueryParamsResponse) XXX_Merge(src proto.Message) {
	xxx_messageInfo_QueryParamsResponse.Merge(m, src)
}

func (m *QueryParamsResponse) XXX_Size() int {
	return m.Size()
}

func (m *QueryParamsResponse) XXX_DiscardUnknown() {
	xxx_messageInfo_QueryParamsResponse.DiscardUnknown(m)
}

var xxx_messageInfo_QueryParamsResponse proto.InternalMessageInfo

func (m *QueryParamsResponse) GetParams() types.Params {
	if m != nil {
		return m.Params
	}
	return types.Params{}
}

// QueryInflationResponse is the response type for the Query/Inflation RPC
// method.
type QueryInflationResponse struct {
	// inflation is the current minting inflation value.
	Inflation github_com_cosmos_cosmos_sdk_types.Dec `protobuf:"bytes,1,opt,name=inflation,proto3,customtype=github.com/cosmos/cosmos-sdk/types.Dec" json:"inflation"`
}

func (m *QueryInflationResponse) Reset()         { *m = QueryInflationResponse{} }
func (m *QueryInflationResponse) String() string { return proto.CompactTextString(m) }
func (*QueryInflationResponse) ProtoMessage()    {}
func (*QueryInflationResponse) Descriptor() ([]byte, []int) {
	return fileDescriptor_b8ebde63af3133dc, []int{1}
}

func (m *QueryInflationResponse) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}

func (m *QueryInflationResponse) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_QueryInflationResponse.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}

func (m *QueryInflationResponse) XXX_Merge(src proto.Message) {
	xxx_messageInfo_QueryInflationResponse.Merge(m, src)
}

func (m *QueryInflationResponse) XXX_Size() int {
	return m.Size()
}

func (m *QueryInflationResponse) XXX_DiscardUnknown() {
	xxx_messageInfo_QueryInflationResponse.DiscardUnknown(m)
}

var xxx_messageInfo_QueryInflationResponse proto.InternalMessageInfo

// QueryAnnualProvisionsResponse is the response type for the
// Query/AnnualProvisions RPC method.
type QueryAnnualProvisionsResponse struct {
	// annual_provisions is the current minting annual provisions value.
	AnnualProvisions github_com_cosmos_cosmos_sdk_types.Dec `protobuf:"bytes,1,opt,name=annual_provisions,json=annualProvisions,proto3,customtype=github.com/cosmos/cosmos-sdk/types.Dec" json:"annual_provisions"`
}

func (m *QueryAnnualProvisionsResponse) Reset()         { *m = QueryAnnualProvisionsResponse{} }
func (m *QueryAnnualProvisionsResponse) String() string { return proto.CompactTextString(m) }
func (*QueryAnnualProvisionsResponse) ProtoMessage()    {}
func (*QueryAnnualProvisionsResponse) Descriptor() ([]byte, []int) {
	return fileDescriptor_b8ebde63af3133dc, []int{2}
}

func (m *QueryAnnualProvisionsResponse) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}

func (m *QueryAnnualProvisionsResponse) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_QueryAnnualProvisionsResponse.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}

func (m *QueryAnnualProvisionsResponse) XXX_Merge(src proto.Message) {
	xxx_messageInfo_QueryAnnualProvisionsResponse.Merge(m, src)
}

func (m *QueryAnnualProvisionsResponse) XXX_Size() int {
	return m.Size()
}

func (m *QueryAnnualProvisionsResponse) XXX_DiscardUnknown() {
	xxx_messageInfo_QueryAnnualProvisionsResponse.DiscardUnknown(m)
}

var xxx_messageInfo_QueryAnnualProvisionsResponse proto.InternalMessageInfo

func init() {
	proto.RegisterType((*QueryParamsResponse)(nil), "cosmwasm.wasm.v1.stargate.mint.QueryParamsResponse")
	proto.RegisterType((*QueryInflationResponse)(nil), "cosmwasm.wasm.v1.stargate.mint.QueryInflationResponse")
	proto.RegisterType((*QueryAnnualProvisionsResponse)(nil), "cosmwasm.wasm.v1.stargate.mint.QueryAnnualProvisionsResponse")
}

func init() {
	proto.RegisterFile("cosmwasm/wasm/v1/stargate/mint/query.proto", fileDescriptor_b8ebde63af3133dc)
}

var fileDescriptor_b8ebde63af3133dc = []byte{
	// 324 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x9c, 0x91, 0xb1, 0x4b, 0x03, 0x31,
	0x14, 0xc6, 0xef, 0x40, 0x0a, 0x46, 0x07, 0xad, 0x22, 0x52, 0x31, 0x95, 0x0e, 0x22, 0x82, 0x09,
	0xa7, 0x93, 0xa3, 0xd5, 0x41, 0xc1, 0xa1, 0x76, 0x11, 0x74, 0x90, 0xb4, 0x4d, 0xcf, 0x60, 0x93,
	0x77, 0x5e, 0xd2, 0xd3, 0x82, 0x7f, 0x84, 0x7f, 0x56, 0xc7, 0x8e, 0xe2, 0x50, 0xa4, 0xfd, 0x47,
	0xe4, 0x5e, 0x7a, 0xad, 0x3a, 0xba, 0xe4, 0xee, 0xde, 0xfb, 0xbe, 0xdf, 0x77, 0xe1, 0x23, 0x87,
	0x6d, 0xb0, 0xfa, 0x45, 0x58, 0xcd, 0xf1, 0xc8, 0x22, 0x6e, 0x9d, 0x48, 0x63, 0xe1, 0x24, 0xd7,
	0xca, 0x38, 0xfe, 0xdc, 0x97, 0xe9, 0x80, 0x25, 0x29, 0x38, 0x28, 0xd3, 0x42, 0xcb, 0xf0, 0xc8,
	0x22, 0x56, 0x68, 0x59, 0xae, 0xad, 0x6c, 0xc6, 0x10, 0x03, 0x4a, 0x79, 0xfe, 0xe6, 0x5d, 0x15,
	0x74, 0x81, 0xf5, 0xb8, 0x2c, 0x6a, 0x49, 0x27, 0x22, 0xfc, 0xf0, 0xfb, 0x5a, 0x83, 0x6c, 0xdc,
	0xe4, 0x21, 0x0d, 0x91, 0x0a, 0x6d, 0x9b, 0xd2, 0x26, 0x60, 0xac, 0x2c, 0x9f, 0x92, 0x52, 0x82,
	0x93, 0xed, 0x70, 0x2f, 0x3c, 0x58, 0x39, 0xde, 0x61, 0x9e, 0x83, 0x51, 0x6c, 0xc6, 0x61, 0xde,
	0x54, 0x5f, 0x1a, 0x8e, 0xab, 0x41, 0x73, 0x66, 0xa8, 0x75, 0xc9, 0x16, 0x12, 0xaf, 0x4c, 0xb7,
	0x27, 0x9c, 0x02, 0x33, 0x87, 0x5e, 0x93, 0x65, 0x55, 0x0c, 0x91, 0xbb, 0x5a, 0x67, 0xb9, 0xf5,
	0x73, 0x5c, 0xdd, 0x8f, 0x95, 0x7b, 0xec, 0xb7, 0x58, 0x1b, 0x34, 0x9f, 0xfd, 0xb1, 0x7f, 0x1c,
	0xd9, 0xce, 0x13, 0x77, 0x83, 0x44, 0x5a, 0x76, 0x21, 0xdb, 0xcd, 0x05, 0xa0, 0xf6, 0x46, 0x76,
	0x31, 0xe7, 0xcc, 0x98, 0xbe, 0xe8, 0x35, 0x52, 0xc8, 0x94, 0x55, 0x60, 0x16, 0x77, 0xb8, 0x27,
	0xeb, 0x02, 0x77, 0x0f, 0xc9, 0x7c, 0xf9, 0xcf, 0xd8, 0x35, 0xf1, 0x27, 0xa4, 0x7e, 0x39, 0x9c,
	0xd0, 0x70, 0x34, 0xa1, 0xe1, 0xd7, 0x84, 0x86, 0xef, 0x53, 0x1a, 0x8c, 0xa6, 0x34, 0xf8, 0x98,
	0xd2, 0xe0, 0x8e, 0xfd, 0x60, 0x9e, 0x83, 0xd5, 0xb7, 0x45, 0xbd, 0x1d, 0xfe, 0xea, 0x6b, 0xfe,
	0xd5, 0x71, 0xab, 0x84, 0x45, 0x9c, 0x7c, 0x07, 0x00, 0x00, 0xff, 0xff, 0xb2, 0x55, 0x7d, 0x13,
	0x0c, 0x02, 0x00, 0x00,
}

func (m *QueryParamsResponse) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *QueryParamsResponse) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *QueryParamsResponse) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	{
		size, err := m.Params.MarshalToSizedBuffer(dAtA[:i])
		if err != nil {
			return 0, err
		}
		i -= size
		i = encodeVarintQuery(dAtA, i, uint64(size))
	}
	i--
	dAtA[i] = 0xa
	return len(dAtA) - i, nil
}

func (m *QueryInflationResponse) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *QueryInflationResponse) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *QueryInflationResponse) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	{
		size := m.Inflation.Size()
		i -= size
		if _, err := m.Inflation.MarshalTo(dAtA[i:]); err != nil {
			return 0, err
		}
		i = encodeVarintQuery(dAtA, i, uint64(size))
	}
	i--
	dAtA[i] = 0xa
	return len(dAtA) - i, nil
}

func (m *QueryAnnualProvisionsResponse) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *QueryAnnualProvisionsResponse) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *QueryAnnualProvisionsResponse) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	{
		size := m.AnnualProvisions.Size()
		i -= size
		if _, err := m.AnnualProvisions.MarshalTo(dAtA[i:]); err != nil {
			return 0, err
		}
		i = encodeVarintQuery(dAtA, i, uint64(size))
	}
	i--
	dAtA[i] = 0xa
	return len(dAtA) - i, nil
}

func encodeVarintQuery(dAtA []byte, offset int, v uint64) int {
	offset -= sovQuery(v)
	base := offset
	for v >= 1<<7 {
		dAtA[offset] = uint8(v&0x7f | 0x80)
		v >>= 7
		offset++
	}
	dAtA[offset] = uint8(v)
	return base
}

func (m *QueryParamsResponse) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	l = m.Params.Size()
	n += 1 + l + sovQuery(uint64(l))
	return n
}

func (m *QueryInflationResponse) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	l = m.Inflation.Size()
	n += 1 + l + sovQuery(uint64(l))
	return n
}

func (m *QueryAnnualProvisionsResponse) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	l = m.AnnualProvisions.Size()
	n += 1 + l + sovQuery(uint64(l))
	return n
}

func sovQuery(x uint64) (n int) {
	return (math_bits.Len64(x|1) + 6) / 7
}

func sozQuery(x uint64) (n int) {
	return sovQuery(uint64((x << 1) ^ uint64((int64(x) >> 63))))
}

func (m *QueryParamsResponse) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowQuery
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
			return fmt.Errorf("proto: QueryParamsResponse: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: QueryParamsResponse: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Params", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowQuery
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				msglen |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if msglen < 0 {
				return ErrInvalidLengthQuery
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthQuery
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if err := m.Params.Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipQuery(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthQuery
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

func (m *QueryInflationResponse) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowQuery
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
			return fmt.Errorf("proto: QueryInflationResponse: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: QueryInflationResponse: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Inflation", wireType)
			}
			var byteLen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowQuery
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
				return ErrInvalidLengthQuery
			}
			postIndex := iNdEx + byteLen
			if postIndex < 0 {
				return ErrInvalidLengthQuery
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if err := m.Inflation.Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipQuery(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthQuery
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

func (m *QueryAnnualProvisionsResponse) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowQuery
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
			return fmt.Errorf("proto: QueryAnnualProvisionsResponse: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: QueryAnnualProvisionsResponse: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field AnnualProvisions", wireType)
			}
			var byteLen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowQuery
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
				return ErrInvalidLengthQuery
			}
			postIndex := iNdEx + byteLen
			if postIndex < 0 {
				return ErrInvalidLengthQuery
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if err := m.AnnualProvisions.Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipQuery(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthQuery
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

func skipQuery(dAtA []byte) (n int, err error) {
	l := len(dAtA)
	iNdEx := 0
	depth := 0
	for iNdEx < l {
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return 0, ErrIntOverflowQuery
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
					return 0, ErrIntOverflowQuery
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
					return 0, ErrIntOverflowQuery
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
				return 0, ErrInvalidLengthQuery
			}
			iNdEx += length
		case 3:
			depth++
		case 4:
			if depth == 0 {
				return 0, ErrUnexpectedEndOfGroupQuery
			}
			depth--
		case 5:
			iNdEx += 4
		default:
			return 0, fmt.Errorf("proto: illegal wireType %d", wireType)
		}
		if iNdEx < 0 {
			return 0, ErrInvalidLengthQuery
		}
		if depth == 0 {
			return iNdEx, nil
		}
	}
	return 0, io.ErrUnexpectedEOF
}

var (
	ErrInvalidLengthQuery        = fmt.Errorf("proto: negative length found during unmarshaling")
	ErrIntOverflowQuery          = fmt.Errorf("proto: integer overflow")
	ErrUnexpectedEndOfGroupQuery = fmt.Errorf("proto: unexpected end of group")
)
