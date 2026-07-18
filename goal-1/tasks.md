# Tasks — goal-1

Status legend: `[ ]` todo · `[~]` in progress · `[x]` done · `[!]` blocked

---

## Product parity (TDD)

### Task 1 — Search command + API
- [x] **Status:** done
- **Goal:** `javdb search` with zone/sort/filter-by/type/page/limit/has-magnets/json; appapi Search; public SDK wrapper; unit tests (params + printers).
- **Done criteria:** `go test` green; CLI help shows flags; offline tests assert query keys.
- **Actual / evidence / risks / next:**
  - Added `BuildSearchParams`, `Client.Search`, `javdb.Search`, CLI `search`, printers + FilterHasMagnets.
  - Evidence: `go test ./...` green; live `search SSIS-589` and `search 巨乳 --type list` OK; commit `feat(search)...`.
  - Risks: none high for search path.
  - Next: Task 2 detail + number resolve.

### Task 2 — Detail + number resolve
- [x] **Status:** done
- **Goal:** `javdb detail` with -i/--magnets/--json; ResolveNumber→id via search zone=all; graph print parity.
- **Done criteria:** unit tests for resolve preference exact number; build green.
- **Actual / evidence / risks / next:**
  - ResolveNumber/ResolveMovieID; MovieDetail; detail CLI + PrintDetail graph.
  - Evidence: unit tests green; live `detail SSIS-589` shows 9DGB5X/dQ5/9Dqpw; `go test ./...` OK.
  - Risks: `--magnets` needs login (token from default account).
  - Next: Task 3 magnets command.

### Task 3 — Magnets command
- [x] **Status:** done
- **Goal:** `javdb magnets` cnsub/hd/min-size/best/json; filter_magnets + pick_best unit tests; movie_magnets endpoint.
- **Done criteria:** unit tests cover best preference; CLI wired.
- **Actual / evidence / risks / next:**
  - FilterMagnets/PickBestMagnet/MagnetURI + CLI magnets (auth via default token).
  - Evidence: unit tests green; live `magnets SSIS-589 --best` → cnsub+hd 5.3GB hash 3251b649…; `go test ./...` OK.
  - Risks: requires logged-in default account.
  - Next: Task 4 CHECKPOINT #1.

### Task 4 — CHECKPOINT / review-debug #1
- [x] **Status:** done
- **Type:** 集中检查-debug 循环
- **Checklist:** input.md alignment; bugs/dead code; go test/race/vet/build; no secrets in logs; docs not yet required.
- **Actual / evidence / risks / next:**
  - Aligned with goal: P0+search/detail/magnets done; docs/CI/release still later tasks.
  - Evidence: `go test ./...` + `-race` green; `go vet` clean; `scripts/build.sh` OK; CLI help lists auth/search/detail/magnets.
  - No secrets in source; no blocking bugs found.
  - Residual: full parity incomplete (browse/entities/user/lists/rankings); README still stub; no remote yet.
  - Next: Task 5 tags + browse.

### Task 5 — Tags + browse
- [x] **Status:** done
- **Goal:** masks package; tags taxonomy cache; browse + tag alias resolve; CLI tags/browse.
- **Done criteria:** mask golden tests; CLI tests offline.
- **Actual / evidence / risks / next:**
  - BuildTagFilter/EntityFilter; tags store; RefreshTagTaxonomy; browse; CLI tags/browse.
  - Evidence: mask unit tests; tags alias resolve; live `tags` + `browse --tag 巨乳` OK; commit ae0fb61.
  - Risks: none high.
  - Next: Task 6 entity filmography.

