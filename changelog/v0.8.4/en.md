# v0.8.4 — 2026-11-17

## Fixed

- Search (including image and integrated magnet search) and movie, actor, and playback rankings use the saved default account token, fixing authentication failures after login on affected routes. Requests stay anonymous when no token is available. Image downloads and reverse-search providers receive no JavDB token; rejected requests return their authentication errors. ([#107](https://github.com/FlanChanXwO/javdb-cli/pull/107))

## New Contributors

- [@LightYear512](https://github.com/LightYear512) made their first contribution. ([#107](https://github.com/FlanChanXwO/javdb-cli/pull/107))

**Full Changelog**: [v0.8.3...v0.8.4](https://github.com/FlanChanXwO/javdb-cli/compare/v0.8.3...v0.8.4)
