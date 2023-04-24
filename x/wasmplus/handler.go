package wasmplus

import (
	"github.com/gogo/protobuf/proto"

	sdk "github.com/Finschia/finschia-sdk/types"

	"github.com/Finschia/wasmd/x/wasm"
	wasmtypes "github.com/Finschia/wasmd/x/wasm/types"
	"github.com/Finschia/wasmd/x/wasmplus/keeper"
	"github.com/Finschia/wasmd/x/wasmplus/types"
)

func NewHandler(k wasmtypes.ContractOpsKeeper) sdk.Handler {
	msgServer := keeper.NewMsgServerImpl(k)
	wasmHandler := wasm.NewHandler(k)

	return func(ctx sdk.Context, msg sdk.Msg) (*sdk.Result, error) {
		var (
			res proto.Message
			err error
		)
		switch msg := msg.(type) {
		case *types.MsgStoreCodeAndInstantiateContract:
			res, err = msgServer.StoreCodeAndInstantiateContract(sdk.WrapSDKContext(ctx), msg)
		default:
			return wasmHandler(ctx, msg)
		}
		return sdk.WrapServiceResult(ctx, res, err)
	}
}
