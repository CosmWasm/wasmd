# Wasm Module

This should be a brief overview of the functionality

## Configuration

You can add the following section to `config/app.toml`:

```toml
[wasm]
# This is the maximum sdk gas (wasm and storage) that we allow for any x/wasm "smart" queries
query_gas_limit = 300000
# This defines the memory size for Wasm modules that we can keep cached to speed-up instantiation
# The value is in MiB not bytes
memory_cache_size = 300
```

The values can also be set via CLI flags on with the `start` command:
```shell script
--wasm.memory_cache_size uint32     Sets the size in MiB (NOT bytes) of an in-memory cache for wasm modules. Set to 0 to disable. (default 100)
--wasm.query_gas_limit uint         Set the max gas that can be spent on executing a query with a Wasm contract (default 3000000)
```

## Events

A number of events are returned to allow good indexing of the transactions from smart contracts.

Every call to Instantiate or Execute will be tagged with the info on the contract that was executed and who executed it.
It should look something like this (with different addresses). The module is always `wasm`, and `code_id` is only present
when Instantiating a contract, so you can subscribe to new instances, it is omitted on Execute. There is also an `action` tag
which is auto-added by the Cosmos SDK and has a value of either `store-code`, `instantiate` or `execute` depending on which message
was sent:

```json
{
    "Type": "message",
    "Attr": [
        {
            "key": "module",
            "value": "wasm"
        },
        {
            "key": "action",
            "value": "instantiate"
        },
        {
            "key": "signer",
            "value": "cosmos1vx8knpllrj7n963p9ttd80w47kpacrhuts497x"
        },
        {
            "key": "code_id",
            "value": "1"
        },
        {
            "key": "_contract_address",
            "value": "cosmos14hj2tavq8fpesdwxxcu44rty3hh90vhujrvcmstl4zr3txmfvw9s4hmalr"
        }
    ]
}
```

If any funds were transferred to the contract as part of the message, or if the contract released funds as part of it's executions,
it will receive the typical events associated with sending tokens from bank. In this case, we instantiate the contract and
provide a initial balance in the same `MsgInstantiateContract`. We see the following events in addition to the above one:

```json
[
    {
        "Type": "transfer",
        "Attr": [
            {
                "key": "recipient",
                "value": "cosmos14hj2tavq8fpesdwxxcu44rty3hh90vhujrvcmstl4zr3txmfvw9s4hmalr"
            },
            {
                "key": "sender",
                "value": "cosmos1ffnqn02ft2psvyv4dyr56nnv6plllf9pm2kpmv"
            },
            {
                "key": "amount",
                "value": "100000denom"
            }
        ]
    }
]
```

Finally, the contract itself can emit a "custom event" on Execute only (not on Init).
There is one event per contract, so if one contract calls a second contract, you may receive
one event for the original contract and one for the re-invoked contract. All attributes from the contract are passed through verbatim,
and we add a `_contract_address` attribute that contains the actual contract that emitted that event.
Here is an example from the escrow contract successfully releasing funds to the destination address:

```json
{
    "Type": "wasm",
    "Attr": [
        {
            "key": "_contract_address",
            "value": "cosmos14hj2tavq8fpesdwxxcu44rty3hh90vhujrvcmstl4zr3txmfvw9s4hmalr"
        },
        {
            "key": "action",
            "value": "release"
        },
        {
            "key": "destination",
            "value": "cosmos14k7v7ms4jxkk2etmg9gljxjm4ru3qjdugfsflq"
        }
    ]
}
```

### Pulling this all together

We will invoke an escrow contract to release to the designated beneficiary.
The escrow was previously loaded with `100000denom` (from the above example).
In this transaction, we send `5000denom` along with the `MsgExecuteContract`
and the contract releases the entire funds (`105000denom`) to the beneficiary.

We will see all the following events, where you should be able to reconstruct the actions
(remember there are two events for each transfer). We see (1) the initial transfer of funds
to the contract, (2) the contract custom event that it released funds (3) the transfer of funds
from the contract to the beneficiary and (4) the generic x/wasm event stating that the contract
was executed (which always appears, while 2 is optional and has information as reliable as the contract):

