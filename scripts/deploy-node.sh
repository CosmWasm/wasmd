# default home is ~/.wasmd
# if you want to setup multiple apps on your local make sure to change this value
APP_HOME="$HOME/.wasmd"
RPC="http://localhost:26657"
CHAIN_ID="localnet"
KEYRING="test"

echo >&1 "installing wasmd"
rm -rf $HOME/.wasmd
make install
# initialize wasmd configuration files
wasmd init localnet --chain-id ${CHAIN_ID}

# Create main address
# --keyring-backend test is for testing purposes
# Change it to --keyring-backend file for secure usage.
wasmd keys add main --keyring-backend $KEYRING

# create validator address
wasmd keys add validator --keyring-backend $KEYRING



# add your wallet addresses to genesis
wasmd add-genesis-account $(wasmd keys show -a main --keyring-backend $KEYRING) 10000000000ucosm,10000000000stake --keyring-backend $KEYRING
wasmd add-genesis-account $(wasmd keys show -a validator --keyring-backend $KEYRING) 10000000000ucosm,10000000000stake --keyring-backend $KEYRING

# add fred's address as validator's address
wasmd gentx validator 1000000000stake --chain-id ${CHAIN_ID} --keyring-backend $KEYRING

# collect gentxs to genesis
wasmd collect-gentxs

# validate the genesis file
wasmd validate-genesis

# run the node
wasmd start --minimum-gas-prices=0.0000stake