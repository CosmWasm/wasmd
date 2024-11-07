# docker build . -t cosmwasm/wasmd:latest
# docker run --rm -it cosmwasm/wasmd:latest /bin/sh

# Using Alpine 3.19+ as the build system is currently broken,
# see https://github.com/CosmWasm/wasmvm/issues/523
FROM golang:1.21-alpine3.18 AS go-builder

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
ADD https://github.com/CosmWasm/wasmvm/releases/download/v2.2.0-rc.2/libwasmvm_muslc.aarch64.a /lib/libwasmvm_muslc.aarch64.a
ADD https://github.com/CosmWasm/wasmvm/releases/download/v2.2.0-rc.2/libwasmvm_muslc.x86_64.a /lib/libwasmvm_muslc.x86_64.a
RUN sha256sum /lib/libwasmvm_muslc.aarch64.a | grep 7af80eb7e79d82789eca0d5512a87dc20e96182590fe88ae5fd0153e31c097c9
RUN sha256sum /lib/libwasmvm_muslc.x86_64.a | grep 2c497b5246c7217a70c447c50117c7fb09909ec23e6e4151a4de3e5f29db8134

# force it to use static lib (from above) not standard libgo_cosmwasm.so file
RUN LEDGER_ENABLED=false BUILD_TAGS=muslc LINK_STATICALLY=true make build
RUN echo "Ensuring binary is statically linked ..." \
  && (file /code/build/wasmd | grep "statically linked")

# --------------------------------------------------------
FROM alpine:3.18

COPY --from=go-builder /code/build/wasmd /usr/bin/layerd

COPY docker/* /opt/
RUN chmod +x /opt/*.sh

WORKDIR /opt

# rest server
EXPOSE 1317
# grpc server
EXPOSE 9090
# grpc-web server
EXPOSE 9091
# tendermint p2p
EXPOSE 26656
# tendermint rpc
EXPOSE 26657

ENTRYPOINT ["/usr/bin/layerd"]
CMD ["start"]