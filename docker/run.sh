#!/bin/bash

if test -n "$1"; then
    # need -R not -r to copy hidden files
    cp -R "$1/.wasmd" /root
    cp -R "$1/.wasmcli" /root
fi

echo Starting Wasmd...

mkdir -p /root/log
wasmd start --rpc.laddr tcp://0.0.0.0:26657 >> /root/log/wasmd.log &

sleep 4
echo Starting Rest Server...

wasmcli rest-server --laddr tcp://0.0.0.0:1317 --trust-node
