#!/bin/bash

# Default home is $HOME/.wasmd
# If you want to setup multiple apps on your local make sure to change this value
export APP_HOME="$HOME/.wasmd"
export RPC="http://localhost:26657"
export CHAIN_ID="localnet"
# --keyring-backend test is for testing purposes
# Change it to --keyring-backend file for secure usage.
export KEYRING="--keyring-backend test --keyring-dir $HOME/.wasmd_keys"

export NODE="--node $RPC"
export TXFLAG="$NODE --chain-id ${CHAIN_ID} --gas-prices 0.01ucosm --gas auto --gas-adjustment 1.3"
