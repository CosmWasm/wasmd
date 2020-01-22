#!/bin/bash

./run_wasmd.sh >> /root/log/wasmd.log &

sleep 4
echo Starting Rest Server...

wasmcli rest-server --laddr tcp://0.0.0.0:1317 --trust-node
