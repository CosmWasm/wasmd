#!/bin/bash
set -o errexit -o nounset -o pipefail -x

echo "-----------------------"
echo "## Add new wallet to state sync node"
oraid keys add wallet --keyring-backend=test --home=.oraid/state_sync
oraid keys add alice --keyring-backend=test --home=.oraid/state_sync
oraid keys add bob --keyring-backend=test --home=.oraid/state_sync

echo "-----------------------"
echo "## Send fund to state sync account"
oraid tx send $(oraid keys show validator1 -a --keyring-backend=test --home=$HOME/.oraid/validator1) $(oraid keys show wallet -a --keyring-backend=test --home=.oraid/state_sync) 500000orai --keyring-backend=test --home=$HOME/.oraid/validator1 --chain-id=testing --broadcast-mode block --gas 200000 --fees 2orai --node http://localhost:26657 --yes

oraid tx send $(oraid keys show validator1 -a --keyring-backend=test --home=$HOME/.oraid/validator1) $(oraid keys show alice -a --keyring-backend=test --home=.oraid/state_sync) 500000orai --keyring-backend=test --home=$HOME/.oraid/validator1 --chain-id=testing --broadcast-mode block --gas 200000 --fees 2orai --node http://localhost:26657 --yes

oraid tx send $(oraid keys show validator1 -a --keyring-backend=test --home=$HOME/.oraid/validator1) $(oraid keys show bob -a --keyring-backend=test --home=.oraid/state_sync) 500000orai --keyring-backend=test --home=$HOME/.oraid/validator1 --chain-id=testing --broadcast-mode block --gas 200000 --fees 2orai --node http://localhost:26657 --yes

echo "-----------------------"
echo "## Create new contract instance"
INIT='{"purchase_price":{"amount":"100","denom":"orai"},"transfer_price":{"amount":"999","denom":"orai"}}'
TXFLAG=${TX_FLAG:-"--node tcp://localhost:26647 --chain-id=testing --gas auto --gas-adjustment 1.3 -b block"}

# Instantiate the first contract. This contract was deploy in multinode-local-testnet.sh script
oraid tx wasm instantiate 1 "$INIT" --from=wallet --admin="$(oraid keys show wallet -a --keyring-backend=test --home=.oraid/state_sync)" --keyring-backend=test --home=.oraid/state_sync --label "name service" $TXFLAG -y

CONTRACT=$(oraid query wasm list-contract-by-code 1 -o json | jq -r '.contracts[-1]')
echo "* Contract address: $CONTRACT"

echo "### Query all"
RESP=$(oraid query wasm contract-state all "$CONTRACT" -o json)
echo "$RESP" | jq

# Excecute the first contract.
# Register a name for the wallet's address
echo "-----------------------"
echo "## Execute contract $CONTRACT"
REGISTER='{"register":{"name":"tony"}}'
oraid tx wasm execute $CONTRACT "$REGISTER" --amount 100orai --from=wallet --keyring-backend=test --home=.oraid/state_sync $TXFLAG -y -b block -o json | jq

# Query the first contract.
# Query the owner of the name record
NAME_QUERY='{"resolve_record": {"name": "tony"}}'
oraid query wasm contract-state smart $CONTRACT "$NAME_QUERY" --node "tcp://localhost:26647" --output json
# Owner is the wallet's address

# Excecute the first contract.
# Transfer the ownership of the name record to bob (change the "to" address to bob generated during environment setup)
# get alice's address
ALICE_ADDRESS=$(oraid keys show alice -a --keyring-backend=test --home=.oraid/state_sync)
TRANSFER={'"transfer"':{'"name"':'"tony"','"to"':\"$ALICE_ADDRESS\"}}
oraid tx wasm execute $CONTRACT $TRANSFER --amount 999orai --from=wallet --gas 1500000 --keyring-backend=test --home=.oraid/state_sync $TXFLAG -y

# Query the first contract.
# Query the record owner again to see the new owner address:
NAME_QUERY='{"resolve_record": {"name": "tony"}}'
oraid query wasm contract-state smart $CONTRACT "$NAME_QUERY" --node "tcp://localhost:26647" --output json
# Owner is the alice's address

# Set new first contract admin.
echo "-----------------------"
echo "## Set new admin"
echo "### Query old admin: $(oraid q wasm contract "$CONTRACT" -o json | jq -r '.contract_info.admin')"
echo "### Update contract"
oraid tx wasm set-contract-admin "$CONTRACT" "$(oraid keys show bob -a --keyring-backend=test --home=.oraid/state_sync)" \
  --from wallet -y --keyring-backend=test --home=.oraid/state_sync --chain-id=testing --gas 200000 --fees 2orai -b block -o json | jq
echo "### Query new admin: $(oraid q wasm contract "$CONTRACT" -o json | jq -r '.contract_info.admin')"

# Migrate the second contract. This contract was deploy in multinode-local-testnet.sh script
DEST_ACCOUNT=$(oraid keys show bob -a --keyring-backend=test --home=.oraid/state_sync)
oraid tx wasm migrate "$CONTRACT" 2 "{\"payout\": \"$DEST_ACCOUNT\"}" --from bob \
  --chain-id=testing --keyring-backend=test --home=.oraid/state_sync --gas 1500000 --fees 150orai -b block -y -o json | jq

# balances of bob: 500000 + 100 + 999 - 150 = 500949
echo "### Query destination account: 2"
oraid q bank balances "$DEST_ACCOUNT" -o json | jq

echo "### Query contract meta data: $CONTRACT"
oraid q wasm contract "$CONTRACT" -o json | jq

echo "### Query contract meta history: $CONTRACT"
oraid q wasm contract-history "$CONTRACT" -o json | jq
