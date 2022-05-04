# mAuth module

This is a simple module that allows any external account to create a new Interchain Account
controlled directly from their private key.

It was copied from [gaia x/mauth](https://github.com/cosmos/gaia/tree/ica-acct-auth/x/mauth) at commit [7fd4255c](https://github.com/cosmos/gaia/commit/7fd4255c8d1afd9b1e133380febda88911dc03e4)

It provides a minimal usable auth setup. We do not want to add gaia (especially a branch) as a dependency. But if you wish to use this in an independent application (your own blockchain), I suggest you check for the most recent version in the gaia repo, or check with the ibc-go team.