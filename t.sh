#!/usr/bin/env bash

for pkg in $(go list ./... ); do
  echo "$pkg"
  go test "$pkg"
done