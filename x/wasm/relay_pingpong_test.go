package wasm_test

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/CosmWasm/go-cosmwasm"
	wasmTypes "github.com/CosmWasm/go-cosmwasm/types"
	"github.com/CosmWasm/wasmd/x/wasm"
	"github.com/CosmWasm/wasmd/x/wasm/ibc_testing"
	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/internal/keeper"
	cosmwasmv2 "github.com/CosmWasm/wasmd/x/wasm/internal/keeper/cosmwasm"
	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	clientexported "github.com/cosmos/cosmos-sdk/x/ibc/02-client/exported"
	channeltypes "github.com/cosmos/cosmos-sdk/x/ibc/04-channel/types"
	host "github.com/cosmos/cosmos-sdk/x/ibc/24-host"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	ping = "ping"
	pong = "pong"
)
const doNotTimeout uint64 = 110000

func TestPinPong(t *testing.T) {
	var (
		coordinator = ibc_testing.NewCoordinator(t, 2)
		chainA      = coordinator.GetChain(ibc_testing.GetChainID(0))
		chainB      = coordinator.GetChain(ibc_testing.GetChainID(1))
	)
	_ = chainB.NewRandomContractInstance() // skip 1 id
	var (
		pingContractAddr = chainA.NewRandomContractInstance()
		pongContractAddr = chainB.NewRandomContractInstance()
	)
	require.NotEqual(t, pingContractAddr, pongContractAddr)

	pingContract := &player{t: t, actor: ping, chain: chainA, contractAddr: pingContractAddr}
	pongContract := &player{t: t, actor: pong, chain: chainB, contractAddr: pongContractAddr}

	wasmkeeper.MockContracts[pingContractAddr.String()] = pingContract
	wasmkeeper.MockContracts[pongContractAddr.String()] = pongContract

	var (
		sourcePortID       = wasmkeeper.PortIDForContract(pingContractAddr)
		counterpartyPortID = wasmkeeper.PortIDForContract(pongContractAddr)
	)
	clientA, clientB, connA, connB := coordinator.SetupClientConnections(chainA, chainB, clientexported.Tendermint)
	connA.NextChannelVersion = ping
	connB.NextChannelVersion = pong

	channelA, channelB := coordinator.CreateChannel(chainA, chainB, connA, connB, sourcePortID, counterpartyPortID, channeltypes.UNORDERED)
	var err error

	const startValue uint64 = 100
	s := startGame{
		Source:       cosmwasmv2.IBCEndpoint{channelA.ID, channelA.PortID},
		CounterParty: cosmwasmv2.IBCEndpoint{channelB.ID, channelB.PortID},
		Value:        startValue,
	}
	startMsg := &wasm.MsgExecuteContract{
		Sender:   chainA.SenderAccount.GetAddress(),
		Contract: pingContractAddr,
		Msg:      s.GetBytes(),
	}
	// send from chainA to chainB
	err = coordinator.SendMsgs(chainA, chainB, clientB, startMsg)
	require.NoError(t, err)

	t.Log("Duplicate messages are due to check/deliver tx calls")

	const rounds = 3
	var (
		activePlayer  = ping
		pingBallValue = startValue
	)
	for i := 1; i <= rounds; i++ {
		t.Logf("++ round: %d\n", i)
		ball := NewHit(activePlayer, pingBallValue)

		seq := uint64(i)
		pkg := channeltypes.NewPacket(ball.GetBytes(), seq, channelA.PortID, channelA.ID, channelB.PortID, channelB.ID, doNotTimeout, 0)
		ack := ball.BuildAck()

		err = coordinator.RelayPacket(chainA, chainB, clientA, clientB, pkg, ack.GetBytes())
		require.NoError(t, err)
		//coordinator.CommitBlock(chainA, chainB)
		err = coordinator.UpdateClient(chainA, chainB, clientA, clientexported.Tendermint)
		require.NoError(t, err)

		// switch side
		activePlayer = counterParty(activePlayer)
		ball = NewHit(activePlayer, uint64(i))
		pkg = channeltypes.NewPacket(ball.GetBytes(), seq, channelB.PortID, channelB.ID, channelA.PortID, channelA.ID, doNotTimeout, 0)
		ack = ball.BuildAck()

		err = coordinator.RelayPacket(chainB, chainA, clientB, clientA, pkg, ack.GetBytes())
		require.NoError(t, err)
		err = coordinator.UpdateClient(chainB, chainA, clientB, clientexported.Tendermint)
		require.NoError(t, err)

		// switch side for next round
		activePlayer = counterParty(activePlayer)
		pingBallValue++
	}
	assert.Equal(t, startValue+rounds, pingContract.QueryState(lastBallSentKey))
	assert.Equal(t, uint64(rounds), pingContract.QueryState(lastBallReceivedKey))
	assert.Equal(t, uint64(rounds+1), pingContract.QueryState(sentBallsCountKey))
	assert.Equal(t, uint64(rounds), pingContract.QueryState(receivedBallsCountKey))
	assert.Equal(t, uint64(rounds), pingContract.QueryState(confirmedBallsCountKey))

	assert.Equal(t, uint64(rounds), pongContract.QueryState(lastBallSentKey))
	assert.Equal(t, startValue+rounds-1, pongContract.QueryState(lastBallReceivedKey))
	assert.Equal(t, uint64(rounds), pongContract.QueryState(sentBallsCountKey))
	assert.Equal(t, uint64(rounds), pongContract.QueryState(receivedBallsCountKey))
	assert.Equal(t, uint64(rounds), pongContract.QueryState(confirmedBallsCountKey))

}

