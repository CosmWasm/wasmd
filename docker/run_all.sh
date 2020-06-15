#!/bin/sh
#set -euo pipefail

mkdir -p /root/log
touch /root/log/wasmd.log
./run_wasmd.sh $1 >> /root/log/wasmd.log &

sleep 4
echo Starting Rest Server...

./run_rest_server.sh
