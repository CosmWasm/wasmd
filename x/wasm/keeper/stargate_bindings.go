package keeper

import (
	"sync"
)

// StargateWhitelist keeps whitelist and its deterministic
// response binding for stargate queries.
//
// The query can be multi-thread, so we have to use
// thread safe sync.Map.
var StargateWhitelist sync.Map

// Define whitelists here as maps using 'StargateWhitelist'
// e.x) StargateWhitelist.Store("/cosmos.auth.v1beta1.Query/Account", &authtypes.QueryAccountResponse{})
func init() {

}
