# docker build . -t finschia/wasmd:latest
# docker run --rm -it finschia/wasmd:latest /bin/sh
FROM golang:1.18-alpine3.15 AS go-builder
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

# See https://github.com/Finschia/wasmvm/releases
ADD https://github.com/Finschia/wasmvm/releases/download/v1.0.0-0.10.0/libwasmvm_static.x86_64.a /lib/libwasmvm_static.x86_64.a
ADD https://github.com/Finschia/wasmvm/releases/download/v1.0.0-0.10.0/libwasmvm_static.aarch64.a /lib/libwasmvm_static.aarch64.a
RUN sha256sum /lib/libwasmvm_static.aarch64.a | grep bc3db72ba32f34ad88ceb1d20479411bd7f50ccd6a5ca50cc8ca462a561e6189
RUN sha256sum /lib/libwasmvm_static.x86_64.a | grep 352fa5de5f9dba66f0a38082541d3e63e21394fee3e577ea35e0906294c61276

# Copy the library you want to the final location that will be found by the linker flag `-lwasmvm_muslc`
RUN cp /lib/libwasmvm_static.${arch}.a /lib/libwasmvm_static.a

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