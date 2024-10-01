#!/bin/bash

# This script should be called when there's already a running local network

set -ux

# hard-coded test private key. DO NOT USE!!
PRIVATE_KEY_ETH=${PRIVATE_KEY_ETH:-"021646C7F742C743E60CC460C56242738A3951667E71C803929CB84B6FA4B0D6"}
# run evm send token test
current_dir=$PWD

clone or pull latest repo
if [ -d "$PWD/../evm-send-token" ]; then
    cd ../evm-send-token
    git pull origin master
    git checkout feat/add-evm-send-token
else
    git clone https://github.com/oraichain/evm-bridge-proxy.git ../evm-send-token
    cd ../evm-send-token
    git checkout feat/add-evm-send-token
fi

# prepare env and chain
yarn && yarn compile
echo "PRIVATE_KEY=$PRIVATE_KEY_ETH" >.env

yarn evm-send-token

echo "EVM Send Token Test Passed"
cd $current_dir