### Task 6 — Entity filmography (actor/series/maker/director/code/list)
- [x] **Status:** done
- **Goal:** entity_movies + resolve_entity including list letter `l`; six commands + shared options.
- **Done criteria:** unit tests letters/masks; CLI spy tests.
- **Actual / evidence / risks / next:**
  - EntityMovies/ResolveEntity/EntityDetail/AllEntityMovies; 6 CLI cmds + shared flags.
  - Evidence: unit tests; live `actor 山手梨愛 --main m`, `series SSS-BODY`, `list RZ8Bm` OK.
  - Risks: resolve falls back to search first hit when name ambiguous.
  - Next: Task 7 CHECKPOINT #2.

### Task 7 — CHECKPOINT / review-debug #2
- [x] **Status:** done
- **Type:** 集中检查-debug 循环
- **Actual / evidence / risks / next:**
  - Post tags/browse/entity: CLI surface now includes actor/series/maker/director/code/list + browse/tags.
  - Evidence: `go test ./...` + `-race` green; `go vet` clean; build OK; help lists all entity cmds.
  - No TODO/FIXME/panic leftovers found; no blocking bugs.
  - Residual: user write paths, rankings/top250, lists group, docs/CI/release still open.
  - Next: Task 8 user write paths + auto_relogin.

### Task 8 — User write paths + auto_relogin
- [x] **Status:** done
- **Goal:** watched/want/mark/unmark/recent/collections; default-account client; auto_relogin once when config on.
- **Done criteria:** unit tests auth retry path; CLI offline spies.
- **Actual / evidence / risks / next:**
  - user.go APIs; withAuthedClient (auto_relogin once); CLI watched/want/recent/collections/mark/unmark.
  - Evidence: unit tests; live recent/collections/watched OK; commit feat(user)…
  - Risks: auto_relogin needs saved password; default off per plan.
  - Next: Task 9 rankings + top250.

