#!/bin/bash
set -u

HIDE_LOGS="/dev/null"
CHAIN_ID=${CHAIN_ID:-testing}
ARGS="--keyring-backend test"
TX_SEND_ARGS="$ARGS --chain-id $CHAIN_ID --gas 200000 --fees 2orai --node http://localhost:26657 --yes"

VALIDATOR1_HOME="$HOME/.oraid/validator1"
VALIDATOR2_HOME="$HOME/.oraid/validator2"
VALIDATOR3_HOME="$HOME/.oraid/validator3"

# always returns true so set -e doesn't exit if it is not running.
killall oraid || true
rm -rf $HOME/.oraid/
killall screen

# make four orai directories
mkdir $HOME/.oraid
mkdir $VALIDATOR1_HOME
mkdir $VALIDATOR2_HOME
mkdir $VALIDATOR3_HOME

# init all three validators
oraid init --chain-id $CHAIN_ID validator1 --home $VALIDATOR1_HOME
oraid init --chain-id $CHAIN_ID validator2 --home $VALIDATOR2_HOME
oraid init --chain-id $CHAIN_ID validator3 --home $VALIDATOR3_HOME

# create keys for all three validators
oraid keys add validator1 $ARGS --home $VALIDATOR1_HOME >$HIDE_LOGS
oraid keys add validator2 $ARGS --home $VALIDATOR2_HOME >$HIDE_LOGS
oraid keys add validator3 $ARGS --home $VALIDATOR3_HOME >$HIDE_LOGS

update_genesis () {
    cat $VALIDATOR1_HOME/config/genesis.json | jq "$1" > $VALIDATOR1_HOME/config/tmp_genesis.json && mv $VALIDATOR1_HOME/config/tmp_genesis.json $VALIDATOR1_HOME/config/genesis.json
}

# change staking denom to orai
update_genesis '.app_state["staking"]["params"]["bond_denom"]="orai"'

# create validator node 1
oraid genesis add-genesis-account $(oraid keys show validator1 -a $ARGS --home $VALIDATOR1_HOME) 1000000000000orai,1000000000000stake --home $VALIDATOR1_HOME >$HIDE_LOGS
oraid genesis gentx validator1 500000000orai $ARGS --home $VALIDATOR1_HOME --chain-id $CHAIN_ID >$HIDE_LOGS
oraid genesis collect-gentxs --home $VALIDATOR1_HOME >$HIDE_LOGS
oraid genesis validate --home $VALIDATOR1_HOME >$HIDE_LOGS

# update staking genesis
update_genesis '.app_state["staking"]["params"]["unbonding_time"]="240s"'
# update crisis variable to orai
update_genesis '.app_state["crisis"]["constant_fee"]["denom"]="orai"'
# udpate gov genesis
update_genesis '.app_state["gov"]["params"]["min_deposit"][0]["denom"]="orai"'
update_genesis '.app_state["gov"]["params"]["expedited_min_deposit"][0]["denom"]="orai"'
update_genesis '.app_state["gov"]["params"]["voting_period"]="6s"'
# update mint genesis
update_genesis '.app_state["mint"]["params"]["mint_denom"]="orai"'
# port key (validator1 uses default ports)
# validator1 1317, 9090, 9091, 26658, 26657, 26656, 6060
# validator2 1316, 9088, 9089, 26655, 26654, 26653, 6061
# validator3 1315, 9086, 9087, 26652, 26651, 26650, 6062

# change app.toml values
VALIDATOR1_APP_TOML=$VALIDATOR1_HOME/config/app.toml
VALIDATOR2_APP_TOML=$VALIDATOR2_HOME/config/app.toml
VALIDATOR3_APP_TOML=$VALIDATOR3_HOME/config/app.toml

# change config.toml values
VALIDATOR1_CONFIG=$VALIDATOR1_HOME/config/config.toml
VALIDATOR2_CONFIG=$VALIDATOR2_HOME/config/config.toml
VALIDATOR3_CONFIG=$VALIDATOR3_HOME/config/config.toml

# validator2
sed -i -E 's|tcp://localhost:1317|tcp://localhost:1316|g' $VALIDATOR2_APP_TOML
sed -i -E 's|localhost:9090|localhost:9088|g' $VALIDATOR2_APP_TOML
# sed -i -E 's|0.0.0.0:9091|0.0.0.0:9089|g' $VALIDATOR2_APP_TOML

# validator3
sed -i -E 's|tcp://localhost:1317|tcp://localhost:1315|g' $VALIDATOR3_APP_TOML
sed -i -E 's|localhost:9090|localhost:9086|g' $VALIDATOR3_APP_TOML
# sed -i -E 's|0.0.0.0:9091|0.0.0.0:9087|g' $VALIDATOR3_APP_TOML

