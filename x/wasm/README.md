# Wasm Module

This should be a brief overview of the functionality

## Configuration

You can add the following section to `config/app.toml`. Below is shown with defaults:

```toml
[wasm]
# This is the maximum sdk gas (wasm and storage) that we allow for any x/wasm "smart" queries
query_gas_limit = 300000
# This is the number of wasm vm instances we keep cached in memory for speed-up
# Warning: this is currently unstable and may lead to crashes, best to keep for 0 unless testing locally
lru_size = 0
```

## Events

A number of events are returned to allow good indexing of the transactions from smart contracts.

Every call to Instantiate or Execute will be tagged with the info on the contract that was executed and who executed it.
It should look something like this (with different addresses). The module is always `wasm`, and `code_id` is only present
when Instantiating a contract, so you can subscribe to new instances, it is omitted on Execute:

```json
{
    "Type": "message",
    "Attr": [
        {
            "key": "module",
            "value": "wasm"
        },
        {
            "key": "signer",
            "value": "cosmos1qua29gv7fqy46q6rnwn66mw35shu7rq80p2hth"
        },
        {
            "key": "code_id",
            "value": "1"
        },
        {
            "key": "contract_address",
            "value": "cosmos18vd8fpwxzck93qlwghaj6arh4p7c5n89uzcee5"
        }
    ]
}
```

If any funds were transferred to the contract as part of the message, or if the contract released funds as part of it's executions,
it will receive the typical events associated with sending tokens from bank:

```json
{
    "Type": "transfer",
    "Attr": [
        {
            "key": "recipient",
            "value": "cosmos18vd8fpwxzck93qlwghaj6arh4p7c5n89uzcee5 /* this is who got the funds */"
        },
        {
            "key": "amount",
            "value": "100000denom"
        }
    ]
    },
    {
    "Type": "message",
    "Attr": [
        {
            "key": "sender",
            "value": "cosmos18kwmngcpvrdl8dwmwjgncfwskxs3wtge2tgpdr /* this is who sent the funds */"
        }
    ]
}
```

This is actually not very ergonomic, as the "sender" (account that sent the funds) is separated from the actual transfer as two separate
events, and this may cause confusion, especially if the sender moves funds to the contract and the contract to another recipient in the
same transaction.

Finally, the contract itself can emit a "custom event" on Execute only (not on Init).
There is one event per contract, so if one contract calls a second contract, you may receive
one event for the original contract and one for the re-invoked contract. All attributes from the contract are passed through verbatim,
and we add a `contract_address` attribute that contains the actual contract that emitted that event.
Here is an example from the escrow contract successfully releasing funds to the destination address:

```json
{
    "Type": "wasm",
    "Attr": [
        {
            "key": "contract_address",
            "value": "cosmos18vd8fpwxzck93qlwghaj6arh4p7c5n89uzcee5"
        },
        {
            "key": "action",
            "value": "release"
        },
        {
            "key": "destination",
            "value": "cosmos1nrskytrh6ce26zk8zqgh6gtmzrc22kd95ljkqp"
        }
    ]
}
```

### Pulling this all together

We will invoke an escrow contract to release to the designated beneficiary.
We send `5000denom` along with the `MsgExecuteContract` and the contract releases `` to the beneficiary.
We will see all the following events:

```json
"TODO"
```

## Messages

TODO

## CLI

TODO

## Rest

TODO
