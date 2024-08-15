#!/bin/sh

VERSION=${VERSION:-"v0.41.2"}
HEIGHT=${HEIGHT:-200}
VALIDATOR_HOME=${VALIDATOR_HOME:-"$HOME/.oraid/validator1"}

oraid tx gov submit-proposal software-upgrade "v0.41.2" --title "foobar" --description "foobar"  --from validator1 --upgrade-height $HEIGHT --upgrade-info "x" --deposit 10000000orai --chain-id testing --keyring-backend test --home $VALIDATOR_HOME -y --fees 2orai -b block

oraid tx gov vote 1 yes --from validator1 --chain-id testing -y --keyring-backend test --home "$HOME/.oraid/validator1" --fees 2orai -b block && oraid tx gov vote 1 yes --from validator2 --chain-id testing -y --keyring-backend test --home "$HOME/.oraid/validator2" --fees 2orai -b block