# Pruning - comment this configuration if you want to run upgrade script
pruning="custom"
pruning_keep_recent="5"
pruning_keep_every="10"
pruning_interval="10000"

sed -i -e "s%^pruning *=.*%pruning = \"$pruning\"%; " $VALIDATOR1_APP_TOML
sed -i -e "s%^pruning-keep-recent *=.*%pruning-keep-recent = \"$pruning_keep_recent\"%; " $VALIDATOR1_APP_TOML
# sed -i -e "s%^pruning-keep-every *=.*%pruning-keep-every = \"$pruning_keep_every\"%; " $VALIDATOR1_APP_TOML
sed -i -e "s%^pruning-interval *=.*%pruning-interval = \"$pruning_interval\"%; " $VALIDATOR1_APP_TOML

sed -i -e "s%^pruning *=.*%pruning = \"$pruning\"%; " $VALIDATOR2_APP_TOML
sed -i -e "s%^pruning-keep-recent *=.*%pruning-keep-recent = \"$pruning_keep_recent\"%; " $VALIDATOR2_APP_TOML
sed -i -e "s%^pruning-keep-every *=.*%pruning-keep-every = \"$pruning_keep_every\"%; " $VALIDATOR2_APP_TOML
sed -i -e "s%^pruning-interval *=.*%pruning-interval = \"$pruning_interval\"%; " $VALIDATOR2_APP_TOML

sed -i -e "s%^pruning *=.*%pruning = \"$pruning\"%; " $VALIDATOR3_APP_TOML
sed -i -e "s%^pruning-keep-recent *=.*%pruning-keep-recent = \"$pruning_keep_recent\"%; " $VALIDATOR3_APP_TOML
# sed -i -e "s%^pruning-keep-every *=.*%pruning-keep-every = \"$pruning_keep_every\"%; " $VALIDATOR3_APP_TOML
sed -i -e "s%^pruning-interval *=.*%pruning-interval = \"$pruning_interval\"%; " $VALIDATOR3_APP_TOML

# state sync  - comment this configuration if you want to run upgrade script
snapshot_interval="10"
snapshot_keep_recent="2"

sed -i -e "s%^snapshot-interval *=.*%snapshot-interval = \"$snapshot_interval\"%; " $VALIDATOR1_APP_TOML
sed -i -e "s%^snapshot-keep-recent *=.*%snapshot-keep-recent = \"$snapshot_keep_recent\"%; " $VALIDATOR1_APP_TOML

sed -i -e "s%^snapshot-interval *=.*%snapshot-interval = \"$snapshot_interval\"%; " $VALIDATOR2_APP_TOML
sed -i -e "s%^snapshot-keep-recent *=.*%snapshot-keep-recent = \"$snapshot_keep_recent\"%; " $VALIDATOR2_APP_TOML

sed -i -e "s%^snapshot-interval *=.*%snapshot-interval = \"$snapshot_interval\"%; " $VALIDATOR3_APP_TOML
sed -i -e "s%^snapshot-keep-recent *=.*%snapshot-keep-recent = \"$snapshot_keep_recent\"%; " $VALIDATOR3_APP_TOML

# validator1
sed -i -E 's|allow_duplicate_ip = false|allow_duplicate_ip = true|g' $VALIDATOR1_CONFIG
sed -i -e "s%^timeout_broadcast_tx_commit *=.*%timeout_broadcast_tx_commit = \"60s\"%; " $VALIDATOR1_CONFIG

# validator2
sed -i -E 's|tcp://127.0.0.1:26658|tcp://0.0.0.0:26655|g' $VALIDATOR2_CONFIG
sed -i -E 's|tcp://127.0.0.1:26657|tcp://0.0.0.0:26654|g' $VALIDATOR2_CONFIG
sed -i -E 's|tcp://0.0.0.0:26656|tcp://0.0.0.0:26653|g' $VALIDATOR2_CONFIG
sed -i -E 's|allow_duplicate_ip = false|allow_duplicate_ip = true|g' $VALIDATOR2_CONFIG

# validator3
sed -i -E 's|tcp://127.0.0.1:26658|tcp://0.0.0.0:26652|g' $VALIDATOR3_CONFIG
sed -i -E 's|tcp://127.0.0.1:26657|tcp://0.0.0.0:26651|g' $VALIDATOR3_CONFIG
sed -i -E 's|tcp://0.0.0.0:26656|tcp://0.0.0.0:26650|g' $VALIDATOR3_CONFIG
sed -i -E 's|allow_duplicate_ip = false|allow_duplicate_ip = true|g' $VALIDATOR3_CONFIG

# enable lcd
sed -i -e "s%^enable *=.*%enable = true%; " $VALIDATOR1_APP_TOML
sed -i -e "s%^enable *=.*%enable = true%; " $VALIDATOR2_APP_TOML
sed -i -e "s%^enable *=.*%enable = true%; " $VALIDATOR3_APP_TOML

