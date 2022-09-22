package keeper

import (
	"bytes"
	"encoding/hex"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/CosmWasm/wasmd/x/wasm/types"
)

func TestBuildContractAddress(t *testing.T) {
	x, y := sdk.GetConfig().GetBech32AccountAddrPrefix(), sdk.GetConfig().GetBech32AccountPubPrefix()
	t.Cleanup(func() {
		sdk.GetConfig().SetBech32PrefixForAccount(x, y)
	})
	sdk.GetConfig().SetBech32PrefixForAccount("purple", "purple")

	// test vectors from https://gist.github.com/webmaster128/e4d401d414bd0e7e6f70482f11877fbe
	specs := map[string]struct {
		checksum   []byte
		salt       []byte
		creator    string
		msg        []byte
		expAddress string
	}{
		"initial account addr example": {
			checksum:   fromHex("13a1fc994cc6d1c81b746ee0c0ff6f90043875e0bf1d9be6b7d779fc978dc2a5"),
			creator:    "purple1nxvenxve42424242hwamhwamenxvenxvhxf2py",
			salt:       fromHex("13a1fc994cc6d1c81b746ee0c0ff6f90043875e0bf1d9be6b7d779fc978dc2a5"),
			expAddress: "purple1t6r960j945lfv8mhl4mage2rg97w63xeynwrupum2s2l7em4lprs9ce5hk",
		},
		"contract addr with different msg": {
			checksum:   fromHex("13a1fc994cc6d1c81b746ee0c0ff6f90043875e0bf1d9be6b7d779fc978dc2a5"),
			creator:    "purple1nxvenxve42424242hwamhwamenxvenxvhxf2py",
			salt:       fromHex("13a1fc994cc6d1c81b746ee0c0ff6f90043875e0bf1d9be6b7d779fc978dc2a5"),
			msg:        []byte(`{}`),
			expAddress: "purple1px25n9sgj3a99q0zcl4awx7my6s6mxqegmdd2lmvf5lwxh080q6suttktr",
		},
		"account addr with different salt": {
			checksum:   fromHex("13A1FC994CC6D1C81B746EE0C0FF6F90043875E0BF1D9BE6B7D779FC978DC2A5"),
			salt:       []byte("instance 2"),
			creator:    "purple1nxvenxve42424242hwamhwamenxvenxvhxf2py",
			expAddress: "purple1jpc2w2vu2t6k7u09p6v8e3w28ptnzvvt2snu5rytlj8qya27dq5sjkwm06",
		},
		"initial contract addr example": {
			checksum:   fromHex("13A1FC994CC6D1C81B746EE0C0FF6F90043875E0BF1D9BE6B7D779FC978DC2A5"),
			salt:       fromHex("13a1fc994cc6d1c81b746ee0c0ff6f90043875e0bf1d9be6b7d779fc978dc2a5"),
			creator:    "purple1nxvenxve42424242hwamhwamenxvenxvmhwamhwaamhwamhwlllsatsy6m",
			expAddress: "purple1wde5zlcveuh79w37w5g244qu9xja3hgfulyfkjvkxvwvjzgd5l3qfraz3c",
		},
		"contract addr with different salt": {
			checksum:   fromHex("13A1FC994CC6D1C81B746EE0C0FF6F90043875E0BF1D9BE6B7D779FC978DC2A5"),
			salt:       []byte("instance 2"),
			creator:    "purple1nxvenxve42424242hwamhwamenxvenxvmhwamhwaamhwamhwlllsatsy6m",
			expAddress: "purple1vae2kf0r3ehtq5q2jmfkg7wp4ckxwrw8dv4pvazz5nlzzu05lxzq878fa9",
		},
		"account addr with different checksum": {
			checksum:   fromHex("1DA6C16DE2CBAF7AD8CBB66F0925BA33F5C278CB2491762D04658C1480EA229B"),
			salt:       fromHex("13a1fc994cc6d1c81b746ee0c0ff6f90043875e0bf1d9be6b7d779fc978dc2a5"),
			creator:    "purple1nxvenxve42424242hwamhwamenxvenxvhxf2py",
			expAddress: "purple1gmgvt9levtn52mpfal3gl5tv60f47zez3wgczrh5c9352sfm9crs47zt0k",
		},
		"account addr with different checksum and salt": {
			checksum:   fromHex("1DA6C16DE2CBAF7AD8CBB66F0925BA33F5C278CB2491762D04658C1480EA229B"),
			salt:       []byte("instance 2"),
			creator:    "purple1nxvenxve42424242hwamhwamenxvenxvhxf2py",
			expAddress: "purple13jrcqxknt05rhdxmegjzjel666yay6fj3xvfp6445k7a9q2km4wqa7ss34",
		},
		"contract addr with different checksum": {
			checksum:   fromHex("1DA6C16DE2CBAF7AD8CBB66F0925BA33F5C278CB2491762D04658C1480EA229B"),
			salt:       fromHex("13a1fc994cc6d1c81b746ee0c0ff6f90043875e0bf1d9be6b7d779fc978dc2a5"),
			creator:    "purple1nxvenxve42424242hwamhwamenxvenxvmhwamhwaamhwamhwlllsatsy6m",
			expAddress: "purple1lu0lf6wmqeuwtrx93ptzvf4l0dyyz2vz6s8h5y9cj42fvhsmracq49pww9",
		},
		"contract addr with different checksum and salt": {
			checksum:   fromHex("1DA6C16DE2CBAF7AD8CBB66F0925BA33F5C278CB2491762D04658C1480EA229B"),
			salt:       []byte("instance 2"),
			creator:    "purple1nxvenxve42424242hwamhwamenxvenxvmhwamhwaamhwamhwlllsatsy6m",
			expAddress: "purple1zmerc8a9ml2au29rq3knuu35fktef3akceurckr6pf370n0wku7sw3c9mj",
		},
		// regression tests
		"min salt size": {
			checksum:   fromHex("13A1FC994CC6D1C81B746EE0C0FF6F90043875E0BF1D9BE6B7D779FC978DC2A5"),
			salt:       []byte("x"),
			creator:    "purple1nxvenxve42424242hwamhwamenxvenxvhxf2py",
			expAddress: "purple16pc8gt824lmp3dh2sxvttj0ykcs02n5p3ldswhv3j7y853gghlfq7mqeh9",
		},
		"max salt size": {
			checksum:   fromHex("13A1FC994CC6D1C81B746EE0C0FF6F90043875E0BF1D9BE6B7D779FC978DC2A5"),
			salt:       bytes.Repeat([]byte("x"), types.MaxSaltSize),
			creator:    "purple1nxvenxve42424242hwamhwamenxvenxvhxf2py",
			expAddress: "purple1hvau0pl7xedxhjjn0gpc86m7d8s9l9cslhyfd2k4c25k075xhugqr6jnr6",
		},
	}
	for name, spec := range specs {
		t.Run(name, func(t *testing.T) {
			creatorAddr, err := sdk.AccAddressFromBech32(spec.creator)
			require.NoError(t, err)

			// when
			gotAddr := BuildContractAddressPredictable(spec.checksum, creatorAddr, spec.salt, spec.msg)

			require.Equal(t, spec.expAddress, gotAddr.String())
			require.NoError(t, sdk.VerifyAddressFormat(gotAddr))
		})
	}
}

func fromHex(s string) []byte {
	r, err := hex.DecodeString(s)
	if err != nil {
		panic(err)
	}
	return r
}
