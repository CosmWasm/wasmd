# Simple usage with a mounted data directory:
# > docker build -t gaia .
# > docker run -it -p 46657:46657 -p 46656:46656 -v ~/.wasmd:/root/.wasmd -v ~/.wasmcli:/root/.wasmcli gaia wasmd init
# > docker run -it -p 46657:46657 -p 46656:46656 -v ~/.wasmd:/root/.wasmd -v ~/.wasmcli:/root/.wasmcli gaia wasmd start
FROM golang:1.13-buster AS build-env

# Install minimum necessary dependencies, build Cosmos SDK, remove packages
RUN apt update
RUN apt install -y curl git build-essential

# Set working directory for the build
WORKDIR /go/src/github.com/cosmwasm/wasmd

# Add source files
COPY . .
#
RUN make tools
RUN make install

COPY docker/* /opt/
RUN chmod +x /opt/*.sh

WORKDIR /opt

# rest server
EXPOSE 1317
# tendermint p2p
EXPOSE 26656
# tendermint rpc
EXPOSE 26657

CMD ["wasmd"]
