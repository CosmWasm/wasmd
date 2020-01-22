#!/bin/sh

PASSWORD=${PASSWORD:-1234567890}

wasmd init --chain-id=testing testing
(echo $PASSWORD; echo $PASSWORD) | wasmcli keys add validator
echo $PASSWORD | wasmd add-genesis-account validator 1000000000stake,1000000000validatortoken
(echo $PASSWORD; echo $PASSWORD; echo $PASSWORD) | wasmd gentx --name validator
wasmd collect-gentxs