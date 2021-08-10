package wasm_test

import (
	"encoding/json"
	"fmt"
	"testing"

	wasmd "github.com/CosmWasm/wasmd/app"
	wasmibctesting "github.com/CosmWasm/wasmd/x/wasm/ibctesting"
	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
	"github.com/CosmWasm/wasmd/x/wasm/keeper/wasmtesting"
	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	wasmvm "github.com/CosmWasm/wasmvm"
	wasmvmtypes "github.com/CosmWasm/wasmvm/types"
	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	clienttypes "github.com/cosmos/ibc-go/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/modules/core/04-channel/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	ping = "ping"
	pong = "pong"
)

var doNotTimeout = clienttypes.NewHeight(1, 1111111)

func TestPingPong(t *testing.T) {
	wasmibctesting.DefaultTestingAppInit = wasmd.SetupTestingApp
	pingContract := &player{t: t, actor: ping}
	pongContract := &player{t: t, actor: pong}

	var (
		chainAOpts = []wasmkeeper.Option{wasmkeeper.WithWasmEngine(
			wasmtesting.NewIBCContractMockWasmer(pingContract)),
		}
		chainBOpts = []wasmkeeper.Option{wasmkeeper.WithWasmEngine(
			wasmtesting.NewIBCContractMockWasmer(pongContract),
		)}
		coordinator = wasmibctesting.NewCoordinator(t, 2, chainAOpts, chainBOpts)
		chainA      = coordinator.GetChain(wasmibctesting.GetChainID(0))
		chainB      = coordinator.GetChain(wasmibctesting.GetChainID(1))
	)
	coordinator.CommitBlock(chainA, chainB)

	_ = chainB.SeedNewContractInstance() // skip 1 instance so that addresses are not the same
	var (
		pingContractAddr = chainA.SeedNewContractInstance()
		pongContractAddr = chainB.SeedNewContractInstance()
	)
	require.NotEqual(t, pingContractAddr, pongContractAddr)

	pingContract.chain = chainA
	pingContract.contractAddr = pingContractAddr

	pongContract.chain = chainB
	pongContract.contractAddr = pongContractAddr

	var (
		sourcePortID       = wasmkeeper.PortIDForContract(pingContractAddr)
		counterpartyPortID = wasmkeeper.PortIDForContract(pongContractAddr)
	)

	path := wasmibctesting.NewPath(chainA, chainB)
	path.EndpointA.ChannelConfig.PortID = sourcePortID
	path.EndpointB.ChannelConfig.PortID = counterpartyPortID
	path.EndpointA.ChannelConfig.Version = ping
	path.EndpointB.ChannelConfig.Version = pong

	coordinator.Setup(path)

	const startValue uint64 = 100
	const rounds = 3
	s := startGame{
		ChannelID: path.EndpointA.ChannelID,
		Value:     startValue,
	}
	startMsg := &wasmtypes.MsgExecuteContract{
		Sender:   chainA.SenderAccount.GetAddress().String(),
		Contract: pingContractAddr.String(),
		Msg:      s.GetBytes(),
	}
	// send message to chainA
	_, err := chainA.SendMsgs(startMsg)
	require.NoError(t, err)

	var (
		activePlayer  = ping
		pingBallValue = startValue
	)
	for i := 1; i <= rounds; i++ {
		t.Logf("++ round: %d\n", i)
		ball := NewHit(activePlayer, pingBallValue)

		pkg := channeltypes.NewPacket(ball.GetBytes(), uint64(i*2),
			path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID,
			path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, doNotTimeout, 0)
		ack := ball.BuildAck()

		err = path.EndpointA.SendPacket(pkg)
		require.NoError(t, err)
		err = path.RelayPacket(pkg, ack.GetBytes())
		require.NoError(t, err)

		// switch side
		activePlayer = counterParty(activePlayer)
		ball = NewHit(activePlayer, uint64(i))
		pkg = channeltypes.NewPacket(ball.GetBytes(), uint64(i*2),
			path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID,
			path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, doNotTimeout, 0)
		ack = ball.BuildAck()

		err = path.EndpointB.SendPacket(pkg)
		require.NoError(t, err)
		err = path.RelayPacket(pkg, ack.GetBytes())
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

var _ wasmtesting.IBCContractCallbacks = &player{}

// player is a (mock) contract that sends and receives ibc packages
type player struct {
	t            *testing.T
	chain        *wasmibctesting.TestChain
	contractAddr sdk.AccAddress
	actor        string // either ping or pong
	execCalls    int    // number of calls to Execute method (checkTx + deliverTx)
}

// Execute starts the ping pong game
// Contracts finds all connected channels and broadcasts a ping message
func (p *player) Execute(code wasmvm.Checksum, env wasmvmtypes.Env, info wasmvmtypes.MessageInfo, executeMsg []byte, store wasmvm.KVStore, goapi wasmvm.GoAPI, querier wasmvm.Querier, gasMeter wasmvm.GasMeter, gasLimit uint64, deserCost wasmvmtypes.UFraction) (*wasmvmtypes.Response, uint64, error) {
	p.execCalls++
	// start game
	var start startGame
	if err := json.Unmarshal(executeMsg, &start); err != nil {
		return nil, 0, err
	}

	if start.MaxValue != 0 {
		store.Set(maxValueKey, sdk.Uint64ToBigEndian(start.MaxValue))
	}
	service := NewHit(p.actor, start.Value)
	p.t.Logf("[%s] starting game with: %d: %v\n", p.actor, start.Value, service)

	p.incrementCounter(sentBallsCountKey, store)
	store.Set(lastBallSentKey, sdk.Uint64ToBigEndian(start.Value))
	return &wasmvmtypes.Response{
		Messages: []wasmvmtypes.SubMsg{
			{
				ReplyOn: wasmvmtypes.ReplyNever,
				Msg: wasmvmtypes.CosmosMsg{
					IBC: &wasmvmtypes.IBCMsg{
						SendPacket: &wasmvmtypes.SendPacketMsg{
							ChannelID: start.ChannelID,
							Data:      service.GetBytes(),
							Timeout: wasmvmtypes.IBCTimeout{Block: &wasmvmtypes.IBCTimeoutBlock{
								Revision: doNotTimeout.RevisionNumber,
								Height:   doNotTimeout.RevisionHeight,
							}},
						},
					},
				},
			},
		},
	}, 0, nil
}

// OnIBCChannelOpen ensures to accept only configured version
func (p player) IBCChannelOpen(codeID wasmvm.Checksum, env wasmvmtypes.Env, channel wasmvmtypes.IBCChannelOpenMsg, store wasmvm.KVStore, goapi wasmvm.GoAPI, querier wasmvm.Querier, gasMeter wasmvm.GasMeter, gasLimit uint64, deserCost wasmvmtypes.UFraction) (uint64, error) {
	return 0, nil
}

// OnIBCChannelConnect persists connection endpoints
func (p player) IBCChannelConnect(codeID wasmvm.Checksum, env wasmvmtypes.Env, channel wasmvmtypes.IBCChannelConnectMsg, store wasmvm.KVStore, goapi wasmvm.GoAPI, querier wasmvm.Querier, gasMeter wasmvm.GasMeter, gasLimit uint64, deserCost wasmvmtypes.UFraction) (*wasmvmtypes.IBCBasicResponse, uint64, error) {
	p.storeEndpoint(store, connectedChannelsModel{Our: channel.GetChannel().Endpoint, Their: channel.GetChannel().CounterpartyEndpoint})
	return &wasmvmtypes.IBCBasicResponse{}, 0, nil
}

// connectedChannelsModel is a simple persistence model to store endpoint addresses within the contract's store
type connectedChannelsModel struct {
	Our   wasmvmtypes.IBCEndpoint
	Their wasmvmtypes.IBCEndpoint
}

var ( // store keys
	ibcEndpointsKey = []byte("ibc-endpoints")
	maxValueKey     = []byte("max-value")
)

func (p player) loadEndpoints(store prefix.Store, channelID string) *connectedChannelsModel {
	var counterparties []connectedChannelsModel
	if bz := store.Get(ibcEndpointsKey); bz != nil {
		require.NoError(p.t, json.Unmarshal(bz, &counterparties))
	}
	for _, v := range counterparties {
		if v.Our.ChannelID == channelID {
			return &v
		}
	}
	p.t.Fatalf("no counterparty found for channel %q", channelID)
	return nil
}

func (p player) storeEndpoint(store wasmvm.KVStore, connectedChannel connectedChannelsModel) {
	var counterparties []connectedChannelsModel
	if b := store.Get(ibcEndpointsKey); b != nil {
		require.NoError(p.t, json.Unmarshal(b, &counterparties))
	}
	counterparties = append(counterparties, connectedChannel)
	bz, err := json.Marshal(&counterparties)
	require.NoError(p.t, err)
	store.Set(ibcEndpointsKey, bz)
}

func (p player) IBCChannelClose(codeID wasmvm.Checksum, env wasmvmtypes.Env, channel wasmvmtypes.IBCChannelCloseMsg, store wasmvm.KVStore, goapi wasmvm.GoAPI, querier wasmvm.Querier, gasMeter wasmvm.GasMeter, gasLimit uint64, deserCost wasmvmtypes.UFraction) (*wasmvmtypes.IBCBasicResponse, uint64, error) {
	panic("implement me")
}

var ( // store keys
	lastBallSentKey        = []byte("lastBallSent")
	lastBallReceivedKey    = []byte("lastBallReceived")
	sentBallsCountKey      = []byte("sentBalls")
	receivedBallsCountKey  = []byte("recvBalls")
	confirmedBallsCountKey = []byte("confBalls")
)

// IBCPacketReceive receives the hit and serves a response hit via `wasmvmtypes.IBCPacket`
func (p player) IBCPacketReceive(codeID wasmvm.Checksum, env wasmvmtypes.Env, packet wasmvmtypes.IBCPacketReceiveMsg, store wasmvm.KVStore, goapi wasmvm.GoAPI, querier wasmvm.Querier, gasMeter wasmvm.GasMeter, gasLimit uint64, deserCost wasmvmtypes.UFraction) (*wasmvmtypes.IBCReceiveResponse, uint64, error) {
	// parse received data and store
	var receivedBall hit
	if err := json.Unmarshal(packet.Packet.Data, &receivedBall); err != nil {
		return &wasmvmtypes.IBCReceiveResponse{
			Acknowledgement: hitAcknowledgement{Error: err.Error()}.GetBytes(),
			// no hit msg, we stop the game
		}, 0, nil
	}
	p.incrementCounter(receivedBallsCountKey, store)

	otherCount := receivedBall[counterParty(p.actor)]
	store.Set(lastBallReceivedKey, sdk.Uint64ToBigEndian(otherCount))

	if maxVal := store.Get(maxValueKey); maxVal != nil && otherCount > sdk.BigEndianToUint64(maxVal) {
		errMsg := fmt.Sprintf("max value exceeded: %d got %d", sdk.BigEndianToUint64(maxVal), otherCount)
		return &wasmvmtypes.IBCReceiveResponse{
			Acknowledgement: receivedBall.BuildError(errMsg).GetBytes(),
		}, 0, nil
	}

	nextValue := p.incrementCounter(lastBallSentKey, store)
	newHit := NewHit(p.actor, nextValue)
	respHit := &wasmvmtypes.IBCMsg{SendPacket: &wasmvmtypes.SendPacketMsg{
		ChannelID: packet.Packet.Src.ChannelID,
		Data:      newHit.GetBytes(),
		Timeout: wasmvmtypes.IBCTimeout{Block: &wasmvmtypes.IBCTimeoutBlock{
			Revision: doNotTimeout.RevisionNumber,
			Height:   doNotTimeout.RevisionHeight,
		}},
	}}
	p.incrementCounter(sentBallsCountKey, store)
	p.t.Logf("[%s] received %d, returning %d: %v\n", p.actor, otherCount, nextValue, newHit)

	return &wasmvmtypes.IBCReceiveResponse{
		Acknowledgement: receivedBall.BuildAck().GetBytes(),
		Messages:        []wasmvmtypes.SubMsg{{ReplyOn: wasmvmtypes.ReplyNever, Msg: wasmvmtypes.CosmosMsg{IBC: respHit}}},
	}, 0, nil
}

// OnIBCPacketAcknowledgement handles the packet acknowledgment frame. Stops the game on an any error
func (p player) IBCPacketAck(codeID wasmvm.Checksum, env wasmvmtypes.Env, packetAck wasmvmtypes.IBCPacketAckMsg, store wasmvm.KVStore, goapi wasmvm.GoAPI, querier wasmvm.Querier, gasMeter wasmvm.GasMeter, gasLimit uint64, deserCost wasmvmtypes.UFraction) (*wasmvmtypes.IBCBasicResponse, uint64, error) {
	// parse received data and store
	var sentBall hit
	if err := json.Unmarshal(packetAck.OriginalPacket.Data, &sentBall); err != nil {
		return nil, 0, err
	}

	var ack hitAcknowledgement
	if err := json.Unmarshal(packetAck.Acknowledgement.Data, &ack); err != nil {
		return nil, 0, err
	}
	if ack.Success != nil {
		confirmedCount := sentBall[p.actor]
		p.t.Logf("[%s] acknowledged %d: %v\n", p.actor, confirmedCount, sentBall)
	} else {
		p.t.Logf("[%s] received app layer error: %s\n", p.actor, ack.Error)

	}

	p.incrementCounter(confirmedBallsCountKey, store)
	return &wasmvmtypes.IBCBasicResponse{}, 0, nil
}

func (p player) IBCPacketTimeout(codeID wasmvm.Checksum, env wasmvmtypes.Env, packet wasmvmtypes.IBCPacketTimeoutMsg, store wasmvm.KVStore, goapi wasmvm.GoAPI, querier wasmvm.Querier, gasMeter wasmvm.GasMeter, gasLimit uint64, deserCost wasmvmtypes.UFraction) (*wasmvmtypes.IBCBasicResponse, uint64, error) {
	panic("implement me")
}

func (p player) incrementCounter(key []byte, store wasmvm.KVStore) uint64 {
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
	raw := wasmd.IBCTestSupport(p.t, p.chain).WasmKeeper().QueryRaw(p.chain.GetContext(), p.contractAddr, key)
	return sdk.BigEndianToUint64(raw)
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

func (h hit) BuildError(errMsg string) hitAcknowledgement {
	return hitAcknowledgement{Error: errMsg}
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
	ChannelID string
	Value     uint64
	// limit above the game is aborted
	MaxValue uint64 `json:"max_value,omitempty"`
}

func (g startGame) GetBytes() json.RawMessage {
	b, err := json.Marshal(g)
	if err != nil {
		panic(err)
	}
	return b
}
