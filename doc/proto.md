# Protocol Documentation
<a name="top"></a>

## Table of Contents

- [x/wasm/internal/types/genesis.proto](#x/wasm/internal/types/genesis.proto)
    - [Code](#wasmd.x.wasmd.v1beta1.Code)
    - [Contract](#wasmd.x.wasmd.v1beta1.Contract)
    - [GenesisState](#wasmd.x.wasmd.v1beta1.GenesisState)
    - [Sequence](#wasmd.x.wasmd.v1beta1.Sequence)
  
- [x/wasm/internal/types/msg.proto](#x/wasm/internal/types/msg.proto)
    - [MsgClearAdmin](#wasmd.x.wasmd.v1beta1.MsgClearAdmin)
    - [MsgExecuteContract](#wasmd.x.wasmd.v1beta1.MsgExecuteContract)
    - [MsgInstantiateContract](#wasmd.x.wasmd.v1beta1.MsgInstantiateContract)
    - [MsgMigrateContract](#wasmd.x.wasmd.v1beta1.MsgMigrateContract)
    - [MsgStoreCode](#wasmd.x.wasmd.v1beta1.MsgStoreCode)
    - [MsgUpdateAdmin](#wasmd.x.wasmd.v1beta1.MsgUpdateAdmin)
  
- [x/wasm/internal/types/proposal.proto](#x/wasm/internal/types/proposal.proto)
    - [ClearAdminProposal](#wasmd.x.wasmd.v1beta1.ClearAdminProposal)
    - [InstantiateContractProposal](#wasmd.x.wasmd.v1beta1.InstantiateContractProposal)
    - [MigrateContractProposal](#wasmd.x.wasmd.v1beta1.MigrateContractProposal)
    - [StoreCodeProposal](#wasmd.x.wasmd.v1beta1.StoreCodeProposal)
    - [UpdateAdminProposal](#wasmd.x.wasmd.v1beta1.UpdateAdminProposal)
  
- [x/wasm/internal/types/query.proto](#x/wasm/internal/types/query.proto)
    - [CodeInfoResponse](#wasmd.x.wasmd.v1beta1.CodeInfoResponse)
    - [ContractInfoWithAddress](#wasmd.x.wasmd.v1beta1.ContractInfoWithAddress)
    - [QueryAllContractStateRequest](#wasmd.x.wasmd.v1beta1.QueryAllContractStateRequest)
    - [QueryAllContractStateResponse](#wasmd.x.wasmd.v1beta1.QueryAllContractStateResponse)
    - [QueryCodeRequest](#wasmd.x.wasmd.v1beta1.QueryCodeRequest)
    - [QueryCodeResponse](#wasmd.x.wasmd.v1beta1.QueryCodeResponse)
    - [QueryCodesResponse](#wasmd.x.wasmd.v1beta1.QueryCodesResponse)
    - [QueryContractHistoryRequest](#wasmd.x.wasmd.v1beta1.QueryContractHistoryRequest)
    - [QueryContractHistoryResponse](#wasmd.x.wasmd.v1beta1.QueryContractHistoryResponse)
    - [QueryContractInfoRequest](#wasmd.x.wasmd.v1beta1.QueryContractInfoRequest)
    - [QueryContractInfoResponse](#wasmd.x.wasmd.v1beta1.QueryContractInfoResponse)
    - [QueryContractsByCodeRequest](#wasmd.x.wasmd.v1beta1.QueryContractsByCodeRequest)
    - [QueryContractsByCodeResponse](#wasmd.x.wasmd.v1beta1.QueryContractsByCodeResponse)
    - [QueryRawContractStateRequest](#wasmd.x.wasmd.v1beta1.QueryRawContractStateRequest)
    - [QueryRawContractStateResponse](#wasmd.x.wasmd.v1beta1.QueryRawContractStateResponse)
    - [QuerySmartContractStateRequest](#wasmd.x.wasmd.v1beta1.QuerySmartContractStateRequest)
    - [QuerySmartContractStateResponse](#wasmd.x.wasmd.v1beta1.QuerySmartContractStateResponse)
  
    - [Query](#wasmd.x.wasmd.v1beta1.Query)
  
- [x/wasm/internal/types/types.proto](#x/wasm/internal/types/types.proto)
    - [AbsoluteTxPosition](#wasmd.x.wasmd.v1beta1.AbsoluteTxPosition)
    - [AccessConfig](#wasmd.x.wasmd.v1beta1.AccessConfig)
    - [AccessTypeParam](#wasmd.x.wasmd.v1beta1.AccessTypeParam)
    - [CodeInfo](#wasmd.x.wasmd.v1beta1.CodeInfo)
    - [ContractCodeHistoryEntry](#wasmd.x.wasmd.v1beta1.ContractCodeHistoryEntry)
    - [ContractHistory](#wasmd.x.wasmd.v1beta1.ContractHistory)
    - [ContractInfo](#wasmd.x.wasmd.v1beta1.ContractInfo)
    - [Model](#wasmd.x.wasmd.v1beta1.Model)
    - [Params](#wasmd.x.wasmd.v1beta1.Params)
  
    - [AccessType](#wasmd.x.wasmd.v1beta1.AccessType)
    - [ContractCodeHistoryOperationType](#wasmd.x.wasmd.v1beta1.ContractCodeHistoryOperationType)
  
- [Scalar Value Types](#scalar-value-types)



<a name="x/wasm/internal/types/genesis.proto"></a>
<p align="right"><a href="#top">Top</a></p>

## x/wasm/internal/types/genesis.proto



<a name="wasmd.x.wasmd.v1beta1.Code"></a>

### Code
Code struct encompasses CodeInfo and CodeBytes


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| code_id | [uint64](#uint64) |  |  |
| code_info | [CodeInfo](#wasmd.x.wasmd.v1beta1.CodeInfo) |  |  |
| code_bytes | [bytes](#bytes) |  |  |






<a name="wasmd.x.wasmd.v1beta1.Contract"></a>

### Contract
Contract struct encompasses ContractAddress, ContractInfo, and ContractState


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| contract_address | [bytes](#bytes) |  |  |
| contract_info | [ContractInfo](#wasmd.x.wasmd.v1beta1.ContractInfo) |  |  |
| contract_state | [Model](#wasmd.x.wasmd.v1beta1.Model) | repeated |  |






<a name="wasmd.x.wasmd.v1beta1.GenesisState"></a>

### GenesisState
GenesisState - genesis state of x/wasm


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| params | [Params](#wasmd.x.wasmd.v1beta1.Params) |  |  |
| codes | [Code](#wasmd.x.wasmd.v1beta1.Code) | repeated |  |
| contracts | [Contract](#wasmd.x.wasmd.v1beta1.Contract) | repeated |  |
| sequences | [Sequence](#wasmd.x.wasmd.v1beta1.Sequence) | repeated |  |






<a name="wasmd.x.wasmd.v1beta1.Sequence"></a>

### Sequence
Sequence id and value of a counter


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id_key | [bytes](#bytes) |  |  |
| value | [uint64](#uint64) |  |  |





 

 

 

 



<a name="x/wasm/internal/types/msg.proto"></a>
<p align="right"><a href="#top">Top</a></p>

## x/wasm/internal/types/msg.proto



<a name="wasmd.x.wasmd.v1beta1.MsgClearAdmin"></a>

### MsgClearAdmin



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| sender | [bytes](#bytes) |  |  |
| contract | [bytes](#bytes) |  |  |






<a name="wasmd.x.wasmd.v1beta1.MsgExecuteContract"></a>

### MsgExecuteContract



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| sender | [bytes](#bytes) |  |  |
| contract | [bytes](#bytes) |  |  |
| msg | [bytes](#bytes) |  |  |
| sent_funds | [cosmos.base.v1beta1.Coin](#cosmos.base.v1beta1.Coin) | repeated |  |






<a name="wasmd.x.wasmd.v1beta1.MsgInstantiateContract"></a>

### MsgInstantiateContract



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| sender | [bytes](#bytes) |  |  |
| admin | [bytes](#bytes) |  | Admin is an optional address that can execute migrations |
| code_id | [uint64](#uint64) |  |  |
| label | [string](#string) |  |  |
| init_msg | [bytes](#bytes) |  |  |
| init_funds | [cosmos.base.v1beta1.Coin](#cosmos.base.v1beta1.Coin) | repeated |  |






<a name="wasmd.x.wasmd.v1beta1.MsgMigrateContract"></a>

### MsgMigrateContract



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| sender | [bytes](#bytes) |  |  |
| contract | [bytes](#bytes) |  |  |
| code_id | [uint64](#uint64) |  |  |
| migrate_msg | [bytes](#bytes) |  |  |






<a name="wasmd.x.wasmd.v1beta1.MsgStoreCode"></a>

### MsgStoreCode



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| sender | [bytes](#bytes) |  |  |
| wasm_byte_code | [bytes](#bytes) |  | WASMByteCode can be raw or gzip compressed |
| source | [string](#string) |  | Source is a valid absolute HTTPS URI to the contract&#39;s source code, optional |
| builder | [string](#string) |  | Builder is a valid docker image name with tag, optional |
| instantiate_permission | [AccessConfig](#wasmd.x.wasmd.v1beta1.AccessConfig) |  | InstantiatePermission to apply on contract creation, optional |






<a name="wasmd.x.wasmd.v1beta1.MsgUpdateAdmin"></a>

### MsgUpdateAdmin



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| sender | [bytes](#bytes) |  |  |
| new_admin | [bytes](#bytes) |  |  |
| contract | [bytes](#bytes) |  |  |





 

 

 

 



<a name="x/wasm/internal/types/proposal.proto"></a>
<p align="right"><a href="#top">Top</a></p>

## x/wasm/internal/types/proposal.proto



<a name="wasmd.x.wasmd.v1beta1.ClearAdminProposal"></a>

### ClearAdminProposal
ClearAdminProposal gov proposal content type to clear the admin of a contract.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| title | [string](#string) |  |  |
| description | [string](#string) |  |  |
| contract | [bytes](#bytes) |  |  |






<a name="wasmd.x.wasmd.v1beta1.InstantiateContractProposal"></a>

### InstantiateContractProposal
InstantiateContractProposal gov proposal content type to instantiate a contract.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| title | [string](#string) |  |  |
| description | [string](#string) |  |  |
| run_as | [bytes](#bytes) |  | RunAs is the address that is passed to the contract&#39;s environment as sender |
| admin | [bytes](#bytes) |  | Admin is an optional address that can execute migrations |
| code_id | [uint64](#uint64) |  |  |
| label | [string](#string) |  |  |
| init_msg | [bytes](#bytes) |  |  |
| init_funds | [cosmos.base.v1beta1.Coin](#cosmos.base.v1beta1.Coin) | repeated |  |






<a name="wasmd.x.wasmd.v1beta1.MigrateContractProposal"></a>

### MigrateContractProposal
MigrateContractProposal gov proposal content type to migrate a contract.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| title | [string](#string) |  |  |
| description | [string](#string) |  |  |
| run_as | [bytes](#bytes) |  | RunAs is the address that is passed to the contract&#39;s environment as sender |
| contract | [bytes](#bytes) |  |  |
| code_id | [uint64](#uint64) |  |  |
| migrate_msg | [bytes](#bytes) |  |  |






<a name="wasmd.x.wasmd.v1beta1.StoreCodeProposal"></a>

### StoreCodeProposal



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| title | [string](#string) |  |  |
| description | [string](#string) |  |  |
| run_as | [bytes](#bytes) |  | RunAs is the address that is passed to the contract&#39;s environment as sender |
| wasm_byte_code | [bytes](#bytes) |  | WASMByteCode can be raw or gzip compressed |
| source | [string](#string) |  | Source is a valid absolute HTTPS URI to the contract&#39;s source code, optional |
| builder | [string](#string) |  | Builder is a valid docker image name with tag, optional |
| instantiate_permission | [AccessConfig](#wasmd.x.wasmd.v1beta1.AccessConfig) |  | InstantiatePermission to apply on contract creation, optional |






<a name="wasmd.x.wasmd.v1beta1.UpdateAdminProposal"></a>

### UpdateAdminProposal
UpdateAdminProposal gov proposal content type to set an admin for a contract.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| title | [string](#string) |  |  |
| description | [string](#string) |  |  |
| new_admin | [bytes](#bytes) |  |  |
| contract | [bytes](#bytes) |  |  |





 

 

 

 



<a name="x/wasm/internal/types/query.proto"></a>
<p align="right"><a href="#top">Top</a></p>

## x/wasm/internal/types/query.proto



<a name="wasmd.x.wasmd.v1beta1.CodeInfoResponse"></a>

### CodeInfoResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| code_id | [uint64](#uint64) |  | id for legacy support |
| creator | [bytes](#bytes) |  |  |
| data_hash | [bytes](#bytes) |  |  |
| source | [string](#string) |  |  |
| builder | [string](#string) |  |  |






<a name="wasmd.x.wasmd.v1beta1.ContractInfoWithAddress"></a>

### ContractInfoWithAddress
ContractInfoWithAddress adds the address (key) to the ContractInfo representation


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| address | [bytes](#bytes) |  |  |
| contract_info | [ContractInfo](#wasmd.x.wasmd.v1beta1.ContractInfo) |  |  |






<a name="wasmd.x.wasmd.v1beta1.QueryAllContractStateRequest"></a>

### QueryAllContractStateRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| address | [bytes](#bytes) |  | address is the address of the contract |






<a name="wasmd.x.wasmd.v1beta1.QueryAllContractStateResponse"></a>

### QueryAllContractStateResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| models | [Model](#wasmd.x.wasmd.v1beta1.Model) | repeated |  |






<a name="wasmd.x.wasmd.v1beta1.QueryCodeRequest"></a>

### QueryCodeRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| code_id | [uint64](#uint64) |  | grpc-gateway_out does not support Go style CodID |






<a name="wasmd.x.wasmd.v1beta1.QueryCodeResponse"></a>

### QueryCodeResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| code_info | [CodeInfoResponse](#wasmd.x.wasmd.v1beta1.CodeInfoResponse) |  |  |
| data | [bytes](#bytes) |  |  |






<a name="wasmd.x.wasmd.v1beta1.QueryCodesResponse"></a>

### QueryCodesResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| code_infos | [CodeInfoResponse](#wasmd.x.wasmd.v1beta1.CodeInfoResponse) | repeated |  |






<a name="wasmd.x.wasmd.v1beta1.QueryContractHistoryRequest"></a>

### QueryContractHistoryRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| address | [bytes](#bytes) |  | address is the address of the contract to query |






<a name="wasmd.x.wasmd.v1beta1.QueryContractHistoryResponse"></a>

### QueryContractHistoryResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| entries | [ContractCodeHistoryEntry](#wasmd.x.wasmd.v1beta1.ContractCodeHistoryEntry) | repeated |  |






<a name="wasmd.x.wasmd.v1beta1.QueryContractInfoRequest"></a>

### QueryContractInfoRequest
QueryContractInfoRequest is the request type for the Query/ContractInfo RPC method


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| address | [bytes](#bytes) |  | address is the address of the contract to query |






<a name="wasmd.x.wasmd.v1beta1.QueryContractInfoResponse"></a>

### QueryContractInfoResponse
QueryContractInfoResponse is the response type for the Query/ContractInfo RPC method


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| address | [bytes](#bytes) |  | address is the address of the contract |
| contract_info | [ContractInfo](#wasmd.x.wasmd.v1beta1.ContractInfo) |  |  |






<a name="wasmd.x.wasmd.v1beta1.QueryContractsByCodeRequest"></a>

### QueryContractsByCodeRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| code_id | [uint64](#uint64) |  | grpc-gateway_out does not support Go style CodID |






<a name="wasmd.x.wasmd.v1beta1.QueryContractsByCodeResponse"></a>

### QueryContractsByCodeResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| contract_infos | [ContractInfoWithAddress](#wasmd.x.wasmd.v1beta1.ContractInfoWithAddress) | repeated |  |






<a name="wasmd.x.wasmd.v1beta1.QueryRawContractStateRequest"></a>

### QueryRawContractStateRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| address | [bytes](#bytes) |  | address is the address of the contract |
| query_data | [bytes](#bytes) |  |  |






<a name="wasmd.x.wasmd.v1beta1.QueryRawContractStateResponse"></a>

### QueryRawContractStateResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| data | [bytes](#bytes) |  |  |






<a name="wasmd.x.wasmd.v1beta1.QuerySmartContractStateRequest"></a>

### QuerySmartContractStateRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| address | [bytes](#bytes) |  | address is the address of the contract |
| query_data | [bytes](#bytes) |  |  |






<a name="wasmd.x.wasmd.v1beta1.QuerySmartContractStateResponse"></a>

### QuerySmartContractStateResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| data | [bytes](#bytes) |  |  |





 

 

 


<a name="wasmd.x.wasmd.v1beta1.Query"></a>

### Query
Query provides defines the gRPC querier service

| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| ContractInfo | [QueryContractInfoRequest](#wasmd.x.wasmd.v1beta1.QueryContractInfoRequest) | [QueryContractInfoResponse](#wasmd.x.wasmd.v1beta1.QueryContractInfoResponse) |  |
| ContractHistory | [QueryContractHistoryRequest](#wasmd.x.wasmd.v1beta1.QueryContractHistoryRequest) | [QueryContractHistoryResponse](#wasmd.x.wasmd.v1beta1.QueryContractHistoryResponse) |  |
| ContractsByCode | [QueryContractsByCodeRequest](#wasmd.x.wasmd.v1beta1.QueryContractsByCodeRequest) | [QueryContractsByCodeResponse](#wasmd.x.wasmd.v1beta1.QueryContractsByCodeResponse) |  |
| AllContractState | [QueryAllContractStateRequest](#wasmd.x.wasmd.v1beta1.QueryAllContractStateRequest) | [QueryAllContractStateResponse](#wasmd.x.wasmd.v1beta1.QueryAllContractStateResponse) |  |
| RawContractState | [QueryRawContractStateRequest](#wasmd.x.wasmd.v1beta1.QueryRawContractStateRequest) | [QueryRawContractStateResponse](#wasmd.x.wasmd.v1beta1.QueryRawContractStateResponse) |  |
| SmartContractState | [QuerySmartContractStateRequest](#wasmd.x.wasmd.v1beta1.QuerySmartContractStateRequest) | [QuerySmartContractStateResponse](#wasmd.x.wasmd.v1beta1.QuerySmartContractStateResponse) |  |
| Code | [QueryCodeRequest](#wasmd.x.wasmd.v1beta1.QueryCodeRequest) | [QueryCodeResponse](#wasmd.x.wasmd.v1beta1.QueryCodeResponse) |  |
| Codes | [.google.protobuf.Empty](#google.protobuf.Empty) | [QueryCodesResponse](#wasmd.x.wasmd.v1beta1.QueryCodesResponse) |  |

 



<a name="x/wasm/internal/types/types.proto"></a>
<p align="right"><a href="#top">Top</a></p>

## x/wasm/internal/types/types.proto



<a name="wasmd.x.wasmd.v1beta1.AbsoluteTxPosition"></a>

### AbsoluteTxPosition
AbsoluteTxPosition can be used to sort contracts


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| block_height | [int64](#int64) |  | BlockHeight is the block the contract was created at |
| tx_index | [uint64](#uint64) |  | TxIndex is a monotonic counter within the block (actual transaction index, or gas consumed) |






<a name="wasmd.x.wasmd.v1beta1.AccessConfig"></a>

### AccessConfig



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| permission | [AccessType](#wasmd.x.wasmd.v1beta1.AccessType) |  |  |
| address | [bytes](#bytes) |  |  |






<a name="wasmd.x.wasmd.v1beta1.AccessTypeParam"></a>

### AccessTypeParam



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| value | [AccessType](#wasmd.x.wasmd.v1beta1.AccessType) |  |  |






<a name="wasmd.x.wasmd.v1beta1.CodeInfo"></a>

### CodeInfo
CodeInfo is data for the uploaded contract WASM code


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| code_hash | [bytes](#bytes) |  |  |
| creator | [bytes](#bytes) |  |  |
| source | [string](#string) |  |  |
| builder | [string](#string) |  |  |
| instantiate_config | [AccessConfig](#wasmd.x.wasmd.v1beta1.AccessConfig) |  |  |






<a name="wasmd.x.wasmd.v1beta1.ContractCodeHistoryEntry"></a>

### ContractCodeHistoryEntry
ContractCodeHistoryEntry stores code updates to a contract.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| operation | [ContractCodeHistoryOperationType](#wasmd.x.wasmd.v1beta1.ContractCodeHistoryOperationType) |  |  |
| code_id | [uint64](#uint64) |  |  |
| updated | [AbsoluteTxPosition](#wasmd.x.wasmd.v1beta1.AbsoluteTxPosition) |  |  |
| msg | [bytes](#bytes) |  |  |






<a name="wasmd.x.wasmd.v1beta1.ContractHistory"></a>

### ContractHistory



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| code_history_entries | [ContractCodeHistoryEntry](#wasmd.x.wasmd.v1beta1.ContractCodeHistoryEntry) | repeated |  |






<a name="wasmd.x.wasmd.v1beta1.ContractInfo"></a>

### ContractInfo
ContractInfo stores a WASM contract instance


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| code_id | [uint64](#uint64) |  |  |
| creator | [bytes](#bytes) |  |  |
| admin | [bytes](#bytes) |  |  |
| label | [string](#string) |  |  |
| created | [AbsoluteTxPosition](#wasmd.x.wasmd.v1beta1.AbsoluteTxPosition) |  | never show this in query results, just use for sorting (Note: when using json tag &#34;-&#34; amino refused to serialize it...) |






<a name="wasmd.x.wasmd.v1beta1.Model"></a>

### Model
Model is a struct that holds a KV pair


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [bytes](#bytes) |  | hex-encode key to read it better (this is often ascii) |
| value | [bytes](#bytes) |  | base64-encode raw value |






<a name="wasmd.x.wasmd.v1beta1.Params"></a>

### Params
Params defines the set of wasm parameters.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| code_upload_access | [AccessConfig](#wasmd.x.wasmd.v1beta1.AccessConfig) |  |  |
| instantiate_default_permission | [AccessType](#wasmd.x.wasmd.v1beta1.AccessType) |  |  |
| max_wasm_code_size | [uint64](#uint64) |  |  |





 


<a name="wasmd.x.wasmd.v1beta1.AccessType"></a>

### AccessType


| Name | Number | Description |
| ---- | ------ | ----------- |
| ACCESS_TYPE_UNSPECIFIED | 0 |  |
| ACCESS_TYPE_NOBODY | 1 |  |
| ACCESS_TYPE_ONLY_ADDRESS | 2 |  |
| ACCESS_TYPE_EVERYBODY | 3 |  |



<a name="wasmd.x.wasmd.v1beta1.ContractCodeHistoryOperationType"></a>

### ContractCodeHistoryOperationType


| Name | Number | Description |
| ---- | ------ | ----------- |
| CONTRACT_CODE_HISTORY_OPERATION_TYPE_UNSPECIFIED | 0 |  |
| CONTRACT_CODE_HISTORY_OPERATION_TYPE_INIT | 1 |  |
| CONTRACT_CODE_HISTORY_OPERATION_TYPE_MIGRATE | 2 |  |
| CONTRACT_CODE_HISTORY_OPERATION_TYPE_GENESIS | 3 |  |


 

 

 



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

