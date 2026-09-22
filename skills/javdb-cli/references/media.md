# Preview Media Assets

Use this workflow only for an explicit request to inspect or save supported thumbnail/preview assets. It does not download full movies or magnet targets.

## Select before writing

1. Confirm the movie reference and requested destination. `javdb assets list ABC-123` lists thumbnail, cover, preview images, then preview videos; indices identify positions after filtering, not persistent IDs.
2. Use `--type image|video` or selectors such as `1-4` and `1,3-5` for the requested subset. Inspect before feeding a write operation; an already explicit target/destination need not be asked again.
3. Pipe the selected `TYPE<TAB>URL` stream into `javdb assets download`. `-d DIR` generates names such as `image-001.jpg` and `video-001.mp4`; `-o PATH` requires exactly one asset. Existing targets are never replaced.

```text
javdb assets list ABC-123 --type image 1-2 | javdb assets download -d ./images
javdb assets list ABC-123 --type video | javdb assets download -o ./preview.mp4
```

Use the second example only after confirming exactly one selected video. Do not substitute raw `preview_images` parsing, handwritten Referer headers, or an ffmpeg pipeline for the supported command.

## Metadata and outcomes

After selection, list performs bounded streaming probes. JSON/NDJSON can include optional `width`, `height`, and integer-second video `duration`; TTY output shows size/duration, while default pipe output stays exactly `TYPE<TAB>URL`. An individual probe failure omits optional metadata; it does not prove an asset is downloadable.

The current `assets.probe.concurrency` default is 4 and must be positive. `assets.probe.enabled=false` disables extra media probes; changing either is a persistent configuration action requiring authorization.

Successful `assets download` stdout contains final file paths, one per line, without decorative text. Check exit status and the reported files. Never present encrypted image bytes, incomplete media, or a partially failed batch as a completed download.

## Format boundaries

MP4 remuxing supports one H.264 PID and at most one AAC-LC PID per segment, one ADTS raw-data block per frame, and channel configurations 1 (mono) or 2 (stereo). Other tracks/profiles/blocks/channels fail explicitly; timed ID3 is not written into MP4. TS publication validates media structure and video timestamps while preserving ADTS and does not inherit the MP4-only AAC restrictions.

Do not add a timeout or alternate encoder just because a legitimate download takes time. Preserve cancellation and real transport/format errors. Attach resulting files only through an available attachment mechanism; a local path alone is not proof the user received a file.
