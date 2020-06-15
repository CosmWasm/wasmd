# docker build . -t cosmwasm/wasm:latest
# docker run --rm -it cosmwasm/wasm:latest /bin/sh
FROM cosmwasm/go-ext-builder:0.8.2-alpine AS builder

RUN apk add git
# without this, build with LEDGER_ENABLED=false
RUN apk add libusb-dev linux-headers

# copy all code into /code
WORKDIR /code
COPY . /code

# download all deps
RUN go mod download
# TODO: how to use this instead of hardcoding GO_COSMWASM
RUN basename $(ls -d /go/pkg/mod/github.com/\!cosm\!wasm/go-cosmwasm@v*)

ENV GO_COSMWASM="v0.8.2-0.20200615221537-0fc920db0349"

# build go-cosmwasm *.a and install it
WORKDIR /go/pkg/mod/github.com/\!cosm\!wasm/go-cosmwasm@${GO_COSMWASM}
RUN cargo build --release --features backtraces --example muslc
RUN mv /go/pkg/mod/github.com/\!cosm\!wasm/go-cosmwasm@${GO_COSMWASM}/target/release/examples/libmuslc.a /lib/libgo_cosmwasm_muslc.a
# I got errors from go mod verify (called from make build) if I didn't clean this up
RUN rm -rf /go/pkg/mod/github.com/\!cosm\!wasm/go-cosmwasm@${GO_COSMWASM}/target

# build the go wasm binary
WORKDIR /code

# force it to use static lib (from above) not standard libgo_cosmwasm.so file
RUN BUILD_TAGS=muslc make build

FROM alpine:3.12

COPY --from=builder /code/build/wasmd /usr/bin/wasmd
COPY --from=builder /code/build/wasmcli /usr/bin/wasmcli

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