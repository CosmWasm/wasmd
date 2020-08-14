package cosmwasm

type IBCQuery struct {
	// example queries available via cli
	ChannelEndQuery         *ChannelEndQuery
	ChannelClientStateQuery *ChannelClientStateQuery
}

type ChannelEndQuery struct {
	PortID, ChannelID string
}
type ChannelClientStateQuery struct {
	PortID, ChannelID string
}

//type AllChannelsToContractQuery struct {
// address string
//}
