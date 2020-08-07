# IBC specification

This documents how CosmWasm contracts are expected to interact with IBC.

## General Concepts

**IBC Enabled** - when instantiating a contract, we detect if it supports IBC messages.
  We require "feature flags" in the contract/vm handshake to ensure compatibility
  for features like staking or chain-specific extensions. IBC functionality will require
  another "feature flag", and the list of "enabled features" can be returned to the `x/wasm`
  module to control conditional IBC behavior.
  
  If this feature is enabled, it is considered "IBC Enabled", and that info will
  be stored in the ContractInfo. (For mock, we assume all contracts are IBC enabled)
  
Also, please read the [IBC Docs](https://docs.cosmos.network/master/ibc/overview.html)
for detailed descriptions of the terms *Port*, *Client*, *Connection*,
and *Channel*
  
## Overview

We use "One Port per Contract", which is the most straight-forward mapping, treating each contract 
like a module. It does lead to very long portIDs however. Pay special attention to both the Channel establishment 
(which should be compatible with standard ICS20 modules without changes on their part), as well
as how contracts can properly identify their counterparty.

(We considered on port for the `x/wasm` module and multiplexing on it, but [dismissed that idea](#rejected-ideas))

* Upon `Instantiate`, if a contract is *IBC Enabled*, we dynamically 
  bind a port for this contract. The port name is `wasm-<contract address>`,
  eg. `wasm-cosmos1hmdudppzceg27qsuq707tjg8rkgj7g5hnvnw29`
* If a *Channel* is being established with a registered `wasm-xyz` port,
  the `x/wasm.Keeper` will handle this and call into the appropriate
  contract to determine supported protocol versions during the
  [`ChanOpenTry` and `ChanOpenAck` phases](https://docs.cosmos.network/master/ibc/overview.html#channels).
  (See [Channel Handshake Version Negotiation](https://docs.cosmos.network/master/ibc/custom.html#channel-handshake-version-negotiation))
* Both the *Port* and the *Channel* are fully owned by one contract.
* `x/wasm` will allow both *ORDERED* and *UNORDERED* channels and pass that mode
  down to the contract in `OnChanOpenTry`, so the contract can decide if it accepts
  the mode. We will recommend the contract developers stick with *ORDERED* channels
  for custom protocols unless they can reason about async packet timing.
* When sending a packet, the CosmWasm contract must specify the local *ChannelID*.
  As there is a unique *PortID* per contract, that is filled in by `x/wasm`
  to produce the globally unique `(PortID, ChannelID)`
* When receiving a Packet (or Ack or Timeout), the contracts receives the
  *ChannelID* it came from, as well as the packet that was sent by the counterparty.
* When receiving an Ack or Timeout packet, the contract also receives the
  original packet that it sent earlier.
* We do not support multihop packets in this model (they are rejected by `x/wasm`).
  They are currently not fully specified nor implemented in IBC 1.0, so let us
  simplify our model until this is well established

## Workflow

Establishing *Clients* and *Connections* is out of the scope of this
module and must be created by the same means as for `ibc-transfer`
(via the cli or otherwise). `x/wasm` will bind a unique *Port* for each
"IBC Enabled" contract.

For mocks, all the Packet Handling and Channel Lifecycle Hooks are routed 
to some Golang stub handler, but containing the contract address, so we
can perform contract-specific actions for each packet.

### Messages

An "IBC Enabled" contract may dispatch the following messages not available
to other contracts:

* `IBCSendMsg` - this sends an IBC packet over an established channel. 
* `IBCOpenChannel` - given a ConnectionID and a remote Port, this will request
  to open a new channel with the given chain
* `IBCCloseChannel` - given an existing channelID bound to this contract's Port,
  initiate the closing sequence and reject all pending packets.

They are returned from `handle` just like any other `CosmosMsg`
(For mocks, we will trigger this externally, later only valid contract addresses 
should be able to do so).

### Packet Handling

An "IBC Enabled" contract must support the following callbacks from the runtime
(we will likely multiplex many over one wasm export, as we do with handle, but these 
are the different calls we must support):

* `IBCRecvPacket` - when another chain sends a packet destined to
  an IBCEnabled contract, this will be routed to the proper contract
  and call a function exposed for this purpose.
* `IBCPacketAck` - The original sender of `IBCSendMsg` will
  get this callback eventually if the message was
  processed on the other chain (this may be either a success or an error,
  but comes from the app-level protocol, not the IBC protocol).
* `IBCPacketTimeout` - The original sender of `IBCSendMsg` will 
  get this callback eventually if the message failed to be
  processed on the other chain (for timeout, closed channel, or 
  other IBC-level failure)
  
Note: We may add some helpers inside the contract to map `IBCPacketAck` / `IBCPacketTimeout`
to `IBCPacketSucceeded` / `IBCPacketFailed` assuming they use the standard envelope. However,
we decided not to enforce this on the Go-level, to allow contracts to communicate using protocols
that do not use this envelope.

### Channel Lifecycle Hooks

If you look at the [4 step process](https://docs.cosmos.network/master/ibc/overview.html#channels) for
channel handshakes, we simplify this from the view of the contract:

1. Channels *cannot* be opened by external clients, only the contract can initiate opening
   a channel, via an `IBCOpenChannel` message. This means that `ChanOpenInit` does not need to
   call the contract, but just verify that this contract did indeed attempt to open this channel.
2. The counterparty has a chance for version negotiation in `OnChanOpenTry`, where the contract
   can apply custom logic. It provides the protocol versions that the initiating party expects, and 
   the contract can reject the connection or accept it and return the protocol version it will communicate with.
3. `OnChanOpened` is called on the contract for both `OnChanOpenAck` and `OnChanOpenConfirm` containing
   the final version string (counterparty version). This gives a chance to abort the process
   if we realize this doesn't work. Or save the info (we may need to define this channel uses an older version
   of the protocol for example).
4. `OnChanClosed` is called on both sides if the channel is closed for any reason, allowing them to
   perform any cleanup. This will be followed by `IBCPacketTimeout` callbacks for all the in-progress 
   packets that were not processed before it closed, as well as all pending acks (pending the relayer
   to provide that).

We require the following callbacks on the contract

* `OnChanNegotiate` - called on receiving end for `OnChanOpenTry`. 
* `OnChanOpened` - called on both sides after the initial 2 steps have passed to confirm the version used
* `OnChanClosed` - this is called when an existing channel is closed for any reason

Note @cwgoes I am rather confused by 
[the handshake docs](https://docs.cosmos.network/master/ibc/custom.html#channel-handshake-version-negotiation)
In particular "ChanOpenTry callback should verify that the MsgChanOpenTry.Version is valid and that 
MsgChanOpenTry.CounterpartyVersion is valid.". Where does the CounterpartyVersion come from?
I thought the module would accept the Version and decide on it's own CounterpartyVersion. We assume the
relayer makes that decision????

### Queries

We may want to expose some basic queries of IBC State to the contract.
We should check if this is useful to the contract and if it opens up
any possible DoS:

* `GetPortID` - return PortID given a contract address
* `ListChannels` - return a list of all (portID, channelID) pairs
  that are bound to a given port.
* `ListPendingPackets` - given a (portID, channelID) identifier, return all packets
  in that channel, that have been sent by this chain, but for which no acknowledgement 
  or timeout has yet been received

## Contract Details

Here we map out the workflow with the exact arguments passed with those calls 
from the Go side (which we will use with our mock), and then a 
proposal for multiplexing this over fewer wasm exports (define some rust types)

Messages:

```go
package messages

type IBCSendMsg struct {
    // This is our contract-local ID
    ChannelID string
    Msg []byte
    // optional fields (or do we need exactly/at least one of these?)
    TimeoutHeight uint64
    TimeoutTimestamp uint64
}

// note that a contract has exactly one port, so the SourcePortID is implied
type IBCOpenChannel struct {
    ConnectionID string
    // this is the remote port ID, local port ID is implied as contract only has one port
    PortID string
    Version Version
    Order channeltypes.Order
    // TODO: more info??
}

type IBCCloseChannel struct {
    ChannelID string
}

// use type from https://github.com/cosmos/cosmos-sdk/blob/master/x/ibc/03-connection/types/version.go ?
// or is there a better generic model?
type Version string
```

Packet callbacks:

```go
package packets

// for reference: this is more like what we pass to go-cosmwasm
// func (c *mockContract) OnReceive(params cosmwasm2.Env, msg []byte, store prefix.Store, api cosmwasm.GoAPI, 
//         querier keeper.QueryHandler, meter sdk.GasMeter, gas uint64) (*cosmwasm2.OnReceiveIBCResponse, uint64, error) {}
// below is how we want to expose it in x/wasm:

// IBCRecvPacket is called when we receive a packet sent from the other end of a channel
// The response bytes listed here will be returned to the caller as "result" in IBCPacketAck
// What do we do with error?
//
// If we were to assume/enforce an envelope, then we could wrap response/error into the acknowledge packet,
// but we delegated that encoding to the contract
func IBCRecvPacket(ctx sdk.Context, k *wasm.Keeper, contractAddress sdk.AccAddress, env IBCPacketInfo, msg []byte) 
    (response []byte, err error) {}

// how to handle error here? if we push it up the ibc stack and fail the transaction (normal handling),
// the packet may be posted again and again. just log and ignore failures here? what does a failure even mean?
// only realistic one I can imagine is OutOfGas (panic) to retry with higher gas limit.
//
// if there any point in returning a response, what does it mean?
func IBCPacketAck(ctx sdk.Context, k *wasm.Keeper, contractAddress sdk.AccAddress, env IBCPacketInfo, 
    originalMsg []byte, result []byte) error {}

// same question as for IBCPacketAck
func IBCPacketDropped(ctx sdk.Context, k *wasm.Keeper, contractAddress sdk.AccAddress, env IBCPacketInfo, 
    originalMsg []byte, errorMsg string) error {}

// do we need/want all this info?
type IBCPacketInfo struct {
	// local id for the Channel packet was sent/received on
	ChannelID string

    // sequence for the packet (will already be enforced in order if ORDERED channel)
	Sequence uint64
	
    // Note: Timeout if guaranteed ONLY to be exceeded iff we are processing IBCPacketDropped
    // otherwise, this is just interesting metadata

	// block height after which the packet times out 
	TimeoutHeight uint64
	// block timestamp (in nanoseconds) after which the packet times out
	TimeoutTimestamp uint64
}
```

Channel Lifecycle:

```go
package lifecycle

// if this returns error, we reject the channel opening
// otherwise response has the versions we accept
//
// It is provided the full ChannelInfo being proposed to do any needed checks
func OnChanNegotiate(ctx sdk.Context, k *wasm.Keeper, contractAddress sdk.AccAddress, request ChannelInfo) 
    (version Version, err error) {}

// This is called with the full ChannelInfo once the other side has agreed.
// An error here will still abort the handshake process. (so you can double check the order/version here)
//
// The main purpose is to allow the contract to set up any internal state knowing the channel was established,
// and keep a registry with that ChannelID
func OnChanOpened(ctx sdk.Context, k *wasm.Keeper, contractAddress sdk.AccAddress, request ChannelInfo) error {}

// This is called when the channel is closed for any reason
// TODO: any meaning to return an error here? we cannot abort closing a channel
func OnChanClosed(ctx sdk.Context, k *wasm.Keeper, contractAddress sdk.AccAddress, request ChannelClosedInfo) error {}

type ChannelInfo struct {
    // key info to enforce (error if not what is expected)
    Order channeltypes.Order
    // the version info the counterparty contract is proposing / agreed to
    CounterpartyVersion Version
    // local id for the Channel that is being initiated
    ChannelID string
    // these two are taken from channeltypes.Counterparty
    RemotePortID string
    RemoteChannelID string
}

type ChannelClosedInfo struct {
    // local id for the Channel that is being shut down
    ChannelID string
}
```

Queries:

These are callbacks that the contract can make, calling into a QueryPlugin.
The general type definition for a `QueryPlugin` is
`func(ctx sdk.Context, request *wasmTypes.IBCQuery) ([]byte, error)`.
All other info (like the contract address we are querying for) must be passed in
from the contract (which knows it's own address).

Here we just defined the request and response types (which will be serialized into `[]byte`)

```go
package queries

type QueryPort struct {
   ContractAddress string
}

type QueryPortResponse struct {
   PortID string
}

type QueryChannels struct {
    // exactly one of these must be set. ContractAddress is a shortcut to save the Contract->PortID mapping
    PortID string
	ContractAddress string
}

type QueryChannelsResponse struct {
   ChannelIDs []ChannelMetadata
}

type ChannelMetadata struct {
    // Local portID, channelID is our unique identifier
    PortID string
    ChannelID string
    RemotePortID string
    Order channeltypes.Order
    CounterpartyVerson Version
}

type QueryPendingPackets struct {
    // Always required
    ChannelID string

    // exactly one of these must be set. ContractAddress is a shortcut to save the Contract->PortID mapping
    PortID string
	ContractAddress string
}

type QueryPendingPacketsResponse struct {
   Packets []PacketMetadata
}

type PacketMetadata struct {
    // The original (serialized) message we sent
    Msg []byte

    Sequence uint64
	// block height after which the packet times out 
	TimeoutHeight uint64
	// block timestamp (in nanoseconds) after which the packet times out
	TimeoutTimestamp uint64
}
```

### Contract (Wasm) entrypoints

**TODO**

## Future Ideas

Here are some ideas we may add in the future

### Dynamic Ports and Channels

* multiple ports per contract
* elastic ports that can be assigned to different contracts
* transfer of channels to another contract

This is inspired by the Agoric design, but also adds considerable complexity to both the `x/wasm`
implementation as well as the correctness reasoning of any given contract. This will not be
available in the first version of our "IBC Enabled contracts", but we can consider it for later,
if there are concrete user cases that would significantly benefit from this added complexity. 

### Add multihop support

Once the ICS and IBC specs fully establish how multihop packets work, we should add support for that.
Both on setting up the routes with OpenChannel, as well as acting as an intermediate relayer (if that is possible)

## Rejected Ideas
  
### One Port per Module

We decided on "one port per contract", especially after the IBC team raised
the max length on port names to allow `wasm-<bech32 address>` to be a valid port.
Here are the arguments for "one port for x/wasm" vs "one port per contract". Here 
was an alternate proposal:

In this approach, the `x/wasm` module just binds one port to handle all
modules. This can be well defined name like `wasm`. Since we always
have `(ChannelID, PortID)` for routing messages, we can reuse one port
for all contracts as long as we have a clear way to map the `ChannelID`
to a specific contract when it is being established.


* On genesis we bind the port `wasm` for all communication with the `x/wasm`
  module.
* The *Port* is fully owned by `x/wasm`
* Each *Channel* is fully owned by one contract.
* `x/wasm` only accepts *ORDERED Channels* for simplicity of contract
  correctness.

To clarify:

* When a *Channel* is being established with port `wasm`, the
  `x/wasm.Keeper` must be able to identify for which contract this
  is destined. **how to do so**??
  * One idea: the channel name must be the contract address. This means
    (`wasm`, `cosmos13d...`) will map to the given contract in the wasm module.
    The problem with this is that if two contracts from chainA want to
    connect to the same contracts on chainB, they will want to claim the
    same *ChannelID* and *PortID*. Not sure how to differentiate multiple
    parties in this way.
  * Other ideas: have a special field we send on `OnChanOpenInit` that
    specifies the destination contract, and allow any *ChannelID*.
    However, looking at [`OnChanOpenInit` function signature](https://docs.cosmos.network/master/ibc/custom.html#implement-ibcmodule-interface-and-callbacks),
    I don't see a place to put this extra info, without abusing the version field,
    which is a [specified field](https://docs.cosmos.network/master/ibc/custom.html#channel-handshake-version-negotiation):
    ```
    Versions must be strings but can implement any versioning structure. 
    If your application plans to have linear releases then semantic versioning is recommended.
    ... 
    Valid version selection includes selecting a compatible version identifier with a subset 
    of features supported by your application for that version.
    ...    
    ICS20 currently implements basic string matching with a
    single supported version.
    ```