#!/bin/bash

source ./localnet_vars.sh

# Initialize wasmd configuration files
wasmd init localnet --chain-id ${CHAIN_ID} --home ${APP_HOME}

# add minimum gas prices config to app configuration file
sed -i -r 's/minimum-gas-prices = ""/minimum-gas-prices = "0.01ucosm"/' ${APP_HOME}/config/app.toml

# Create main address
wasmd keys add main $KEYRING

# create your address
wasmd keys add $USER $KEYRING

# Add wallet addresses to genesis
wasmd add-genesis-account $(wasmd keys show -a main $KEYRING) 10000000000ucosm,10000000000stake --home ${APP_HOME}
wasmd add-genesis-account $(wasmd keys show -a $USER $KEYRING) 10000000000ucosm,10000000000stake --home ${APP_HOME}

# Add your address as a validator
wasmd gentx $USER $KEYRING 1000000000stake --home ${APP_HOME} --chain-id ${CHAIN_ID}

# Collect gentxs to genesis
wasmd collect-gentxs --home ${APP_HOME}

# Validate the genesis file
wasmd validate-genesis --home ${APP_HOME}
