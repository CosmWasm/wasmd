<!-- This file is auto-generated. Please do not modify it yourself. -->

# Protobuf Documentation

<a name="top"></a>

## Table of Contents

- [cosmwasm/tokenfactory/v1beta1/authorityMetadata.proto](#cosmwasm/tokenfactory/v1beta1/authorityMetadata.proto)
  - [DenomAuthorityMetadata](#cosmwasm.tokenfactory.v1beta1.DenomAuthorityMetadata)
- [cosmwasm/tokenfactory/v1beta1/params.proto](#cosmwasm/tokenfactory/v1beta1/params.proto)
  - [Params](#cosmwasm.tokenfactory.v1beta1.Params)
- [cosmwasm/tokenfactory/v1beta1/genesis.proto](#cosmwasm/tokenfactory/v1beta1/genesis.proto)
  - [GenesisDenom](#cosmwasm.tokenfactory.v1beta1.GenesisDenom)
  - [GenesisState](#cosmwasm.tokenfactory.v1beta1.GenesisState)
- [cosmwasm/tokenfactory/v1beta1/query.proto](#cosmwasm/tokenfactory/v1beta1/query.proto)

  - [QueryDenomAuthorityMetadataRequest](#cosmwasm.tokenfactory.v1beta1.QueryDenomAuthorityMetadataRequest)
  - [QueryDenomAuthorityMetadataResponse](#cosmwasm.tokenfactory.v1beta1.QueryDenomAuthorityMetadataResponse)
  - [QueryDenomsFromCreatorRequest](#cosmwasm.tokenfactory.v1beta1.QueryDenomsFromCreatorRequest)
  - [QueryDenomsFromCreatorResponse](#cosmwasm.tokenfactory.v1beta1.QueryDenomsFromCreatorResponse)
  - [QueryParamsRequest](#cosmwasm.tokenfactory.v1beta1.QueryParamsRequest)
  - [QueryParamsResponse](#cosmwasm.tokenfactory.v1beta1.QueryParamsResponse)

  - [Query](#cosmwasm.tokenfactory.v1beta1.Query)

- [cosmwasm/tokenfactory/v1beta1/tx.proto](#cosmwasm/tokenfactory/v1beta1/tx.proto)

  - [MsgBurn](#cosmwasm.tokenfactory.v1beta1.MsgBurn)
  - [MsgBurnResponse](#cosmwasm.tokenfactory.v1beta1.MsgBurnResponse)
  - [MsgChangeAdmin](#cosmwasm.tokenfactory.v1beta1.MsgChangeAdmin)
  - [MsgChangeAdminResponse](#cosmwasm.tokenfactory.v1beta1.MsgChangeAdminResponse)
  - [MsgCreateDenom](#cosmwasm.tokenfactory.v1beta1.MsgCreateDenom)
  - [MsgCreateDenomResponse](#cosmwasm.tokenfactory.v1beta1.MsgCreateDenomResponse)
  - [MsgForceTransfer](#cosmwasm.tokenfactory.v1beta1.MsgForceTransfer)
  - [MsgForceTransferResponse](#cosmwasm.tokenfactory.v1beta1.MsgForceTransferResponse)
  - [MsgMint](#cosmwasm.tokenfactory.v1beta1.MsgMint)
  - [MsgMintResponse](#cosmwasm.tokenfactory.v1beta1.MsgMintResponse)
  - [MsgSetDenomMetadata](#cosmwasm.tokenfactory.v1beta1.MsgSetDenomMetadata)
  - [MsgSetDenomMetadataResponse](#cosmwasm.tokenfactory.v1beta1.MsgSetDenomMetadataResponse)

  - [Msg](#cosmwasm.tokenfactory.v1beta1.Msg)

- [Scalar Value Types](#scalar-value-types)

<a name="cosmwasm/tokenfactory/v1beta1/authorityMetadata.proto"></a>

<p align="right"><a href="#top">Top</a></p>

## cosmwasm/tokenfactory/v1beta1/authorityMetadata.proto

<a name="cosmwasm.tokenfactory.v1beta1.DenomAuthorityMetadata"></a>

### DenomAuthorityMetadata

DenomAuthorityMetadata specifies metadata for addresses that have specific
capabilities over a token factory denom. Right now there is only one Admin
permission, but is planned to be extended to the future.

| Field   | Type              | Label | Description                                            |
| ------- | ----------------- | ----- | ------------------------------------------------------ |
| `admin` | [string](#string) |       | Can be empty for no admin, or a valid cosmwasm address |

 <!-- end messages -->

 <!-- end enums -->

 <!-- end HasExtensions -->

 <!-- end services -->

<a name="cosmwasm/tokenfactory/v1beta1/params.proto"></a>

<p align="right"><a href="#top">Top</a></p>

## cosmwasm/tokenfactory/v1beta1/params.proto

<a name="cosmwasm.tokenfactory.v1beta1.Params"></a>

### Params

Params defines the parameters for the tokenfactory module.

| Field                        | Type                                                  | Label    | Description                                 |
| ---------------------------- | ----------------------------------------------------- | -------- | ------------------------------------------- |
| `denom_creation_fee`         | [cosmos.base.v1beta1.Coin](#cosmos.base.v1beta1.Coin) | repeated |                                             |
| `denom_creation_gas_consume` | [uint64](#uint64)                                     |          | https://github.com/CosmWasm/wasmd/issues/11 |

 <!-- end messages -->

 <!-- end enums -->

 <!-- end HasExtensions -->

 <!-- end services -->

<a name="cosmwasm/tokenfactory/v1beta1/genesis.proto"></a>

<p align="right"><a href="#top">Top</a></p>

## cosmwasm/tokenfactory/v1beta1/genesis.proto

<a name="cosmwasm.tokenfactory.v1beta1.GenesisDenom"></a>

### GenesisDenom

GenesisDenom defines a tokenfactory denom that is defined within genesis
state. The structure contains DenomAuthorityMetadata which defines the
denom's admin.

| Field                | Type                                                                            | Label | Description |
| -------------------- | ------------------------------------------------------------------------------- | ----- | ----------- |
| `denom`              | [string](#string)                                                               |       |             |
| `authority_metadata` | [DenomAuthorityMetadata](#cosmwasm.tokenfactory.v1beta1.DenomAuthorityMetadata) |       |             |

<a name="cosmwasm.tokenfactory.v1beta1.GenesisState"></a>

### GenesisState

GenesisState defines the tokenfactory module's genesis state.

| Field            | Type                                                        | Label    | Description                                  |
| ---------------- | ----------------------------------------------------------- | -------- | -------------------------------------------- |
| `params`         | [Params](#cosmwasm.tokenfactory.v1beta1.Params)             |          | params defines the paramaters of the module. |
| `factory_denoms` | [GenesisDenom](#cosmwasm.tokenfactory.v1beta1.GenesisDenom) | repeated |                                              |

 <!-- end messages -->

 <!-- end enums -->

 <!-- end HasExtensions -->

 <!-- end services -->

<a name="cosmwasm/tokenfactory/v1beta1/query.proto"></a>

<p align="right"><a href="#top">Top</a></p>

## cosmwasm/tokenfactory/v1beta1/query.proto

<a name="cosmwasm.tokenfactory.v1beta1.QueryDenomAuthorityMetadataRequest"></a>

### QueryDenomAuthorityMetadataRequest

QueryDenomAuthorityMetadataRequest defines the request structure for the
DenomAuthorityMetadata gRPC query.

| Field   | Type              | Label | Description |
| ------- | ----------------- | ----- | ----------- |
| `denom` | [string](#string) |       |             |

<a name="cosmwasm.tokenfactory.v1beta1.QueryDenomAuthorityMetadataResponse"></a>

### QueryDenomAuthorityMetadataResponse

QueryDenomAuthorityMetadataResponse defines the response structure for the
DenomAuthorityMetadata gRPC query.

| Field                | Type                                                                            | Label | Description |
| -------------------- | ------------------------------------------------------------------------------- | ----- | ----------- |
| `authority_metadata` | [DenomAuthorityMetadata](#cosmwasm.tokenfactory.v1beta1.DenomAuthorityMetadata) |       |             |

<a name="cosmwasm.tokenfactory.v1beta1.QueryDenomsFromCreatorRequest"></a>

### QueryDenomsFromCreatorRequest

QueryDenomsFromCreatorRequest defines the request structure for the
DenomsFromCreator gRPC query.

| Field     | Type              | Label | Description |
| --------- | ----------------- | ----- | ----------- |
| `creator` | [string](#string) |       |             |

<a name="cosmwasm.tokenfactory.v1beta1.QueryDenomsFromCreatorResponse"></a>

### QueryDenomsFromCreatorResponse

QueryDenomsFromCreatorRequest defines the response structure for the
DenomsFromCreator gRPC query.

| Field    | Type              | Label    | Description |
| -------- | ----------------- | -------- | ----------- |
| `denoms` | [string](#string) | repeated |             |

<a name="cosmwasm.tokenfactory.v1beta1.QueryParamsRequest"></a>

### QueryParamsRequest

QueryParamsRequest is the request type for the Query/Params RPC method.

<a name="cosmwasm.tokenfactory.v1beta1.QueryParamsResponse"></a>

### QueryParamsResponse

QueryParamsResponse is the response type for the Query/Params RPC method.

| Field    | Type                                            | Label | Description                                  |
| -------- | ----------------------------------------------- | ----- | -------------------------------------------- |
| `params` | [Params](#cosmwasm.tokenfactory.v1beta1.Params) |       | params defines the parameters of the module. |

 <!-- end messages -->

 <!-- end enums -->

 <!-- end HasExtensions -->

<a name="cosmwasm.tokenfactory.v1beta1.Query"></a>

### Query

Query defines the gRPC querier service.

| Method Name              | Request Type                                                                                            | Response Type                                                                                             | Description                                                                                                       | HTTP Verb | Endpoint                                                         |
| ------------------------ | ------------------------------------------------------------------------------------------------------- | --------------------------------------------------------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------- | --------- | ---------------------------------------------------------------- |
| `Params`                 | [QueryParamsRequest](#cosmwasm.tokenfactory.v1beta1.QueryParamsRequest)                                 | [QueryParamsResponse](#cosmwasm.tokenfactory.v1beta1.QueryParamsResponse)                                 | Params defines a gRPC query method that returns the tokenfactory module's parameters.                             | GET       | /cosmwasm/tokenfactory/v1beta1/params                            |
| `DenomAuthorityMetadata` | [QueryDenomAuthorityMetadataRequest](#cosmwasm.tokenfactory.v1beta1.QueryDenomAuthorityMetadataRequest) | [QueryDenomAuthorityMetadataResponse](#cosmwasm.tokenfactory.v1beta1.QueryDenomAuthorityMetadataResponse) | DenomAuthorityMetadata defines a gRPC query method for fetching DenomAuthorityMetadata for a particular denom.    | GET       | /cosmwasm/tokenfactory/v1beta1/denoms/{denom}/authority_metadata |
| `DenomsFromCreator`      | [QueryDenomsFromCreatorRequest](#cosmwasm.tokenfactory.v1beta1.QueryDenomsFromCreatorRequest)           | [QueryDenomsFromCreatorResponse](#cosmwasm.tokenfactory.v1beta1.QueryDenomsFromCreatorResponse)           | DenomsFromCreator defines a gRPC query method for fetching all denominations created by a specific admin/creator. | GET       | /cosmwasm/tokenfactory/v1beta1/denoms_from_creator/{creator}     |

 <!-- end services -->

<a name="cosmwasm/tokenfactory/v1beta1/tx.proto"></a>

<p align="right"><a href="#top">Top</a></p>

## cosmwasm/tokenfactory/v1beta1/tx.proto

<a name="cosmwasm.tokenfactory.v1beta1.MsgBurn"></a>

### MsgBurn

MsgBurn is the sdk.Msg type for allowing an admin account to burn
a token. For now, we only support burning from the sender account.

| Field             | Type                                                  | Label | Description |
| ----------------- | ----------------------------------------------------- | ----- | ----------- |
| `sender`          | [string](#string)                                     |       |             |
| `amount`          | [cosmos.base.v1beta1.Coin](#cosmos.base.v1beta1.Coin) |       |             |
| `burnFromAddress` | [string](#string)                                     |       |             |

<a name="cosmwasm.tokenfactory.v1beta1.MsgBurnResponse"></a>

### MsgBurnResponse

<a name="cosmwasm.tokenfactory.v1beta1.MsgChangeAdmin"></a>

### MsgChangeAdmin

MsgChangeAdmin is the sdk.Msg type for allowing an admin account to reassign
adminship of a denom to a new account

| Field       | Type              | Label | Description |
| ----------- | ----------------- | ----- | ----------- |
| `sender`    | [string](#string) |       |             |
| `denom`     | [string](#string) |       |             |
| `new_admin` | [string](#string) |       |             |

<a name="cosmwasm.tokenfactory.v1beta1.MsgChangeAdminResponse"></a>

### MsgChangeAdminResponse

MsgChangeAdminResponse defines the response structure for an executed
MsgChangeAdmin message.

<a name="cosmwasm.tokenfactory.v1beta1.MsgCreateDenom"></a>

### MsgCreateDenom

MsgCreateDenom defines the message structure for the CreateDenom gRPC service
method. It allows an account to create a new denom. It requires a sender
address and a sub denomination. The (sender_address, sub_denomination) tuple
must be unique and cannot be re-used.

The resulting denom created is defined as
<factory/{creatorAddress}/{subdenom}>. The resulting denom's admin is
originally set to be the creator, but this can be changed later. The token
denom does not indicate the current admin.

| Field      | Type              | Label | Description                                              |
| ---------- | ----------------- | ----- | -------------------------------------------------------- |
| `sender`   | [string](#string) |       |                                                          |
| `subdenom` | [string](#string) |       | subdenom can be up to 44 "alphanumeric" characters long. |

<a name="cosmwasm.tokenfactory.v1beta1.MsgCreateDenomResponse"></a>

### MsgCreateDenomResponse

MsgCreateDenomResponse is the return value of MsgCreateDenom
It returns the full string of the newly created denom

| Field             | Type              | Label | Description |
| ----------------- | ----------------- | ----- | ----------- |
| `new_token_denom` | [string](#string) |       |             |

<a name="cosmwasm.tokenfactory.v1beta1.MsgForceTransfer"></a>

### MsgForceTransfer

| Field                 | Type                                                  | Label | Description |
| --------------------- | ----------------------------------------------------- | ----- | ----------- |
| `sender`              | [string](#string)                                     |       |             |
| `amount`              | [cosmos.base.v1beta1.Coin](#cosmos.base.v1beta1.Coin) |       |             |
| `transferFromAddress` | [string](#string)                                     |       |             |
| `transferToAddress`   | [string](#string)                                     |       |             |

<a name="cosmwasm.tokenfactory.v1beta1.MsgForceTransferResponse"></a>

### MsgInstantiateContract2

MsgInstantiateContract2 create a new smart contract instance for the given
code id with a predictable address.

| Field           | Type                                                  | Label | Description |
| --------------- | ----------------------------------------------------- | ----- | ----------- |
| `sender`        | [string](#string)                                     |       |             |
| `amount`        | [cosmos.base.v1beta1.Coin](#cosmos.base.v1beta1.Coin) |       |             |
| `mintToAddress` | [string](#string)                                     |       |             |

<a name="cosmwasm.tokenfactory.v1beta1.MsgMintResponse"></a>

### MsgMintResponse

<a name="cosmwasm.tokenfactory.v1beta1.MsgSetDenomMetadata"></a>

### MsgSetDenomMetadata

MsgSetDenomMetadata is the sdk.Msg type for allowing an admin account to set
the denom's bank metadata

| Field      | Type                                                          | Label | Description |
| ---------- | ------------------------------------------------------------- | ----- | ----------- |
| `sender`   | [string](#string)                                             |       |             |
| `metadata` | [cosmos.bank.v1beta1.Metadata](#cosmos.bank.v1beta1.Metadata) |       |             |

<a name="cosmwasm.tokenfactory.v1beta1.MsgSetDenomMetadataResponse"></a>

### MsgSetDenomMetadataResponse

MsgSetDenomMetadataResponse defines the response structure for an executed
MsgSetDenomMetadata message.

 <!-- end messages -->

 <!-- end enums -->

 <!-- end HasExtensions -->

<a name="cosmwasm.tokenfactory.v1beta1.Msg"></a>

### Msg

Msg defines the tokefactory module's gRPC message service.

| Method Name        | Request Type                                                              | Response Type                                                                             | Description | HTTP Verb | Endpoint |
| ------------------ | ------------------------------------------------------------------------- | ----------------------------------------------------------------------------------------- | ----------- | --------- | -------- |
| `CreateDenom`      | [MsgCreateDenom](#cosmwasm.tokenfactory.v1beta1.MsgCreateDenom)           | [MsgCreateDenomResponse](#cosmwasm.tokenfactory.v1beta1.MsgCreateDenomResponse)           |             |           |
| `Mint`             | [MsgMint](#cosmwasm.tokenfactory.v1beta1.MsgMint)                         | [MsgMintResponse](#cosmwasm.tokenfactory.v1beta1.MsgMintResponse)                         |             |           |
| `Burn`             | [MsgBurn](#cosmwasm.tokenfactory.v1beta1.MsgBurn)                         | [MsgBurnResponse](#cosmwasm.tokenfactory.v1beta1.MsgBurnResponse)                         |             |           |
| `ChangeAdmin`      | [MsgChangeAdmin](#cosmwasm.tokenfactory.v1beta1.MsgChangeAdmin)           | [MsgChangeAdminResponse](#cosmwasm.tokenfactory.v1beta1.MsgChangeAdminResponse)           |             |           |
| `SetDenomMetadata` | [MsgSetDenomMetadata](#cosmwasm.tokenfactory.v1beta1.MsgSetDenomMetadata) | [MsgSetDenomMetadataResponse](#cosmwasm.tokenfactory.v1beta1.MsgSetDenomMetadataResponse) |             |           |
| `ForceTransfer`    | [MsgForceTransfer](#cosmwasm.tokenfactory.v1beta1.MsgForceTransfer)       | [MsgForceTransferResponse](#cosmwasm.tokenfactory.v1beta1.MsgForceTransferResponse)       |             |           |

 <!-- end services -->

## Scalar Value Types

| .proto Type                    | Notes                                                                                                                                           | C++    | Java       | Python      | Go      | C#         | PHP            | Ruby                           |
| ------------------------------ | ----------------------------------------------------------------------------------------------------------------------------------------------- | ------ | ---------- | ----------- | ------- | ---------- | -------------- | ------------------------------ |
| <a name="double" /> double     |                                                                                                                                                 | double | double     | float       | float64 | double     | float          | Float                          |
| <a name="float" /> float       |                                                                                                                                                 | float  | float      | float       | float32 | float      | float          | Float                          |
| <a name="int32" /> int32       | Uses variable-length encoding. Inefficient for encoding negative numbers – if your field is likely to have negative values, use sint32 instead. | int32  | int        | int         | int32   | int        | integer        | Bignum or Fixnum (as required) |
| <a name="int64" /> int64       | Uses variable-length encoding. Inefficient for encoding negative numbers – if your field is likely to have negative values, use sint64 instead. | int64  | long       | int/long    | int64   | long       | integer/string | Bignum                         |
| <a name="uint32" /> uint32     | Uses variable-length encoding.                                                                                                                  | uint32 | int        | int/long    | uint32  | uint       | integer        | Bignum or Fixnum (as required) |
| <a name="uint64" /> uint64     | Uses variable-length encoding.                                                                                                                  | uint64 | long       | int/long    | uint64  | ulong      | integer/string | Bignum or Fixnum (as required) |
| <a name="sint32" /> sint32     | Uses variable-length encoding. Signed int value. These more efficiently encode negative numbers than regular int32s.                            | int32  | int        | int         | int32   | int        | integer        | Bignum or Fixnum (as required) |
| <a name="sint64" /> sint64     | Uses variable-length encoding. Signed int value. These more efficiently encode negative numbers than regular int64s.                            | int64  | long       | int/long    | int64   | long       | integer/string | Bignum                         |
| <a name="fixed32" /> fixed32   | Always four bytes. More efficient than uint32 if values are often greater than 2^28.                                                            | uint32 | int        | int         | uint32  | uint       | integer        | Bignum or Fixnum (as required) |
| <a name="fixed64" /> fixed64   | Always eight bytes. More efficient than uint64 if values are often greater than 2^56.                                                           | uint64 | long       | int/long    | uint64  | ulong      | integer/string | Bignum                         |
| <a name="sfixed32" /> sfixed32 | Always four bytes.                                                                                                                              | int32  | int        | int         | int32   | int        | integer        | Bignum or Fixnum (as required) |
| <a name="sfixed64" /> sfixed64 | Always eight bytes.                                                                                                                             | int64  | long       | int/long    | int64   | long       | integer/string | Bignum                         |
| <a name="bool" /> bool         |                                                                                                                                                 | bool   | boolean    | boolean     | bool    | bool       | boolean        | TrueClass/FalseClass           |
| <a name="string" /> string     | A string must always contain UTF-8 encoded or 7-bit ASCII text.                                                                                 | string | String     | str/unicode | string  | string     | string         | String (UTF-8)                 |
| <a name="bytes" /> bytes       | May contain any arbitrary sequence of bytes.                                                                                                    | string | ByteString | str         | []byte  | ByteString | string         | String (ASCII-8BIT)            |
