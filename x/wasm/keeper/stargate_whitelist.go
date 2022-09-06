package keeper

import (
	"sync"
)

// AcceptList keeps whitelist and its deterministic
// response binding for stargate queries.
//
// The query can be multi-thread, so we have to use
// thread safe sync.Map.
var AcceptList sync.Map

// Define AcceptList here as maps using 'AcceptList'
// e.x) AcceptList.Store("/cosmos.auth.v1beta1.Query/Account", &authtypes.QueryAccountResponse{})
func init() {

}
