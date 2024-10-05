#!/bin/bash
# Before running this script, you must setup local network:
# sh $PWD/scripts/multinode-local-testnet.sh
# oraiswap-token.wasm source code: https://github.com/oraichain/oraiswap.git

set -eux

# hard-coded test private key. DO NOT USE!!
PRIVATE_KEY_ETH=${PRIVATE_KEY_ETH:-"021646C7F742C743E60CC460C56242738A3951667E71C803929CB84B6FA4B0D6"}
PRIVATE_KEY_EVM_ADDRESS=${PRIVATE_KEY_EVM_ADDRESS:-"0xB0ac9d216b303a32907632731a93356228CAEE87"}
current_dir=$PWD
WASM_PATH=${WASM_PATH:-"$PWD/scripts/wasm_file/oraiswap-token.wasm"}
ARGS="--chain-id testing -y --keyring-backend test --gas auto --gas-adjustment 1.5 -b sync"
VALIDATOR1_ARGS=${VALIDATOR1_ARGS:-"--from validator1 --home $HOME/.oraid/validator1"}

HIDE_LOGS="/dev/null"

store_ret=$(oraid tx wasm store $WASM_PATH $VALIDATOR1_ARGS $ARGS --output json)
store_txhash=$(echo $store_ret | jq -r '.txhash')
# need to sleep 1s for tx already in block
sleep 2
code_id=$(oraid query tx $store_txhash --output json | jq -r '.events[4].attributes[] | select(.key | contains("code_id")).value')

# 2 addresses that map with the hard-coded private key above
INSTANTIATE_MSG='{"name":"OraichainToken","symbol":"ORAI","decimals":6,"initial_balances":[{"amount":"1000000000","address":"orai1kzkf6gttxqar9yrkxfe34ye4vg5v4m588ew7c9"},{"amount":"1000000000","address":"orai1hgscrqcd2kmju4t5akujeugwrfev7uxv66lnuu"}]}'
oraid tx wasm instantiate $code_id $INSTANTIATE_MSG --label 'cw20 ORAI' $VALIDATOR1_ARGS --admin $(oraid keys show validator1 --keyring-backend test --home $HOME/.oraid/validator1 -a) $ARGS > $HIDE_LOGS
# need to sleep 1s for tx already in block
sleep 2
contract_address=$(oraid query wasm list-contract-by-code $code_id --output json | jq -r '.contracts | last')
echo "cw-stargate-staking-query contract address: $contract_address"

# clone or pull latest repo
if [ -d "$PWD/../evm-bridge-proxy" ]; then
  cd ../evm-bridge-proxy
  git pull origin master
else
  git clone https://github.com/oraichain/evm-bridge-proxy.git ../evm-bridge-proxy
  cd ../evm-bridge-proxy
fi

# prepare env and chain
yarn && yarn compile;
echo "PRIVATE_KEY=$PRIVATE_KEY_ETH" > .env

# before deploying erc20, we need to fund the private key's address first
oraid tx bank send $USER orai1kzkf6gttxqar9yrkxfe34ye4vg5v4m588ew7c9 100000orai $VALIDATOR1_ARGS $ARGS > $HIDE_LOGS
sleep 2 # wait for tx

# deploy cw20erc20 contract
output=$(CW20_ADDRESS=$contract_address yarn hardhat run scripts/cw20erc20-deploy.ts --network testing)
# collect only the contract address part
contract_addr=$(echo "$output" | grep -oE '0x[0-9a-fA-F]+')
echo "ERC20 contract addr: $contract_addr"

# validate
contract_addr_len=${#contract_addr}
if [ $contract_addr_len -ne 42 ] ; then
   echo "Couldn't deploy ERC20 contract. CW20-ERC20 Test Failed"; 
fi

# try querying balance of owner -> get balance of from cosmwasm contract
output=$(ERC20_ADDRESS=$contract_addr yarn hardhat run scripts/cw20erc20-query-balance-of.ts --network testing)
balance_of=$(echo "$output" | awk '/^[0-9]+$/ { print $1 }')
if ! [ $balance_of == "1000000000" ] ; then
   echo "Could not query balance of CW20ERC20 address. CW20-ERC20 Test Failed"; exit 1
fi

# try querying decimals -> get decimals from cosmwasm contract
orai_balance_before_transfer=$(oraid query bank balance orai1kzkf6gttxqar9yrkxfe34ye4vg5v4m588ew7c9 orai --output json | jq '.balance.amount')
ERC20_ADDRESS=$contract_addr yarn hardhat run scripts/cw20erc20-transfer.ts --network testing
orai_balance_after_transfer=$(oraid query bank balance orai1kzkf6gttxqar9yrkxfe34ye4vg5v4m588ew7c9 orai --output json | jq '.balance.amount')

# try querying balance of owner after transfer -> should drop
output=$(ERC20_ADDRESS=$contract_addr yarn hardhat run scripts/cw20erc20-query-balance-of.ts --network testing)
balance_of=$(echo "$output" | awk '/^[0-9]+$/ { print $1 }')
if [ $balance_of == "1000000000" ] ; then
   echo "Could not transfer CW20 token from ERC20 contract. CW20-ERC20 Test Failed"; exit 1
fi 

if [ $orai_balance_before_transfer == $orai_balance_after_transfer ] ; then
   echo "Could not transfer native ORAI token from ERC20 contract. CW20-ERC20 Test Failed"; exit 1
fi 

echo "CW20-ERC20 Test Passed"; cd $current_dir