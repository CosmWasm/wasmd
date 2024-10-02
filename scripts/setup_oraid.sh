#!/bin/bash
set -ux

CHAIN_ID=${CHAIN_ID:-testing}
USER=${USER:-tupt}
MONIKER=${MONIKER:-node001}
HIDE_LOGS="/dev/null"
# PASSWORD=${PASSWORD:-$1}
NODE_HOME="$PWD/.oraid"
ARGS="--keyring-backend test --home $NODE_HOME"
START_ARGS="--json-rpc.address="0.0.0.0:8545" --json-rpc.ws-address="0.0.0.0:8546" --json-rpc.api="eth,web3,net,txpool,debug" --json-rpc.enable --home $NODE_HOME"

rm -rf $NODE_HOME

oraid init --chain-id "$CHAIN_ID" "$MONIKER" --home $NODE_HOME >$HIDE_LOGS

oraid keys add $USER $ARGS 2>&1 | tee account.txt
oraid keys add $USER-eth $ARGS --eth 2>&1 | tee account-eth.txt
oraid keys unsafe-export-eth-key $USER-eth $ARGS 2>&1 | tee priv-eth.txt

# hardcode the validator account for this instance
oraid genesis add-genesis-account $USER "100000000000000orai" $ARGS
oraid genesis add-genesis-account $USER-eth "100000000000000orai" $ARGS
oraid genesis add-genesis-account orai1kzkf6gttxqar9yrkxfe34ye4vg5v4m588ew7c9 "100000000000000orai" $ARGS

# submit a genesis validator tx
# Workraround for https://github.com/cosmos/cosmos-sdk/issues/8251
oraid genesis gentx $USER "250000000orai" --chain-id="$CHAIN_ID" -y $ARGS >$HIDE_LOGS

oraid genesis collect-gentxs --home $NODE_HOME >$HIDE_LOGS

oraid start $START_ARGS
