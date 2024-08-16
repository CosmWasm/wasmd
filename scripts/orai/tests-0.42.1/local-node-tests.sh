#!/bin/bash

CHAIN_ID=${CHAIN_ID:-testing}
USER=${USER:-tupt}
MONIKER=${MONIKER:-node001}
WASM_PATH=${WASM_PATH:-"$PWD/scripts/wasm_file/swapmap.wasm"}
EXECUTE_MSG=${EXECUTE_MSG:-'{"ping":{}}'}
ARGS="--chain-id testing -y --keyring-backend test --gas auto --gas-adjustment 1.5 -b block"
HIDE_LOGS="/dev/null"

pkill oraid
rm -rf .oraid/

update_genesis () {    
    cat $PWD/.oraid/config/genesis.json | jq "$1" > $PWD/.oraid/config/tmp_genesis.json && mv $PWD/.oraid/config/tmp_genesis.json $PWD/.oraid/config/genesis.json
}

oraid init --chain-id "$CHAIN_ID" "$MONIKER"

# 2s for fast test
update_genesis '.app_state["gov"]["voting_params"]["voting_period"]="2s"'

oraid keys add $USER --keyring-backend test 2>&1 | tee account.txt > $HIDE_LOGS
oraid keys add $USER-eth --keyring-backend test --eth 2>&1 | tee account-eth.txt > $HIDE_LOGS
oraid keys unsafe-export-eth-key $USER-eth --keyring-backend test 2>&1 | tee priv-eth.txt > $HIDE_LOGS

# hardcode the validator account for this instance
oraid add-genesis-account $USER "1000000000orai" --keyring-backend test > $HIDE_LOGS

# submit a genesis validator tx
oraid gentx $USER "2500000orai" --chain-id="$CHAIN_ID" --amount="2500000orai" -y --keyring-backend test > $HIDE_LOGS

oraid collect-gentxs

screen -S test-gasless -d -m oraid start --json-rpc.address="0.0.0.0:8545" --json-rpc.ws-address="0.0.0.0:8546" --json-rpc.api="eth,web3,net,txpool,debug" --json-rpc.enable

# wait for the node to start
sleep 2

sh $PWD/scripts/tests-0.42.1/test-gasless.sh
sh $PWD/scripts/tests-0.42.1/test-tokenfactory.sh
sh $PWD/scripts/tests-0.42.1/test-tokenfactory-bindings.sh
sh $PWD/scripts/tests-0.42.1/test-evm-cosmos-mapping.sh
bash $PWD/scripts/tests-0.42.1/test-evm-cosmos-mapping-complex.sh