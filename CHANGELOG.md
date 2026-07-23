# Changelog

[English](CHANGELOG.md) | [简体中文](CHANGELOG.zh-CN.md)

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.2.0] - 2026-07-24

### Added

- `javdb update` now checks for and installs newer GitHub Releases while
  preserving the Homebrew, `go install`, or Release-archive installation
  channel. The read-only `--check --json` form supports automation.
- Release-archive updates select the running OS/architecture, verify the
  published SHA-256, preflight `javdb version --json`, and replace the binary
  only after those checks pass.

## [0.1.1] - 2026-07-23

### Added

- `skills/javdb-cli`: an agent operator skill with credential, state-change,
  discovery, installation, and troubleshooting guidance.
- Native packaged-binary smoke coverage for macOS (Intel/Apple Silicon), Linux
  (amd64/arm64), and Windows (amd64/arm64).

### Changed

- Release builds now use explicit single-target build/package scripts, rebuild
  immutable release tags on fresh native runners, verify the asset set before
  publishing, and validate the generated Homebrew formula before optional tap
  deployment.

### Fixed

- Credential-store permission tests now account for Windows not exposing POSIX
  mode bits, while retaining the `0600` assertion on supported platforms.
- Release checks preserve LF source bytes on Windows so Git checkout conversion
  cannot be mistaken for an unformatted Go source file.
- Production release builds now directly depend on validation, so their exact
  version input is available on every native runner.

## [0.1.0] - 2026-07-18

First public release.

### Added

- Full CLI surface: auth, search, detail, magnets, tags, browse, entity filmography, user lists, rankings, top250, lists (合集).
- Multi-account password login with local `auth.json` and optional `auto_relogin`.
- Public Go SDK package `javdb`.
- `javdb version --json` for Homebrew formula tests.
- Bilingual README / CONTRIBUTING / docs.
- CI quality gate and multi-arch GitHub Release workflow.
- Homebrew formula for `FlanChanXwO/tap/javdb-cli`.
