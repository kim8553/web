#!/usr/bin/env bash
set -euo pipefail

ROOT="$(pwd)"
BUILD="$ROOT/buildtree"

# Stage20 adds a dormant post-commit client publication boundary for ordinary
# NPC shop purchase. Exact-current ordinary-shop selector authority remains
# unverified, so production handler wiring stays OFF.
bash tools/stage37_current_recovery_stage19.sh

grep -F 'current_client_selector_verified=0' "$ROOT/stage19_regular_shop_commit_boundary_status.log"
grep -F 'wire_gate_status=BLOCKED_NO_EXACT_CURRENT_CLIENT_AUTHORITY_IN_REPO' "$ROOT/stage19_regular_shop_commit_boundary_status.log"
grep -F 'production_handler_wiring=OFF' "$ROOT/stage19_regular_shop_commit_boundary_status.log"
grep -F 'client_publish_wiring=OFF' "$ROOT/stage19_regular_shop_commit_boundary_status.log"

cat > "$ROOT/stage20_regular_shop_publish_boundary_status.log" <<'STATUS'
stage20_scope=ordinary_npc_shop_post_commit_client_publish_boundary
current_client_selector_verified=0
wire_gate_status=BLOCKED_NO_EXACT_CURRENT_CLIENT_AUTHORITY_IN_REPO
production_handler_wiring=OFF
client_publish_wiring=READY_DORMANT
publish_callsite_in_production=OFF
frame_order=CURRENCY_THEN_LATEST_CURRENT_BAG_FRAMES
currency_frame_source=FROZEN_COMMITTED_CURRENCY_VIA_EXISTING_ENCODER
bag_frame_source=LATEST_CLIENT_CURRENT_BAG_FRAMES
publish_failure=ERR_RESYNC_REQUIRED_NO_DB_OR_MEMORY_ROLLBACK
resync_or_reconnect_action=CALLER_REQUIRED_NOT_WIRED
atomic_persistence_target=MYSQL_SHARED_DB_ONLY
json_fallback_atomic_persistence=UNSUPPORTED_FAIL_CLOSED
split_mysql_db_atomic_persistence=UNSUPPORTED_FAIL_CLOSED
legacy_or_historical_selector_promotion=OFF
gm_grant_changes=0
STATUS

for spec in \
  '3e4994c962432f7c9406f777b89eb9895320c39a recovery/postbuild_files/internal__shopbuypublish__publish.go' \
  'efec6d0189ee32dbef4e2980ca77d2a31da1feff recovery/postbuild_files/internal__shopbuypublish__publish_test.go' \
  '786d84a1cf5295fc1932c172413115fd672e804b recovery/postbuild_files/cmd__protocol-probe__latest_client_shop_buy_publish.go'; do
  expected="${spec%% *}"
  path="${spec#* }"
  actual="$(git hash-object "$path")"
  echo "stage20_input_blob path=$path expected=$expected actual=$actual"
  test "$actual" = "$expected"
done

mkdir -p "$BUILD/internal/shopbuypublish"
cp recovery/postbuild_files/internal__shopbuypublish__publish.go "$BUILD/internal/shopbuypublish/publish.go"
cp recovery/postbuild_files/internal__shopbuypublish__publish_test.go "$BUILD/internal/shopbuypublish/publish_test.go"
cp recovery/postbuild_files/cmd__protocol-probe__latest_client_shop_buy_publish.go "$BUILD/cmd/protocol-probe/latest_client_shop_buy_publish.go"
gofmt -w "$BUILD/internal/shopbuypublish" "$BUILD/cmd/protocol-probe/latest_client_shop_buy_publish.go"

(cd "$BUILD" && find . -type f ! -name 'ci.real.mod' ! -name 'ci.real.sum' ! -name 'protocol-probe' -printf '%P\0' | sort -z | xargs -0 sha256sum > "$ROOT/generated-manifest.txt")
sha256sum "$ROOT/generated-manifest.txt" > "$ROOT/generated-manifest.sha256"
source_files=$(find "$BUILD" -type f ! -name 'ci.real.mod' ! -name 'ci.real.sum' ! -name 'protocol-probe' | wc -l | tr -d ' ')
echo "stage20_source_files=$source_files"
echo "stage20_manifest_sha256=$(awk '{print $1}' "$ROOT/generated-manifest.sha256")"
if [ "$source_files" != "228" ]; then
  echo "unexpected Stage20 source count: $source_files" >&2
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

run_gate shopbuypublish go test -modfile=ci.real.mod ./internal/shopbuypublish -count=1
run_gate shopbuypublish_race go test -modfile=ci.real.mod -race ./internal/shopbuypublish -count=1
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
for x in shopbuypublish shopbuypublish_race shopbuycommit shopbuycommit_race shopbuyplan shopbuyplan_race shopbuyatomic shopbuyatomic_race build protocol_compile protocol_race_compile nonprotocol race vet windows; do
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
echo 'client_publish_wiring=READY_DORMANT'
exit "$failed"
