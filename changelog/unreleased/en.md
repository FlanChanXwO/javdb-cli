# Unreleased

> Optional manual drafting area. The release workflow does not read this file; move finalized bilingual notes
> into the target `changelog/vX.Y.Z/` directory before creating the release-prep PR.

- Machine-contract migration: lists/collections NDJSON moves from aggregate payloads to per-record
  envelopes. Old fields were `data.lists` and `data.items`; new records use `data.list` and
  `data.entity`. Legacy human output and explicit aggregate `--json` retain their existing
  output shapes. Stable list/entity IDs are now required, and records without one fail explicitly.
