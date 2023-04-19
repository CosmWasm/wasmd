# wasmplus

Extended module of [finschia/wasmd/x/wasm](https://github.com/Finschia/wasmd/tree/cae21ecd251cea44f56209e0a4586ca2979c6c87/x/wasm) module.

## Concepts

Implements the difference part with cosmwasm/wasmd.

* Add smart contract Inactivate function
* Add `Msg/StoreCodeAndInstantiateContract` tx message

### Inactivate
Add a function to deactivate or disable a specific smart contract address as a proposal.
Inactive smart contract address are stored and managed as `inactive_contract_addresses` in the genesis state.

Inactive smart contract address restricts execution of `ExecuteContract`, `MigrateContract`, `UpdateAdmin`, `ClearAdmin`.
Through `ActivateContractProposal`, you can release restrictions on the use of inactive smart contract address.

#### Proposal
##### DeactivateContractProposal
* If the deactivation succeeds, the [`EventDeactivateContractProposal`](../../docs/proto/proto-docs.md#eventdeactivatecontractproposal) event is emitted.
##### ActivateContractProposal
* Proposals activation of disabled smart contract address.
* If the activation succeeds, the [`EventActivateContractProposal`](../../docs/proto/proto-docs.md#eventactivatecontractproposal) event is emitted.

#### queries
##### InactiveContracts
* Query API to query a list of all disabled smart contract addresses with pagination
* [Detailed specification](../../docs/proto/proto-docs.md#activatecontractproposal)
##### InactiveContract
* Query API to check if a specific smart contract address is disabled
* [Detailed specification](../../docs/proto/proto-docs.md#deactivatecontractproposal)

### Msg/StoreCodeAndInstantiateContract
`Msg/StoreCodeAndInstantiateContract` allows `StoreCode` and `InstantiateContract` to be processed as one tx message.
More information can be found [here](../../docs/proto/proto-docs.md#msgstorecodeandinstantiatecontract)
