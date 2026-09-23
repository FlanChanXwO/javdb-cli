# Errors, Network, and Routing

Inspect exit status and actual stderr before parsing output. An authentication, network, validation, or filesystem failure is not an empty successful result.

- **Missing binary:** report it; use [install.md](install.md) only for an explicit install/repair request.
- **Authentication:** diagnose with `auth check --json` when needed. Use [auth.md](auth.md) for requested login; do not read `auth.json` or enable auto-relogin automatically.
- **Host routing:** CLI `host` defaults to `auto`; it validates/reuses a cached route or explicitly discovers a replacement. Fixed `main`, `mirror`, or a custom URL changes the destination. Do not confuse this with the SDK's mirror default or change it without the requested scope.
- **Proxy:** use `--proxy URL` only for the supplied/authorized proxy on this invocation. Persisting `https_proxy` is a separate configuration write. Do not assume a developer's loopback proxy exists on the execution host, and do not replace a rejected blank proxy with a silent direct connection.
- **Optional magnets auth:** magnets and `detail --magnets` can retry anonymously after a rejected optional token. Authenticated lists/marks do not inherit that contract; report their actual failure.
- **API/status/format failures:** report the redacted cause. Do not invent retry counts, request caps, browser scraping, or a silent source fallback.
- **Output parsing:** failed commands need not produce JSON. Default piped text and `--ndjson` are different protocols; match the consumer and retain per-item failures and final status.
- **Image search:** upload requires an authorized source; cache hits do not prove current upstream availability. A partial candidate failure can coexist with valid output and a nonzero exit.
- **Assets:** see [media.md](media.md) for optional probe metadata versus verified download success. Existing targets and unsupported media must fail visibly, not be overwritten or disguised as empty files.
- **Updates:** use the read-only check first; a signature/hash/source failure blocks replacement. Do not reroute data-host configuration or manually execute an unverified archive as a workaround.

Keep credentials, source images, signed/private URLs, account files, and raw responses out of diagnostic artifacts. Report what remains unverified rather than claiming a repair after changing unrelated configuration.
