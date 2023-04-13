# docker build . -t cosmwasm/wasmd:latest
# docker run --rm -it cosmwasm/wasmd:latest /bin/sh
FROM golang:1.19-alpine3.15 AS go-builder
ARG arch=x86_64

# this comes from standard alpine nightly file
#  https://github.com/rust-lang/docker-rust-nightly/blob/master/alpine3.12/Dockerfile
# with some changes to support our toolchain, etc
RUN set -eux; apk add --no-cache ca-certificates build-base;

RUN apk add git
# NOTE: add these to run with LEDGER_ENABLED=true
# RUN apk add libusb-dev linux-headers

WORKDIR /code
COPY . /code/
# See https://github.com/CosmWasm/wasmvm/releases
ADD https://github.com/CosmWasm/wasmvm/releases/download/v1.2.1/libwasmvm_muslc.aarch64.a /lib/libwasmvm_muslc.aarch64.a
ADD https://github.com/CosmWasm/wasmvm/releases/download/v1.2.1/libwasmvm_muslc.x86_64.a /lib/libwasmvm_muslc.x86_64.a
RUN sha256sum /lib/libwasmvm_muslc.aarch64.a | grep 86bc5fdc0f01201481c36e17cd3dfed6e9650d22e1c5c8983a5b78c231789ee0
RUN sha256sum /lib/libwasmvm_muslc.x86_64.a | grep a00700aa19f5bfe0f46290ddf69bf51eb03a6dfcd88b905e1081af2e42dbbafc

# Copy the library you want to the final location that will be found by the linker flag `-lwasmvm_muslc`
RUN cp /lib/libwasmvm_muslc.${arch}.a /lib/libwasmvm_muslc.a

# force it to use static lib (from above) not standard libgo_cosmwasm.so file
RUN LEDGER_ENABLED=false BUILD_TAGS=muslc LINK_STATICALLY=true make build
RUN echo "Ensuring binary is statically linked ..." \
  && (file /code/build/wasmd | grep "statically linked")

# --------------------------------------------------------
FROM alpine:3.15

COPY --from=go-builder /code/build/wasmd /usr/bin/wasmd

COPY docker/* /opt/
RUN chmod +x /opt/*.sh

WORKDIR /opt

# rest server
EXPOSE 1317
# tendermint p2p
EXPOSE 26656
# tendermint rpc
EXPOSE 26657

CMD ["/usr/bin/wasmd", "version"]