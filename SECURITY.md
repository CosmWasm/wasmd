# Security Policy

## Reporting a Vulnerability

Please report any security issues via email to security@confio.gmbh. 

You will receive a response from us within 2 working days. If the issue is confirmed, we will release a patch as soon as possible depending on complexity but historically within a few days.

Please avoid opening public issues on GitHub that contain information about a potential security vulnerability as this makes it difficult to reduce the impact and harm of valid security issues.

## Supported Versions

This is alpha software, do not run on a production system. Notably, we currently provide no migration path not even "dump state and restart" to move to future versions.

We will have a stable v0.x version before the final v1.0.0 version with the same API as the v1.0 version in order to run last testnets and manual testing on it. We have not yet committed to that version number. wasmd 0.22 will support Cosmos SDK 0.44/0.45 and should be quite close to a final API, minus some minor details.

Our v1.0.0 release plans were also delayed by upstream release cycles, and we have continued to refine APIs while we can.

## Coordinated Vulnerability Disclosure Policy

We ask security researchers to keep vulnerabilities and communications around vulnerability submissions private and confidential until a patch is developed. In addition to this, we ask that you:

 - Allow us a reasonable amount of time to correct or address security vulnerabilities.
 - Avoid exploiting any vulnerabilities that you discover.
 - Demonstrate good faith by not disrupting or degrading services built on top of this software.

## Vulnerability Disclosure Process

Confio uses the following disclosure process for the various CosmWasm-related repos:

 - Once a security report is received, the core development team works to verify the issue.
 - Patches are prepared for eligible releases in private repositories.
 - We notify the community that a security release is coming, to give users time to prepare their systems for the update. Notifications can include Discord messages, tweets, and emails to partners and validators. Please also see [CosmWasm/advisories](https://github.com/CosmWasm/advisories) if you want to receive notifications.
 - No less than 24 hours following this notification, the fixes are applied publicly and new releases are issued.
 - Once releases are available, we notify the community, again, through the same channels as above.
 - Once the patches have been properly rolled out, we will publish a post with further details on the vulnerability as well as our response to it.
 - Note that we are working on a concept for bug bounties and they are not currently available.

 This process can take some time. Every effort will be made to handle the bug as quickly and thoroughly as possible. However, it's important that we follow the process described above to ensure that disclosures are handled consistently and to keep this codebase and the projects that depend on them secure.