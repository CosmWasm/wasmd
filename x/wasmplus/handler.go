package wasmplus

import (
	"strings"

	"github.com/gogo/protobuf/proto"

	sdk "github.com/line/lbm-sdk/types"

	"github.com/line/wasmd/x/wasm"
	wasmtypes "github.com/line/wasmd/x/wasm/types"
	"github.com/line/wasmd/x/wasmplus/keeper"
	"github.com/line/wasmd/x/wasmplus/types"
)

func NewHandler(k wasmtypes.ContractOpsKeeper) sdk.Handler {
	msgServer := keeper.NewMsgServerImpl(k)
	wasmHandler := wasm.NewHandler(k)

	return func(ctx sdk.Context, msg sdk.Msg) (*sdk.Result, error) {
		var (
			res proto.Message
			err error
		)
		res, err = wasmHandler(ctx, msg)
		if err != nil && strings.Contains(err.Error(), "MsgStoreCodeAndInstantiateContract") {
			// handle wasmplus service
			msg2, ok := msg.(*types.MsgStoreCodeAndInstantiateContract)
			if ok {
				res, err = msgServer.StoreCodeAndInstantiateContract(sdk.WrapSDKContext(ctx), msg2)
			}
		}
		return sdk.WrapServiceResult(ctx, res, err)
	}
}
