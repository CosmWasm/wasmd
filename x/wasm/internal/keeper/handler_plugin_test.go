package keeper

import (
	"encoding/json"
	"github.com/CosmWasm/wasmd/x/wasm/internal/keeper/wasmtesting"
	capabilitytypes "github.com/cosmos/cosmos-sdk/x/capability/types"
	ibctransfertypes "github.com/cosmos/cosmos-sdk/x/ibc/applications/transfer/types"
	clienttypes "github.com/cosmos/cosmos-sdk/x/ibc/core/02-client/types"
	channeltypes "github.com/cosmos/cosmos-sdk/x/ibc/core/04-channel/types"
	ibcexported "github.com/cosmos/cosmos-sdk/x/ibc/core/exported"
	"github.com/golang/protobuf/proto"
	"testing"

	"github.com/CosmWasm/wasmd/x/wasm/internal/types"
	wasmvmtypes "github.com/CosmWasm/wasmvm/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	distributiontypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEncoding(t *testing.T) {
	addr1 := RandomAccountAddress(t)
	addr2 := RandomAccountAddress(t)
	invalidAddr := "xrnd1d02kd90n38qvr3qb9qof83fn2d2"
	valAddr := make(sdk.ValAddress, sdk.AddrLen)
	valAddr[0] = 12
	valAddr2 := make(sdk.ValAddress, sdk.AddrLen)
	valAddr2[1] = 123
	var timeoutVal uint64 = 100

	jsonMsg := json.RawMessage(`{"foo": 123}`)

	bankMsg := &banktypes.MsgSend{
		FromAddress: addr2.String(),
		ToAddress:   addr1.String(),
		Amount: sdk.Coins{
			sdk.NewInt64Coin("uatom", 12345),
			sdk.NewInt64Coin("utgd", 54321),
		},
	}
	bankMsgBin, err := proto.Marshal(bankMsg)
	require.NoError(t, err)

	cases := map[string]struct {
		sender     sdk.AccAddress
		srcMsg     wasmvmtypes.CosmosMsg
		srcIBCPort string
		// set if valid
		output []sdk.Msg
		// set if invalid
		isError bool
	}{
		"simple send": {
			sender: addr1,
			srcMsg: wasmvmtypes.CosmosMsg{
				Bank: &wasmvmtypes.BankMsg{
					Send: &wasmvmtypes.SendMsg{
						ToAddress: addr2.String(),
						Amount: []wasmvmtypes.Coin{
							{
								Denom:  "uatom",
								Amount: "12345",
							},
							{
								Denom:  "usdt",
								Amount: "54321",
							},
						},
					},
				},
			},
			output: []sdk.Msg{
				&banktypes.MsgSend{
					FromAddress: addr1.String(),
					ToAddress:   addr2.String(),
					Amount: sdk.Coins{
						sdk.NewInt64Coin("uatom", 12345),
						sdk.NewInt64Coin("usdt", 54321),
					},
				},
			},
		},
		"invalid send amount": {
			sender: addr1,
			srcMsg: wasmvmtypes.CosmosMsg{
				Bank: &wasmvmtypes.BankMsg{
					Send: &wasmvmtypes.SendMsg{
						ToAddress: addr2.String(),
						Amount: []wasmvmtypes.Coin{
							{
								Denom:  "uatom",
								Amount: "123.456",
							},
						},
					},
				},
			},
			isError: true,
		},
		"invalid address": {
			sender: addr1,
			srcMsg: wasmvmtypes.CosmosMsg{
				Bank: &wasmvmtypes.BankMsg{
					Send: &wasmvmtypes.SendMsg{
						ToAddress: invalidAddr,
						Amount: []wasmvmtypes.Coin{
							{
								Denom:  "uatom",
								Amount: "7890",
							},
						},
					},
				},
			},
			isError: false, // addresses are checked in the handler
			output: []sdk.Msg{
				&banktypes.MsgSend{
					FromAddress: addr1.String(),
					ToAddress:   invalidAddr,
					Amount: sdk.Coins{
						sdk.NewInt64Coin("uatom", 7890),
					},
				},
			},
		},
		"wasm execute": {
			sender: addr1,
			srcMsg: wasmvmtypes.CosmosMsg{
				Wasm: &wasmvmtypes.WasmMsg{
					Execute: &wasmvmtypes.ExecuteMsg{
						ContractAddr: addr2.String(),
						Msg:          jsonMsg,
						Send: []wasmvmtypes.Coin{
							wasmvmtypes.NewCoin(12, "eth"),
						},
					},
				},
			},
			output: []sdk.Msg{
				&types.MsgExecuteContract{
					Sender:   addr1.String(),
					Contract: addr2.String(),
					Msg:      jsonMsg,
					Funds:    sdk.NewCoins(sdk.NewInt64Coin("eth", 12)),
				},
			},
		},
		"wasm instantiate": {
			sender: addr1,
			srcMsg: wasmvmtypes.CosmosMsg{
				Wasm: &wasmvmtypes.WasmMsg{
					Instantiate: &wasmvmtypes.InstantiateMsg{
						CodeID: 7,
						Msg:    jsonMsg,
						Send: []wasmvmtypes.Coin{
							wasmvmtypes.NewCoin(123, "eth"),
						},
						Label: "myLabel",
					},
				},
			},
			output: []sdk.Msg{
				&types.MsgInstantiateContract{
					Sender:  addr1.String(),
					CodeID:  7,
					Label:   "myLabel",
					InitMsg: jsonMsg,
					Funds:   sdk.NewCoins(sdk.NewInt64Coin("eth", 123)),
				},
			},
		},
		"staking delegate": {
			sender: addr1,
			srcMsg: wasmvmtypes.CosmosMsg{
				Staking: &wasmvmtypes.StakingMsg{
					Delegate: &wasmvmtypes.DelegateMsg{
						Validator: valAddr.String(),
						Amount:    wasmvmtypes.NewCoin(777, "stake"),
					},
				},
			},
			output: []sdk.Msg{
				&stakingtypes.MsgDelegate{
					DelegatorAddress: addr1.String(),
					ValidatorAddress: valAddr.String(),
					Amount:           sdk.NewInt64Coin("stake", 777),
				},
			},
		},
		"staking delegate to non-validator": {
			sender: addr1,
			srcMsg: wasmvmtypes.CosmosMsg{
				Staking: &wasmvmtypes.StakingMsg{
					Delegate: &wasmvmtypes.DelegateMsg{
						Validator: addr2.String(),
						Amount:    wasmvmtypes.NewCoin(777, "stake"),
					},
				},
			},
			isError: false, // fails in the handler
			output: []sdk.Msg{
				&stakingtypes.MsgDelegate{
					DelegatorAddress: addr1.String(),
					ValidatorAddress: addr2.String(),
					Amount:           sdk.NewInt64Coin("stake", 777),
				},
			},
		},
		"staking undelegate": {
			sender: addr1,
			srcMsg: wasmvmtypes.CosmosMsg{
				Staking: &wasmvmtypes.StakingMsg{
					Undelegate: &wasmvmtypes.UndelegateMsg{
						Validator: valAddr.String(),
						Amount:    wasmvmtypes.NewCoin(555, "stake"),
					},
				},
			},
			output: []sdk.Msg{
				&stakingtypes.MsgUndelegate{
					DelegatorAddress: addr1.String(),
					ValidatorAddress: valAddr.String(),
					Amount:           sdk.NewInt64Coin("stake", 555),
				},
			},
		},
		"staking redelegate": {
			sender: addr1,
			srcMsg: wasmvmtypes.CosmosMsg{
				Staking: &wasmvmtypes.StakingMsg{
					Redelegate: &wasmvmtypes.RedelegateMsg{
						SrcValidator: valAddr.String(),
						DstValidator: valAddr2.String(),
						Amount:       wasmvmtypes.NewCoin(222, "stake"),
					},
				},
			},
			output: []sdk.Msg{
				&stakingtypes.MsgBeginRedelegate{
					DelegatorAddress:    addr1.String(),
					ValidatorSrcAddress: valAddr.String(),
					ValidatorDstAddress: valAddr2.String(),
					Amount:              sdk.NewInt64Coin("stake", 222),
				},
			},
		},
		"staking withdraw (implicit recipient)": {
			sender: addr1,
			srcMsg: wasmvmtypes.CosmosMsg{
				Staking: &wasmvmtypes.StakingMsg{
					Withdraw: &wasmvmtypes.WithdrawMsg{
						Validator: valAddr2.String(),
					},
				},
			},
			output: []sdk.Msg{
				&distributiontypes.MsgSetWithdrawAddress{
					DelegatorAddress: addr1.String(),
					WithdrawAddress:  addr1.String(),
				},
				&distributiontypes.MsgWithdrawDelegatorReward{
					DelegatorAddress: addr1.String(),
					ValidatorAddress: valAddr2.String(),
				},
			},
		},
		"staking withdraw (explicit recipient)": {
			sender: addr1,
			srcMsg: wasmvmtypes.CosmosMsg{
				Staking: &wasmvmtypes.StakingMsg{
					Withdraw: &wasmvmtypes.WithdrawMsg{
						Validator: valAddr2.String(),
						Recipient: addr2.String(),
					},
				},
			},
			output: []sdk.Msg{
				&distributiontypes.MsgSetWithdrawAddress{
					DelegatorAddress: addr1.String(),
					WithdrawAddress:  addr2.String(),
				},
				&distributiontypes.MsgWithdrawDelegatorReward{
					DelegatorAddress: addr1.String(),
					ValidatorAddress: valAddr2.String(),
				},
			},
		},
		"stargate encoded bank msg": {
			sender: addr2,
			srcMsg: wasmvmtypes.CosmosMsg{
				Stargate: &wasmvmtypes.StargateMsg{
					TypeURL: "/cosmos.bank.v1beta1.MsgSend",
					Value:   bankMsgBin,
				},
			},
			output: []sdk.Msg{bankMsg},
		},
		"stargate encoded invalid typeUrl": {
			sender: addr2,
			srcMsg: wasmvmtypes.CosmosMsg{
				Stargate: &wasmvmtypes.StargateMsg{
					TypeURL: "/cosmos.bank.v2.MsgSend",
					Value:   bankMsgBin,
				},
			},
			isError: true,
		},
		"IBC transfer with block timeout": {
			sender:     addr1,
			srcIBCPort: "myIBCPort",
			srcMsg: wasmvmtypes.CosmosMsg{
				IBC: &wasmvmtypes.IBCMsg{
					Transfer: &wasmvmtypes.TransferMsg{
						ChannelID: "myChanID",
						ToAddress: addr2.String(),
						Amount: wasmvmtypes.Coin{
							Denom:  "ALX",
							Amount: "1",
						},
						TimeoutBlock: &wasmvmtypes.IBCTimeoutBlock{Revision: 1, Height: 2},
					},
				},
			},
			output: []sdk.Msg{
				&ibctransfertypes.MsgTransfer{
					SourcePort:    "transfer",
					SourceChannel: "myChanID",
					Token: sdk.Coin{
						Denom:  "ALX",
						Amount: sdk.NewInt(1),
					},
					Sender:        addr1.String(),
					Receiver:      addr2.String(),
					TimeoutHeight: clienttypes.Height{RevisionNumber: 1, RevisionHeight: 2},
				},
			},
		},
		"IBC transfer with time timeout": {
			sender:     addr1,
			srcIBCPort: "myIBCPort",
			srcMsg: wasmvmtypes.CosmosMsg{
				IBC: &wasmvmtypes.IBCMsg{
					Transfer: &wasmvmtypes.TransferMsg{
						ChannelID: "myChanID",
						ToAddress: addr2.String(),
						Amount: wasmvmtypes.Coin{
							Denom:  "ALX",
							Amount: "1",
						},
						TimeoutTimestamp: &timeoutVal,
					},
				},
			},
			output: []sdk.Msg{
				&ibctransfertypes.MsgTransfer{
					SourcePort:    "transfer",
					SourceChannel: "myChanID",
					Token: sdk.Coin{
						Denom:  "ALX",
						Amount: sdk.NewInt(1),
					},
					Sender:           addr1.String(),
					Receiver:         addr2.String(),
					TimeoutTimestamp: 100,
				},
			},
		},
		"IBC close channel": {
			sender:     addr1,
			srcIBCPort: "myIBCPort",
			srcMsg: wasmvmtypes.CosmosMsg{
				IBC: &wasmvmtypes.IBCMsg{
					CloseChannel: &wasmvmtypes.CloseChannelMsg{
						ChannelID: "channel-1",
					},
				},
			},
			output: []sdk.Msg{
				&channeltypes.MsgChannelCloseInit{
					PortId:    "wasm." + addr1.String(),
					ChannelId: "channel-1",
					Signer:    addr1.String(),
				},
			},
		},
	}
	encodingConfig := MakeEncodingConfig(t)
	encoder := DefaultEncoders(nil, nil, encodingConfig.Marshaler)
	for name, tc := range cases {
		tc := tc
		t.Run(name, func(t *testing.T) {
			var ctx sdk.Context
			res, err := encoder.Encode(ctx, tc.sender, tc.srcIBCPort, tc.srcMsg)
			if tc.isError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tc.output, res)
			}
		})
	}
}