### Task 9 — Rankings + top250
- [x] **Status:** done
- **Goal:** rankings movies/actors/playback; top250 filters; period day→daily mapping.
- **Done criteria:** param unit tests; CLI wired.
- **Actual / evidence / risks / next:**
  - RankingsMovies/Actors/Playback + Top250; ActorPeriod; rankings/* + top250 CLI.
  - Evidence: unit tests; live rankings movies/actors + top250 #1..3 OK.
  - Risks: none high.
  - Next: Task 10 lists 合集 group.

### Task 10 — Lists 合集 group
- [x] **Status:** done
- **Goal:** lists default my (sort_by required); show/search/related; list filmography already in task 6.
- **Done criteria:** unit + CLI tests.
- **Actual / evidence / risks / next:**
  - MyLists/ListInfo/RelatedLists; lists + show/search/related CLI.
  - Evidence: unit tests; live lists/search/show/related OK; list filmography already via task6.
  - Risks: none high.
  - Next: Task 11 CHECKPOINT #3.

### Task 11 — CHECKPOINT / review-debug #3
- [x] **Status:** done
- **Type:** 集中检查-debug 循环 (+ optional live smoke if creds present, never log secrets)
- **Actual / evidence / risks / next:**
  - Product parity commands present: search/detail/magnets/browse/tags/entities/user/rankings/top250/lists.
  - Evidence: `go test` + `-race` green; `go vet` clean; build OK; live smoke search/detail/magnets/browse/actor/rankings/top250/lists OK.
  - No blocking bugs. Residual: version --json/buildinfo polish; bilingual docs; GitHub/CI/release/brew.
  - Next: Task 12 parity polish + version --json + buildinfo.

### Task 12 — Parity polish + version --json + buildinfo
- [x] **Status:** done
- **Goal:** printers/has-magnets consistency; internal/buildinfo; `javdb version [--json]` for brew; root flags verified.
- **Done criteria:** version json shape `{"version":"v..."}`; full go test green.
- **Actual / evidence / risks / next:**
  - buildinfo package; version text/JSON; build.sh ldflags.
  - Evidence: unit tests; `VERSION=0.1.0 sh scripts/build.sh` → `version --json` has `"version":"v0.1.0"`; full go test green.
  - Risks: none. Product parity phase complete.
  - Next: Task 13 LICENSE + bilingual README/CONTRIBUTING/CHANGELOG.

---

## Docs, license, repo, CI, release, brew

### Task 13 — LICENSE + bilingual README (disclaimer) + CONTRIBUTING + CHANGELOG
- [x] **Status:** done
- **Goal:** MIT LICENSE; README.md + README.zh-CN.md with disclaimer; CONTRIBUTING EN/ZH; CHANGELOG EN/ZH skeleton.
- **Done criteria:** files exist; grep no Frida/blutter/apk reverse narrative.
- **Actual / evidence / risks / next:**
  - LICENSE MIT; README + zh-CN with disclaimer; CONTRIBUTING + CHANGELOG EN/ZH.
  - Evidence: RE grep clean on those files; commit docs task13.
  - Risks: README links to docs/* filled in task 14.
  - Next: Task 14 docs/ usage + development + sdk EN+ZH.

### Task 14 — docs/ usage + development + sdk (EN+ZH)
- [x] **Status:** done
- **Goal:** docs/index, usage, development, sdk each EN+ZH; product-only.
- **Done criteria:** RE grep clean; links from README.
- **Actual / evidence / risks / next:**
  - docs/index|usage|development|sdk (+ .zh-CN) product-only.
  - Evidence: RE grep clean on docs/; files linked from README; commit docs task14.
  - Risks: none.
  - Next: Task 15 CHECKPOINT #4 (docs + product).

### Task 15 — CHECKPOINT / review-debug #4 (docs + product)
- [x] **Status:** done
- **Type:** 集中检查-debug 循环
- **Actual / evidence / risks / next:**
  - Product CLI complete + bilingual docs/LICENSE in place.
  - Evidence: `go test` + `-race` green; `go vet` clean; build OK; RE grep clean on docs/README; all README-linked files exist; full command surface listed.
  - Residual: no GitHub remote/CI/release/brew yet (tasks 16–20).
  - Next: Task 16 create public GitHub repo + push.

### Task 16 — GitHub public repo create + push
- [x] **Status:** done
- **Goal:** `gh repo create FlanChanXwO/javdb-cli --public` from local tree; push main; no secrets.
- **Done criteria:** repo URL reachable; remote origin set.
- **Actual / evidence / risks / next:**
  - Created public repo + pushed main; origin tracks origin/main.
  - Evidence: https://github.com/FlanChanXwO/javdb-cli ; no auth.json/secrets tracked.
  - Risks: none.
  - Next: Task 17 CI workflow quality gate.

### Task 17 — CI workflow quality gate
- [ ] **Status:** pending
- **Goal:** `.github/workflows/ci.yml` test/race/vet/build; push; confirm green run.
- **Done criteria:** gh run success on main.
- **Actual / evidence / risks / next:**

### Task 18 — Release workflow + package helpers + homebrew template
- [ ] **Status:** pending
- **Goal:** release.yml matrix 6 arches; checksums; formula tmpl; render script.
- **Done criteria:** workflow files valid; local package dry-run if possible.
- **Actual / evidence / risks / next:**

### Task 19 — CHECKPOINT / review-debug #5 (release readiness)
- [ ] **Status:** pending
- **Type:** 集中检查-debug 循环
- **Actual / evidence / risks / next:**

### Task 20 — Tag v0.1.0, publish Release, update homebrew-tap formula
- [ ] **Status:** pending
- **Goal:** full suite green; tag v0.1.0; wait release assets; deploy Formula/javdb-cli.rb to homebrew-tap; document brew install.
- **Done criteria:** release has 6 archives+checksums; formula on tap; brew install path documented.
- **Actual / evidence / risks / next:**

### Task 21 — FINAL REVIEW
- [ ] **Status:** pending
- **Type:** 终审
- **Checklist:** parity complete; docs EN/ZH; no RE; CI green; release; brew; residual risks listed.
- **Actual / evidence / risks / next:**
