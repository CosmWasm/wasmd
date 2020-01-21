# Simple usage with a mounted data directory:
# > docker build -t gaia .
# > docker run -it -p 46657:46657 -p 46656:46656 -v ~/.wasmd:/root/.wasmd -v ~/.wasmcli:/root/.wasmcli gaia wasmd init
# > docker run -it -p 46657:46657 -p 46656:46656 -v ~/.wasmd:/root/.wasmd -v ~/.wasmcli:/root/.wasmcli gaia wasmd start
FROM golang:1.13-buster AS build-env

# Set up dependencies
ENV PACKAGES curl git build-essential

# Set working directory for the build
WORKDIR /go/src/github.com/cosmwasm/wasmd

# Add source files
COPY . .

# Install minimum necessary dependencies, build Cosmos SDK, remove packages
RUN apt update
RUN apt install -y $PACKAGES
RUN make tools
RUN make install

# Final image
FROM debian:buster

# Install ca-certificates
# RUN apk add --update ca-certificates
WORKDIR /root

# Copy over binaries from the build-env
COPY --from=build-env /go/bin/wasmd /usr/bin/wasmd
COPY --from=build-env /go/bin/wasmcli /usr/bin/wasmcli

# Run wasmd by default, omit entrypoint to ease using container with wasmcli
CMD ["wasmd"]
