package keeper

import (
	"fmt"
	"sort"
	"strings"

	wasmvmtypes "github.com/CosmWasm/wasmvm/v3/types"
	abci "github.com/cometbft/cometbft/abci/types"

	errorsmod "cosmossdk.io/errors"
	storetypes "cosmossdk.io/store/types"

	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	"github.com/CosmWasm/wasmd/x/wasm/keeper/wasmtesting"
	"github.com/CosmWasm/wasmd/x/wasm/types"
)

var (
	_ Messenger = &wasmtesting.MockMessageHandler{}
	_ Messenger = MessageHandlerChain{}
	_ Messenger = SDKMessageHandler{}
)

// Messenger is an extension point for custom wasmd message handling
type Messenger interface {
	// DispatchMsg encodes the wasmVM message and dispatches it.
	DispatchMsg(ctx sdk.Context, contractAddr sdk.AccAddress, contractIBCPortID string, msg wasmvmtypes.CosmosMsg) (events []sdk.Event, data [][]byte, msgResponses [][]*codectypes.Any, err error)
}

// replyer is a subset of keeper that can handle replies to submessages
type replyer interface {
	reply(ctx sdk.Context, contractAddress sdk.AccAddress, reply wasmvmtypes.Reply) ([]byte, error)
}

// MessageDispatcher coordinates message sending and submessage reply/ state commits
type MessageDispatcher struct {
	messenger Messenger
	keeper    replyer
}

// NewMessageDispatcher constructor
func NewMessageDispatcher(messenger Messenger, keeper replyer) *MessageDispatcher {
	return &MessageDispatcher{messenger: messenger, keeper: keeper}
}

// DispatchMessages sends all messages.
func (d MessageDispatcher) DispatchMessages(ctx sdk.Context, contractAddr sdk.AccAddress, ibcPort string, msgs []wasmvmtypes.CosmosMsg) error {
	for _, msg := range msgs {
		events, _, _, err := d.messenger.DispatchMsg(ctx, contractAddr, ibcPort, msg)
		if err != nil {
			return err
		}
		// redispatch all events, (type sdk.EventTypeMessage will be filtered out in the handler)
		ctx.EventManager().EmitEvents(events)
	}
	return nil
}

// dispatchMsgWithGasLimit sends a message with gas limit applied
func (d MessageDispatcher) dispatchMsgWithGasLimit(ctx sdk.Context, contractAddr sdk.AccAddress, ibcPort string, msg wasmvmtypes.CosmosMsg, gasLimit uint64) (events []sdk.Event, data [][]byte, msgResponses [][]*codectypes.Any, err error) {
	limitedMeter := storetypes.NewGasMeter(gasLimit)
	subCtx := ctx.WithGasMeter(limitedMeter)

	// catch out of gas panic and just charge the entire gas limit
	defer func() {
		if r := recover(); r != nil {
			if _, ok := r.(storetypes.ErrorOutOfGas); ok {
				// consume the gas limit for the submessage and turn panic into error
				ctx.GasMeter().ConsumeGas(gasLimit, "Sub-Message OutOfGas panic")
				err = errorsmod.Wrap(sdkerrors.ErrOutOfGas, "SubMsg hit gas limit")
			} else {
				// if it's not an ErrorOutOfGas, consume the gas used in the sub-context and raise it again
				spent := subCtx.GasMeter().GasConsumed()
				ctx.GasMeter().ConsumeGas(spent, "From limited Sub-Message")
				// log it to get the original stack trace somewhere (as panic(r) keeps message but stacktrace to here
				moduleLogger(ctx).Info("SubMsg rethrowing panic: %#v", r)
				panic(r)
			}
		}
	}()
	events, data, msgResponses, err = d.messenger.DispatchMsg(subCtx, contractAddr, ibcPort, msg)

	// make sure we charge the parent what was spent
	spent := subCtx.GasMeter().GasConsumed()
	ctx.GasMeter().ConsumeGas(spent, "From limited Sub-Message")

	return events, data, msgResponses, err
}

