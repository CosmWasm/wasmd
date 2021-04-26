package keeper

import (
	"errors"
	"fmt"
	"github.com/CosmWasm/wasmd/x/wasm/keeper/wasmtesting"
	wasmvmtypes "github.com/CosmWasm/wasmvm/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	abci "github.com/tendermint/tendermint/abci/types"
	"testing"
)

func TestDispatchSubmessages(t *testing.T) {
	noReplyCalled := &mockReplyer{}
	var anyGasLimit uint64 = 1
	specs := map[string]struct {
		msgs       []wasmvmtypes.SubMsg
		replyer    *mockReplyer
		msgHandler *wasmtesting.MockMessageHandler
		expErr     bool
		expData    []byte
		expCommits []bool
		expEvents  sdk.Events
	}{
		"no reply on error without error": {
			msgs:    []wasmvmtypes.SubMsg{{ReplyOn: wasmvmtypes.ReplyError}},
			replyer: noReplyCalled,
			msgHandler: &wasmtesting.MockMessageHandler{
				DispatchMsgFn: func(ctx sdk.Context, contractAddr sdk.AccAddress, contractIBCPortID string, msg wasmvmtypes.CosmosMsg) (events []sdk.Event, data [][]byte, err error) {
					return nil, [][]byte{[]byte("myData")}, nil
				},
			},
			expCommits: []bool{true},
		},
		"no reply on success without success": {
			msgs:    []wasmvmtypes.SubMsg{{ReplyOn: wasmvmtypes.ReplySuccess}},
			replyer: noReplyCalled,
			msgHandler: &wasmtesting.MockMessageHandler{
				DispatchMsgFn: func(ctx sdk.Context, contractAddr sdk.AccAddress, contractIBCPortID string, msg wasmvmtypes.CosmosMsg) (events []sdk.Event, data [][]byte, err error) {
					return nil, nil, errors.New("test, ignore")
				},
			},
			expCommits: []bool{false},
			expErr:     true,
		},
		"reply on success - received": {
			msgs: []wasmvmtypes.SubMsg{{
				ReplyOn: wasmvmtypes.ReplySuccess,
			}},
			replyer: &mockReplyer{
				replyFn: func(ctx sdk.Context, contractAddress sdk.AccAddress, reply wasmvmtypes.Reply) (*sdk.Result, error) {
					return &sdk.Result{Data: []byte("myReplyData")}, nil
				},
			},
			msgHandler: &wasmtesting.MockMessageHandler{
				DispatchMsgFn: func(ctx sdk.Context, contractAddr sdk.AccAddress, contractIBCPortID string, msg wasmvmtypes.CosmosMsg) (events []sdk.Event, data [][]byte, err error) {
					return nil, [][]byte{[]byte("myData")}, nil
				},
			},
			expData:    []byte("myReplyData"),
			expCommits: []bool{true},
		},
		"reply on error - handled": {
			msgs: []wasmvmtypes.SubMsg{{
				ReplyOn: wasmvmtypes.ReplyError,
			}},
			replyer: &mockReplyer{
				replyFn: func(ctx sdk.Context, contractAddress sdk.AccAddress, reply wasmvmtypes.Reply) (*sdk.Result, error) {
					return &sdk.Result{Data: []byte("myReplyData")}, nil
				},
			},
			msgHandler: &wasmtesting.MockMessageHandler{
				DispatchMsgFn: func(ctx sdk.Context, contractAddr sdk.AccAddress, contractIBCPortID string, msg wasmvmtypes.CosmosMsg) (events []sdk.Event, data [][]byte, err error) {
					return nil, nil, errors.New("my error")
				},
			},
			expData:    []byte("myReplyData"),
			expCommits: []bool{false},
		},
		"with reply events": {
			msgs: []wasmvmtypes.SubMsg{{
				ReplyOn: wasmvmtypes.ReplySuccess,
			}},
			replyer: &mockReplyer{
				replyFn: func(ctx sdk.Context, contractAddress sdk.AccAddress, reply wasmvmtypes.Reply) (*sdk.Result, error) {
					return &sdk.Result{Data: []byte("myReplyData")}, nil
				},
			},
			msgHandler: &wasmtesting.MockMessageHandler{
				DispatchMsgFn: func(ctx sdk.Context, contractAddr sdk.AccAddress, contractIBCPortID string, msg wasmvmtypes.CosmosMsg) (events []sdk.Event, data [][]byte, err error) {
					myEvents := []sdk.Event{{Type: "myEvent", Attributes: []abci.EventAttribute{{Key: []byte("foo"), Value: []byte("bar")}}}}
					return myEvents, [][]byte{[]byte("myData")}, nil
				},
			},
			expData:    []byte("myReplyData"),
			expCommits: []bool{true},
			expEvents: []sdk.Event{{
				Type:       "myEvent",
				Attributes: []abci.EventAttribute{{Key: []byte("foo"), Value: []byte("bar")}},
			}},
		},
		"reply returns error": {
			msgs: []wasmvmtypes.SubMsg{{
				ReplyOn: wasmvmtypes.ReplySuccess,
			}},
			replyer: &mockReplyer{
				replyFn: func(ctx sdk.Context, contractAddress sdk.AccAddress, reply wasmvmtypes.Reply) (*sdk.Result, error) {
					return nil, errors.New("reply failed")
				},
			},
			msgHandler: &wasmtesting.MockMessageHandler{
				DispatchMsgFn: func(ctx sdk.Context, contractAddr sdk.AccAddress, contractIBCPortID string, msg wasmvmtypes.CosmosMsg) (events []sdk.Event, data [][]byte, err error) {
					return nil, nil, nil
				},
			},
			expCommits: []bool{false},
			expErr:     true,
		},
		"with gas limit - out of gas": {
			msgs: []wasmvmtypes.SubMsg{{
				GasLimit: &anyGasLimit,
				ReplyOn:  wasmvmtypes.ReplyError,
			}},
			replyer: &mockReplyer{
				replyFn: func(ctx sdk.Context, contractAddress sdk.AccAddress, reply wasmvmtypes.Reply) (*sdk.Result, error) {
					return &sdk.Result{Data: []byte("myReplyData")}, nil
				},
			},
			msgHandler: &wasmtesting.MockMessageHandler{
				DispatchMsgFn: func(ctx sdk.Context, contractAddr sdk.AccAddress, contractIBCPortID string, msg wasmvmtypes.CosmosMsg) (events []sdk.Event, data [][]byte, err error) {
					ctx.GasMeter().ConsumeGas(sdk.Gas(101), "testing")
					return nil, [][]byte{[]byte("someData")}, nil
				},
			},
			expData:    []byte("myReplyData"),
			expCommits: []bool{false},
		},
		"with gas limit - within limit no error": {
			msgs: []wasmvmtypes.SubMsg{{
				GasLimit: &anyGasLimit,
				ReplyOn:  wasmvmtypes.ReplyError,
			}},
			replyer: &mockReplyer{},
			msgHandler: &wasmtesting.MockMessageHandler{
				DispatchMsgFn: func(ctx sdk.Context, contractAddr sdk.AccAddress, contractIBCPortID string, msg wasmvmtypes.CosmosMsg) (events []sdk.Event, data [][]byte, err error) {
					ctx.GasMeter().ConsumeGas(sdk.Gas(1), "testing")
					return nil, [][]byte{[]byte("someData")}, nil
				},
			},
			expCommits: []bool{true},
		},
		"multiple msg - last reply": {
			msgs: []wasmvmtypes.SubMsg{{ID: 1, ReplyOn: wasmvmtypes.ReplyError}, {ID: 2, ReplyOn: wasmvmtypes.ReplyError}},
			replyer: &mockReplyer{
				replyFn: func(ctx sdk.Context, contractAddress sdk.AccAddress, reply wasmvmtypes.Reply) (*sdk.Result, error) {
					return &sdk.Result{Data: []byte(fmt.Sprintf("myReplyData:%d", reply.ID))}, nil
				},
			},
			msgHandler: &wasmtesting.MockMessageHandler{
				DispatchMsgFn: func(ctx sdk.Context, contractAddr sdk.AccAddress, contractIBCPortID string, msg wasmvmtypes.CosmosMsg) (events []sdk.Event, data [][]byte, err error) {
					return nil, nil, errors.New("my error")
				},
			},
			expData:    []byte("myReplyData:2"),
			expCommits: []bool{false, false},
		},
		"multiple msg - last reply with non nil": {
			msgs: []wasmvmtypes.SubMsg{{ID: 1, ReplyOn: wasmvmtypes.ReplyError}, {ID: 2, ReplyOn: wasmvmtypes.ReplyError}},
			replyer: &mockReplyer{
				replyFn: func(ctx sdk.Context, contractAddress sdk.AccAddress, reply wasmvmtypes.Reply) (*sdk.Result, error) {
					if reply.ID == 2 {
						return &sdk.Result{}, nil
					}
					return &sdk.Result{Data: []byte("myReplyData:1")}, nil
				},
			},
			msgHandler: &wasmtesting.MockMessageHandler{
				DispatchMsgFn: func(ctx sdk.Context, contractAddr sdk.AccAddress, contractIBCPortID string, msg wasmvmtypes.CosmosMsg) (events []sdk.Event, data [][]byte, err error) {
					return nil, nil, errors.New("my error")
				},
			},
			expData:    []byte("myReplyData:1"),
			expCommits: []bool{false, false},
		},
		"empty replyOn rejected": {
			msgs:       []wasmvmtypes.SubMsg{{}},
			replyer:    noReplyCalled,
			msgHandler: &wasmtesting.MockMessageHandler{},
			expErr:     true,
		},
		"invalid replyOn rejected": {
			msgs:       []wasmvmtypes.SubMsg{{ReplyOn: "invalid"}},
			replyer:    noReplyCalled,
			msgHandler: &wasmtesting.MockMessageHandler{},
			expCommits: []bool{false},
			expErr:     true,
		},
	}
	for name, spec := range specs {
		t.Run(name, func(t *testing.T) {
			var mockStore wasmtesting.MockCommitMultiStore
			em := sdk.NewEventManager()
			ctx := sdk.Context{}.WithMultiStore(&mockStore).
				WithGasMeter(sdk.NewGasMeter(100)).
				WithEventManager(em)
			d := NewMessageDispatcher(spec.msgHandler, spec.replyer)
			gotData, gotErr := d.DispatchSubmessages(ctx, RandomAccountAddress(t), "any_port", spec.msgs)
			if spec.expErr {
				require.Error(t, gotErr)
				return
			} else {
				require.NoError(t, gotErr)
				assert.Equal(t, spec.expData, gotData)
			}
			assert.Equal(t, spec.expCommits, mockStore.Committed)
			if len(spec.expEvents) == 0 {
				assert.Empty(t, em.Events())
			} else {
				assert.Equal(t, spec.expEvents, em.Events())
			}
		})
	}
}

type mockReplyer struct {
	replyFn func(ctx sdk.Context, contractAddress sdk.AccAddress, reply wasmvmtypes.Reply) (*sdk.Result, error)
}

func (m mockReplyer) reply(ctx sdk.Context, contractAddress sdk.AccAddress, reply wasmvmtypes.Reply) (*sdk.Result, error) {
	if m.replyFn == nil {
		panic("not expected to be called")
	}
	return m.replyFn(ctx, contractAddress, reply)
}