// hit is ibc packet payload
type hit map[string]uint64

func NewHit(player string, count uint64) hit {
	return map[string]uint64{
		player: count,
	}
}
func (h hit) GetBytes() []byte {
	b, err := json.Marshal(h)
	if err != nil {
		panic(err)
	}
	return b
}
func (h hit) String() string {
	return fmt.Sprintf("Ball %s", string(h.GetBytes()))
}

func (h hit) BuildAck() hitAcknowledgement {
	return hitAcknowledgement{Success: &h}
}

// hitAcknowledgement is ibc acknowledgment payload
type hitAcknowledgement struct {
	Error   string `json:"error,omitempty"`
	Success *hit   `json:"success,omitempty"`
}

func (a hitAcknowledgement) GetBytes() []byte {
	b, err := json.Marshal(a)
	if err != nil {
		panic(err)
	}
	return b
}

// startGame is an execute message payload
type startGame struct {
	Source, CounterParty cosmwasmv2.IBCEndpoint
	Value                uint64
}

func (g startGame) GetBytes() json.RawMessage {
	b, err := json.Marshal(g)
	if err != nil {
		panic(err)
	}
	return b
}

// player is a (mock) contract that sends and receives ibc packages
type player struct {
	t            *testing.T
	chain        *ibc_testing.TestChain
	contractAddr sdk.AccAddress
	actor        string
	execCalls    int
}

func (p *player) Execute(hash []byte, params wasmTypes.Env, data []byte, store prefix.Store, api cosmwasm.GoAPI, querier wasmkeeper.QueryHandler, meter sdk.GasMeter, gas uint64) (*cosmwasmv2.HandleResponse, uint64, error) {
	p.execCalls++
	if p.execCalls%2 == 1 { // skip checkTx step because of no rollback with `chain.GetContext()`
		return &cosmwasmv2.HandleResponse{}, 0, nil
	}
	// start game
	var start startGame
	if err := json.Unmarshal(data, &start); err != nil {
		return nil, 0, err
	}

	ctx := p.chain.GetContext()
	channelCap, ok := p.chain.App.WasmKeeper.ScopedKeeper.GetCapability(ctx, host.ChannelCapabilityPath(start.Source.Port, start.Source.Channel))
	if !ok {
		return nil, 0, sdkerrors.Wrap(channeltypes.ErrChannelCapabilityNotFound, "module does not own channel capability")
	}
	service := NewHit(p.actor, start.Value)
	p.t.Logf("[%s] starting game with: %d: %v\n", p.actor, start.Value, service)

	var seq uint64 = 1
	packet := channeltypes.NewPacket(service.GetBytes(), seq, start.Source.Port, start.Source.Channel, start.CounterParty.Port, start.CounterParty.Channel, doNotTimeout, 0)
	err := p.chain.App.WasmKeeper.ChannelKeeper.SendPacket(ctx, channelCap, packet)
	if err != nil {
		return nil, 0, err
	}

	p.IncrementCounter(sentBallsCountKey, store)
	store.Set(lastBallSentKey, sdk.Uint64ToBigEndian(start.Value))
	return &cosmwasmv2.HandleResponse{}, 0, nil
}

func (p player) OnIBCChannelOpen(hash []byte, params cosmwasmv2.Env, channel cosmwasmv2.IBCChannel, store prefix.Store, api cosmwasm.GoAPI, querier wasmkeeper.QueryHandler, meter sdk.GasMeter, gas uint64) (*cosmwasmv2.IBCChannelOpenResponse, uint64, error) {
	if channel.Version != p.actor {
		return &cosmwasmv2.IBCChannelOpenResponse{Success: false, Reason: fmt.Sprintf("expected %q but got %q", p.actor, channel.Version)}, 0, nil
	}
	return &cosmwasmv2.IBCChannelOpenResponse{Success: true}, 0, nil
}

func (p player) OnIBCChannelConnect(hash []byte, params cosmwasmv2.Env, channel cosmwasmv2.IBCChannel, store prefix.Store, api cosmwasm.GoAPI, querier wasmkeeper.QueryHandler, meter sdk.GasMeter, gas uint64) (*cosmwasmv2.IBCChannelConnectResponse, uint64, error) {
	return &cosmwasmv2.IBCChannelConnectResponse{}, 0, nil
}

