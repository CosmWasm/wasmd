package keeper

import (
	"errors"
	"testing"

	wasmvmtypes "github.com/CosmWasm/wasmvm/v2/types"
	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"cosmossdk.io/log"
	storetypes "cosmossdk.io/store/types"

	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/CosmWasm/wasmd/x/wasm/keeper/wasmtesting"
)

func setupDispatchTest(t *testing.T) (sdk.Context, *wasmtesting.MockCommitMultiStore, *sdk.EventManager) {
	mockStore := &wasmtesting.MockCommitMultiStore{}
	em := sdk.NewEventManager()
	ctx := sdk.Context{}.WithMultiStore(mockStore).
		WithGasMeter(storetypes.NewGasMeter(100)).
		WithEventManager(em).WithLogger(log.NewTestLogger(t))
	return ctx, mockStore, em
}

func TestDispatchSubmessagesBasicReplies(t *testing.T) {
	noReplyCalled := &mockReplyer{}
	specs := map[string]struct {
		msgs       []wasmvmtypes.SubMsg
		replyer    *mockReplyer
		msgHandler *wasmtesting.MockMessageHandler
		expErr     bool

		expData    []byte
		expCommits []bool
	}{
		"no reply on error without error": {
			msgs:    []wasmvmtypes.SubMsg{{ReplyOn: wasmvmtypes.ReplyError}},
			replyer: noReplyCalled,
			msgHandler: &wasmtesting.MockMessageHandler{
				DispatchMsgFn: func(ctx sdk.Context, contractAddr sdk.AccAddress, contractIBCPortID string, msg wasmvmtypes.CosmosMsg) (events []sdk.Event, data [][]byte, msgResponses [][]*codectypes.Any, err error) {
					return nil, [][]byte{[]byte("myData")}, [][]*codectypes.Any{}, nil
				},
			},
			expCommits: []bool{true},
		},
		"no reply on success without success": {
			msgs:    []wasmvmtypes.SubMsg{{ReplyOn: wasmvmtypes.ReplySuccess}},
			replyer: noReplyCalled,
			msgHandler: &wasmtesting.MockMessageHandler{
				DispatchMsgFn: func(ctx sdk.Context, contractAddr sdk.AccAddress, contractIBCPortID string, msg wasmvmtypes.CosmosMsg) (events []sdk.Event, data [][]byte, msgResponses [][]*codectypes.Any, err error) {
					return nil, nil, [][]*codectypes.Any{}, errors.New("test, ignore")
				},
			},
			expCommits: []bool{false},
			expErr:     true,
		},
	}

	for name, spec := range specs {
		t.Run(name, func(t *testing.T) {
			ctx, mockStore, em := setupDispatchTest(t)
			d := NewMessageDispatcher(spec.msgHandler, spec.replyer)

			gotData, gotErr := d.DispatchSubmessages(ctx, RandomAccountAddress(t), "any_port", spec.msgs)
			if spec.expErr {
				require.Error(t, gotErr)
				assert.Empty(t, em.Events())
				return
			}
			require.NoError(t, gotErr)
			assert.Equal(t, spec.expData, gotData)
			assert.Equal(t, spec.expCommits, mockStore.Committed)
		})
	}
}

