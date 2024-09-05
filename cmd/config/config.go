package config

var (
	Bech32Prefix = "orai"
	MinimalDenom = "orai"
	CosmosDenom  = Bech32Prefix
	EvmDenom     = "aorai" // atto orai. This will be converted automatically by evmutil of kava
)
