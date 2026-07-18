# Plan — goal-1

## Goal

Ship a public Go `javdb` CLI with full parity vs Python `javdb-cli`, bilingual non-RE docs, MIT license, GitHub CI + multi-arch Release, and Homebrew formula on `FlanChanXwO/homebrew-tap`.

## Context

- Local repo: `javdb-cli-go` (git main, commit `0200fb7` P0).
- Python oracle: `../javdb-cli` (read-only for behavior).
- Tap exists: `FlanChanXwO/homebrew-tap` (pixiv-cli formula pattern).
- gh authenticated as FlanChanXwO with repo+workflow scopes.
- User id source proven: JWT `id` + `/api/v1/users`.

## Assumptions (no further user questions)

1. First public tag is **v0.1.0** only after parity + docs + green CI.
2. Formula name **javdb-cli**, binary **javdb**.
3. Release targets: darwin/linux/windows × amd64/arm64, CGO_ENABLED=0.
4. Tap auto-deploy may lack secrets on new repo → first formula update via `gh` PR/push if token allows; else open PR.
5. `javdb version --json` → `{"version":"vX.Y.Z"}` for brew test.
6. Docs never mention reverse engineering process.
7. Dual-repo: Python tree stays; Go owns GitHub name `javdb-cli`.
8. TDD: write failing tests first when adding endpoints/commands; green before commit.
9. Checkpoint every 3 product tasks: full `go test ./...`, `go vet`, build.
10. Live e2e optional/manual with local credentials; CI stays offline unit tests.

## Execution phases

### A — Product parity (TDD)

A1 search → A2 detail/resolve → A3 magnets → **CHK1** → A5 tags/browse → A6 entities → **CHK2** → A8 user write paths → A9 rankings/top250 → A10 lists 合集 → **CHK3** → A12 polish/version json

### B — Docs + LICENSE

Bilingual README (disclaimer), CONTRIBUTING, CHANGELOG, docs/usage|development|sdk, LICENSE MIT. Grep ban list for RE terms.

### C — GitHub

`gh repo create FlanChanXwO/javdb-cli --public --source=. --remote=origin` + push.

### D — CI

`.github/workflows/ci.yml`: test, race, vet, build on PR/push main.

### E — Release

`.github/workflows/release.yml` on tags v*; matrix build; checksums; gh release.

### F — Homebrew

Template + render; deploy Formula/javdb-cli.rb to homebrew-tap.

### G — Gate + tag v0.1.0

Full suite + smoke + tag + verify brew install path documented.

## Verification

- `go test ./... -count=1` and race/vet/build green
- Parity command table complete
- Docs EN+ZH, no RE
- Public repo + green CI
- Release assets ×6 + checksums
- brew formula installs javdb

## Rollback

- Do not tag until G green.
- If bad release: delete tag/release (careful), fix, retag patch.
- Tap formula can pin previous version SHA.

## Risks

- Large scope → goal mode one-task discipline.
- Tap secrets → manual PR fallback.
- Docs RE leak → mandatory grep in check tasks.
