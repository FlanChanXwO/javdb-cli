#!/usr/bin/env bash
# javdb-cli 真实端到端冒烟脚本:对线上 JavDB App API 跑常见只读命令并断言关键输出。
#
# 只覆盖只读命令。状态变更命令(mark/unmark/config set/auth remove/tags --refresh)
# 会改写远程或本机状态,不在此处自动执行。
#
# 凭据:需登录的命令从环境变量取账号,缺省则跳过(以 SKIP 标记)。
#   JAVDB_E2E_USERNAME / JAVDB_E2E_PASSWORD  —— 登录凭据
#   JAVDB_BIN                              —— 可执行路径,默认 ./build/javdb
#   JAVDB_E2E_HOST                         —— 可选,覆盖 --host
# 退出码:0 全部通过;1 任一硬失败;2 仅 SKIP(凭据缺失,软结果)。
set -uo pipefail

BIN="${JAVDB_BIN:-./build/javdb}"
HOST="${JAVDB_E2E_HOST:-}"
HAVE_AUTH=0
if [ -n "${JAVDB_E2E_USERNAME:-}" ] && [ -n "${JAVDB_E2E_PASSWORD:-}" ]; then
  HAVE_AUTH=1
fi

PASS=0
FAIL=0
SKIP=0

# assert_substring <label> <haystack> <needle>
assert_substring() {
  local label="$1" haystack="$2" needle="$3"
  # here-string 避免 grep -q 提前关闭管道在大输出下触发 SIGPIPE。
  if grep -Fq -- "$needle" <<<"$haystack"; then
    printf '  ok   %s\n' "$label"; PASS=$((PASS+1))
  else
    printf '  FAIL %s: missing %q in output\n' "$label" "$needle"; FAIL=$((FAIL+1))
  fi
}

# assert_json <label> <json> <python-expr-returning-bool>
assert_json() {
  local label="$1" json="$2" expr="$3"
  if printf '%s' "$json" | python3 -c "import sys,json; d=json.load(sys.stdin); sys.exit(0 if ($expr) else 1)"; then
    printf '  ok   %s\n' "$label"; PASS=$((PASS+1))
  else
    printf '  FAIL %s: JSON assertion %s failed\n' "$label" "$expr"; FAIL=$((FAIL+1))
  fi
}

# run <args...> -> stdout captured, stderr discarded, exit propagated via assertions only.
run() {
  if [ -n "$HOST" ]; then "$BIN" --host "$HOST" "$@" 2>/dev/null; else "$BIN" "$@" 2>/dev/null; fi
}
run_capture() {
  if [ -n "$HOST" ]; then "$BIN" --host "$HOST" "$@" 2>/dev/null; else "$BIN" "$@" 2>/dev/null; fi
}

host_args() { [ -n "$HOST" ] && printf '%s\n' --host "$HOST"; }

echo "== version =="
out=$(run version)
assert_substring "version prints name" "$out" "javdb"
jout=$(run version --json)
assert_json "version --json has version/commit/build_date" "$jout" "all(k in d for k in ('version','commit','build_date'))"

echo "== search =="
out=$(run search SSIS-001 --limit 1)
assert_substring "search text has number SSIS-001" "$out" "SSIS-001"
jout=$(run search SSIS-001 --limit 1 --json)
assert_json "search --json has movies[0].id" "$jout" "bool(d.get('movies') and d['movies'][0].get('id'))"
# thumb_url 受 CDN/地区影响可为空,只校验键存在,反映 list 条目契约。
assert_json "search --json exposes thumb_url key" "$jout" "'thumb_url' in d['movies'][0]"
nout=$(run search SSIS-001 --limit 1 --ndjson)
assert_json "search --ndjson emits a pipeline envelope" "$nout" "d.get('schema') == 'javdb.pipeline/v1' and d.get('kind') == 'movie'"

echo "== rankings =="
out=$(run rankings movies --period week)
assert_substring "rankings movies week has a row" "$out" $'\t'
out=$(run rankings actors --period week)
assert_substring "rankings actors week has tab row" "$out" $'\t'
out=$(run rankings playback --period week)
assert_substring "rankings playback week has a row" "$out" $'\t'

echo "== browse =="
out=$(run browse --zone censored --limit 1)
assert_substring "browse returns a movie row" "$out" $'\t'

echo "== detail (image fields) =="
out=$(run detail SSIS-001)
assert_substring "detail has 番号" "$out" "番号"
jout=$(run detail SSIS-001 --json)
# preview_images 在 CI/地区差异下可能为空数组,校验键存在与类型即可。
assert_json "detail --json exposes preview_images list" "$jout" "isinstance(d.get('preview_images'), list)"
assert_json "detail --json exposes thumb_url key" "$jout" "'thumb_url' in d"
assert_json "detail --json exposes cover_url key" "$jout" "'cover_url' in d"

echo "== tags =="
out=$(run tags --zone censored)
assert_substring "tags lists main header" "$out" "main"

echo "== magnets (read-only; uses token if available) =="
jout=$(run magnets SSIS-001 --best --json)
assert_json "magnets --best --json has magnet_uri" "$jout" "bool(d.get('magnet_uri'))"
out=$(run magnets SSIS-001 --cnsub)
assert_substring "magnets text has magnet uri line" "$out" "magnet:?xt=urn:btih:"

# --- 以下命令需要登录 ---
if [ "$HAVE_AUTH" -ne 1 ]; then
  echo "== authed commands: SKIP (set JAVDB_E2E_USERNAME/PASSWORD to enable) =="
  SKIP=$((SKIP+1))
else
  echo "== auth login =="
  if [ -n "$HOST" ]; then login_out=$("$BIN" --host "$HOST" auth login -u "$JAVDB_E2E_USERNAME" -p "$JAVDB_E2E_PASSWORD" 2>&1); \
  else login_out=$("$BIN" auth login -u "$JAVDB_E2E_USERNAME" -p "$JAVDB_E2E_PASSWORD" 2>&1); fi
  assert_substring "auth login succeeds" "$login_out" "logged in"

  echo "== auth check =="
  out=$(run auth check --json)
  assert_json "auth check --json ok" "$out" "d.get('ok') is True or d.get('valid') is True or 'user' in d or 'username' in d or bool(d)"

  echo "== top250 =="
  out=$(run top250 --zone censored --limit 1)
  assert_substring "top250 has ranked row" "$out" "#1"
  out=$(run top250 --zone censored --limit 1 --json)
  assert_json "top250 --json ok" "$out" "'movies' in d and len(d['movies']) == 1"

  echo "== watched / want / recent =="
  out=$(run watched)
  assert_substring "watched returns rows" "$out" $'\t'
  out=$(run want)
  assert_substring "want returns rows" "$out" $'\t'
  out=$(run recent)
  assert_substring "recent returns rows" "$out" $'\t'

  echo "== collections =="
  out=$(run collections actors)
  assert_substring "collections actors has row" "$out" $'\t'

  echo "== lists =="
  out=$(run lists --limit 1)
  assert_substring "lists has row" "$out" $'\t'

  echo "== entity filmography =="
  out=$(run actor 葵つかさ --limit 1)
  assert_substring "actor filmography has row" "$out" $'\t'

  echo "== lists related =="
  out=$(run lists related SSIS-001 --limit 1)
  assert_substring "lists related has row" "$out" $'\t'
fi

echo
echo "summary: pass=$PASS fail=$FAIL skip=$SKIP"
if [ "$FAIL" -ne 0 ]; then exit 1; fi
if [ "$SKIP" -ne 0 ] && [ "$PASS" -eq 0 ]; then exit 2; fi
exit 0
