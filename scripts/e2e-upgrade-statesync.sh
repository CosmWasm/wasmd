#!/bin/bash

set -eu

# setup the network using the old binary

ARGS="--chain-id testing -y --keyring-backend test --gas auto --gas-adjustment 1.5 -b block"
VALIDATOR_HOME=${VALIDATOR_HOME:-"$HOME/.oraid/validator1"}
EXECUTE_MSG=${EXECUTE_MSG:-'{"ping":{}}'}
STATE_SYNC_HOME=${STATE_SYNC_HOME:-".oraid/state_sync"}

# run e2e upgrade before testing statesync
sh $PWD/scripts/e2e-upgrade.sh

# sleep a bit for the network to start 
echo "Sleep to wait for the network to start and wait for new snapshot intervals are after the upgrade to take place..."
sleep 60s

# now we setup statesync node
sh $PWD/scripts/state_sync.sh

echo "Sleep 1 min to get statesync done..."
sleep 1m

# add new key so we test sending wasm transaction afters statesync
# create new key
oraid keys add alice --keyring-backend=test --home=$STATE_SYNC_HOME

echo "## Send fund to state sync account"
oraid tx send $(oraid keys show validator1 -a --keyring-backend=test --home=$VALIDATOR_HOME) $(oraid keys show alice -a --keyring-backend=test --home=$STATE_SYNC_HOME) 500000orai --home=$VALIDATOR_HOME --node http://localhost:26657 $ARGS

echo "Sleep 2s to prevent account sequence error"
sleep 2s

# test wasm transaction using statesync node (port 26647)
contract_address=$(oraid query wasm list-contract-by-code 1 --output json | jq -r '.contracts[0]')
echo "## Test execute wasm transaction"
oraid tx wasm execute $contract_address $EXECUTE_MSG --from=validator1 --home=$VALIDATOR_HOME --node tcp://localhost:26647 $ARGS