# Changelog

## [Unreleased](https://github.com/CosmWasm/wasmd/tree/HEAD)

[Full Changelog](https://github.com/CosmWasm/wasmd/compare/v0.14.0...HEAD)

## [v0.14.0](https://github.com/CosmWasm/wasmd/tree/v0.14.0) (2021-01-11)

[Full Changelog](https://github.com/CosmWasm/wasmd/compare/v0.13.0...v0.14.0)

**Features:**
- Upgrade to cosmos-sdk v0.40.0 final [\#354](https://github.com/CosmWasm/wasmd/pull/369)
- Refactor to GRPC message server [\#366](https://github.com/CosmWasm/wasmd/pull/366)
- Make it easy to initialize contracts in genesis file with new CLI commands[\#326](https://github.com/CosmWasm/wasmd/issues/326)
- Upgrade to WasmVM v0.13.0 [\#358](https://github.com/CosmWasm/wasmd/pull/358)
- Upgrade to cosmos-sdk v0.40.0-rc6 [\#354](https://github.com/CosmWasm/wasmd/pull/354)
- Upgrade to cosmos-sdk v0.40.0-rc5 [\#344](https://github.com/CosmWasm/wasmd/issues/344)
- Add Dependabot to keep dependencies secure and up-to-date [\#336](https://github.com/CosmWasm/wasmd/issues/336)

**Fixed bugs:**

- Dependabot can't resolve your Go dependency files [\#339](https://github.com/CosmWasm/wasmd/issues/339)
- Errors in `InitGenesis` [\#335](https://github.com/CosmWasm/wasmd/issues/335)
- Invalid homeDir for export command [\#334](https://github.com/CosmWasm/wasmd/issues/334)

## [v0.13.0](https://github.com/CosmWasm/wasmd/tree/v0.13.0) (2020-12-04)

[Full Changelog](https://github.com/CosmWasm/wasmd/compare/v0.12.1...v0.13.0)

**Fixed bugs:**

- REST handler wrong `Sender` source [\#324](https://github.com/CosmWasm/wasmd/issues/324)

**Closed issues:**

- Change proto package to match \<organisation\>.\<module\>.\<version\> [\#329](https://github.com/CosmWasm/wasmd/issues/329)
- Out of gas causes panic when external contract store query executed [\#321](https://github.com/CosmWasm/wasmd/issues/321)
- Check codecov report [\#298](https://github.com/CosmWasm/wasmd/issues/298)
- cosmwasm.GoAPI will not work on sdk.ValAddress [\#264](https://github.com/CosmWasm/wasmd/issues/264)
- Stargate: Add pagination support for queries [\#242](https://github.com/CosmWasm/wasmd/issues/242)

**Merged pull requests:**

- Rename protobuf package [\#330](https://github.com/CosmWasm/wasmd/pull/330) ([alpe](https://github.com/alpe))
- Use base request data for sender [\#325](https://github.com/CosmWasm/wasmd/pull/325) ([alpe](https://github.com/alpe))
- Handle panics in query contract smart [\#322](https://github.com/CosmWasm/wasmd/pull/322) ([alpe](https://github.com/alpe))

## [v0.12.1](https://github.com/CosmWasm/wasmd/tree/v0.12.1) (2020-11-23)

[Full Changelog](https://github.com/CosmWasm/wasmd/compare/v0.12.0...v0.12.1)

**Closed issues:**

- Complete IBC Mock testing [\#255](https://github.com/CosmWasm/wasmd/issues/255)
- Idea: do multiple queries in one API call [\#72](https://github.com/CosmWasm/wasmd/issues/72)

**Merged pull requests:**

- Exclude generate proto code files in coverage [\#320](https://github.com/CosmWasm/wasmd/pull/320) ([alpe](https://github.com/alpe))
- Upgrade wasmvm to 0.12.0 [\#319](https://github.com/CosmWasm/wasmd/pull/319) ([webmaster128](https://github.com/webmaster128))
- Fix chain id setup in contrib/local/setup\_wasmd.sh [\#318](https://github.com/CosmWasm/wasmd/pull/318) ([orkunkl](https://github.com/orkunkl))
- Add pagination to grpc queries [\#317](https://github.com/CosmWasm/wasmd/pull/317) ([alpe](https://github.com/alpe))

## [v0.12.0](https://github.com/CosmWasm/wasmd/tree/v0.12.0) (2020-11-17)

[Full Changelog](https://github.com/CosmWasm/wasmd/compare/v0.12.0-alpha1...v0.12.0)

**Closed issues:**

- Merge wasmd and wasmcli into a single binary [\#308](https://github.com/CosmWasm/wasmd/issues/308)
- Change bech32 prefix for wasmd [\#313](https://github.com/CosmWasm/wasmd/issues/313)
- Upgrade go-cowasmwasm to wasmvm 0.12 [\#309](https://github.com/CosmWasm/wasmd/issues/309)
- Use string type for AccAddresses in proto  [\#306](https://github.com/CosmWasm/wasmd/issues/306)
- Upgrade to cosmos/sdk v0.40.0-rc2 [\#296](https://github.com/CosmWasm/wasmd/issues/296)
- Generate protobuf outputs in a container [\#295](https://github.com/CosmWasm/wasmd/issues/295)
- Instantiate contract process ordering [\#292](https://github.com/CosmWasm/wasmd/issues/292)
- Make Wasm maxSize a configuration option [\#289](https://github.com/CosmWasm/wasmd/issues/289)
- Return error if wasm to big [\#287](https://github.com/CosmWasm/wasmd/issues/287)

**Merged pull requests:**

- Set bech32 prefix [\#316](https://github.com/CosmWasm/wasmd/pull/316) ([alpe](https://github.com/alpe))
- Replace sdk.AccAddress with bech32 string [\#314](https://github.com/CosmWasm/wasmd/pull/314) ([alpe](https://github.com/alpe))
- Integrate wasmcli into wasmd [\#312](https://github.com/CosmWasm/wasmd/pull/312) ([alpe](https://github.com/alpe))
- Upgrade wasmvm aka go-cosmwasm [\#311](https://github.com/CosmWasm/wasmd/pull/311) ([alpe](https://github.com/alpe))
- Upgrade to Stargate RC3 [\#305](https://github.com/CosmWasm/wasmd/pull/305) ([alpe](https://github.com/alpe))
- Containerized Protobuf generation  [\#304](https://github.com/CosmWasm/wasmd/pull/304) ([alpe](https://github.com/alpe))
- Reject wasm code exceeding limit  [\#302](https://github.com/CosmWasm/wasmd/pull/302) ([alpe](https://github.com/alpe))
- Support self calling contract on instantiation [\#300](https://github.com/CosmWasm/wasmd/pull/300) ([alpe](https://github.com/alpe))
- Upgrade to Stargate RC2 [\#299](https://github.com/CosmWasm/wasmd/pull/299) ([alpe](https://github.com/alpe))
