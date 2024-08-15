#!/bin/bash
set -u

HIDE_LOGS="/dev/null"

# always returns true so set -e doesn't exit if it is not running.
killall oraid || true
rm -rf $HOME/.oraid/
killall screen

# make four orai directories
mkdir $HOME/.oraid
mkdir $HOME/.oraid/validator1
mkdir $HOME/.oraid/validator2
mkdir $HOME/.oraid/validator3

# init all three validators
oraid init --chain-id=testing validator1 --home=$HOME/.oraid/validator1
oraid init --chain-id=testing validator2 --home=$HOME/.oraid/validator2
oraid init --chain-id=testing validator3 --home=$HOME/.oraid/validator3

# create keys for all three validators
oraid keys add validator1 --keyring-backend=test --home=$HOME/.oraid/validator1 > $HIDE_LOGS
oraid keys add validator2 --keyring-backend=test --home=$HOME/.oraid/validator2 > $HIDE_LOGS
oraid keys add validator3 --keyring-backend=test --home=$HOME/.oraid/validator3 > $HIDE_LOGS

update_genesis () {    
    cat $HOME/.oraid/validator1/config/genesis.json | jq "$1" > $HOME/.oraid/validator1/config/tmp_genesis.json && mv $HOME/.oraid/validator1/config/tmp_genesis.json $HOME/.oraid/validator1/config/genesis.json
}

# change staking denom to orai
update_genesis '.app_state["staking"]["params"]["bond_denom"]="orai"'

# create validator node 1
oraid add-genesis-account $(oraid keys show validator1 -a --keyring-backend=test --home=$HOME/.oraid/validator1) 1000000000000orai,1000000000000stake --home=$HOME/.oraid/validator1 > $HIDE_LOGS
oraid gentx validator1 500000000orai --keyring-backend=test --home=$HOME/.oraid/validator1 --chain-id=testing > $HIDE_LOGS
oraid collect-gentxs --home=$HOME/.oraid/validator1 > $HIDE_LOGS
oraid validate-genesis --home=$HOME/.oraid/validator1 > $HIDE_LOGS

# update staking genesis
update_genesis '.app_state["staking"]["params"]["unbonding_time"]="240s"'
# update crisis variable to orai
update_genesis '.app_state["crisis"]["constant_fee"]["denom"]="orai"'
# udpate gov genesis
update_genesis '.app_state["gov"]["deposit_params"]["min_deposit"][0]["denom"]="orai"'
# update mint genesis
update_genesis '.app_state["mint"]["params"]["mint_denom"]="orai"'
update_genesis '.app_state["gov"]["voting_params"]["voting_period"]="5s"'
# port key (validator1 uses default ports)
# validator1 1317, 9090, 9091, 26658, 26657, 26656, 6060
# validator2 1316, 9088, 9089, 26655, 26654, 26653, 6061
# validator3 1315, 9086, 9087, 26652, 26651, 26650, 6062


# change app.toml values
VALIDATOR1_APP_TOML=$HOME/.oraid/validator1/config/app.toml
VALIDATOR2_APP_TOML=$HOME/.oraid/validator2/config/app.toml
VALIDATOR3_APP_TOML=$HOME/.oraid/validator3/config/app.toml

# change config.toml values
VALIDATOR1_CONFIG=$HOME/.oraid/validator1/config/config.toml
VALIDATOR2_CONFIG=$HOME/.oraid/validator2/config/config.toml
VALIDATOR3_CONFIG=$HOME/.oraid/validator3/config/config.toml

# validator2
sed -i -E 's|tcp://0.0.0.0:1317|tcp://0.0.0.0:1316|g' $VALIDATOR2_APP_TOML
sed -i -E 's|0.0.0.0:9090|0.0.0.0:9088|g' $VALIDATOR2_APP_TOML
sed -i -E 's|0.0.0.0:9091|0.0.0.0:9089|g' $VALIDATOR2_APP_TOML

# validator3
sed -i -E 's|tcp://0.0.0.0:1317|tcp://0.0.0.0:1315|g' $VALIDATOR3_APP_TOML
sed -i -E 's|0.0.0.0:9090|0.0.0.0:9086|g' $VALIDATOR3_APP_TOML
sed -i -E 's|0.0.0.0:9091|0.0.0.0:9087|g' $VALIDATOR3_APP_TOML

# Pruning - comment this configuration if you want to run upgrade script
pruning="custom"
pruning_keep_recent="5"
pruning_keep_every="10"
pruning_interval="10000"

sed -i -e "s%^pruning *=.*%pruning = \"$pruning\"%; " $VALIDATOR1_APP_TOML
sed -i -e "s%^pruning-keep-recent *=.*%pruning-keep-recent = \"$pruning_keep_recent\"%; " $VALIDATOR1_APP_TOML
sed -i -e "s%^pruning-keep-every *=.*%pruning-keep-every = \"$pruning_keep_every\"%; " $VALIDATOR1_APP_TOML
sed -i -e "s%^pruning-interval *=.*%pruning-interval = \"$pruning_interval\"%; " $VALIDATOR1_APP_TOML

