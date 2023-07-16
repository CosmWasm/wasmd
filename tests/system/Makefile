#!/usr/bin/make -f

WAIT_TIME ?= 45s

all: test

test:
	go test -mod=readonly -failfast -tags='system_test' ./... --wait-time=$(WAIT_TIME) --verbose

format:
	find . -name '*.go' -type f -not -path "./vendor*" -not -path "*.git*" -not -path "./client/lcd/statik/statik.go" | xargs gofumpt -w
	find . -name '*.go' -type f -not -path "./vendor*" -not -path "*.git*" -not -path "./client/lcd/statik/statik.go" | xargs misspell -w
	find . -name '*.go' -type f -not -path "./vendor*" -not -path "*.git*" -not -path "./client/lcd/statik/statik.go" | xargs goimports -w -local github.com/CosmWasm/wasmd


.PHONY: all test format
