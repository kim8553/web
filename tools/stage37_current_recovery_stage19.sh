#!/usr/bin/env bash
set -euo pipefail

ROOT="$(pwd)"
BUILD="$ROOT/buildtree"

# Stage19 builds the concurrency-safe durable/live commit boundary for ordinary
# NPC shop purchase, but keeps it dormant. The exact-current ordinary-shop wire
# selector is still unverified, so production handler and client publication
# remain OFF.
bash tools/stage37_current_recovery_stage18.sh

grep -F 'current_client_selector_verified=0' "$ROOT/stage18_regular_shop_purchase_staging_status.log"
grep -F 'wire_gate_status=BLOCKED_NO_EXACT_CURRENT_CLIENT_AUTHORITY_IN_REPO' "$ROOT/stage18_regular_shop_purchase_staging_status.log"
grep -F 'production_handler_wiring=OFF' "$ROOT/stage18_regular_shop_purchase_staging_status.log"
grep -F 'client_publish_wiring=OFF' "$ROOT/stage18_regular_shop_purchase_staging_status.log"

cat > "$ROOT/stage19_regular_shop_commit_boundary_status.log" <<'STATUS'
stage19_scope=ordinary_npc_shop_concurrency_safe_persistence_live_apply_boundary
current_client_selector_verified=0
wire_gate_status=BLOCKED_NO_EXACT_CURRENT_CLIENT_AUTHORITY_IN_REPO
production_handler_wiring=OFF
client_publish_wiring=OFF
live_apply_helper=READY_DORMANT
atomic_persistence_target=MYSQL_SHARED_DB_ONLY
json_fallback_atomic_persistence=UNSUPPORTED_FAIL_CLOSED
split_mysql_db_atomic_persistence=UNSUPPORTED_FAIL_CLOSED
lock_boundary=PLAYER_MUTEX_SPANS_LIVE_SNAPSHOT_PURE_STAGE_ATOMIC_PERSIST_LIVE_APPLY
publish_boundary=AFTER_UNLOCK_NOT_WIRED
gate_order=REQUIRE_BEFORE_LOCK_STAGE_PERSIST_APPLY
persistence_failure=NO_LIVE_APPLY
stage_failure=NO_PERSIST_NO_LIVE_APPLY
db_and_memory_single_atomic_transaction=NO
legacy_or_historical_selector_promotion=OFF
gm_grant_changes=0
STATUS

for spec in \
  'a4d652df34d5f8ed2642d126b8ef5a18feee67fc recovery/postbuild_files/internal__shopbuycommit__commit.go' \
  'c9fe5a893ccf0e1f4f4b59ddb6ed39fe0dada4ec recovery/postbuild_files/internal__shopbuycommit__commit_test.go' \
  '4db98ae7116b05a1ff705ff483d86495bcfefe01 recovery/postbuild_files/cmd__protocol-probe__latest_client_shop_buy_commit.go'; do
  expected="${spec%% *}"
  path="${spec#* }"
  actual="$(git hash-object "$path")"
  echo "stage19_input_blob path=$path expected=$expected actual=$actual"
  test "$actual" = "$expected"
done

mkdir -p "$BUILD/internal/shopbuycommit"
cp recovery/postbuild_files/internal__shopbuycommit__commit.go "$BUILD/internal/shopbuycommit/commit.go"
cp recovery/postbuild_files/internal__shopbuycommit__commit_test.go "$BUILD/internal/shopbuycommit/commit_test.go"
cp recovery/postbuild_files/cmd__protocol-probe__latest_client_shop_buy_commit.go "$BUILD/cmd/protocol-probe/latest_client_shop_buy_commit.go"
gofmt -w "$BUILD/internal/shopbuycommit" "$BUILD/cmd/protocol-probe/latest_client_shop_buy_commit.go"

