#!/bin/bash
# Before running this script, you must setup local network:

set -eux

ARGS="--chain-id testing -y --keyring-backend test --gas auto --gas-adjustment 1.5 -b sync"
VALIDATOR1_ARGS=${VALIDATOR1_ARGS:-"--from validator1 --home $HOME/.oraid/validator1"}

HIDE_LOGS="/dev/null"

add_globalfee() {
   amount=$1
   ret=$(oraid tx globalfee add-globalfee orai $amount foobar 10000000orai foobar $VALIDATOR1_ARGS $ARGS --output json --fees 200000orai)
   proposal_txhash=$(echo $ret | jq -r '.txhash')
   sleep 3

   proposal_id=$(oraid query tx $proposal_txhash --output json | jq '.events[8].attributes[0].value | tonumber')

   oraid tx gov vote $proposal_id yes --from validator1 --home "$HOME/.oraid/validator1" $ARGS --fees 200000orai > $HIDE_LOGS && oraid tx gov vote $proposal_id yes --from validator2 --home "$HOME/.oraid/validator2" $ARGS --fees 200000orai > $HIDE_LOGS
}

validate_globalfee() {
   expected_amount="\"$1\""
   actual_globalfee_amount=$(oraid query globalfee minimum-gas-prices --output json | jq '.minimum_gas_prices[0].amount')
   if ! [[ $expected_amount == $actual_globalfee_amount ]] ; then
      echo "Globalfee amount is not correct. Global fee test failed"; exit 1
   fi
}

add_globalfee 0.0001
# wait til proposal passes
sleep 5
validate_globalfee 0.000100000000000000
add_globalfee 0.0002
# wait til proposal passes
sleep 5
validate_globalfee 0.000200000000000000

globalfee_error_message=$(oraid tx bank send validator1 orai1kzkf6gttxqar9yrkxfe34ye4vg5v4m588ew7c9 1orai $VALIDATOR1_ARGS --keyring-backend test --chain-id testing --gas 200000 -y --output json | jq '.raw_log')

# Check if the string contains "40orai" = 0.000200000000000000 (gas price) * 200000 (gas)
if ! [[ "$globalfee_error_message" == *"40orai"* ]]; then
    echo "Minimum global fee is not correct. Test global fee failed!"
fi

# reset globalfee to 0 for future tests
add_globalfee 0

echo "Global Fee test passed"
