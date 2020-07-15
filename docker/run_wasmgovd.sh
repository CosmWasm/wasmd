#!/bin/sh

if test -n "$1"; then
    # need -R not -r to copy hidden files
    cp -R "$1/.wasmd" /root
    cp -R "$1/.wasmcli" /root
fi

mkdir -p /root/log
wasmgovd start --rpc.laddr tcp://0.0.0.0:26657 --trace
