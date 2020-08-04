# IBC specification

This documents how CosmWasm contracts are expected to interact with IBC.

## General Concepts

**IBC Enabled** - when instantiating a contract, we detect if it supports IBC messages
  (this is an optional extension). If so, it is considered "IBC Enabled".
  This info should be public on all contracts. (For mock, we assume all
  contracts are IBC enabled)
  
Also, please read the [IBC Docs](https://docs.cosmos.network/master/ibc/overview.html)
for detailed descriptions of the terms *Port*, *Client*, *Connection*,
and *Channel*
  
## Workflow

Establishing *Clients* and *Connections* is out of the scope of this
module and must be created by the same means as for `ibc-transfer`
(via the cli or otherwise). `x/wasm` will bind a unique *Port* for each
"IBC Enabled" contract.

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

An "IBC Enabled" contract must support the following callbacks from the runtime
(we will likely multiplex many over one wasm export, as we do with handle, but these 
are the different calls we must support):

Packet Lifecycle:

* `IBCRecvPacket` - when another chain sends a packet destined to
  an IBCEnabled contract, this will be routed to the proper contract
  and call an exposed function for this purpose.
* `IBCPacketAck` - The original sender of `IBCSendMsg` will
  get this callback eventually if the message was successfully
  processed on the other chain (this implies using an envelope for the 
  IBC acknowledge message, so the Go code can differentiate between success and error)
* `IBCPacketFailed` - The original sender of `IBCSendMsg` will 
  get this callback eventually if the message failed to be
  processed on the other chain (for timeout, closed channel, or the 
  other side returning an error message in the IBC acknowledgement)

Channel Lifecycle Hooks:

* `OnChanOpenTry` - this is called when another chain attempts to open
  a channel with this contract. It provides the protocol versions that
  the initiate expects, and the contract can reject it or accept it
  and return the protocol version it will communicate with.
* `OnChanOpenConfirm` - this is called when the other party has agreed
  to the channel and all negotiation is over. This will be called exactly
  once for any established channel, and the contract should record these
  open channels in the internal state.
* `OnChanClosed` - this is called when an existing channel is closed
  for any reason, allowing the client to update the state there.
  This will (likely? @cwgoes?) be accompanied by `IBCPacketFailed`
  callbacks for all the in-progress packets that were not processed before
  it closed, as well as all pending acks.

We may want to expose some basic queries of IBC State to the contract.
We should check if this is useful to the contract and if it opens up
any possible DoS:

* `ListMyChannels` - return a list of all channel IDs (along with remote PortID)
  that are bound to this contract's Port.
* `ListPendingPackets` - given a channelID owned by the contract, return all packets
  in that channel, that were sent by this chain, for which no acknowledgement or timeout
  has been received

For mocks, all the `IBCxxxPacket` commands are routed to some
Golang stub handler, but containing the contract address, so we
can perform contract-specific actions for each packet.

## Contract Details

Here we map out the workflow with the exact arguments passed with those calls 
from the Go side (which we will use with our mock), and then a 
proposal for multiplexing this over fewer wasm exports (define some rust types)

Messages:

```go
type IBCSendMsg struct {
    ChannelID string
    RemotePort string // do we need both? isn't channel enough once it is established???
    Msg []byte
    // optional fields
    TimeoutHeight uint64
    TimeoutTimestamp uint64
}

// note that a contract has exactly one port, so the SourcePortID is implied
type IBCOpenChannel struct {
    ConnectionID string
    // this is the remote port ID
    PortID string
    Versions []Version
    // more info??
}

type IBCCloseChannel struct {
    ChannelID string
    // more info??
}

// use type from https://github.com/cosmos/cosmos-sdk/blob/master/x/ibc/03-connection/types/version.go ?
// or is there a better generic model?
type Version string
```

Packet callbacks:

```go
// for reference: this is more like what we pass to go-cosmwasm
// func (c *mockContract) OnReceive(params cosmwasm2.Env, msg []byte, store prefix.Store, api cosmwasm.GoAPI, 
//         querier keeper.QueryHandler, meter sdk.GasMeter, gas uint64) (*cosmwasm2.OnReceiveIBCResponse, uint64, error) {}

// we can imagine it like this in go
// an error here will not be passed up to the transaction level, but will be encoded in an envelope
// and send packet as an IBCAckPacket, to trigger IBCPacketFailed on the other end
func (c *mockContract) IBCRecvPacket(ctx sdk.Context, k *wasm.Keeper, env IBCInfo, msg []byte) (response []byte, err error) {}

// how to handle error here? if we push it up the ibc stack and fail the transaction (normal handling),
// the packet may be posted again and again. just log and ignore failures here? what does a failure even mean?
func (c *mockContract) IBCPacketAck(ctx sdk.Context, k *wasm.Keeper, env IBCInfo, originalMsg []byte, result []byte) (response []byte, err error) {}

// same question as for IBCPacketAck
func (c *mockContract) IBCPacketFailed(ctx sdk.Context, k *wasm.Keeper, env IBCInfo, originalMsg []byte, errorMsg string) (response []byte, err error) {}

type IBCInfo struct {
	// PortID of the remote contract???
	RemotePortID string
	// PortID of the our contract???
	OurPortID string
	// ChannelID packet was sent on
	ChannelID string
	Packet    *IBCPacketInfo `json:"packet,omitempty"`
}

// do we need/want all this info?
type IBCPacketInfo struct {
	Sequence uint64
	// identifies the port on the sending chain.
	SourcePort string
	// identifies the channel end on the sending chain.
	SourceChannel string
	// block height after which the packet times out
	TimeoutHeight uint64
	// block timestamp (in nanoseconds) after which the packet times out
	TimeoutTimestamp uint64
}
```

Channel Lifecycle:

**TODO**

Queries:

**TODO**

### Contract (Wasm) entrypoints

**TODO**

## Ports and Channels Discussion

We decided on "one port per contract", especially after the IBC team raised
the max length on port names to allow `wasm-<bech32 address>` to be a valid port.
Here are the arguments for "one port for x/wasm" vs "one port per contract"

### One Port per Contract

This may be the most straight-forward mapping, treating each contract 
like a module. It does lead to very long portIDs however. Make special
attention to both the Channel establishment (which should be compatible
with standard ICS20 modules without changes on their part), as well
as how contracts can properly identify their counterparty.

* Upon `Instantiate`, if a contract is *IBC Enabled*, we dynamically 
  bind a port for this contract. The port name is `wasm-<contract address>`,
  eg. `wasm-cosmos1hmdudppzceg27qsuq707tjg8rkgj7g5hnvnw29`
* If a *Channel* is being established with a registered `wasm-xyz` port,
  the `x/wasm.Keeper` will handle this and call into the appropriate
  contract to determine supported protocol versions during the
  [`ChanOpenTry` and `ChanOpenAck` phases](https://docs.cosmos.network/master/ibc/overview.html#channels).
  (See [Channel Handshake Version Negotiation](https://docs.cosmos.network/master/ibc/custom.html#channel-handshake-version-negotiation))
* Both the *Port* and the *Channel* are fully owned by one contract.
* Question: `x/wasm` only accepts *ORDERED Channels* for simplicity of contract
  correctness? Or should we allow the contract to decide?
* When sending a packet, the CosmWasm contract must specify the *ChannelID*
  and remote *PortID*, which is generally provided by the external client
  which understands which channels it wishes to communicate over.
* When sending a packet, the contract can set a custom identifier (of
  max 64 characters) that it can use to look this up later.
* When receiving a Packet (or Ack or Error), the contracts receives the
  *ChannelID* as well as remote *PortID* (and *ClientID*???) that is
  communicating.
* When receiving an Ack or Error packet, the contract also receives the
  original packet that it set on Send 
  
### One Port per Module

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