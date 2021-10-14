package codec

import (
	"bytes"
	"fmt"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/gogo/protobuf/jsonpb"
	"github.com/gogo/protobuf/proto"
)

var _ codec.Marshaler = (*ProtoCodec)(nil)

// ProtoCodec that omits empty values.
// This Marshaler can be used globally when setting up the client context or individually
// for each command via `clientCtx.WithJSONMarshaler(myMarshaler)`.
type ProtoCodec struct {
	codec.Marshaler
	interfaceRegistry types.InterfaceRegistry
}

func NewProtoCodec(marshaler codec.Marshaler, registry types.InterfaceRegistry) *ProtoCodec {
	return &ProtoCodec{Marshaler: marshaler, interfaceRegistry: registry}
}

// MarshalJSON implements JSONMarshaler.MarshalJSON method,
// it marshals to JSON using proto codec.
func (pc *ProtoCodec) MarshalJSON(o proto.Message) ([]byte, error) {
	m, ok := o.(codec.ProtoMarshaler)
	if !ok {
		return nil, fmt.Errorf("cannot protobuf JSON encode unsupported type: %T", o)
	}
	return ProtoMarshalJSON(m, pc.interfaceRegistry)
}

// MustMarshalJSON implements JSONMarshaler.MustMarshalJSON method,
// it executes MarshalJSON except it panics upon failure.
func (pc *ProtoCodec) MustMarshalJSON(o proto.Message) []byte {
	bz, err := pc.MarshalJSON(o)
	if err != nil {
		panic(err)
	}
	return bz
}

// MarshalInterfaceJSON is a convenience function for proto marshalling interfaces. It
// packs the provided value in an Any and then marshals it to bytes.
// NOTE: to marshal a concrete type, you should use MarshalJSON instead
func (pc *ProtoCodec) MarshalInterfaceJSON(x proto.Message) ([]byte, error) {
	any, err := types.NewAnyWithValue(x)
	if err != nil {
		return nil, err
	}
	return pc.MarshalJSON(any)
}

// ProtoMarshalJSON provides an auxiliary function to return Proto3 JSON encoded
// bytes of a message.
func ProtoMarshalJSON(msg proto.Message, resolver jsonpb.AnyResolver) ([]byte, error) {
	// method copied from sdk codec/json.go with EmitDefaults set to `false`
	// so that empty fields are not rendered

	// We use the OrigName because camel casing fields just doesn't make sense.
	// EmitDefaults is also often the more expected behavior for CLI users
	jm := &jsonpb.Marshaler{OrigName: true, EmitDefaults: false, AnyResolver: resolver}
	err := types.UnpackInterfaces(msg, types.ProtoJSONPacker{JSONPBMarshaler: jm})
	if err != nil {
		return nil, err
	}

	buf := new(bytes.Buffer)

	if err := jm.Marshal(buf, msg); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}
