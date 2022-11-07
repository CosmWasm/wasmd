package e2e

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/CosmWasm/wasmd/x/wasm/ibctesting"
)

// CCVTestSuite is an in-mem test suite which implements the standard group of tests validating
// the e2e functionality of ccv enabled chains.
// Any method implemented for this struct will be ran when suite.Run() is called.
type CCVTestSuite struct {
	suite.Suite
	// coordinator   *ibctesting.Coordinator
	// providerChain *ibctesting.TestChain
	// consumerChain *ibctesting.TestChain
	// providerApp       e2e.ProviderApp
	// consumerApp       e2e.ConsumerApp
	// providerClient    *ibctmtypes.ClientState
	// providerConsState *ibctmtypes.ConsensusState
	// path              *ibctesting.Path
	// transferPath  *ibctesting.Path
	setupCallback SetupCallback
}

// NewCCVTestSuite returns a new instance of CCVTestSuite, ready to be tested against using suite.Run().
func NewCCVTestSuite(setupCallback SetupCallback) *CCVTestSuite {
	ccvSuite := new(CCVTestSuite)
	ccvSuite.setupCallback = setupCallback
	return ccvSuite
}

// Callback for instantiating a new coordinator, provider/consumer test chains, and provider/consumer app
// before every test defined on the suite.
type SetupCallback func(t *testing.T) (
	coord *ibctesting.Coordinator,
	providerChain *ibctesting.TestChain,
	consumerChain *ibctesting.TestChain,
	// providerApp e2e.ProviderApp,
	// consumerApp e2e.ConsumerApp,
)
