package xadr8

import (
	"reflect"

	"github.com/cosmos/cosmos-sdk/types/address"
	icatypes "github.com/cosmos/ibc-go/v7/modules/apps/27-interchain-accounts/types"

	"github.com/cosmos/cosmos-sdk/codec"
	gogoproto "github.com/cosmos/gogoproto/proto"
	channeltypes "github.com/cosmos/ibc-go/v7/modules/core/04-channel/types"
)

type SenderAuthorized interface {
	gogoproto.Message
	GetSender() string
}

// NewGenericPacketDecoder constructor
func NewGenericPacketDecoder[T SenderAuthorized](cdc codec.Codec) PacketDecoder {
	return PacketDecoderFn(func(packet channeltypes.Packet) (string, bool) {
		var data T
		if x := reflect.ValueOf(data); x.Kind() == reflect.Ptr && x.IsNil() {
			v := reflect.New(reflect.TypeOf(data).Elem())
			data = v.Interface().(T)
		}
		if err := cdc.UnmarshalJSON(packet.GetData(), data); err != nil {
			return "", false
		}
		sender := data.GetSender()
		return sender, sender != "" // sanity check for non empty addresses
	})
}

var portPrefixLen = len(icatypes.ControllerPortPrefix)

func NewICAControllerPacketDecoder() PacketDecoder {
	return PacketDecoderFn(func(packet channeltypes.Packet) (string, bool) {
		if len(packet.SourcePort) < portPrefixLen+address.Len {
			return "", false
		}
		return packet.SourcePort[portPrefixLen:], true
	})
}