// DispatchSubmessages builds a sandbox to execute these messages and returns the execution result to the contract
// that dispatched them, both on success as well as failure
func (d MessageDispatcher) DispatchSubmessages(ctx sdk.Context, contractAddr sdk.AccAddress, ibcPort string, msgs []wasmvmtypes.SubMsg) ([]byte, error) {
	var rsp []byte
	for _, msg := range msgs {
		switch msg.ReplyOn {
		case wasmvmtypes.ReplySuccess, wasmvmtypes.ReplyError, wasmvmtypes.ReplyAlways, wasmvmtypes.ReplyNever:
		default:
			return nil, errorsmod.Wrap(types.ErrInvalid, "replyOn value")
		}
		// first, we build a sub-context which we can use inside the submessages
		subCtx, commit := ctx.CacheContext()
		em := sdk.NewEventManager()
		subCtx = subCtx.WithEventManager(em)

		// check how much gas left locally, optionally wrap the gas meter
		gasRemaining := ctx.GasMeter().Limit() - ctx.GasMeter().GasConsumed()
		limitGas := msg.GasLimit != nil && (*msg.GasLimit < gasRemaining)

		var err error
		var events []sdk.Event
		var data [][]byte
		var msgResponses [][]*codectypes.Any
		if limitGas {
			events, data, msgResponses, err = d.dispatchMsgWithGasLimit(subCtx, contractAddr, ibcPort, msg.Msg, *msg.GasLimit)
		} else {
			events, data, msgResponses, err = d.messenger.DispatchMsg(subCtx, contractAddr, ibcPort, msg.Msg)
		}

		// if it succeeds, commit state changes from submessage, and pass on events to Event Manager
		var filteredEvents []sdk.Event
		if err == nil {
			commit()
			filteredEvents = filterEvents(append(em.Events(), events...))
			ctx.EventManager().EmitEvents(filteredEvents)
			if msg.Msg.Wasm == nil {
				filteredEvents = []sdk.Event{}
			} else {
				for _, e := range filteredEvents {
					attributes := e.Attributes
					sort.SliceStable(attributes, func(i, j int) bool {
						return strings.Compare(attributes[i].Key, attributes[j].Key) < 0
					})
				}
			}
		} // on failure, revert state from sandbox, and ignore events (just skip doing the above)

		// we only callback if requested. Short-circuit here the cases we don't want to
		if (msg.ReplyOn == wasmvmtypes.ReplySuccess || msg.ReplyOn == wasmvmtypes.ReplyNever) && err != nil {
			return nil, err
		}
		if msg.ReplyOn == wasmvmtypes.ReplyNever || (msg.ReplyOn == wasmvmtypes.ReplyError && err == nil) {
			continue
		}

		// otherwise, we create a SubMsgResult and pass it into the calling contract
		var result wasmvmtypes.SubMsgResult
		if err == nil {
			// just take the first one for now if there are multiple sub-sdk messages
			// and safely return nothing if no data
			var responseData []byte
			if len(data) > 0 {
				responseData = data[0]
			}

			// For msgResponses we flatten the nested list into a flat list. In the majority of cases
			// we only expect one message to be emitted and one response per message. But it might be possible
			// to create multiple SDK messages from one CosmWasm message or we have multiple responses for one message.
			// See https://github.com/CosmWasm/cosmwasm/issues/2009 for more information.
			var msgResponsesFlattened []wasmvmtypes.MsgResponse
			for _, singleMsgResponses := range msgResponses {
				for _, singleMsgResponse := range singleMsgResponses {
					msgResponsesFlattened = append(msgResponsesFlattened, wasmvmtypes.MsgResponse{
						TypeURL: singleMsgResponse.TypeUrl,
						Value:   singleMsgResponse.Value,
					})
				}
			}

			result = wasmvmtypes.SubMsgResult{
				Ok: &wasmvmtypes.SubMsgResponse{
					Events:       sdkEventsToWasmVMEvents(filteredEvents),
					Data:         responseData,
					MsgResponses: msgResponsesFlattened,
				},
			}
		} else {
			// Issue #759 - we don't return error string for worries of non-determinism
			moduleLogger(ctx).Debug("Redacting submessage error", "cause", err)
			result = wasmvmtypes.SubMsgResult{
				Err: redactError(err).Error(),
			}
		}

		// now handle the reply, we use the parent context, and abort on error
		reply := wasmvmtypes.Reply{
			ID:      msg.ID,
			Result:  result,
			Payload: msg.Payload,
		}

		// we can ignore any result returned as there is nothing to do with the data
		// and the events are already in the ctx.EventManager()
		rspData, err := d.keeper.reply(ctx, contractAddr, reply)
		switch {
		case err != nil:
			return nil, errorsmod.Wrap(err, "reply")
		case rspData != nil:
			rsp = rspData
		}
	}
	return rsp, nil
}

// Issue #759 - we don't return error string for worries of non-determinism
func redactError(err error) error {
	// Do not redact system errors
	// SystemErrors must be created in x/wasm and we can ensure determinism
	if wasmvmtypes.ToSystemError(err) != nil {
		return err
	}

	// If it is a DeterministicError, we can safely return it without redaction.
	// We only check the top level error to avoid changes in the error chain becoming
	// consensus-breaking.
	if _, ok := err.(types.DeterministicError); ok {
		return err
	}

	// FIXME: do we want to hardcode some constant string mappings here as well?
	// Or better document them? (SDK error string may change on a patch release to fix wording)
	// sdk/11 is out of gas
	// sdk/5 is insufficient funds (on bank send)
	// (we can theoretically redact less in the future, but this is a first step to safety)
	codespace, code, _ := errorsmod.ABCIInfo(err, false)
	return fmt.Errorf("codespace: %s, code: %d", codespace, code)
}

func filterEvents(events []sdk.Event) []sdk.Event {
	// pre-allocate space for efficiency
	res := make([]sdk.Event, 0, len(events))
	for _, ev := range events {
		if ev.Type != "message" {
			res = append(res, ev)
		}
	}
	return res
}

func sdkEventsToWasmVMEvents(events []sdk.Event) []wasmvmtypes.Event {
	res := make([]wasmvmtypes.Event, len(events))
	for i, ev := range events {
		res[i] = wasmvmtypes.Event{
			Type:       ev.Type,
			Attributes: sdkAttributesToWasmVMAttributes(ev.Attributes),
		}
	}
	return res
}

func sdkAttributesToWasmVMAttributes(attrs []abci.EventAttribute) []wasmvmtypes.EventAttribute {
	res := make([]wasmvmtypes.EventAttribute, len(attrs))
	for i, attr := range attrs {
		res[i] = wasmvmtypes.EventAttribute{
			Key:   attr.Key,
			Value: attr.Value,
		}
	}
	return res
}
