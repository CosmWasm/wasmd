# Governance

This document gives an overview of how the various governance
proposals interact with the CosmWasm contract lifecycle. It is
a high-level, technical introduction meant to provide context before
looking into the code, or constructing proposals. 

## Proposal Types
We have added 5 new wasm specific proposal types that cover the contract's live cycle and authorization:
 
* `StoreCodeProposal` - upload a wasm binary
* `InstantiateContractProposal` - instantiate a wasm contract
* `MigrateContractProposal` - migrate a wasm contract to a new code version
* `UpdateAdminProposal` - set a new admin for a contract
* `ClearAdminProposal` - clear admin for a contract to prevent further migrations

For details see the proposal type [implementation](https://github.com/CosmWasm/wasmd/blob/master/x/wasm/internal/types/proposal.go)

A wasm message but no proposal type: 
* `ExecuteContract` - execute a command on a wasm contract

### Unit tests
[Proposal type validations](https://github.com/CosmWasm/wasmd/blob/master/x/wasm/internal/types/proposal_test.go)

## Proposal Handler
The [wasmd proposal_handler](https://github.com/CosmWasm/wasmd/blob/master/x/wasm/internal/keeper/proposal_handler.go) implements the `gov.Handler` function
and executes the wasmd proposal types after a successful tally.
 
The proposal handler uses a [`GovAuthorizationPolicy`](https://github.com/CosmWasm/wasmd/blob/master/x/wasm/internal/keeper/authz_policy.go#L29) to bypass the existing contract's authorization policy.

### Tests
* [Integration: Submit and execute proposal](https://github.com/CosmWasm/wasmd/blob/master/x/wasm/internal/keeper/proposal_integration_test.go)

## Gov Integration
The wasmd proposal handler can be added to the gov router in the [abci app](https://github.com/CosmWasm/wasmd/blob/master/app/app.go#L306)
to receive proposal execution calls. 
```go
govRouter.AddRoute(wasm.RouterKey, wasm.NewWasmProposalHandler(app.wasmKeeper, enabledProposals))
```

## Wasmd Authorization Settings

Settings via sdk `params` module: 
- `code_upload_access` - who can upload a wasm binary: `Nobody`, `Everybody`, `OnlyAddress`
- `instantiate_default_permission` - platform default, who can instantiate a wasm binary when the code owner has not set it 

See [params.go](https://github.com/CosmWasm/wasmd/blob/master/x/wasm/internal/types/params.go)

### Init Params Via Genesis 

```json
    "wasm": {
      "params": {
        "code_upload_access": {
          "permission": "Everybody"
        },
        "instantiate_default_permission": "Everybody"
      }
    },
```

The values can be updated via gov proposal implemented in the `params` module.

### Enable gov proposals at **compile time**. 
As gov proposals bypass the existing authorzation policy they are diabled and require to be enabled at compile time. 
```
-X github.com/CosmWasm/wasmd/app.ProposalsEnabled=true - enable all x/wasm governance proposals (default false)
-X github.com/CosmWasm/wasmd/app.EnableSpecificProposals=MigrateContract,UpdateAdmin,ClearAdmin - enable a subset of the x/wasm governance proposal types (overrides ProposalsEnabled)
```

### Tests
* [params validation unit tests](https://github.com/CosmWasm/wasmd/blob/master/x/wasm/internal/types/params_test.go)
* [genesis validation tests](https://github.com/CosmWasm/wasmd/blob/master/x/wasm/internal/types/genesis_test.go)
* [policy integration tests](https://github.com/CosmWasm/wasmd/blob/master/x/wasm/internal/keeper/keeper_test.go)

## CLI

```shell script
  wasmcli tx gov submit-proposal [command]

Available Commands:
  wasm-store           Submit a wasm binary proposal
  instantiate-contract Submit an instantiate wasm contract proposal
  migrate-contract     Submit a migrate wasm contract to a new code version proposal
  set-contract-admin   Submit a new admin for a contract proposal
  clear-contract-admin Submit a clear admin for a contract to prevent further migrations proposal
...
```
## Rest
New [`ProposalHandlers`](https://github.com/CosmWasm/wasmd/blob/master/x/wasm/client/proposal_handler.go)

* Integration
```shell script
gov.NewAppModuleBasic(append(wasmclient.ProposalHandlers, paramsclient.ProposalHandler, distr.ProposalHandler, upgradeclient.ProposalHandler)...),
```
In [abci app](https://github.com/CosmWasm/wasmd/blob/master/app/app.go#L109)

### Tests
* [Rest Unit tests](https://github.com/CosmWasm/wasmd/blob/master/x/wasm/client/proposal_handler_test.go)
* [Rest smoke LCD test](https://github.com/CosmWasm/wasmd/blob/master/lcd_test/wasm_test.go)



## Pull requests
* https://github.com/CosmWasm/wasmd/pull/190
* https://github.com/CosmWasm/wasmd/pull/186
* https://github.com/CosmWasm/wasmd/pull/183
* https://github.com/CosmWasm/wasmd/pull/180
* https://github.com/CosmWasm/wasmd/pull/179
* https://github.com/CosmWasm/wasmd/pull/173
