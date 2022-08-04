package keeper

//DONTCOVER

import (
	"sync"
)

// StargateLayerBindings keeps whitelist and its deterministic
// response binding for stargate queries.
//
// The query can be multi-thread, so we have to use
// thread safe sync.Map.
var StargateWhitelist sync.Map

func init() {

}