func (p player) OnIBCChannelClose(ctx sdk.Context, hash []byte, params cosmwasmv2.Env, channel cosmwasmv2.IBCChannel, meter sdk.GasMeter, gas uint64) (*cosmwasmv2.IBCChannelCloseResponse, uint64, error) {
	panic("implement me")
}

var ( // store keys
	lastBallSentKey        = []byte("lastBallSent")
	lastBallReceivedKey    = []byte("lastBallReceived")
	sentBallsCountKey      = []byte("sentBalls")
	receivedBallsCountKey  = []byte("recvBalls")
	confirmedBallsCountKey = []byte("confBalls")
)

func (p player) OnIBCPacketReceive(hash []byte, params cosmwasmv2.Env, packet cosmwasmv2.IBCPacket, store prefix.Store, api cosmwasm.GoAPI, querier wasmkeeper.QueryHandler, meter sdk.GasMeter, gas uint64) (*cosmwasmv2.IBCPacketReceiveResponse, uint64, error) {
	// parse received data and store
	var receivedBall hit
	if err := json.Unmarshal(packet.Data, &receivedBall); err != nil {
		return &cosmwasmv2.IBCPacketReceiveResponse{
			Acknowledgement: hitAcknowledgement{Error: err.Error()}.GetBytes(),
			// no hit msg, we stop the game
		}, 0, nil
	}

	otherCount := receivedBall[counterParty(p.actor)]
	store.Set(lastBallReceivedKey, sdk.Uint64ToBigEndian(otherCount))

	nextValue := p.IncrementCounter(lastBallSentKey, store)
	newHit := NewHit(p.actor, nextValue)
	respHit := &cosmwasmv2.IBCMsg{SendPacket: &cosmwasmv2.IBCSendMsg{
		ChannelID:     packet.Source.Channel,
		Data:          newHit.GetBytes(),
		TimeoutHeight: doNotTimeout,
	}}

	p.IncrementCounter(receivedBallsCountKey, store)
	p.IncrementCounter(sentBallsCountKey, store)
	p.t.Logf("[%s] received %d, returning %d: %v\n", p.actor, otherCount, nextValue, newHit)

	return &cosmwasmv2.IBCPacketReceiveResponse{
		Acknowledgement: receivedBall.BuildAck().GetBytes(),
		Messages:        []cosmwasmv2.CosmosMsg{{IBC: respHit}},
	}, 0, nil
}

func (p player) OnIBCPacketAcknowledgement(hash []byte, params cosmwasmv2.Env, packetAck cosmwasmv2.IBCAcknowledgement, store prefix.Store, api cosmwasm.GoAPI, querier wasmkeeper.QueryHandler, meter sdk.GasMeter, gas uint64) (*cosmwasmv2.IBCPacketAcknowledgementResponse, uint64, error) {
	// parse received data and store
	var sentBall hit
	if err := json.Unmarshal(packetAck.OriginalPacket.Data, &sentBall); err != nil {
		return nil, 0, err
	}

	var ack hitAcknowledgement
	if err := json.Unmarshal(packetAck.Acknowledgement, &ack); err != nil {
		return nil, 0, err
	}
	if ack.Success != nil {
		confirmedCount := sentBall[p.actor]
		p.t.Logf("[%s] acknowledged %d: %v\n", p.actor, confirmedCount, sentBall)
	} else {
		p.t.Logf("[%s] received app layer error: %s\n", p.actor, ack.Error)

	}

	p.IncrementCounter(confirmedBallsCountKey, store)
	return &cosmwasmv2.IBCPacketAcknowledgementResponse{}, 0, nil
}

func (p player) OnIBCPacketTimeout(hash []byte, params cosmwasmv2.Env, packet cosmwasmv2.IBCPacket, store prefix.Store, api cosmwasm.GoAPI, querier wasmkeeper.QueryHandler, meter sdk.GasMeter, gas uint64) (*cosmwasmv2.IBCPacketTimeoutResponse, uint64, error) {
	panic("implement me")
}

func (p player) IncrementCounter(key []byte, store prefix.Store) uint64 {
	var count uint64
	bz := store.Get(key)
	if bz != nil {
		count = sdk.BigEndianToUint64(bz)
	}
	count++
	store.Set(key, sdk.Uint64ToBigEndian(count))
	return count
}

func (p player) QueryState(key []byte) uint64 {
	models := p.chain.App.WasmKeeper.QueryRaw(p.chain.GetContext(), p.contractAddr, key)
	require.Len(p.t, models, 1)
	return sdk.BigEndianToUint64(models[0].Value)
}

func counterParty(s string) string {
	switch s {
	case ping:
		return pong
	case pong:
		return ping
	default:
		panic(fmt.Sprintf("unsupported: %q", s))
	}
}
