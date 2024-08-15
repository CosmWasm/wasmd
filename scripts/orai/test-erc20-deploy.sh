#!/bin/bash

# This script should be called when there's already a running local network

set -eu

# hard-coded test private key. DO NOT USE!!
PRIVATE_KEY_ETH=${PRIVATE_KEY_ETH:-"021646C7F742C743E60CC460C56242738A3951667E71C803929CB84B6FA4B0D6"}
# run erc20 tests
current_dir=$PWD

# clone repo
rm -rf ../../erc20-deploy/ && git clone https://github.com/oraichain/evm-bridge-proxy.git ../../erc20-deploy && cd ../../erc20-deploy

# prepare env and chain
yarn && yarn compile;
echo "PRIVATE_KEY=$PRIVATE_KEY_ETH" > .env

# run the test
output=$(yarn erc20-deploy)
# collect only the contract address part
contract_addr=$(echo "$output" | grep -oE '0x[0-9a-fA-F]+')
echo "ERC20 contract addr: $contract_addr"

# validate
contract_addr_len=${#contract_addr}
if [ $contract_addr_len -ne 42 ] ; then
   echo "ERC20 Test Failed"; 
   # clean up
   rm -rf ../../erc20-deploy/ && exit 1
fi

echo "ERC20 Test Passed"; rm -rf ../../erc20-deploy/ && cd $current_dir