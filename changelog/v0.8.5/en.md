# v0.8.5 — 2026-11-20

## Added

- Select stderr diagnostics with `--log-format text|json` and `--log-level debug|info|warning|error`, or the matching `log_format` / `log_level` configuration. Diagnostics include time, command, source, and one error per failed input. ([#111](https://github.com/FlanChanXwO/javdb-cli/pull/111))
- Help, prompts, and status/error messages follow the system locale in English or Simplified Chinese; API `lang` keeps its existing meaning. Command syntax, JSON/NDJSON fields, and the asset pipe format stay stable. ([#111](https://github.com/FlanChanXwO/javdb-cli/pull/111))

## Changed

- `auth check` verifies the saved token with the server and exits 1 when it is rejected, including with `--json`. Scripts should check the exit status and JSON result. ([#111](https://github.com/FlanChanXwO/javdb-cli/pull/111))

## Fixed

- Actor, series, maker, director, code, list, browse, tags, and collection queries consistently use the saved default account. This fixes routes that still rejected requests after signing in. ([#111](https://github.com/FlanChanXwO/javdb-cli/pull/111))
- Missing or rejected credentials produce actionable `javdb auth login` guidance in regular and piped commands, with the underlying error retained. Explicit `auto_relogin=true` shares one sign-in attempt per invocation and saves the refreshed token before retrying the failed operation. ([#111](https://github.com/FlanChanXwO/javdb-cli/pull/111))
- Align Chinese asset table headings with their data columns, translate grouped help headings and default-value annotations, and use consistent bilingual command terminology. ([#111](https://github.com/FlanChanXwO/javdb-cli/pull/111))

**Full Changelog**: [v0.8.4...v0.8.5](https://github.com/FlanChanXwO/javdb-cli/compare/v0.8.4...v0.8.5)

<!-- internal: https://github.com/FlanChanXwO/javdb-cli/pull/109 -->