func TestEncodeIBCSendPacket(t *testing.T) {
	ibcPort := "contractsIBCPort"
	var ctx sdk.Context
	specs := map[string]struct {
		srcMsg        wasmvmtypes.SendPacketMsg
		expPacketSent channeltypes.Packet
	}{
		"all good": {
			srcMsg: wasmvmtypes.SendPacketMsg{
				ChannelID:    "channel-1",
				Data:         []byte("myData"),
				TimeoutBlock: &wasmvmtypes.IBCTimeoutBlock{Revision: 1, Height: 2},
			},
			expPacketSent: channeltypes.Packet{
				Sequence:           1,
				SourcePort:         ibcPort,
				SourceChannel:      "channel-1",
				DestinationPort:    "other-port",
				DestinationChannel: "other-channel-1",
				Data:               []byte("myData"),
				TimeoutHeight:      clienttypes.Height{RevisionNumber: 1, RevisionHeight: 2},
			},
		},
	}
	for name, spec := range specs {
		t.Run(name, func(t *testing.T) {
			var gotPacket ibcexported.PacketI

			var chanKeeper types.ChannelKeeper = &wasmtesting.MockChannelKeeper{
				GetNextSequenceSendFn: func(ctx sdk.Context, portID, channelID string) (uint64, bool) {
					return 1, true
				},
				GetChannelFn: func(ctx sdk.Context, srcPort, srcChan string) (channeltypes.Channel, bool) {
					return channeltypes.Channel{
						Counterparty: channeltypes.NewCounterparty(
							"other-port",
							"other-channel-1",
						)}, true
				},
				SendPacketFn: func(ctx sdk.Context, channelCap *capabilitytypes.Capability, packet ibcexported.PacketI) error {
					gotPacket = packet
					return nil
				},
			}
			var capKeeper types.CapabilityKeeper = &wasmtesting.MockCapabilityKeeper{
				GetCapabilityFn: func(ctx sdk.Context, name string) (*capabilitytypes.Capability, bool) {
					return &capabilitytypes.Capability{}, true
				},
			}
			sender := RandomAccountAddress(t)
			res, err := EncodeIBCMsg(chanKeeper, capKeeper)(ctx, sender, ibcPort, &wasmvmtypes.IBCMsg{SendPacket: &spec.srcMsg})

			require.NoError(t, err)
			assert.Nil(t, res)
			assert.Equal(t, spec.expPacketSent, gotPacket)
		})
	}
}
