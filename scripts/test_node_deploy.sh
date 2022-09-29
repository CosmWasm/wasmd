#!/bin/bash

KEY="test"
CHAINID="wasmd-testnet-1"
KEYRING="test"
MONIKER="localtestnet"
KEYALGO="secp256k1"
LOGLEVEL="info"

# retrieve all args
WILL_START_FRESH=0
WILL_RECOVER=0
WILL_INSTALL=0
WILL_CONTINUE=0
# $# is to check number of arguments
if [ $# -gt 0 ];
then
    # $@ is for getting list of arguments
    for arg in "$@"; do
        case $arg in
        --fresh)
            WILL_START_FRESH=1
            shift
            ;;
        --recover)
            WILL_RECOVER=1
            shift
            ;;
        --install)
            WILL_INSTALL=1
            shift
            ;;
        --continue)
            WILL_CONTINUE=1
            shift
            ;;
        *)
            printf >&2 "wrong argument somewhere"; exit 1;
            ;;
        esac
    done
fi


echo >&1 "installing wasmd"
rm -rf $HOME/.wasmd*
go install ./...

wasmd config keyring-backend $KEYRING
wasmd config chain-id $CHAINID

# determine if user wants to recorver or create new
rm debug/keys.txt

if [ $WILL_RECOVER -eq 0 ];
then
    KEY_INFO=$(wasmd keys add $KEY --keyring-backend $KEYRING --algo $KEYALGO)
    echo $KEY_INFO >> debug/keys.txt
else
    KEY_INFO=$(wasmd keys add $KEY --keyring-backend $KEYRING --algo $KEYALGO --recover)
    echo $KEY_INFO >> debug/keys.txt
fi

echo >&1 "\n"

# init chain
wasmd init $MONIKER --chain-id $CHAINID

# enable rest server and swagger
toml set --toml-path $HOME/.wasmd/config/app.toml api.swagger true
toml set --toml-path $HOME/.wasmd/config/app.toml api.enable true

# Allocate genesis accounts (cosmos formatted addresses)
wasmd add-genesis-account $KEY 1000000000000stake --keyring-backend $KEYRING

# Sign genesis transaction
wasmd gentx $KEY 1000000stake --keyring-backend $KEYRING --chain-id $CHAINID

# Collect genesis tx
wasmd collect-gentxs

# Run this to ensure everything worked and that the genesis file is setup correctly
wasmd validate-genesis

# Start the node (remove the --pruning=nothing flag if historical queries are not needed)
wasmd start --pruning=nothing --log_level $LOGLEVEL --minimum-gas-prices=0stake --rpc.laddr tcp://0.0.0.0:26657