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
- [ ] **Status:** pending
- **Type:** 集中检查-debug 循环
- **Checklist:** input.md alignment; bugs/dead code; go test/race/vet/build; no secrets in logs; docs not yet required.
- **Actual / evidence / risks / next:**

### Task 5 — Tags + browse
- [ ] **Status:** pending
- **Goal:** masks package; tags taxonomy cache; browse + tag alias resolve; CLI tags/browse.
- **Done criteria:** mask golden tests; CLI tests offline.
- **Actual / evidence / risks / next:**

### Task 6 — Entity filmography (actor/series/maker/director/code/list)
- [ ] **Status:** pending
- **Goal:** entity_movies + resolve_entity including list letter `l`; six commands + shared options.
- **Done criteria:** unit tests letters/masks; CLI spy tests.
- **Actual / evidence / risks / next:**

### Task 7 — CHECKPOINT / review-debug #2
- [ ] **Status:** pending
- **Type:** 集中检查-debug 循环
- **Actual / evidence / risks / next:**

### Task 8 — User write paths + auto_relogin
- [ ] **Status:** pending
- **Goal:** watched/want/mark/unmark/recent/collections; default-account client; auto_relogin once when config on.
- **Done criteria:** unit tests auth retry path; CLI offline spies.
- **Actual / evidence / risks / next:**

### Task 9 — Rankings + top250
- [ ] **Status:** pending
- **Goal:** rankings movies/actors/playback; top250 filters; period day→daily mapping.
- **Done criteria:** param unit tests; CLI wired.
- **Actual / evidence / risks / next:**

### Task 10 — Lists 合集 group
- [ ] **Status:** pending
- **Goal:** lists default my (sort_by required); show/search/related; list filmography already in task 6.
- **Done criteria:** unit + CLI tests.
- **Actual / evidence / risks / next:**

### Task 11 — CHECKPOINT / review-debug #3
- [ ] **Status:** pending
- **Type:** 集中检查-debug 循环 (+ optional live smoke if creds present, never log secrets)
- **Actual / evidence / risks / next:**

### Task 12 — Parity polish + version --json + buildinfo
- [ ] **Status:** pending
- **Goal:** printers/has-magnets consistency; internal/buildinfo; `javdb version [--json]` for brew; root flags verified.
- **Done criteria:** version json shape `{"version":"v..."}`; full go test green.
- **Actual / evidence / risks / next:**

---

## Docs, license, repo, CI, release, brew

### Task 13 — LICENSE + bilingual README (disclaimer) + CONTRIBUTING + CHANGELOG
- [ ] **Status:** pending
- **Goal:** MIT LICENSE; README.md + README.zh-CN.md with disclaimer; CONTRIBUTING EN/ZH; CHANGELOG EN/ZH skeleton.
- **Done criteria:** files exist; grep no Frida/blutter/apk reverse narrative.
- **Actual / evidence / risks / next:**

### Task 14 — docs/ usage + development + sdk (EN+ZH)
- [ ] **Status:** pending
- **Goal:** docs/index, usage, development, sdk each EN+ZH; product-only.
- **Done criteria:** RE grep clean; links from README.
- **Actual / evidence / risks / next:**

### Task 15 — CHECKPOINT / review-debug #4 (docs + product)
- [ ] **Status:** pending
- **Type:** 集中检查-debug 循环
- **Actual / evidence / risks / next:**

### Task 16 — GitHub public repo create + push
- [ ] **Status:** pending
- **Goal:** `gh repo create FlanChanXwO/javdb-cli --public` from local tree; push main; no secrets.
- **Done criteria:** repo URL reachable; remote origin set.
- **Actual / evidence / risks / next:**

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