```json
[
    {
        "Type": "transfer",
        "Attr": [
            {
                "key": "recipient",
                "value": "cosmos14hj2tavq8fpesdwxxcu44rty3hh90vhujrvcmstl4zr3txmfvw9s4hmalr"
            },
            {
                "key": "sender",
                "value": "cosmos1zm074khx32hqy20hlshlsd423n07pwlu9cpt37"
            },
            {
                "key": "amount",
                "value": "5000denom"
            }
        ]
    },
    {
        "Type": "wasm",
        "Attr": [
            {
                "key": "_contract_address",
                "value": "cosmos14hj2tavq8fpesdwxxcu44rty3hh90vhujrvcmstl4zr3txmfvw9s4hmalr"
            },
            {
                "key": "action",
                "value": "release"
            },
            {
                "key": "destination",
                "value": "cosmos14k7v7ms4jxkk2etmg9gljxjm4ru3qjdugfsflq"
            }
        ]
    },
    {
        "Type": "transfer",
        "Attr": [
            {
                "key": "recipient",
                "value": "cosmos14k7v7ms4jxkk2etmg9gljxjm4ru3qjdugfsflq"
            },
            {
                "key": "sender",
                "value": "cosmos14hj2tavq8fpesdwxxcu44rty3hh90vhujrvcmstl4zr3txmfvw9s4hmalr"
            },
            {
                "key": "amount",
                "value": "105000denom"
            }
        ]
    },
    {
        "Type": "message",
        "Attr": [
            {
                "key": "module",
                "value": "wasm"
            },
            {
                "key": "action",
                "value": "execute"
            },
            {
                "key": "signer",
                "value": "cosmos1zm074khx32hqy20hlshlsd423n07pwlu9cpt37"
            },
            {
                "key": "_contract_address",
                "value": "cosmos14hj2tavq8fpesdwxxcu44rty3hh90vhujrvcmstl4zr3txmfvw9s4hmalr"
            }
        ]
    }
]
```

A note on this format. This is what we return from our module. However, it seems to me that many events with the same `Type`
get merged together somewhere along the stack, so in this case, you *may* end up with one "transfer" event with the info for
both transfers. Double check when evaluating the event logs, I will document better with more experience, especially when I
find out the entire path for the events.

## Messages

The Wasm module defines the following messages:

### MsgStoreCode

`MsgStoreCode` is used to upload new code to the blockchain. This can only be done by authorized addresses if permissioning is enabled.

```go
type MsgStoreCode struct {
    Sender       string  // Account address of the signer
    WASMByteCode []byte  // Raw contract code
    InstantiatePermission *AccessConfig // Optional access control configuration for instantiation
}
```

### MsgInstantiateContract

`MsgInstantiateContract` is used to create a new contract instance from already submitted code.

```go
type MsgInstantiateContract struct {
    Sender    string    // Account address of the signer
    Admin     string    // Optional address for contract migrations/admin functions
    CodeID    uint64    // ID of the code to be instantiated
    Label     string    // Human-readable label for this instance
    Msg       []byte    // JSON-encoded instantiate message
    Funds     sdk.Coins // Funds to be transferred to the contract on instantiation
}
```

### MsgExecuteContract

`MsgExecuteContract` is used to execute functions on an existing contract.

```go
type MsgExecuteContract struct {
    Sender  string    // Account address of the signer
    Contract string   // Contract address to execute
    Msg     []byte    // JSON-encoded execution message
    Funds   sdk.Coins // Funds to be transferred to the contract on execution
}
```

### MsgMigrateContract

`MsgMigrateContract` is used to migrate a contract to a new code ID. This can only be executed by the contract admin.

```go
type MsgMigrateContract struct {
    Sender   string // Account address of the signer (must be contract admin)
    Contract string // Contract address to migrate
    CodeID   uint64 // New code ID to migrate to
    Msg      []byte // JSON-encoded migration message
}
```

### MsgUpdateAdmin

`MsgUpdateAdmin` is used to change a contract's admin.

```go
type MsgUpdateAdmin struct {
    Sender   string // Account address of the signer (must be current contract admin)
    NewAdmin string // Address of the new admin
    Contract string // Contract address to update
}
```

### MsgClearAdmin