# modify jsonrpc ports to avoid clashing
sed -i -E 's|127.0.0.1:8545|0.0.0.0:7545|g' $VALIDATOR2_APP_TOML
sed -i -e "s%^ws-address *=.*%ws-address = \"0.0.0.0:7546\"%; " $VALIDATOR2_APP_TOML

sed -i -E 's|127.0.0.1:8545|0.0.0.0:6545|g' $VALIDATOR3_APP_TOML
sed -i -e "s%^ws-address *=.*%ws-address = \"0.0.0.0:6546\"%; " $VALIDATOR3_APP_TOML

# copy validator1 genesis file to validator2-3
cp $VALIDATOR1_HOME/config/genesis.json $VALIDATOR2_HOME/config/genesis.json
cp $VALIDATOR1_HOME/config/genesis.json $VALIDATOR3_HOME/config/genesis.json

# copy tendermint node id of validator1 to persistent peers of validator2-3
sed -i -E "s|persistent_peers = \"\"|persistent_peers = \"$(oraid comet show-node-id --home $VALIDATOR1_HOME)@localhost:26656\"|g" $VALIDATOR2_CONFIG
sed -i -E "s|persistent_peers = \"\"|persistent_peers = \"$(oraid comet show-node-id --home $VALIDATOR1_HOME)@localhost:26656\"|g" $VALIDATOR3_CONFIG

# start all three validators
screen -S validator1 -d -m oraid start --home $VALIDATOR1_HOME
screen -S validator2 -d -m oraid start --home $VALIDATOR2_HOME
screen -S validator3 -d -m oraid start --home $VALIDATOR3_HOME

# send orai from first validator to second validator
echo "Waiting 6 seconds to start the validators..."
sleep 5

VALIDATOR1_ADDRESS=$(oraid keys show validator1 -a $ARGS --home $VALIDATOR1_HOME)
VALIDATOR2_ADDRESS=$(oraid keys show validator2 -a $ARGS --home $VALIDATOR2_HOME)
VALIDATOR3_ADDRESS=$(oraid keys show validator3 -a $ARGS --home $VALIDATOR3_HOME)

oraid tx bank send $VALIDATOR1_ADDRESS $VALIDATOR2_ADDRESS 5000000000orai --home $VALIDATOR1_HOME $TX_SEND_ARGS >$HIDE_LOGS
# need to sleep to send fund to validator3
sleep 1
oraid tx bank send $VALIDATOR1_ADDRESS $VALIDATOR3_ADDRESS 5000000000orai --home $VALIDATOR1_HOME $TX_SEND_ARGS >$HIDE_LOGS
# send test orai to a test account
sleep 1
oraid tx bank send $VALIDATOR1_ADDRESS orai1kzkf6gttxqar9yrkxfe34ye4vg5v4m588ew7c9 5000000000orai --home $VALIDATOR1_HOME $TX_SEND_ARGS > $HIDE_LOGS

echo "Waiting 1 second to create two new validators..."
sleep 1

validator='{"pubkey":{"@type":"/cosmos.crypto.ed25519.PubKey","key":"xj740yWkYQbJCNkof2m7hQWpyaO6eFQ8qvGmYrtsqjQ="},"amount":"500000000orai","moniker":"validator3","identity":"","website":"","security":"","details":"","commission-rate":"0.1","commission-max-rate":"0.2","commission-max-change-rate":"0.05","min-self-delegation":"500000000"}'
validator_info_temp_path=$PWD/scripts/json/validator.json

echo $validator > $validator_info_temp_path

update_validator () {
    cat $validator_info_temp_path | jq "$1" > $PWD/scripts/json/temp_validator.json && mv $PWD/scripts/json/temp_validator.json $validator_info_temp_path
}

VALIDATOR2_PUBKEY=$(oraid comet show-validator --home $VALIDATOR2_HOME | jq -r '.key')
VALIDATOR3_PUBKEY=$(oraid comet show-validator --home $VALIDATOR3_HOME | jq -r '.key')

# create second validator
update_validator ".pubkey[\"key\"]=\"$VALIDATOR2_PUBKEY\""
update_validator '.moniker="validator2"'
update_validator '.amount="500000000orai"'
oraid tx staking create-validator $PWD/scripts/json/validator.json --from validator2 --home $VALIDATOR2_HOME $TX_SEND_ARGS >$HIDE_LOGS

# create third validator
update_validator ".pubkey[\"key\"]=\"$VALIDATOR3_PUBKEY\""
update_validator '.moniker="validator3"'
update_validator '.amount="500000000orai"'
oraid tx staking create-validator $PWD/scripts/json/validator.json  --from validator3 --home $VALIDATOR3_HOME $TX_SEND_ARGS  > $HIDE_LOGS

echo "All 3 Validators are up and running!"
# cleanup validator.json
rm $validator_info_temp_path