func TestDispatchSubmessagesWithEvents(t *testing.T) {
	specs := map[string]struct {
		msgs       []wasmvmtypes.SubMsg
		replyer    *mockReplyer
		msgHandler *wasmtesting.MockMessageHandler
		expEvents  sdk.Events
		expCommits []bool
	}{
		"with reply events": {
			msgs: []wasmvmtypes.SubMsg{{ReplyOn: wasmvmtypes.ReplySuccess}},
			replyer: &mockReplyer{
				replyFn: func(ctx sdk.Context, contractAddress sdk.AccAddress, reply wasmvmtypes.Reply) ([]byte, error) {
					ctx.EventManager().EmitEvent(sdk.NewEvent("wasm-reply"))
					return []byte("myReplyData"), nil
				},
			},
			msgHandler: &wasmtesting.MockMessageHandler{
				DispatchMsgFn: func(ctx sdk.Context, contractAddr sdk.AccAddress, contractIBCPortID string, msg wasmvmtypes.CosmosMsg) (events []sdk.Event, data [][]byte, msgResponses [][]*codectypes.Any, err error) {
					myEvents := []sdk.Event{{Type: "myEvent", Attributes: []abci.EventAttribute{{Key: "foo", Value: "bar"}}}}
					return myEvents, [][]byte{[]byte("myData")}, [][]*codectypes.Any{}, nil
				},
			},
			expCommits: []bool{true},
			expEvents: []sdk.Event{
				{Type: "myEvent", Attributes: []abci.EventAttribute{{Key: "foo", Value: "bar"}}},
				sdk.NewEvent("wasm-reply"),
			},
		},
	}

	for name, spec := range specs {
		t.Run(name, func(t *testing.T) {
			ctx, mockStore, em := setupDispatchTest(t)
			d := NewMessageDispatcher(spec.msgHandler, spec.replyer)

			_, err := d.DispatchSubmessages(ctx, RandomAccountAddress(t), "any_port", spec.msgs)
			require.NoError(t, err)
			assert.Equal(t, spec.expCommits, mockStore.Committed)
			if len(spec.expEvents) == 0 {
				assert.Empty(t, em.Events())
			} else {
				assert.Equal(t, spec.expEvents, em.Events())
			}
		})
	}
}

func TestDispatchSubmessagesWithGasLimit(t *testing.T) {
	var anyGasLimit uint64 = 1
	specs := map[string]struct {
		msgs       []wasmvmtypes.SubMsg
		replyer    *mockReplyer
		msgHandler *wasmtesting.MockMessageHandler
		expData    []byte
		expCommits []bool
	}{
		"with gas limit - out of gas": {
			msgs: []wasmvmtypes.SubMsg{{
				GasLimit: &anyGasLimit,
				ReplyOn:  wasmvmtypes.ReplyError,
			}},
			replyer: &mockReplyer{
				replyFn: func(ctx sdk.Context, contractAddress sdk.AccAddress, reply wasmvmtypes.Reply) ([]byte, error) {
					return []byte("myReplyData"), nil
				},
			},
			msgHandler: &wasmtesting.MockMessageHandler{
				DispatchMsgFn: func(ctx sdk.Context, contractAddr sdk.AccAddress, contractIBCPortID string, msg wasmvmtypes.CosmosMsg) (events []sdk.Event, data [][]byte, msgResponses [][]*codectypes.Any, err error) {
					ctx.GasMeter().ConsumeGas(storetypes.Gas(101), "testing")
					return nil, [][]byte{[]byte("someData")}, [][]*codectypes.Any{}, nil
				},
			},
			expData:    []byte("myReplyData"),
			expCommits: []bool{false},
		},
	}

	for name, spec := range specs {
		t.Run(name, func(t *testing.T) {
			ctx, mockStore, _ := setupDispatchTest(t)
			d := NewMessageDispatcher(spec.msgHandler, spec.replyer)

			gotData, err := d.DispatchSubmessages(ctx, RandomAccountAddress(t), "any_port", spec.msgs)
			require.NoError(t, err)
			assert.Equal(t, spec.expData, gotData)
			assert.Equal(t, spec.expCommits, mockStore.Committed)
		})
	}
}

type mockReplyer struct {
	replyFn func(ctx sdk.Context, contractAddress sdk.AccAddress, reply wasmvmtypes.Reply) ([]byte, error)
}

func (m mockReplyer) reply(ctx sdk.Context, contractAddress sdk.AccAddress, reply wasmvmtypes.Reply) ([]byte, error) {
	if m.replyFn == nil {
		panic("not expected to be called")
	}
	return m.replyFn(ctx, contractAddress, reply)
}
