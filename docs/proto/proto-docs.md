<!-- This file is auto-generated. Please do not modify it yourself. -->
# Protobuf Documentation
<a name="top"></a>

## Table of Contents

- [cosmwasm/evmutil/v1beta1/conversion_pair.proto](#cosmwasm/evmutil/v1beta1/conversion_pair.proto)
    - [ConversionPair](#cosmwasm.evmutil.v1beta1.ConversionPair)
  
- [cosmwasm/evmutil/v1beta1/genesis.proto](#cosmwasm/evmutil/v1beta1/genesis.proto)
    - [Account](#cosmwasm.evmutil.v1beta1.Account)
    - [GenesisState](#cosmwasm.evmutil.v1beta1.GenesisState)
    - [Params](#cosmwasm.evmutil.v1beta1.Params)
  
- [cosmwasm/evmutil/v1beta1/query.proto](#cosmwasm/evmutil/v1beta1/query.proto)
    - [QueryParamsRequest](#cosmwasm.evmutil.v1beta1.QueryParamsRequest)
    - [QueryParamsResponse](#cosmwasm.evmutil.v1beta1.QueryParamsResponse)
  
    - [Query](#cosmwasm.evmutil.v1beta1.Query)
  
- [cosmwasm/evmutil/v1beta1/tx.proto](#cosmwasm/evmutil/v1beta1/tx.proto)
    - [MsgConvertCoinToERC20](#cosmwasm.evmutil.v1beta1.MsgConvertCoinToERC20)
    - [MsgConvertCoinToERC20Response](#cosmwasm.evmutil.v1beta1.MsgConvertCoinToERC20Response)
    - [MsgConvertERC20ToCoin](#cosmwasm.evmutil.v1beta1.MsgConvertERC20ToCoin)
    - [MsgConvertERC20ToCoinResponse](#cosmwasm.evmutil.v1beta1.MsgConvertERC20ToCoinResponse)
  
    - [Msg](#cosmwasm.evmutil.v1beta1.Msg)
  
- [Scalar Value Types](#scalar-value-types)



<a name="cosmwasm/evmutil/v1beta1/conversion_pair.proto"></a>
<p align="right"><a href="#top">Top</a></p>

## cosmwasm/evmutil/v1beta1/conversion_pair.proto



<a name="cosmwasm.evmutil.v1beta1.ConversionPair"></a>

### ConversionPair
ConversionPair defines a Kava ERC20 address and corresponding denom that is
allowed to be converted between ERC20 and sdk.Coin


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `orai_erc20_address` | [bytes](#bytes) |  | ERC20 address of the token on the Kava EVM |
| `denom` | [string](#string) |  | Denom of the corresponding sdk.Coin |





 <!-- end messages -->

 <!-- end enums -->

 <!-- end HasExtensions -->

 <!-- end services -->



<a name="cosmwasm/evmutil/v1beta1/genesis.proto"></a>
<p align="right"><a href="#top">Top</a></p>

## cosmwasm/evmutil/v1beta1/genesis.proto



<a name="cosmwasm.evmutil.v1beta1.Account"></a>

### Account
BalanceAccount defines an account in the evmutil module.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `address` | [bytes](#bytes) |  |  |
| `balance` | [string](#string) |  | balance indicates the amount of a orai owned by the address. |






<a name="cosmwasm.evmutil.v1beta1.GenesisState"></a>

### GenesisState
GenesisState defines the evmutil module's genesis state.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `accounts` | [Account](#cosmwasm.evmutil.v1beta1.Account) | repeated |  |
| `params` | [Params](#cosmwasm.evmutil.v1beta1.Params) |  | params defines all the parameters of the module. |






<a name="cosmwasm.evmutil.v1beta1.Params"></a>

### Params
Params defines the evmutil module params


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `enabled_conversion_pairs` | [ConversionPair](#cosmwasm.evmutil.v1beta1.ConversionPair) | repeated | enabled_conversion_pairs defines the list of conversion pairs allowed to be converted between Kava ERC20 and sdk.Coin |





 <!-- end messages -->

 <!-- end enums -->

 <!-- end HasExtensions -->

 <!-- end services -->



<a name="cosmwasm/evmutil/v1beta1/query.proto"></a>
<p align="right"><a href="#top">Top</a></p>

## cosmwasm/evmutil/v1beta1/query.proto



<a name="cosmwasm.evmutil.v1beta1.QueryParamsRequest"></a>

### QueryParamsRequest
QueryParamsRequest defines the request type for querying x/evmutil
parameters.






<a name="cosmwasm.evmutil.v1beta1.QueryParamsResponse"></a>

### QueryParamsResponse
QueryParamsResponse defines the response type for querying x/evmutil
parameters.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `params` | [Params](#cosmwasm.evmutil.v1beta1.Params) |  |  |





 <!-- end messages -->

 <!-- end enums -->

 <!-- end HasExtensions -->


<a name="cosmwasm.evmutil.v1beta1.Query"></a>

### Query
Query defines the gRPC querier service for evmutil module

| Method Name | Request Type | Response Type | Description | HTTP Verb | Endpoint |
| ----------- | ------------ | ------------- | ------------| ------- | -------- |
| `Params` | [QueryParamsRequest](#cosmwasm.evmutil.v1beta1.QueryParamsRequest) | [QueryParamsResponse](#cosmwasm.evmutil.v1beta1.QueryParamsResponse) | Params queries all parameters of the evmutil module. | GET|/kava/evmutil/v1beta1/params|

 <!-- end services -->



<a name="cosmwasm/evmutil/v1beta1/tx.proto"></a>
<p align="right"><a href="#top">Top</a></p>

## cosmwasm/evmutil/v1beta1/tx.proto



<a name="cosmwasm.evmutil.v1beta1.MsgConvertCoinToERC20"></a>

### MsgConvertCoinToERC20
MsgConvertCoinToERC20 defines a conversion from sdk.Coin to Kava ERC20.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `initiator` | [string](#string) |  | Kava bech32 address initiating the conversion. |
| `receiver` | [string](#string) |  | EVM 0x hex address that will receive the converted Kava ERC20 tokens. |
| `amount` | [cosmos.base.v1beta1.Coin](#cosmos.base.v1beta1.Coin) |  | Amount is the sdk.Coin amount to convert. |






<a name="cosmwasm.evmutil.v1beta1.MsgConvertCoinToERC20Response"></a>

### MsgConvertCoinToERC20Response
MsgConvertCoinToERC20Response defines the response value from
Msg/ConvertCoinToERC20.






<a name="cosmwasm.evmutil.v1beta1.MsgConvertERC20ToCoin"></a>

### MsgConvertERC20ToCoin
MsgConvertERC20ToCoin defines a conversion from Kava ERC20 to sdk.Coin.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `initiator` | [string](#string) |  | EVM 0x hex address initiating the conversion. |
| `receiver` | [string](#string) |  | Kava bech32 address that will receive the converted sdk.Coin. |
| `orai_erc20_address` | [string](#string) |  | EVM 0x hex address of the ERC20 contract. |
| `amount` | [string](#string) |  | ERC20 token amount to convert. |






<a name="cosmwasm.evmutil.v1beta1.MsgConvertERC20ToCoinResponse"></a>

### MsgConvertERC20ToCoinResponse
MsgConvertERC20ToCoinResponse defines the response value from
Msg/MsgConvertERC20ToCoin.





 <!-- end messages -->

 <!-- end enums -->

 <!-- end HasExtensions -->


<a name="cosmwasm.evmutil.v1beta1.Msg"></a>

### Msg
Msg defines the evmutil Msg service.

| Method Name | Request Type | Response Type | Description | HTTP Verb | Endpoint |
| ----------- | ------------ | ------------- | ------------| ------- | -------- |
| `ConvertCoinToERC20` | [MsgConvertCoinToERC20](#cosmwasm.evmutil.v1beta1.MsgConvertCoinToERC20) | [MsgConvertCoinToERC20Response](#cosmwasm.evmutil.v1beta1.MsgConvertCoinToERC20Response) | ConvertCoinToERC20 defines a method for converting sdk.Coin to Kava ERC20. | |
| `ConvertERC20ToCoin` | [MsgConvertERC20ToCoin](#cosmwasm.evmutil.v1beta1.MsgConvertERC20ToCoin) | [MsgConvertERC20ToCoinResponse](#cosmwasm.evmutil.v1beta1.MsgConvertERC20ToCoinResponse) | ConvertERC20ToCoin defines a method for converting Kava ERC20 to sdk.Coin. | |

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

