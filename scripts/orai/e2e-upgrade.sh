#!/bin/bash

set -eu

# setup the network using the old binary

OLD_VERSION=${OLD_VERSION:-"v0.42.1"}
WASM_PATH=${WASM_PATH:-"$PWD/scripts/wasm_file/swapmap.wasm"}
ARGS="--chain-id testing -y --keyring-backend test --gas auto --gas-adjustment 1.5 -b block"
NEW_VERSION=${NEW_VERSION:-"v0.42.2"}
VALIDATOR_HOME=${VALIDATOR_HOME:-"$HOME/.oraid/validator1"}
MIGRATE_MSG=${MIGRATE_MSG:-'{}'}
EXECUTE_MSG=${EXECUTE_MSG:-'{"ping":{}}'}
HIDE_LOGS="/dev/null"

# kill all running binaries
pkill oraid && sleep 2

# download current production binary
current_dir=$PWD
rm -rf ../../orai-old/ && git clone https://github.com/oraichain/orai.git ../../orai-old && cd ../../orai-old/orai && git checkout $OLD_VERSION && go mod tidy && GOTOOLCHAIN=go1.21.4 make install

cd $current_dir

# setup local network
sh $PWD/scripts/multinode-local-testnet.sh

# deploy new contract
store_ret=$(oraid tx wasm store $WASM_PATH --from validator1 --home $VALIDATOR_HOME $ARGS --output json)
code_id=$(echo $store_ret | jq -r '.logs[0].events[1].attributes[] | select(.key | contains("code_id")).value')
oraid tx wasm instantiate $code_id '{}' --label 'testing' --from validator1 --home $VALIDATOR_HOME -b block --admin $(oraid keys show validator1 --keyring-backend test --home $VALIDATOR_HOME -a) $ARGS > $HIDE_LOGS
contract_address=$(oraid query wasm list-contract-by-code $code_id --output json | jq -r '.contracts[0]')

echo "contract address: $contract_address"

# create new upgrade proposal
UPGRADE_HEIGHT=${UPGRADE_HEIGHT:-35}
oraid tx gov submit-proposal software-upgrade $NEW_VERSION --title "foobar" --description "foobar"  --from validator1 --upgrade-height $UPGRADE_HEIGHT --upgrade-info "x" --deposit 10000000orai $ARGS --home $VALIDATOR_HOME > $HIDE_LOGS
oraid tx gov vote 1 yes --from validator1 --home $VALIDATOR_HOME $ARGS > $HIDE_LOGS && oraid tx gov vote 1 yes --from validator2 --home "$HOME/.oraid/validator2" $ARGS > $HIDE_LOGS

# sleep to wait til the proposal passes
echo "Sleep til the proposal passes..."

# Check if latest height is less than the upgrade height
latest_height=$(curl --no-progress-meter http://localhost:1317/blocks/latest | jq '.block.header.height | tonumber')
while [ $latest_height -lt $UPGRADE_HEIGHT ];
do
   sleep 5
   ((latest_height=$(curl --no-progress-meter http://localhost:1317/blocks/latest | jq '.block.header.height | tonumber')))
   echo $latest_height
done

# kill all processes
pkill oraid

# install new binary for the upgrade
echo "install new binary"
GOTOOLCHAIN=go1.21.4 make install

# re-run all validators. All should run
screen -S validator1 -d -m oraid start --home=$HOME/.oraid/validator1
screen -S validator2 -d -m oraid start --home=$HOME/.oraid/validator2
screen -S validator3 -d -m oraid start --home=$HOME/.oraid/validator3

# sleep a bit for the network to start 
echo "Sleep to wait for the network to start..."
sleep 3

# test contract migration
echo "Migrate the contract..."
store_ret=$(oraid tx wasm store $WASM_PATH --from validator1 --home $VALIDATOR_HOME $ARGS --output json)
echo "Execute the contract..."
new_code_id=$(echo $store_ret | jq -r '.logs[0].events[1].attributes[] | select(.key | contains("code_id")).value')

# oraid tx wasm migrate $contract_address $new_code_id $MIGRATE_MSG --from validator1 $ARGS --home $VALIDATOR_HOME
oraid tx wasm execute $contract_address $EXECUTE_MSG --from validator1 $ARGS --home $VALIDATOR_HOME > $HIDE_LOGS

# sleep about 5 secs to wait for the rest & json rpc server to be u
echo "Waiting for the REST & JSONRPC servers to be up ..."
sleep 5

oraid_version=$(oraid version)
if [[ $oraid_version =~ $OLD_VERSION ]] ; then
   echo "The chain has not upgraded yet. There's something wrong!"; exit 1
fi

height_before=$(curl --no-progress-meter http://localhost:1317/blocks/latest | jq '.block.header.height | tonumber')

re='^[0-9]+([.][0-9]+)?$'
if ! [[ $height_before =~ $re ]] ; then
   echo "error: Not a number" >&2; exit 1
fi

sleep 5

height_after=$(curl --no-progress-meter http://localhost:1317/blocks/latest | jq '.block.header.height | tonumber')

if ! [[ $height_after =~ $re ]] ; then
   echo "error: Not a number" >&2; exit 1
fi

if [ $height_after -gt $height_before ]
then
echo "Test Passed"
else
echo "Test Failed"
fi

inflation=$(curl --no-progress-meter http://localhost:1317/cosmos/mint/v1beta1/inflation | jq '.inflation | tonumber')
if ! [[ $inflation =~ $re ]] ; then
   echo "Error: Cannot query inflation => Potentially missing Go GRPC backport" >&2;
   echo "Tests Failed"; exit 1
fi

evm_denom=$(curl --no-progress-meter http://localhost:1317/ethermint/evm/v1/params | jq '.params.evm_denom')
if ! [[ $evm_denom =~ "aorai" ]] ; then
   echo "Error: EVM denom is not correct. The upgraded version is not the latest!" >&2;
   echo "Tests Failed"; exit 1
fi

sh $PWD/scripts/test_clock_counter_contract.sh

# test gasless
NODE_HOME=$VALIDATOR_HOME USER=validator1 sh $PWD/scripts/tests-0.42.1/test-gasless.sh
NODE_HOME=$VALIDATOR_HOME USER=validator1 sh $PWD/scripts/tests-0.42.1/test-tokenfactory.sh
NODE_HOME=$VALIDATOR_HOME USER=validator1 sh $PWD/scripts/tests-0.42.1/test-tokenfactory-bindings.sh
NODE_HOME=$VALIDATOR_HOME USER=validator1 sh $PWD/scripts/tests-0.42.1/test-evm-cosmos-mapping.sh
NODE_HOME=$VALIDATOR_HOME USER=validator1 sh $PWD/scripts/tests-0.42.1/test-evm-cosmos-mapping-complex.sh
NODE_HOME=$VALIDATOR_HOME USER=validator1 sh $PWD/scripts/tests-0.42.2/test-multi-sig.sh

echo "Tests Passed!!"