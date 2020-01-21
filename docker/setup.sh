#!/bin/sh

wasmd init --chain-id=testing testing
wasmcli keys add validator
wasmd add-genesis-account validator 1000000000stake,1000000000validatortoken
wasmd gentx --name validator
wasmd collect-gentxs