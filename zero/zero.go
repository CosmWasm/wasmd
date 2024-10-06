package symbols

import (
	wasmvm "github.com/CosmWasm/wasmvm/v2"
	wasmvmtypes "github.com/CosmWasm/wasmvm/v2/types"
)

// These variables are the variables and functions used in wasmd from wasmvm

var (
	// functions

	CreateChecksum   = wasmvm.CreateChecksum
	LibwasmvmVersion = wasmvm.LibwasmvmVersion
	NewVM            = wasmvm.NewVM
	ToSystemError    = wasmvmtypes.ToSystemError
	NewCoin          = wasmvmtypes.NewCoin

	// types

	GasMeter wasmvm.GasMeter
	GoAPI    wasmvm.GoAPI
	KVStore  wasmvm.KVStore
	Querier  wasmvm.Querier
	WasmCode wasmvm.WasmCode

	AllBalancesResponse              wasmvmtypes.AllBalancesResponse
	AllDelegationsResponse           wasmvmtypes.AllDelegationsResponse
	AllDenomMetadataQuery            wasmvmtypes.AllDenomMetadataQuery
	AllValidatorsResponse            wasmvmtypes.AllValidatorsResponse
	AnalysisReport                   wasmvmtypes.AnalysisReport
	AnyMsg                           wasmvmtypes.AnyMsg
	BalanceResponse                  wasmvmtypes.BalanceResponse
	BankMsg                          wasmvmtypes.BankMsg
	BankQuery                        wasmvmtypes.BankQuery
	BondedDenomResponse              wasmvmtypes.BondedDenomResponse
	ChannelResponse                  wasmvmtypes.ChannelResponse
	Checksum                         wasmvmtypes.Checksum
	CodeInfoResponse                 wasmvmtypes.CodeInfoResponse
	Coin                             wasmvmtypes.Coin
	ContractInfo                     wasmvmtypes.ContractInfo
	ContractInfoResponse             wasmvmtypes.ContractInfoResponse
	ContractResult                   wasmvmtypes.ContractResult
	CosmosMsg                        wasmvmtypes.CosmosMsg
	DecCoin                          wasmvmtypes.DecCoin
	Delegation                       wasmvmtypes.Delegation
	DelegationResponse               wasmvmtypes.DelegationResponse
	DelegationRewardsResponse        wasmvmtypes.DelegationRewardsResponse
	DelegationTotalRewardsResponse   wasmvmtypes.DelegationTotalRewardsResponse
	DelegatorReward                  wasmvmtypes.DelegatorReward
	DelegatorValidatorsResponse      wasmvmtypes.DelegatorValidatorsResponse
	DelegatorWithdrawAddressResponse wasmvmtypes.DelegatorWithdrawAddressResponse
	DenomMetadata                    wasmvmtypes.DenomMetadata
	DenomMetadataResponse            wasmvmtypes.DenomMetadataResponse
	DistributionMsg                  wasmvmtypes.DistributionMsg
	DistributionQuery                wasmvmtypes.DistributionQuery
	Env                              wasmvmtypes.Env
	Event                            wasmvmtypes.Event
	EventAttribute                   wasmvmtypes.EventAttribute
	FullDelegation                   wasmvmtypes.FullDelegation
	GovMsg                           wasmvmtypes.GovMsg
	GrpcQuery                        wasmvmtypes.GrpcQuery
	IBCAckCallbackMsg                wasmvmtypes.IBCAckCallbackMsg
	IBCAcknowledgement               wasmvmtypes.IBCAcknowledgement
	IBCBasicResponse                 wasmvmtypes.IBCBasicResponse
	IBCBasicResult                   wasmvmtypes.IBCBasicResult
	IBCChannel                       wasmvmtypes.IBCChannel
	IBCChannelCloseMsg               wasmvmtypes.IBCChannelCloseMsg
	IBCChannelConnectMsg             wasmvmtypes.IBCChannelConnectMsg
	IBCChannelOpenMsg                wasmvmtypes.IBCChannelOpenMsg
	IBCChannelOpenResult             wasmvmtypes.IBCChannelOpenResult
	IBCCloseConfirm                  wasmvmtypes.IBCCloseConfirm
	IBCCloseInit                     wasmvmtypes.IBCCloseInit
	IBCDestinationCallbackMsg        wasmvmtypes.IBCDestinationCallbackMsg
	IBCEndpoint                      wasmvmtypes.IBCEndpoint
	IBCMsg                           wasmvmtypes.IBCMsg
	IBCOpenAck                       wasmvmtypes.IBCOpenAck
	IBCOpenConfirm                   wasmvmtypes.IBCOpenConfirm
	IBCOpenInit                      wasmvmtypes.IBCOpenInit
	IBCOpenTry                       wasmvmtypes.IBCOpenTry
	IBCPacket                        wasmvmtypes.IBCPacket
	IBCPacketAckMsg                  wasmvmtypes.IBCPacketAckMsg
	IBCPacketReceiveMsg              wasmvmtypes.IBCPacketReceiveMsg
	IBCPacketTimeoutMsg              wasmvmtypes.IBCPacketTimeoutMsg
	IBCQuery                         wasmvmtypes.IBCQuery
	IBCReceiveResult                 wasmvmtypes.IBCReceiveResult
	IBCSourceCallbackMsg             wasmvmtypes.IBCSourceCallbackMsg
	IBCTimeout                       wasmvmtypes.IBCTimeout
	IBCTimeoutBlock                  wasmvmtypes.IBCTimeoutBlock
	IBCTimeoutCallbackMsg            wasmvmtypes.IBCTimeoutCallbackMsg
	Iterator                         wasmvmtypes.Iterator
	ListChannelsResponse             wasmvmtypes.ListChannelsResponse
	MessageInfo                      wasmvmtypes.MessageInfo
	Metrics                          wasmvmtypes.Metrics
	MsgResponse                      wasmvmtypes.MsgResponse
	PinnedMetrics                    wasmvmtypes.PinnedMetrics
	PortIDResponse                   wasmvmtypes.PortIDResponse
	QueryRequest                     wasmvmtypes.QueryRequest
	Reply                            wasmvmtypes.Reply
	Response                         wasmvmtypes.Response
	StakingMsg                       wasmvmtypes.StakingMsg
	StakingQuery                     wasmvmtypes.StakingQuery
	StargateQuery                    wasmvmtypes.StargateQuery
	SubMsg                           wasmvmtypes.SubMsg
	SubMsgResponse                   wasmvmtypes.SubMsgResponse
	SubMsgResult                     wasmvmtypes.SubMsgResult
	SupplyResponse                   wasmvmtypes.SupplyResponse
	UFraction                        wasmvmtypes.UFraction
	Unknown                          wasmvmtypes.Unknown
	UnsupportedRequest               wasmvmtypes.UnsupportedRequest
	Validator                        wasmvmtypes.Validator
	ValidatorResponse                wasmvmtypes.ValidatorResponse
	WasmMsg                          wasmvmtypes.WasmMsg
	WasmQuery                        wasmvmtypes.WasmQuery
	arrayEvent                       wasmvmtypes.Array[wasmvmtypes.Event]
	NoSuchContract                   wasmvmtypes.NoSuchContract
	NoSuchCode                       wasmvmtypes.NoSuchCode

	Yes          = wasmvmtypes.Yes
	No           = wasmvmtypes.No
	NoWithVeto   = wasmvmtypes.NoWithVeto
	Abstain      = wasmvmtypes.Abstain
	ReplySuccess = wasmvmtypes.ReplySuccess
	ReplyError   = wasmvmtypes.ReplyError
	ReplyAlways  = wasmvmtypes.ReplyAlways
	ReplyNever   = wasmvmtypes.ReplyNever
)
