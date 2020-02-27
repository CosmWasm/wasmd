package keeper

import (
	wasmTypes "github.com/confio/go-cosmwasm/types"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

// ToOpaqueMsg encodes the msg using amino json encoding.
// Then it wraps it in base64 to make a string to include in OpaqueMsg
func ToOpaqueMsg(cdc *codec.Codec, msg sdk.Msg) (*wasmTypes.OpaqueMsg, error) {
	opaqueBz, err := cdc.MarshalJSON(msg)
	if err != nil {
		return nil, sdkerrors.Wrap(sdkerrors.ErrJSONMarshal, err.Error())
	}
	res := &wasmTypes.OpaqueMsg{
		Data: opaqueBz,
	}
	return res, nil
}

// ParseOpaqueMsg parses msg.Data as a base64 string
// it then decodes to an sdk.Msg using amino json encoding.
func ParseOpaqueMsg(cdc *codec.Codec, msg *wasmTypes.OpaqueMsg) (sdk.Msg, error) {
	// until more is changes, format is amino json encoding, wrapped base64
	var sdkmsg sdk.Msg
	err := cdc.UnmarshalJSON(msg.Data, &sdkmsg)
	if err != nil {
		return nil, sdkerrors.Wrap(sdkerrors.ErrJSONUnmarshal, err.Error())
	}
	return sdkmsg, nil
}

// TODO: remove
func EncodeCosmosMsgContract(raw string) []byte {
	return []byte(raw)
}
