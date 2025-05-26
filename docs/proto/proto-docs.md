<!-- This file is auto-generated. Please do not modify it yourself. -->
# Protobuf Documentation
<a name="top"></a>

## Table of Contents

- [wasmvm.proto](#wasmvm.proto)
    - [AnalyzeCodeRequest](#cosmwasm.AnalyzeCodeRequest)
    - [AnalyzeCodeResponse](#cosmwasm.AnalyzeCodeResponse)
    - [CallHostFunctionRequest](#cosmwasm.CallHostFunctionRequest)
    - [CallHostFunctionResponse](#cosmwasm.CallHostFunctionResponse)
    - [CanonicalizeAddressRequest](#cosmwasm.CanonicalizeAddressRequest)
    - [CanonicalizeAddressResponse](#cosmwasm.CanonicalizeAddressResponse)
    - [ConsumeGasRequest](#cosmwasm.ConsumeGasRequest)
    - [ConsumeGasResponse](#cosmwasm.ConsumeGasResponse)
    - [Context](#cosmwasm.Context)
    - [ExecuteRequest](#cosmwasm.ExecuteRequest)
    - [ExecuteResponse](#cosmwasm.ExecuteResponse)
    - [ExtendedContext](#cosmwasm.ExtendedContext)
    - [ExtendedExecuteRequest](#cosmwasm.ExtendedExecuteRequest)
    - [ExtendedInstantiateRequest](#cosmwasm.ExtendedInstantiateRequest)
    - [ExtendedMigrateRequest](#cosmwasm.ExtendedMigrateRequest)
    - [ExtendedQueryRequest](#cosmwasm.ExtendedQueryRequest)
    - [GetCodeRequest](#cosmwasm.GetCodeRequest)
    - [GetCodeResponse](#cosmwasm.GetCodeResponse)
    - [GetGasRemainingRequest](#cosmwasm.GetGasRemainingRequest)
    - [GetGasRemainingResponse](#cosmwasm.GetGasRemainingResponse)
    - [GetMetricsRequest](#cosmwasm.GetMetricsRequest)
    - [GetMetricsResponse](#cosmwasm.GetMetricsResponse)
    - [GetPinnedMetricsRequest](#cosmwasm.GetPinnedMetricsRequest)
    - [GetPinnedMetricsResponse](#cosmwasm.GetPinnedMetricsResponse)
    - [HumanizeAddressRequest](#cosmwasm.HumanizeAddressRequest)
    - [HumanizeAddressResponse](#cosmwasm.HumanizeAddressResponse)
    - [IbcMsgRequest](#cosmwasm.IbcMsgRequest)
    - [IbcMsgResponse](#cosmwasm.IbcMsgResponse)
    - [InstantiateRequest](#cosmwasm.InstantiateRequest)
    - [InstantiateResponse](#cosmwasm.InstantiateResponse)
    - [LoadModuleRequest](#cosmwasm.LoadModuleRequest)
    - [LoadModuleResponse](#cosmwasm.LoadModuleResponse)
    - [Metrics](#cosmwasm.Metrics)
    - [MigrateRequest](#cosmwasm.MigrateRequest)
    - [MigrateResponse](#cosmwasm.MigrateResponse)
    - [PerModuleMetrics](#cosmwasm.PerModuleMetrics)
    - [PinModuleRequest](#cosmwasm.PinModuleRequest)
    - [PinModuleResponse](#cosmwasm.PinModuleResponse)
    - [PinnedMetrics](#cosmwasm.PinnedMetrics)
    - [PinnedMetrics.PerModuleEntry](#cosmwasm.PinnedMetrics.PerModuleEntry)
    - [QueryChainRequest](#cosmwasm.QueryChainRequest)
    - [QueryChainResponse](#cosmwasm.QueryChainResponse)
    - [QueryRequest](#cosmwasm.QueryRequest)
    - [QueryResponse](#cosmwasm.QueryResponse)
    - [RemoveModuleRequest](#cosmwasm.RemoveModuleRequest)
    - [RemoveModuleResponse](#cosmwasm.RemoveModuleResponse)
    - [ReplyRequest](#cosmwasm.ReplyRequest)
    - [ReplyResponse](#cosmwasm.ReplyResponse)
    - [StorageDeleteRequest](#cosmwasm.StorageDeleteRequest)
    - [StorageDeleteResponse](#cosmwasm.StorageDeleteResponse)
    - [StorageGetRequest](#cosmwasm.StorageGetRequest)
    - [StorageGetResponse](#cosmwasm.StorageGetResponse)
    - [StorageIteratorRequest](#cosmwasm.StorageIteratorRequest)
    - [StorageIteratorResponse](#cosmwasm.StorageIteratorResponse)
    - [StorageReverseIteratorRequest](#cosmwasm.StorageReverseIteratorRequest)
    - [StorageReverseIteratorResponse](#cosmwasm.StorageReverseIteratorResponse)
    - [StorageSetRequest](#cosmwasm.StorageSetRequest)
    - [StorageSetResponse](#cosmwasm.StorageSetResponse)
    - [SudoRequest](#cosmwasm.SudoRequest)
    - [SudoResponse](#cosmwasm.SudoResponse)
    - [UnpinModuleRequest](#cosmwasm.UnpinModuleRequest)
    - [UnpinModuleResponse](#cosmwasm.UnpinModuleResponse)
  
    - [HostService](#cosmwasm.HostService)
    - [WasmVMService](#cosmwasm.WasmVMService)
  
- [Scalar Value Types](#scalar-value-types)



<a name="wasmvm.proto"></a>
<p align="right"><a href="#top">Top</a></p>

## wasmvm.proto



<a name="cosmwasm.AnalyzeCodeRequest"></a>

### AnalyzeCodeRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `checksum` | [string](#string) |  | Hex encoded checksum of the WASM module |






<a name="cosmwasm.AnalyzeCodeResponse"></a>

### AnalyzeCodeResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `required_capabilities` | [string](#string) | repeated | Comma-separated list of required capabilities |
| `has_ibc_entry_points` | [bool](#bool) |  | True if IBC entry points are detected |
| `error` | [string](#string) |  |  |






<a name="cosmwasm.CallHostFunctionRequest"></a>

### CallHostFunctionRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `function_name` | [string](#string) |  |  |
| `args` | [bytes](#bytes) |  | Binary arguments specific to the host function |
| `context` | [Context](#cosmwasm.Context) |  |  |
| `request_id` | [string](#string) |  |  |






<a name="cosmwasm.CallHostFunctionResponse"></a>

### CallHostFunctionResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `result` | [bytes](#bytes) |  |  |
| `error` | [string](#string) |  |  |






<a name="cosmwasm.CanonicalizeAddressRequest"></a>

### CanonicalizeAddressRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `request_id` | [string](#string) |  |  |
| `human` | [string](#string) |  |  |






<a name="cosmwasm.CanonicalizeAddressResponse"></a>

### CanonicalizeAddressResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `canonical` | [bytes](#bytes) |  |  |
| `gas_used` | [uint64](#uint64) |  |  |
| `error` | [string](#string) |  |  |






<a name="cosmwasm.ConsumeGasRequest"></a>

### ConsumeGasRequest
Gas meter messages


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `request_id` | [string](#string) |  |  |
| `amount` | [uint64](#uint64) |  |  |
| `descriptor` | [string](#string) |  |  |






<a name="cosmwasm.ConsumeGasResponse"></a>

### ConsumeGasResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `error` | [string](#string) |  |  |






<a name="cosmwasm.Context"></a>

### Context
Context message for blockchain-related information


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `block_height` | [uint64](#uint64) |  |  |
| `sender` | [string](#string) |  |  |
| `chain_id` | [string](#string) |  |  |






<a name="cosmwasm.ExecuteRequest"></a>

### ExecuteRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `contract_id` | [string](#string) |  | Hex encoded checksum of the WASM module |
| `context` | [Context](#cosmwasm.Context) |  |  |
| `msg` | [bytes](#bytes) |  |  |
| `gas_limit` | [uint64](#uint64) |  |  |
| `request_id` | [string](#string) |  |  |






<a name="cosmwasm.ExecuteResponse"></a>

### ExecuteResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `data` | [bytes](#bytes) |  |  |
| `gas_used` | [uint64](#uint64) |  |  |
| `error` | [string](#string) |  |  |






<a name="cosmwasm.ExtendedContext"></a>

### ExtendedContext
ExtendedContext includes callback service information for storage support


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `context` | [Context](#cosmwasm.Context) |  |  |
| `callback_service` | [string](#string) |  | Address of the HostService for callbacks |






<a name="cosmwasm.ExtendedExecuteRequest"></a>

### ExtendedExecuteRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `contract_id` | [string](#string) |  |  |
| `context` | [ExtendedContext](#cosmwasm.ExtendedContext) |  |  |
| `msg` | [bytes](#bytes) |  |  |
| `gas_limit` | [uint64](#uint64) |  |  |
| `request_id` | [string](#string) |  |  |






<a name="cosmwasm.ExtendedInstantiateRequest"></a>

### ExtendedInstantiateRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `checksum` | [string](#string) |  |  |
| `context` | [ExtendedContext](#cosmwasm.ExtendedContext) |  |  |
| `init_msg` | [bytes](#bytes) |  |  |
| `gas_limit` | [uint64](#uint64) |  |  |
| `request_id` | [string](#string) |  |  |






<a name="cosmwasm.ExtendedMigrateRequest"></a>

### ExtendedMigrateRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `contract_id` | [string](#string) |  |  |
| `checksum` | [string](#string) |  |  |
| `context` | [ExtendedContext](#cosmwasm.ExtendedContext) |  |  |
| `migrate_msg` | [bytes](#bytes) |  |  |
| `gas_limit` | [uint64](#uint64) |  |  |
| `request_id` | [string](#string) |  |  |






<a name="cosmwasm.ExtendedQueryRequest"></a>

### ExtendedQueryRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `contract_id` | [string](#string) |  |  |
| `context` | [ExtendedContext](#cosmwasm.ExtendedContext) |  |  |
| `query_msg` | [bytes](#bytes) |  |  |
| `request_id` | [string](#string) |  |  |






<a name="cosmwasm.GetCodeRequest"></a>

### GetCodeRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `checksum` | [string](#string) |  | Hex encoded checksum of the WASM module to retrieve |






<a name="cosmwasm.GetCodeResponse"></a>

### GetCodeResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `module_bytes` | [bytes](#bytes) |  | Raw WASM bytes |
| `error` | [string](#string) |  |  |






<a name="cosmwasm.GetGasRemainingRequest"></a>

### GetGasRemainingRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `request_id` | [string](#string) |  |  |






<a name="cosmwasm.GetGasRemainingResponse"></a>

### GetGasRemainingResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `gas_remaining` | [uint64](#uint64) |  |  |
| `error` | [string](#string) |  |  |






<a name="cosmwasm.GetMetricsRequest"></a>

### GetMetricsRequest







<a name="cosmwasm.GetMetricsResponse"></a>

### GetMetricsResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `metrics` | [Metrics](#cosmwasm.Metrics) |  |  |
| `error` | [string](#string) |  |  |






<a name="cosmwasm.GetPinnedMetricsRequest"></a>

### GetPinnedMetricsRequest







<a name="cosmwasm.GetPinnedMetricsResponse"></a>

### GetPinnedMetricsResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `pinned_metrics` | [PinnedMetrics](#cosmwasm.PinnedMetrics) |  |  |
| `error` | [string](#string) |  |  |






<a name="cosmwasm.HumanizeAddressRequest"></a>

### HumanizeAddressRequest
GoAPI messages


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `request_id` | [string](#string) |  |  |
| `canonical` | [bytes](#bytes) |  |  |






<a name="cosmwasm.HumanizeAddressResponse"></a>

### HumanizeAddressResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `human` | [string](#string) |  |  |
| `gas_used` | [uint64](#uint64) |  |  |
| `error` | [string](#string) |  |  |






<a name="cosmwasm.IbcMsgRequest"></a>

### IbcMsgRequest
Generalized IBC Message Request/Response for various IBC entry points
This structure is reused across all IBC-related RPC calls in WasmVMService


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `checksum` | [string](#string) |  | Hex encoded checksum of the WASM module |
| `context` | [Context](#cosmwasm.Context) |  |  |
| `msg` | [bytes](#bytes) |  | Binary message for the IBC call |
| `gas_limit` | [uint64](#uint64) |  |  |
| `request_id` | [string](#string) |  |  |






<a name="cosmwasm.IbcMsgResponse"></a>

### IbcMsgResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `data` | [bytes](#bytes) |  | Binary response data from the contract |
| `gas_used` | [uint64](#uint64) |  |  |
| `error` | [string](#string) |  |  |






<a name="cosmwasm.InstantiateRequest"></a>

### InstantiateRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `checksum` | [string](#string) |  | Hex encoded checksum of the WASM module |
| `context` | [Context](#cosmwasm.Context) |  |  |
| `init_msg` | [bytes](#bytes) |  |  |
| `gas_limit` | [uint64](#uint64) |  |  |
| `request_id` | [string](#string) |  |  |






<a name="cosmwasm.InstantiateResponse"></a>

### InstantiateResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `contract_id` | [string](#string) |  | Identifier for the instantiated contract, typically |
| `data` | [bytes](#bytes) |  | derived from request_id or a unique hash

Binary response data from the contract |
| `gas_used` | [uint64](#uint64) |  |  |
| `error` | [string](#string) |  |  |






<a name="cosmwasm.LoadModuleRequest"></a>

### LoadModuleRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `module_bytes` | [bytes](#bytes) |  |  |






<a name="cosmwasm.LoadModuleResponse"></a>

### LoadModuleResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `checksum` | [string](#string) |  | SHA256 checksum of the module (hex encoded) |
| `error` | [string](#string) |  |  |






<a name="cosmwasm.Metrics"></a>

### Metrics



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `hits_pinned_memory_cache` | [uint32](#uint32) |  |  |
| `hits_memory_cache` | [uint32](#uint32) |  |  |
| `hits_fs_cache` | [uint32](#uint32) |  |  |
| `misses` | [uint32](#uint32) |  |  |
| `elements_pinned_memory_cache` | [uint64](#uint64) |  |  |
| `elements_memory_cache` | [uint64](#uint64) |  |  |
| `size_pinned_memory_cache` | [uint64](#uint64) |  |  |
| `size_memory_cache` | [uint64](#uint64) |  |  |






<a name="cosmwasm.MigrateRequest"></a>

### MigrateRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `contract_id` | [string](#string) |  | Hex encoded checksum of the existing contract |
| `checksum` | [string](#string) |  | Hex encoded checksum of the new WASM module for migration |
| `context` | [Context](#cosmwasm.Context) |  |  |
| `migrate_msg` | [bytes](#bytes) |  |  |
| `gas_limit` | [uint64](#uint64) |  |  |
| `request_id` | [string](#string) |  |  |






<a name="cosmwasm.MigrateResponse"></a>

### MigrateResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `data` | [bytes](#bytes) |  |  |
| `gas_used` | [uint64](#uint64) |  |  |
| `error` | [string](#string) |  |  |






<a name="cosmwasm.PerModuleMetrics"></a>

### PerModuleMetrics



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `hits` | [uint32](#uint32) |  |  |
| `size` | [uint64](#uint64) |  | Size of the module in bytes |






<a name="cosmwasm.PinModuleRequest"></a>

### PinModuleRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `checksum` | [string](#string) |  | Hex encoded checksum of the WASM module to pin |






<a name="cosmwasm.PinModuleResponse"></a>

### PinModuleResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `error` | [string](#string) |  | Error message if pinning failed |






<a name="cosmwasm.PinnedMetrics"></a>

### PinnedMetrics



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `per_module` | [PinnedMetrics.PerModuleEntry](#cosmwasm.PinnedMetrics.PerModuleEntry) | repeated | Map from hex-encoded checksum to its metrics |






<a name="cosmwasm.PinnedMetrics.PerModuleEntry"></a>

### PinnedMetrics.PerModuleEntry



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `key` | [string](#string) |  |  |
| `value` | [PerModuleMetrics](#cosmwasm.PerModuleMetrics) |  |  |






<a name="cosmwasm.QueryChainRequest"></a>

### QueryChainRequest
Query messages


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `request_id` | [string](#string) |  |  |
| `query` | [bytes](#bytes) |  | Serialized QueryRequest |
| `gas_limit` | [uint64](#uint64) |  |  |






<a name="cosmwasm.QueryChainResponse"></a>

### QueryChainResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `result` | [bytes](#bytes) |  |  |
| `error` | [string](#string) |  |  |






<a name="cosmwasm.QueryRequest"></a>

### QueryRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `contract_id` | [string](#string) |  | Hex encoded checksum of the WASM module |
| `context` | [Context](#cosmwasm.Context) |  |  |
| `query_msg` | [bytes](#bytes) |  |  |
| `request_id` | [string](#string) |  |  |






<a name="cosmwasm.QueryResponse"></a>

### QueryResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `result` | [bytes](#bytes) |  | Binary query response data |
| `error` | [string](#string) |  |  |






<a name="cosmwasm.RemoveModuleRequest"></a>

### RemoveModuleRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `checksum` | [string](#string) |  | Hex encoded checksum of the WASM module to remove |






<a name="cosmwasm.RemoveModuleResponse"></a>

### RemoveModuleResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `error` | [string](#string) |  | Error message if removal failed |






<a name="cosmwasm.ReplyRequest"></a>

### ReplyRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `contract_id` | [string](#string) |  | Hex encoded checksum of the WASM module |
| `context` | [Context](#cosmwasm.Context) |  |  |
| `reply_msg` | [bytes](#bytes) |  |  |
| `gas_limit` | [uint64](#uint64) |  |  |
| `request_id` | [string](#string) |  |  |






<a name="cosmwasm.ReplyResponse"></a>

### ReplyResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `data` | [bytes](#bytes) |  |  |
| `gas_used` | [uint64](#uint64) |  |  |
| `error` | [string](#string) |  |  |






<a name="cosmwasm.StorageDeleteRequest"></a>

### StorageDeleteRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `request_id` | [string](#string) |  |  |
| `key` | [bytes](#bytes) |  |  |






<a name="cosmwasm.StorageDeleteResponse"></a>

### StorageDeleteResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `error` | [string](#string) |  |  |






<a name="cosmwasm.StorageGetRequest"></a>

### StorageGetRequest
Storage messages


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `request_id` | [string](#string) |  |  |
| `key` | [bytes](#bytes) |  |  |






<a name="cosmwasm.StorageGetResponse"></a>

### StorageGetResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `value` | [bytes](#bytes) |  |  |
| `exists` | [bool](#bool) |  |  |
| `error` | [string](#string) |  |  |






<a name="cosmwasm.StorageIteratorRequest"></a>

### StorageIteratorRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `request_id` | [string](#string) |  |  |
| `start` | [bytes](#bytes) |  |  |
| `end` | [bytes](#bytes) |  |  |






<a name="cosmwasm.StorageIteratorResponse"></a>

### StorageIteratorResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `key` | [bytes](#bytes) |  |  |
| `value` | [bytes](#bytes) |  |  |
| `done` | [bool](#bool) |  |  |
| `error` | [string](#string) |  |  |






<a name="cosmwasm.StorageReverseIteratorRequest"></a>

### StorageReverseIteratorRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `request_id` | [string](#string) |  |  |
| `start` | [bytes](#bytes) |  |  |
| `end` | [bytes](#bytes) |  |  |






<a name="cosmwasm.StorageReverseIteratorResponse"></a>

### StorageReverseIteratorResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `key` | [bytes](#bytes) |  |  |
| `value` | [bytes](#bytes) |  |  |
| `done` | [bool](#bool) |  |  |
| `error` | [string](#string) |  |  |






<a name="cosmwasm.StorageSetRequest"></a>

### StorageSetRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `request_id` | [string](#string) |  |  |
| `key` | [bytes](#bytes) |  |  |
| `value` | [bytes](#bytes) |  |  |






<a name="cosmwasm.StorageSetResponse"></a>

### StorageSetResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `error` | [string](#string) |  |  |






<a name="cosmwasm.SudoRequest"></a>

### SudoRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `contract_id` | [string](#string) |  | Hex encoded checksum of the WASM module |
| `context` | [Context](#cosmwasm.Context) |  |  |
| `msg` | [bytes](#bytes) |  |  |
| `gas_limit` | [uint64](#uint64) |  |  |
| `request_id` | [string](#string) |  |  |






<a name="cosmwasm.SudoResponse"></a>

### SudoResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `data` | [bytes](#bytes) |  |  |
| `gas_used` | [uint64](#uint64) |  |  |
| `error` | [string](#string) |  |  |






<a name="cosmwasm.UnpinModuleRequest"></a>

### UnpinModuleRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `checksum` | [string](#string) |  | Hex encoded checksum of the WASM module to unpin |






<a name="cosmwasm.UnpinModuleResponse"></a>

### UnpinModuleResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `error` | [string](#string) |  | Error message if unpinning failed |





 <!-- end messages -->

 <!-- end enums -->

 <!-- end HasExtensions -->


<a name="cosmwasm.HostService"></a>

### HostService
HostService: Enhanced RPC interface for host function callbacks
This service is called by the VM to interact with storage, query chain state,
and use other host-provided functionality

| Method Name | Request Type | Response Type | Description | HTTP Verb | Endpoint |
| ----------- | ------------ | ------------- | ------------| ------- | -------- |
| `CallHostFunction` | [CallHostFunctionRequest](#cosmwasm.CallHostFunctionRequest) | [CallHostFunctionResponse](#cosmwasm.CallHostFunctionResponse) | Legacy generic host function call | |
| `StorageGet` | [StorageGetRequest](#cosmwasm.StorageGetRequest) | [StorageGetResponse](#cosmwasm.StorageGetResponse) | Storage operations | |
| `StorageSet` | [StorageSetRequest](#cosmwasm.StorageSetRequest) | [StorageSetResponse](#cosmwasm.StorageSetResponse) |  | |
| `StorageDelete` | [StorageDeleteRequest](#cosmwasm.StorageDeleteRequest) | [StorageDeleteResponse](#cosmwasm.StorageDeleteResponse) |  | |
| `StorageIterator` | [StorageIteratorRequest](#cosmwasm.StorageIteratorRequest) | [StorageIteratorResponse](#cosmwasm.StorageIteratorResponse) stream |  | |
| `StorageReverseIterator` | [StorageReverseIteratorRequest](#cosmwasm.StorageReverseIteratorRequest) | [StorageReverseIteratorResponse](#cosmwasm.StorageReverseIteratorResponse) stream |  | |
| `QueryChain` | [QueryChainRequest](#cosmwasm.QueryChainRequest) | [QueryChainResponse](#cosmwasm.QueryChainResponse) | Query operations | |
| `HumanizeAddress` | [HumanizeAddressRequest](#cosmwasm.HumanizeAddressRequest) | [HumanizeAddressResponse](#cosmwasm.HumanizeAddressResponse) | GoAPI operations | |
| `CanonicalizeAddress` | [CanonicalizeAddressRequest](#cosmwasm.CanonicalizeAddressRequest) | [CanonicalizeAddressResponse](#cosmwasm.CanonicalizeAddressResponse) |  | |
| `ConsumeGas` | [ConsumeGasRequest](#cosmwasm.ConsumeGasRequest) | [ConsumeGasResponse](#cosmwasm.ConsumeGasResponse) | Gas meter operations | |
| `GetGasRemaining` | [GetGasRemainingRequest](#cosmwasm.GetGasRemainingRequest) | [GetGasRemainingResponse](#cosmwasm.GetGasRemainingResponse) |  | |


<a name="cosmwasm.WasmVMService"></a>

### WasmVMService
WasmVMService: RPC interface for wasmvm

| Method Name | Request Type | Response Type | Description | HTTP Verb | Endpoint |
| ----------- | ------------ | ------------- | ------------| ------- | -------- |
| `LoadModule` | [LoadModuleRequest](#cosmwasm.LoadModuleRequest) | [LoadModuleResponse](#cosmwasm.LoadModuleResponse) | Module lifecycle management | |
| `RemoveModule` | [RemoveModuleRequest](#cosmwasm.RemoveModuleRequest) | [RemoveModuleResponse](#cosmwasm.RemoveModuleResponse) |  | |
| `PinModule` | [PinModuleRequest](#cosmwasm.PinModuleRequest) | [PinModuleResponse](#cosmwasm.PinModuleResponse) |  | |
| `UnpinModule` | [UnpinModuleRequest](#cosmwasm.UnpinModuleRequest) | [UnpinModuleResponse](#cosmwasm.UnpinModuleResponse) |  | |
| `GetCode` | [GetCodeRequest](#cosmwasm.GetCodeRequest) | [GetCodeResponse](#cosmwasm.GetCodeResponse) | Retrieve raw WASM bytes | |
| `Instantiate` | [InstantiateRequest](#cosmwasm.InstantiateRequest) | [InstantiateResponse](#cosmwasm.InstantiateResponse) | Contract execution calls | |
| `Execute` | [ExecuteRequest](#cosmwasm.ExecuteRequest) | [ExecuteResponse](#cosmwasm.ExecuteResponse) |  | |
| `Query` | [QueryRequest](#cosmwasm.QueryRequest) | [QueryResponse](#cosmwasm.QueryResponse) |  | |
| `Migrate` | [MigrateRequest](#cosmwasm.MigrateRequest) | [MigrateResponse](#cosmwasm.MigrateResponse) |  | |
| `Sudo` | [SudoRequest](#cosmwasm.SudoRequest) | [SudoResponse](#cosmwasm.SudoResponse) |  | |
| `Reply` | [ReplyRequest](#cosmwasm.ReplyRequest) | [ReplyResponse](#cosmwasm.ReplyResponse) |  | |
| `InstantiateWithStorage` | [ExtendedInstantiateRequest](#cosmwasm.ExtendedInstantiateRequest) | [InstantiateResponse](#cosmwasm.InstantiateResponse) | Storage-aware contract execution calls (enhanced versions) | |
| `ExecuteWithStorage` | [ExtendedExecuteRequest](#cosmwasm.ExtendedExecuteRequest) | [ExecuteResponse](#cosmwasm.ExecuteResponse) |  | |
| `QueryWithStorage` | [ExtendedQueryRequest](#cosmwasm.ExtendedQueryRequest) | [QueryResponse](#cosmwasm.QueryResponse) |  | |
| `MigrateWithStorage` | [ExtendedMigrateRequest](#cosmwasm.ExtendedMigrateRequest) | [MigrateResponse](#cosmwasm.MigrateResponse) |  | |
| `AnalyzeCode` | [AnalyzeCodeRequest](#cosmwasm.AnalyzeCodeRequest) | [AnalyzeCodeResponse](#cosmwasm.AnalyzeCodeResponse) | Code analysis | |
| `GetMetrics` | [GetMetricsRequest](#cosmwasm.GetMetricsRequest) | [GetMetricsResponse](#cosmwasm.GetMetricsResponse) | Metrics | |
| `GetPinnedMetrics` | [GetPinnedMetricsRequest](#cosmwasm.GetPinnedMetricsRequest) | [GetPinnedMetricsResponse](#cosmwasm.GetPinnedMetricsResponse) |  | |
| `IbcChannelOpen` | [IbcMsgRequest](#cosmwasm.IbcMsgRequest) | [IbcMsgResponse](#cosmwasm.IbcMsgResponse) | IBC Entry Points All IBC calls typically share a similar request/response structure with checksum, context, message, gas limit, and request ID. Their responses usually contain data, gas used, and an error. | |
| `IbcChannelConnect` | [IbcMsgRequest](#cosmwasm.IbcMsgRequest) | [IbcMsgResponse](#cosmwasm.IbcMsgResponse) |  | |
| `IbcChannelClose` | [IbcMsgRequest](#cosmwasm.IbcMsgRequest) | [IbcMsgResponse](#cosmwasm.IbcMsgResponse) |  | |
| `IbcPacketReceive` | [IbcMsgRequest](#cosmwasm.IbcMsgRequest) | [IbcMsgResponse](#cosmwasm.IbcMsgResponse) |  | |
| `IbcPacketAck` | [IbcMsgRequest](#cosmwasm.IbcMsgRequest) | [IbcMsgResponse](#cosmwasm.IbcMsgResponse) |  | |
| `IbcPacketTimeout` | [IbcMsgRequest](#cosmwasm.IbcMsgRequest) | [IbcMsgResponse](#cosmwasm.IbcMsgResponse) |  | |
| `IbcSourceCallback` | [IbcMsgRequest](#cosmwasm.IbcMsgRequest) | [IbcMsgResponse](#cosmwasm.IbcMsgResponse) |  | |
| `IbcDestinationCallback` | [IbcMsgRequest](#cosmwasm.IbcMsgRequest) | [IbcMsgResponse](#cosmwasm.IbcMsgResponse) |  | |
| `Ibc2PacketReceive` | [IbcMsgRequest](#cosmwasm.IbcMsgRequest) | [IbcMsgResponse](#cosmwasm.IbcMsgResponse) |  | |
| `Ibc2PacketAck` | [IbcMsgRequest](#cosmwasm.IbcMsgRequest) | [IbcMsgResponse](#cosmwasm.IbcMsgResponse) |  | |
| `Ibc2PacketTimeout` | [IbcMsgRequest](#cosmwasm.IbcMsgRequest) | [IbcMsgResponse](#cosmwasm.IbcMsgResponse) |  | |
| `Ibc2PacketSend` | [IbcMsgRequest](#cosmwasm.IbcMsgRequest) | [IbcMsgResponse](#cosmwasm.IbcMsgResponse) |  | |

 <!-- end services -->



## Scalar Value Types

| .proto Type | Notes | C++ | Java | Python | Go | C# | PHP | Ruby |
| ----------- | ----- | --- | ---- | ------ | -- | -- | --- | ---- |
| <a name="double" /> double |  | double | double | float | float64 | double | float | Float |
| <a name="float" /> float |  | float | float | float | float32 | float | float | Float |
| <a name="int32" /> int32 | Uses variable-length encoding. Inefficient for encoding negative numbers – if your field is likely to have negative values, use sint32 instead. | int32 | int | int | int32 | int | integer | Bignum or Fixnum (as required) |
| <a name="int64" /> int64 | Uses variable-length encoding. Inefficient for encoding negative numbers – if your field is likely to have negative values, use sint64 instead. | int64 | long | int/long | int64 | long | integer/string | Bignum |
| <a name="uint32" /> uint32 | Uses variable-length encoding. | uint32 | int | int/long | uint32 | uint | integer | Bignum or Fixnum (as required) |
| <a name="uint64" /> uint64 | Uses variable-length encoding. | uint64 | long | int/long | uint64 | ulong | integer/string | Bignum or Fixnum (as required) |
| <a name="sint32" /> sint32 | Uses variable-length encoding. Signed int value. These more efficiently encode negative numbers than regular int32s. | int32 | int | int | int32 | int | integer | Bignum or Fixnum (as required) |
| <a name="sint64" /> sint64 | Uses variable-length encoding. Signed int value. These more efficiently encode negative numbers than regular int64s. | int64 | long | int/long | int64 | long | integer/string | Bignum |
| <a name="fixed32" /> fixed32 | Always four bytes. More efficient than uint32 if values are often greater than 2^28. | uint32 | int | int | uint32 | uint | integer | Bignum or Fixnum (as required) |
| <a name="fixed64" /> fixed64 | Always eight bytes. More efficient than uint64 if values are often greater than 2^56. | uint64 | long | int/long | uint64 | ulong | integer/string | Bignum |
| <a name="sfixed32" /> sfixed32 | Always four bytes. | int32 | int | int | int32 | int | integer | Bignum or Fixnum (as required) |
| <a name="sfixed64" /> sfixed64 | Always eight bytes. | int64 | long | int/long | int64 | long | integer/string | Bignum |
| <a name="bool" /> bool |  | bool | boolean | boolean | bool | bool | boolean | TrueClass/FalseClass |
| <a name="string" /> string | A string must always contain UTF-8 encoded or 7-bit ASCII text. | string | String | str/unicode | string | string | string | String (UTF-8) |
| <a name="bytes" /> bytes | May contain any arbitrary sequence of bytes. | string | ByteString | str | []byte | ByteString | string | String (ASCII-8BIT) |

