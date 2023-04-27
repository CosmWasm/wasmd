package custom

type CustomADR8Msg struct {
	RegisterPacketProcessedCallback           *RegisterPacketProcessedCallback   `json:"register_packet_processed_callback,omitempty"`
	UnregisterRegisterPacketProcessedCallback *UnRegisterPacketProcessedCallback `json:"unregister_register_packet_processed_callback,omitempty"`
}

type RegisterPacketProcessedCallback struct {
	// Sequence is the ibc packet sequence number
	Sequence            uint64 `json:"sequence"`
	ChannelID           string `json:"channel_id"`
	PortID              string `json:"port_id"`
	MaxCallbackGasLimit uint64 `json:"max_callback_gas_limit"`
}

type UnRegisterPacketProcessedCallback struct {
	Sequence  uint64 `json:"sequence"`
	ChannelID string `json:"channel_id"`
}
