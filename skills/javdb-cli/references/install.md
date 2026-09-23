# Installation and Updates

Use only for an explicit installation, repair, or upgrade request. Confirm platform, architecture, target version/channel, and destination; installation changes files or package-manager state and must not be a side effect of a query.

For a published version, use the platform archive from the official `FlanChanXwO/javdb-cli` GitHub Releases and verify the published checksums before placing `javdb` (Windows: `javdb.exe`) in the user's selected PATH directory. A checksum establishes integrity relative to that downloaded checksum file; it is not an independent trust root. Do not substitute an unreviewed mirror or unsigned custom build.

When the user chooses Homebrew on macOS/Linux, use `brew install FlanChanXwO/tap/javdb-cli`. For an explicit source build in an existing checkout, use `sh scripts/build.sh` from its root and the Go version in `go.mod`. Missing prerequisites require approval before installation.

After success, run `javdb --version` and report the binary path/version and any PATH change. Do not inspect accounts as part of installation.

For an existing installation, `javdb update --check --json` is the read-only inspection route. Actual `javdb update` uses detected Homebrew, `go install`, or the Release archive. Release replacement validates the signed manifest, source/version/platform, archive checksum, and extracted binary hash; do not bypass those checks or execute a downloaded candidate manually to replace verification.

Use `--prerelease` only for an explicit prerelease request; Homebrew updates do not support it. Development builds refuse self-update. Report failures without switching installation channel or replacing an existing executable by hand.
