#!/bin/bash

CHAIN_ID=${CHAIN_ID:-testing}
USER=${USER:-tupt}
MONIKER=${MONIKER:-node001}
NODE_HOME="$PWD/.oraid"
ARGS="--keyring-backend test --home $NODE_HOME"
START_ARGS="--json-rpc.address="0.0.0.0:8545" --json-rpc.ws-address="0.0.0.0:8546" --json-rpc.api="eth,web3,net,txpool,debug" --json-rpc.enable --home $NODE_HOME"
HIDE_LOGS="/dev/null"

pkill oraid
rm -rf $NODE_HOME

update_genesis() {
    cat $NODE_HOME/config/genesis.json | jq "$1" >$NODE_HOME/config/tmp_genesis.json && mv $NODE_HOME/config/tmp_genesis.json $NODE_HOME/config/genesis.json
}

oraid init --chain-id "$CHAIN_ID" "$MONIKER" --home $NODE_HOME &>$HIDE_LOGS

# 2s for fast test
update_genesis '.app_state["gov"]["voting_params"]["voting_period"]="2s"'

oraid keys add $USER $ARGS 2>&1 | tee account.txt
oraid keys add $USER-eth $ARGS --eth 2>&1 | tee account-eth.txt
oraid keys unsafe-export-eth-key $USER-eth $ARGS 2>&1 | tee priv-eth.txt

# hardcode the validator account for this instance
oraid genesis add-genesis-account $USER "100000000000000orai" $ARGS

# submit a genesis validator tx
oraid genesis gentx $USER "250000000orai" --chain-id="$CHAIN_ID" -y $ARGS &>$HIDE_LOGS

oraid genesis collect-gentxs --home $NODE_HOME &>$HIDE_LOGS

screen -S test-gasless -d -m oraid start $START_ARGS

# wait for the node to start
sleep 2

# sh $PWD/scripts/tests-0.42.1/test-gasless.sh
sh $PWD/scripts/tests-0.42.1/test-tokenfactory.sh
# sh $PWD/scripts/tests-0.42.1/test-tokenfactory-bindings.sh
# sh $PWD/scripts/tests-0.42.1/test-evm-cosmos-mapping.sh
# bash $PWD/scripts/tests-0.42.1/test-evm-cosmos-mapping-complex.sh
