---
name: javdb-cli-media
description: Develop or review javdb-cli media assets, image decoding, HLS decryption, TS/MP4 remuxing, streaming probes, and no-replace file publication. Use for asset/download correctness, media corruption, memory regressions, or transport changes; not for full-movie or magnet downloading.
---

# Maintain javdb-cli Media

Read `AGENTS.md`, `internal/cli/commands/assets`, the public asset SDK methods, and the affected files in `internal/javdb/appapi/media`. Trace the real path before choosing a fix. Keep CLI selection/presentation separate from SDK/media transfer and validation.

## Preserve the contract

- `assets list` applies selectors before optional streaming metadata probes. Probe failures can leave optional width/height/duration absent; do not confuse that documented result with a successfully downloaded asset.
- Piped list output is `TYPE<TAB>URL`, not `javdb.pipeline/v1`. `assets download` consumes that stream; `-d` generates names, `-o` accepts exactly one asset, and successful stdout lists final paths only.
- Images must be decoded/validated after any required XOR restoration. HLS processing owns playlist, key, IV, padding, segment, media-model, and final-container integrity. Never publish encrypted, truncated, or partially verified bytes as success.
- Preserve the supported MP4 H.264/AAC-LC, PID, ADTS-block, and mono/stereo constraints in the current parser and tests. TS-preservation mode has a different contract; do not impose MP4-only restrictions on it. Timed ID3 is not an MP4 audio track.
- Keep no-replace atomic publication and cleanup: existing targets survive, partial files remain private and are removed on failure, and cancellation is distinguishable from corruption. Cross-filesystem and permission errors must remain visible.
- Spooling reduces retained payload, but the implementation still keeps the current PES and sample metadata. Do not claim constant memory without measurement, or add buffering limits based on a guessed file size.

## Test before changing behavior

Create the smallest complete synthetic fixture that reproduces the defect; run it and observe the expected failure. Use existing media test builders rather than downloaded production media or a new codec dependency. Then implement the narrow correction and run related media, SDK, and CLI asset tests.

Depending on the change, cover valid and corrupt image input, playlist/key/IV handling, missing or malformed segments, timestamp normalization, sample boundaries, repeated track changes, cancellation, short writes, existing targets, and cleanup. Select cases from the actual failure/contract, not every imagined format combination. Assert produced media semantics and status, not exact internal function order.

For transport/probe/concurrency changes, run the relevant race checks and show request count/order or measured memory evidence where that is the requirement. A local remux fixture is not proof of a live upstream preview; live requests require separate authorization and must not leak signed URLs or keys.

## Finish

Use [javdb-cli-test](../javdb-cli-test/SKILL.md) and [javdb-cli-review](../javdb-cli-review/SKILL.md), then update the affected SDK/CLI documentation and product skill. Keep the supported thumbnail/preview scope explicit. Do not add transcoding, ffmpeg, full-movie downloading, or a new background service as an incidental fix.
