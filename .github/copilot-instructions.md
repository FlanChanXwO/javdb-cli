# javdb-cli contributor notes

Read `AGENTS.md` first. Keep `cmd/javdb` thin, route remote operations through
the public `javdb` facade, and keep protocol code under `internal/javdb/`.
Preserve CLI/JSON compatibility, never expose credentials, update focused tests
and the routed documentation for behavior changes.