`MsgClearAdmin` is used to clear a contract's admin, removing admin privileges permanently.

```go
type MsgClearAdmin struct {
    Sender   string // Account address of the signer (must be contract admin)
    Contract string // Contract address to clear admin for
}
```

## CLI

The Wasm module provides several commands for interacting with the blockchain via the CLI. Here are the primary commands:

### Upload contract code

```bash
wasmd tx wasm store contract.wasm --from myKey --chain-id=myChainID --gas auto --gas-adjustment 1.3
```

If you want to specify instantiation permissions:

```bash
wasmd tx wasm store contract.wasm --instantiate-only-address cosmos1address... --from myKey --chain-id=myChainID
```

### Instantiate a contract

```bash
wasmd tx wasm instantiate 1 '{"owner":"cosmos1address...", "other_param":"value"}' --label "my contract" --from myKey --chain-id=myChainID
```

For contracts that require funds on initialization:

```bash
wasmd tx wasm instantiate 1 '{"owner":"cosmos1address..."}' --amount 1000uatom --label "my contract" --from myKey --chain-id=myChainID
```

### Execute a contract

```bash
wasmd tx wasm execute cosmos14hj2... '{"action":"do_something","param":"value"}' --from myKey --chain-id=myChainID
```

With funds:

```bash
wasmd tx wasm execute cosmos14hj2... '{"action":"do_something"}' --amount 500uatom --from myKey --chain-id=myChainID
```

### Query a contract

```bash
wasmd query wasm contract-state smart cosmos14hj2... '{"get_config":{}}'
```

### List contracts

List all contracts:

```bash
wasmd query wasm list-contract-by-code 1
```

List all code IDs:

```bash
wasmd query wasm list-code
```

### Contract migrations and admin functions

Migrate a contract to a new code ID:

```bash
wasmd tx wasm migrate cosmos14hj2... 2 '{"new_parameter":"new_value"}' --from myAdminKey --chain-id=myChainID
```

Update a contract's admin:

```bash
wasmd tx wasm update-admin cosmos14hj2... cosmos1newadmin... --from myAdminKey --chain-id=myChainID
```

Clear a contract's admin:

```bash
wasmd tx wasm clear-admin cosmos14hj2... --from myAdminKey --chain-id=myChainID
```

## REST API

The Wasm module provides a comprehensive REST API that can be accessed via the Cosmos SDK API server (typically on port 1317). Below are the key endpoints:

### Code Management

- `GET /wasm/code`: List all codes stored
- `GET /wasm/code/{codeID}`: Get details about a specific code ID
- `POST /wasm/code`: Upload a new contract (multipart request with wasm binary)

### Contract Management

- `GET /wasm/contract/{contractAddr}`: Get contract information
- `GET /wasm/contract/{contractAddr}/state`: Get all contract state
- `GET /wasm/contract/{contractAddr}/smart/{query-data}`: Query contract using smart queries
- `GET /wasm/contract/{contractAddr}/raw/{key}`: Query raw contract state by key
- `GET /wasm/contract/{contractAddr}/history`: Get contract history (migrations)
- `GET /wasm/contract/list/{codeID}`: List all contract instances for a code ID

### Transactions (via REST)

- `POST /wasm/contract/{contractAddr}/execute`: Execute a contract
- `POST /wasm/code/{codeID}/instantiate`: Instantiate a contract
- `POST /wasm/contract/{contractAddr}/migrate`: Migrate a contract to a new code ID
- `POST /wasm/contract/{contractAddr}/admin`: Update contract admin
- `POST /wasm/contract/{contractAddr}/clear-admin`: Clear contract admin

For all transaction endpoints, the payload should be wrapped in a `BaseReq` object with authentication and transaction parameters.

Example request to execute a contract:

```json
{
  "base_req": {
    "from": "cosmos1sender...",
    "chain_id": "myChainID",
    "gas": "auto",
    "gas_adjustment": "1.3"
  },
  "msg": {"action": "transfer", "recipient": "cosmos1recipient...", "amount": "100"},
  "funds": [{"denom": "uatom", "amount": "50"}]
}
```

This REST API provides a comprehensive interface for dApps and other client applications to interact with smart contracts on the blockchain.
