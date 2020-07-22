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
(via the cli or otherwise). `x/wasm` will bind one or more *Ports* to
handle messages for the contracts. How *Channels* are established
is an open question for discussion.

The following actions are only available for `IBCEnabled` contracts:

* Dispatch `IBCSendMsg` - this sends an IBC packet over an established
  channel. This is returned from `handle` just like any other `CosmosMsg`
  (For mocks, we will trigger this externally, later only
  valid contract addresses should be able to do so).
* Handle `IBCRecvPacket` - when another chain sends a packet destined to
  an IBCEnabled contract, this will be routed to the proper contract
  and call an exposed function for this purpose.
* Handle `IBCAckPacket` - The original sender of `IBCSendMsg` will
  get this callback eventually if the message was successfully
  processed on the other chain
* Handle `IBCErrorPacket` - The original sender of `IBCSendMsg` will 
  get this callback eventually if the message failed to be
  processed on the other chain (for any reason)

For mocks, all the `IBCxxxPacket` commands are routed to some
Golang stub handler, but containing the contract address, so we
can perform contract-specific actions for each packet.

There are open questions on how exactly the various `IBCxxPacket`
messages are routed to a contract, as well as related to the
channel lifecycle. (Can a contract open a channel? How do we
allow contracts to participate in protocol version negotiation
when establishing a channel?)

## Ports and Channels

There are two main proposals on how to map these concepts to
contracts. We will describe both in detail, with arguments
pro and contra, to select one approach.

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
* `x/wasm` only accepts *ORDERED Channels* for simplicity of contract
  correctness.
* When sending a packet, the CosmWasm contract must specify the *ChannelID*
  (and remote *PortID*?), which is generally provided by the external client
  which understands which channels it wishes to communicate over.
* When sending a packet, the contract can set a custom identifier (of
  max 64 characters) that it can use to look this up later.
* When receiving a Packet (or Ack or Error), the contracts receives the
  *ChannelID* as well as remote *PortID* (and *ClientID*???) that is
  communicating.
* When receiving an Ack or Error packet, the contract also receives the
  same identifier that it set on Send (`x/wasm` handles mapping between this
  and the native IBC packet IDs).
  
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