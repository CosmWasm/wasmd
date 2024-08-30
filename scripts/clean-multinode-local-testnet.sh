#!/bin/bash
set -ux

# always returns true so set -e doesn't exit if it is not running.
killall oraid || true
rm -rf $HOME/.oraid/
killall screen