sed -i -e "s%^pruning *=.*%pruning = \"$pruning\"%; " $VALIDATOR2_APP_TOML
sed -i -e "s%^pruning-keep-recent *=.*%pruning-keep-recent = \"$pruning_keep_recent\"%; " $VALIDATOR2_APP_TOML
sed -i -e "s%^pruning-keep-every *=.*%pruning-keep-every = \"$pruning_keep_every\"%; " $VALIDATOR2_APP_TOML
sed -i -e "s%^pruning-interval *=.*%pruning-interval = \"$pruning_interval\"%; " $VALIDATOR2_APP_TOML

sed -i -e "s%^pruning *=.*%pruning = \"$pruning\"%; " $VALIDATOR3_APP_TOML
sed -i -e "s%^pruning-keep-recent *=.*%pruning-keep-recent = \"$pruning_keep_recent\"%; " $VALIDATOR3_APP_TOML
sed -i -e "s%^pruning-keep-every *=.*%pruning-keep-every = \"$pruning_keep_every\"%; " $VALIDATOR3_APP_TOML
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

# modify jsonrpc ports to avoid clashing
sed -i -E 's|0.0.0.0:8545|0.0.0.0:7545|g' $VALIDATOR2_APP_TOML
sed -i -e "s%^ws-address *=.*%ws-address = \"0.0.0.0:7546\"%; " $VALIDATOR2_APP_TOML

sed -i -E 's|0.0.0.0:8545|0.0.0.0:6545|g' $VALIDATOR3_APP_TOML
sed -i -e "s%^ws-address *=.*%ws-address = \"0.0.0.0:6546\"%; " $VALIDATOR3_APP_TOML

# copy validator1 genesis file to validator2-3
cp $HOME/.oraid/validator1/config/genesis.json $HOME/.oraid/validator2/config/genesis.json
cp $HOME/.oraid/validator1/config/genesis.json $HOME/.oraid/validator3/config/genesis.json

# copy tendermint node id of validator1 to persistent peers of validator2-3
sed -i -E "s|persistent_peers = \"\"|persistent_peers = \"$(oraid tendermint show-node-id --home=$HOME/.oraid/validator1)@localhost:26656\"|g" $VALIDATOR2_CONFIG
sed -i -E "s|persistent_peers = \"\"|persistent_peers = \"$(oraid tendermint show-node-id --home=$HOME/.oraid/validator1)@localhost:26656\"|g" $VALIDATOR3_CONFIG

# start all three validators
screen -S validator1 -d -m oraid start --home=$HOME/.oraid/validator1
screen -S validator2 -d -m oraid start --home=$HOME/.oraid/validator2
screen -S validator3 -d -m oraid start --home=$HOME/.oraid/validator3

# send orai from first validator to second validator
echo "Waiting 7 seconds to send funds to validators 2 and 3..."
sleep 7

oraid tx send $(oraid keys show validator1 -a --keyring-backend=test --home=$HOME/.oraid/validator1) $(oraid keys show validator2 -a --keyring-backend=test --home=$HOME/.oraid/validator2) 5000000000orai --keyring-backend=test --home=$HOME/.oraid/validator1 --chain-id=testing --broadcast-mode block --gas 200000 --fees 2orai --node http://localhost:26657 --yes > $HIDE_LOGS
oraid tx send $(oraid keys show validator1 -a --keyring-backend=test --home=$HOME/.oraid/validator1) $(oraid keys show validator3 -a --keyring-backend=test --home=$HOME/.oraid/validator3) 4000000000orai --keyring-backend=test --home=$HOME/.oraid/validator1 --chain-id=testing --broadcast-mode block --gas 200000 --fees 2orai --node http://localhost:26657 --yes > $HIDE_LOGS
# send test orai to a test account
oraid tx send $(oraid keys show validator1 -a --keyring-backend=test --home=$HOME/.oraid/validator1) orai14n3tx8s5ftzhlxvq0w5962v60vd82h30rha573 5000000000orai --keyring-backend=test --home=$HOME/.oraid/validator1 --chain-id=testing --broadcast-mode block --gas 200000 --fees 2orai --node http://localhost:26657 --yes > $HIDE_LOGS

# create second & third validator
oraid tx staking create-validator --amount=500000000orai --from=validator2 --pubkey=$(oraid tendermint show-validator --home=$HOME/.oraid/validator2) --moniker="validator2" --chain-id="testing" --commission-rate="0.1" --commission-max-rate="0.2" --commission-max-change-rate="0.05" --min-self-delegation="500000000" --keyring-backend=test --home=$HOME/.oraid/validator2 --broadcast-mode block --gas 200000 --fees 2orai --node http://localhost:26657 --yes > $HIDE_LOGS
oraid tx staking create-validator --amount=400000000orai --from=validator3 --pubkey=$(oraid tendermint show-validator --home=$HOME/.oraid/validator3) --moniker="validator3" --chain-id="testing" --commission-rate="0.1" --commission-max-rate="0.2" --commission-max-change-rate="0.05" --min-self-delegation="400000000" --keyring-backend=test --home=$HOME/.oraid/validator3 --broadcast-mode block --gas 200000 --fees 2orai --node http://localhost:26657 --yes > $HIDE_LOGS

echo "All 3 Validators are up and running!"