package xadr8

import (
	"reflect"

	"github.com/cosmos/cosmos-sdk/codec"
	gogoproto "github.com/cosmos/gogoproto/proto"
	channeltypes "github.com/cosmos/ibc-go/v7/modules/core/04-channel/types"
)

type SenderAuthorized interface {
	gogoproto.Message
	GetSender() string
}

type GenericPacketDecoder[T SenderAuthorized] struct {
	cdc codec.Codec
}

// NewGenericPacketDecoder constructor
func NewGenericPacketDecoder[T SenderAuthorized](cdc codec.Codec) *GenericPacketDecoder[T] {
	return &GenericPacketDecoder[T]{cdc: cdc}
}

func (m GenericPacketDecoder[T]) DecodeSender(packet channeltypes.Packet) (string, bool) {
	var data T
	if x := reflect.ValueOf(data); x.Kind() == reflect.Ptr && x.IsNil() {
		v := reflect.New(reflect.TypeOf(data).Elem())
		data = v.Interface().(T)
	}
	if err := m.cdc.UnmarshalJSON(packet.GetData(), data); err != nil {
		return "", false
	}
	sender := data.GetSender()
	return sender, sender != "" // sanity check for non empty addresses
}
