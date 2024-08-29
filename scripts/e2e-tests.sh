#!/bin/bash

set -eu

# setup the network using the old binary

WASM_PATH=${WASM_PATH:-"$PWD/scripts/wasm_file/swapmap.wasm"}
ARGS="--chain-id testing -y --keyring-backend test --gas auto --gas-adjustment 1.5 -b block"
NEW_VERSION=${NEW_VERSION:-"v0.42.3"}
VALIDATOR_HOME=${VALIDATOR_HOME:-"$HOME/.oraid/validator1"}
re='^[0-9]+([.][0-9]+)?$'

# setup local network
sh $PWD/scripts/multinode-local-testnet.sh

# sleep about 5 secs to wait for the rest & json rpc server to be u
echo "Waiting for the REST & JSONRPC servers to be up ..."
sleep 5

oraid_version=$(oraid version)
if [[ $oraid_version =~ $NEW_VERSION ]] ; then
   echo "The chain version is not latest yet. There's something wrong!"; exit 1
fi

inflation=$(curl --no-progress-meter http://localhost:1317/cosmos/mint/v1beta1/inflation | jq '.inflation | tonumber')
if ! [[ $inflation =~ $re ]] ; then
   echo "Error: Cannot query inflation => Potentially missing Go GRPC backport" >&2;
   echo "Tests Failed"; exit 1
fi

evm_denom=$(curl --no-progress-meter http://localhost:1317/ethermint/evm/v1/params | jq '.params.evm_denom')
if ! [[ $evm_denom =~ "aorai" ]] ; then
   echo "Error: EVM denom is not correct. The current chain version is not the latest!" >&2;
   echo "Tests Failed"; exit 1
fi

sh $PWD/scripts/test_clock_counter_contract.sh

# test gasless
NODE_HOME=$VALIDATOR_HOME USER=validator1 WASM_PATH="$PWD/scripts/wasm_file/counter_high_gas_cost.wasm" sh $PWD/scripts/tests-0.42.1/test-gasless.sh
NODE_HOME=$VALIDATOR_HOME USER=validator1 sh $PWD/scripts/tests-0.42.1/test-tokenfactory.sh
NODE_HOME=$VALIDATOR_HOME USER=validator1 sh $PWD/scripts/tests-0.42.1/test-tokenfactory-bindings.sh
NODE_HOME=$VALIDATOR_HOME USER=validator1 sh $PWD/scripts/tests-0.42.1/test-evm-cosmos-mapping.sh
NODE_HOME=$VALIDATOR_HOME USER=validator1 sh $PWD/scripts/tests-0.42.1/test-evm-cosmos-mapping-complex.sh
NODE_HOME=$VALIDATOR_HOME USER=validator1 sh $PWD/scripts/tests-0.42.2/test-multi-sig.sh
NODE_HOME=$VALIDATOR_HOME sh $PWD/scripts/tests-0.42.3/test-commit-timeout.sh

echo "Tests Passed!!"