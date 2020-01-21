#!/bin/bash

echo Starting Wasmd...

mkdir -p /root/log
wasmd start --rpc.laddr tcp://0.0.0.0:26657 > /root/log/wasmd.log &

sleep 10
echo Starting Rest Server...

wasmcli rest-server --laddr tcp://0.0.0.0:1317 --trust-node