(cd "$BUILD" && find . -type f ! -name 'ci.real.mod' ! -name 'ci.real.sum' ! -name 'protocol-probe' -printf '%P\0' | sort -z | xargs -0 sha256sum > "$ROOT/generated-manifest.txt")
sha256sum "$ROOT/generated-manifest.txt" > "$ROOT/generated-manifest.sha256"
source_files=$(find "$BUILD" -type f ! -name 'ci.real.mod' ! -name 'ci.real.sum' ! -name 'protocol-probe' | wc -l | tr -d ' ')
echo "stage19_source_files=$source_files"
echo "stage19_manifest_sha256=$(awk '{print $1}' "$ROOT/generated-manifest.sha256")"
if [ "$source_files" != "225" ]; then
  echo "unexpected Stage19 source count: $source_files" >&2
  exit 96
fi

cd "$BUILD"
set +e
run_gate() {
  local name="$1"
  shift
  "$@" >"$ROOT/${name}.log" 2>&1
  echo $? >"$ROOT/${name}.exit"
}

run_gate shopbuycommit go test -modfile=ci.real.mod ./internal/shopbuycommit -count=1
run_gate shopbuycommit_race go test -modfile=ci.real.mod -race ./internal/shopbuycommit -count=1
run_gate shopbuyplan go test -modfile=ci.real.mod ./internal/shopbuyplan -count=1
run_gate shopbuyplan_race go test -modfile=ci.real.mod -race ./internal/shopbuyplan -count=1
run_gate shopbuyatomic go test -modfile=ci.real.mod ./internal/shopbuypersist -count=1
run_gate shopbuyatomic_race go test -modfile=ci.real.mod -race ./internal/shopbuypersist -count=1
run_gate build go build -modfile=ci.real.mod ./cmd/protocol-probe
run_gate protocol_compile go test -modfile=ci.real.mod -c -o "$ROOT/protocol-probe.test" ./cmd/protocol-probe
run_gate protocol_race_compile go test -modfile=ci.real.mod -race -c -o "$ROOT/protocol-probe-race.test" ./cmd/protocol-probe

mapfile -t pkgs < <(go list -modfile=ci.real.mod ./... | grep -v '^github.com/local/9yin-go-server/cmd/protocol-probe$')
go test -modfile=ci.real.mod "${pkgs[@]}" -count=1 > "$ROOT/nonprotocol.log" 2>&1
echo $? > "$ROOT/nonprotocol.exit"
go test -modfile=ci.real.mod -race "${pkgs[@]}" -count=1 > "$ROOT/race.log" 2>&1
echo $? > "$ROOT/race.exit"
go vet -modfile=ci.real.mod ./... > "$ROOT/vet.log" 2>&1
echo $? > "$ROOT/vet.exit"

GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -modfile=ci.real.mod -o "$ROOT/stage37-current-windows-amd64.exe" ./cmd/protocol-probe > "$ROOT/windows.log" 2>&1
echo $? > "$ROOT/windows.exit"
if [ -f "$ROOT/stage37-current-windows-amd64.exe" ]; then
  sha256sum "$ROOT/stage37-current-windows-amd64.exe" > "$ROOT/stage37-current-windows-amd64.sha256"
  file "$ROOT/stage37-current-windows-amd64.exe" > "$ROOT/stage37-current-windows-amd64.file.txt"
  go version -m "$ROOT/stage37-current-windows-amd64.exe" > "$ROOT/stage37-current-windows-amd64.goversion.txt"
fi

failed=0
for x in shopbuycommit shopbuycommit_race shopbuyplan shopbuyplan_race shopbuyatomic shopbuyatomic_race build protocol_compile protocol_race_compile nonprotocol race vet windows; do
  code=$(cat "$ROOT/$x.exit")
  echo "$x=$code"
  if [ "$code" -ne 0 ]; then failed=1; fi
done
protocol=$(cat "$ROOT/protocol_runtime.exit")
echo "protocol_runtime=$protocol"
if [ "$protocol" -ne 0 ] && [ "$protocol" -ne 125 ]; then failed=1; fi

echo 'current_client_selector_verified=0'
echo 'wire_gate_status=BLOCKED_NO_EXACT_CURRENT_CLIENT_AUTHORITY_IN_REPO'
echo 'production_handler_wiring=OFF'
echo 'client_publish_wiring=OFF'
exit "$failed"
