# docker build . -t cosmwasm/wasmd:latest
# docker run --rm -it cosmwasm/wasmd:latest /bin/sh
FROM cosmwasm/go-ext-builder:0.8.2-alpine AS rust-builder

RUN apk add git
# without this, build with LEDGER_ENABLED=false
RUN apk add libusb-dev linux-headers

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
# without this, build with LEDGER_ENABLED=false
RUN apk add libusb-dev linux-headers

WORKDIR /code
COPY . /code/

COPY --from=rust-builder /lib/libgo_cosmwasm_muslc.a /lib/libgo_cosmwasm_muslc.a

# force it to use static lib (from above) not standard libgo_cosmwasm.so file
RUN BUILD_TAGS=muslc make build

# --------------------------------------------------------
FROM alpine:3.12

COPY --from=go-builder /code/build/wasmd /usr/bin/wasmd
COPY --from=go-builder /code/build/wasmgovd /usr/bin/wasmgovd
COPY --from=go-builder /code/build/wasmcli /usr/bin/wasmcli

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