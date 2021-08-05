# Event System

## Usage in the SDK

Events are an essential part of the Cosmos SDK. They are similar to "logs" in Ethereum and allow a blockchain
app to attach key-value pairs to a transaction that can later be used to search for it or extract some information
in human readable form. Events are not written to the application state, nor do they form part of the AppHash,
but mainly intended for client use (and become an essential API for any reactive app or app that searches for txs). 

In contrast, transactions also have a binary "data" field that is part of the AppHash (provable with light client proofs,
part of consensus). This data is not searchable, but given a tx hash, you can be gauranteed what the data returned is.
This is often empty, but sometimes custom protobuf formats to return essential information from an execution.

Every message in the SDK may add events to the EventManager and these are then added to the final ABCI result that is returned
to Tendermint. Events are exposed in 3 different ways over the Tendermint API (which is the only way a client can query).
First of all is the `events` field on the transaction result (when you query a transaction by hash, you can see all event emitted
by it). Secondly is the `log` field on the same transaction result. And third is the query interface to search or subscribe for
transactions. 

The `log` field actually has the best data. It contains an array of array of events. The first array is one entry per incoming message.
Transactions in the Cosmos SDK may consist of multiple messages that are executed atomically. Maybe we send tokens, then issue a swap
on a DEX. Each action would return it's own list of Events and in the logs, these are separated. For each message, it maintains a list
of Events, exactly in the order returned by the application. This is JSON encoded and can be parsed by a client.

In Tendermint 0.35, the `events` field will be one flattened list of events over all messages. Just appending the lists returned
from each message. However, currently (until Tendermint 0.34 used in Cosmos SDK 0.40-0.43), they are flattened on type. Meaning all events
with type `wasm` get merged into one. This makes the API not very useful to understanding more complex events currently. (TODO: link PR fixing this)

In the search/subscribe interface, you can query for transactions by `AND`ing a number of conditions. Each is expressed like
`<type>.<key>=<value>`. For example, `message.signer=cosmos1234567890`. It will return all transactions that emitted an event matching this filter.

### Examples

TODO: show event structure.

TODO: contrast flattened/unflattened events

### Standard Events in the SDK

TODO: what is added by the AnteHandlers (message.signer? auth?)

TODO: what is emitted by bank (transfer event), as this is a very important base event

## Usage in wasmd

In `x/wasm` we also use Events system. On one hand, `x/wasm` emits standard event for each message it processes to convey,
for example, "uploaded code, id 6" or "executed code, address wasm1234567890". Furthermore, it allows contracts to
emit custom events based on their execution state, so they can for example say "dex swap, BTC-ATOM, in 0.23, out 512"
which require internal knowledge of the contract and is very useful for custom dApp UIs.

In addition, when a smart contract executes a SubMsg and processes the reply, it receives not only the `data` response
from the message exection, but also the list of events 

### Standard Events in x/wasm

TODO: document what we emit in `x/wasm` regardless of the contract return results

### Emitted Custom Events from a Contract

TODO: document how we process attributes and events fields in the Response.

* Base event `wasm`
* Event name mangling (prepend `wasm-`)
* Append trusted attribute _contract_address
* Validation requirements (_ reserved, non-empty, min-length 2 for type)

## Event Details for wasmd

Beyond the basic Event system and emitted events, we must handle more advanced cases in `x/wasm`
and thus add some more logic to the event processing. Remember that CosmWasm contracts dispatch other
messages themselves, so far from the flattened event structure, or even a list of list (separated
by message index in the tx), we actually have a tree of messages, each with their own events. And we must
flatten that in a meaningful way to feed it into the event system.

Furthermore, with the sub-message reply handlers, we end up with eg. "Contract A execute", "Contract B execute",
"Contract A reply". If we return all events by all of these, we may end up with many repeated event types and
a confusing results, especially for Tendermint 0.34 where they are merged together.

While designing this, we wish to make something that is usable with Tendermint 0.34, but focus on using the
behavior of Tendermint 0.35+ (which is the same behavior as we have internally in the SDK... submessages
all have their own list of Events). Thus, we may emit more events than in previous wasmd versions (as we assume
they will be returned in an ordered list rather than merged).

### Combining Events from Sub-Messages

TODO
### Exposing Events to Reply

TODO
