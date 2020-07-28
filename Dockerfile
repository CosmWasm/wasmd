# docker build . -t cosmwasm/wasmd:latest
# docker run --rm -it cosmwasm/wasmd:latest /bin/sh
FROM cosmwasm/go-ext-builder:0.8.2-alpine AS rust-builder

RUN apk add git

# copy all code into /code
WORKDIR /code
COPY go.* /code/

# download all deps
RUN go mod download github.com/CosmWasm/go-cosmwasm

# build go-cosmwasm *.a and install it
RUN export GO_WASM_DIR=$(go list -f "{{ .Dir }}" -m github.com/CosmWasm/go-cosmwasm) && \
    cd ${GO_WASM_DIR} && \
    cargo build --release --features backtraces --example muslc && \
    mv ${GO_WASM_DIR}/target/release/examples/libmuslc.a /lib/libgo_cosmwasm_muslc.a


# --------------------------------------------------------
FROM cosmwasm/go-ext-builder:0.8.2-alpine AS go-builder

RUN apk add git
# NOTE: add these to run with LEDGER_ENABLED=true
# RUN apk add libusb-dev linux-headers

WORKDIR /code
COPY . /code/

COPY --from=rust-builder /lib/libgo_cosmwasm_muslc.a /lib/libgo_cosmwasm_muslc.a

# force it to use static lib (from above) not standard libgo_cosmwasm.so file
RUN LEDGER_ENABLED=false BUILD_TAGS=muslc make build
# we also (temporarily?) build the testnet binaries here
RUN LEDGER_ENABLED=false BUILD_TAGS=muslc make build-coral
RUN LEDGER_ENABLED=false BUILD_TAGS=muslc make build-gaiaflex

# --------------------------------------------------------
FROM alpine:3.12

COPY --from=go-builder /code/build/wasmd /usr/bin/wasmd
COPY --from=go-builder /code/build/wasmcli /usr/bin/wasmcli

# testnet
COPY --from=go-builder /code/build/coral /usr/bin/coral
COPY --from=go-builder /code/build/corald /usr/bin/corald
COPY --from=go-builder /code/build/gaiaflex /usr/bin/gaiaflex
COPY --from=go-builder /code/build/gaiaflexd /usr/bin/gaiaflexd

COPY docker/* /opt/
RUN chmod +x /opt/*.sh

WORKDIR /opt

# rest server
EXPOSE 1317
# tendermint p2p
EXPOSE 26656
# tendermint rpc
EXPOSE 26657

CMD ["/usr/bin/wasmd version"]