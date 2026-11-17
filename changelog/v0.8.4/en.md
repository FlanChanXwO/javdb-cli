# v0.8.4 — 2026-11-17

## Fixed

- Search (including image search and integrated magnet search) and movie, actor, and playback rankings now use the saved default account token, fixing authentication failures on affected JavDB routes after login. Requests remain anonymous when no token is available; image downloads and reverse-search providers receive no JavDB token. Authentication errors remain visible without automatic re-login or anonymous retry. ([#107](https://github.com/FlanChanXwO/javdb-cli/pull/107))

**Full Changelog**: [v0.8.3...v0.8.4](https://github.com/FlanChanXwO/javdb-cli/compare/v0.8.3...v0.8.4)
