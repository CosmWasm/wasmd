# Integration

If you want to use Wasm in your own app, here is how you can get this working
quickly and easily. First, check to make sure you fit the pre-requisites,
then integrate the `x/wasm` module as described below, and finally, you
can add custom messages and queries to your custom Go/SDK modules, exposing
them to any chain-specific contract.

## Prerequisites

The pre-requisites of integrating `x/wasm` into your custom app is to be using 
a compatible version of the Cosmos SDK, and to accept some limits to the
hardware it runs on.

| wasmd | Cosmos SDK |
|:-----:|:----------:|
| v0.7 | v0.38 |
| v0.8 | v0.38 |

We currently only support Intel/AMD64 CPUs and OSX or Linux (with glibc - not muslc like alpine).
This limit comes from the Rust dll we use to run the wasm code. There are open issues
for adding [ARM support](https://github.com/CosmWasm/go-cosmwasm/issues/53),
adding [Windows support](https://github.com/CosmWasm/go-cosmwasm/issues/28),
and [compiling static binaries](https://github.com/CosmWasm/go-cosmwasm/issues/45), so it works on both glibc and muslc systems.
However, these issues are not high on the roadmap and unless you are championing
them, please count on the current limits for the near future.

## Quick Trial

The simplest way to try out CosmWasm is simply to run `wasmd` out of the box,
and focus on writing, uploading, and using your custom contracts. There is
plenty that can be done there, and lots to learn. 

Once you are happy with it and want to use a custom Cosmos SDK app, 
you may consider simply forking `wasmd`. *I highly advise against this*. 
You should try one of the methods below.

## Integrating wasmd

### As external module

The simplest way to use `wasmd` is just to import `x/wasm` and wire it up
in `app.go`.  You now have access to the whole module and you custom modules
running side by side. (But the CosmWasm contracts will only have access
to `bank` and `staking`... more below on [customization](#Adding-Custom-Hooks)).

The requirement here is that you have imported the standard sdk modules
from the Cosmos SDK, and enabled them in `app.go`. If so, you can just look
at [`wasmd/app/app.go`](https://github.com/CosmWasm/wasmd/blob/master/app/app.go#)
for how to do so (just search there for lines with `wasm`).

### Copied into your app

Sometimes, however, you will need to copy `x/wasm` into your app. This should
be in limited cases, and makes upgrading more difficult, so please take the
above path if possible. This is required if you have either disabled some key
SDK modules in your app (eg. using PoA not staking and need to disable those
callbacks and feature support), or if you have copied in the core `x/*` modules
from the Cosmos SDK into your application and customized them somehow.

In either case, your best approach is to copy the `x/wasm` module from the
latest release into your application. Your goal is to make **minimal changes**
in this module, and rather add your customizations in a separate module.
This is due to the fact that you will have to copy and customize `x/wasm`
from upstream on all future `wasmd` releases, and this should be as simple
as possible.

If, for example, you have forked the standard SDK libs, you just want to
change the imports (from eg. `github.com/cosmos/cosmos-sdk/x/bank` to
`github.com/YOUR/APP/x/bank`), and adjust any calls if there are compiler 
errors due to differing APIs (maybe you use Decimals not Ints for currencies?).

By the end of this, you should be able to run the standard CosmWasm contracts
in your application, alongside all your custom logic.

## Adding custom hooks

Once you have gotten this integration working and are happy with the
flexibility it offers you, you will probably start wishing for deeper
integration with your custom SDK modules. "It sure it nice to have custom
tokens with a bonding curve from my native token, but I would love
to trade them on the exchange I wrote as a Go module. Or maybe use them
to add options to the exchange."

At this point, you need to dig down deeper and see how you can add this
power without forking either CosmWasm or `wasmd`. 

### Calling contracts from native code

This is perhaps the easiest part. Let's say your native exchange module
wants to call into a token that lives as a CosmWasm module. You need to
pass the `wasm.Keeper` into your `exchange.Keeper`. If you know the format
for sending messages and querying the contract (exported as json schema 
from each contract), and have a way of configuring addresses of supported
token contracts, your exchange code can simply call `wasm.Keeper.Execute`
with a properly formatted message to move funds, or `wasm.Keeper.SmartQuery`
to check balances.

If you look at the unit tests in [`x/wasm/internal/keeper`](https://github.com/CosmWasm/wasmd/tree/master/x/wasm/internal/keeper),
it should be pretty straight forward.

### Extending the Contract Interface

If you want to let the contracts access your native modules, the first
step is to define a set of Messages and Queries that you want to expose,
and then add them as `CosmosMsg::Custom` and `QueryRequest::Custom`
variants. You can see an example of the [bindings for Terra](https://github.com/CosmWasm/terra-contracts/tree/master/packages/bindings).

Once you have those bindings, use them to build a 
[simple contact using much of the API](https://github.com/CosmWasm/terra-contracts/tree/master/contracts/maker).
Don't worry too much about the details, this should be usable, but mainly
you will want to upload it to your chain and use for integration tests
with your native Cosmos SDK modules. Once that is solid, then add more
and more complex contracts.

You will then likely want to add a `mocks` package so you can provide
mocks for the functionality of your native modules when unit testing
the contracts (provide static data for exchange rates when your contracts
query it). You can see an example of [mocks for Terra contracts](https://github.com/CosmWasm/terra-contracts/tree/master/packages/mocks) 

### Calling into the SDK

Before I show how this works, I want to remind you, if you have copied `x/wasm`,
please **do not make these changes to `x/wasm`**.

We will add a new module, eg. `x/contracts`, that will contain custom
bindings between CosmWasm contracts and your native modules.

TODO

### Wiring it all together

TODO