#!/usr/bin/env bash
set -euo pipefail

ROOT="$(pwd)"
BUILD="$ROOT/buildtree"

# Stage17 keeps the exact-current ordinary-shop selector blocked. It adds only
# a dormant MySQL persistence target that can save bag + currency through one
# transaction on one shared *sql.DB. The transaction core lives in an internal
# package so it can be UNIT/RACE tested without starting protocol-probe package
# init, which correctly requires exact-current skill resources.
bash tools/stage37_current_recovery_stage16.sh

grep -F 'current_client_selector_verified=0' "$ROOT/stage16_regular_shop_transaction_status.log"
grep -F 'wire_gate_status=BLOCKED_NO_EXACT_CURRENT_CLIENT_AUTHORITY_IN_REPO' "$ROOT/stage16_regular_shop_transaction_status.log"
grep -F 'production_handler_wiring=OFF' "$ROOT/stage16_regular_shop_transaction_status.log"
grep -F 'github.com/DATA-DOG/go-sqlmock v1.5.2' "$BUILD/ci.real.mod"

cat > "$ROOT/stage17_regular_shop_atomic_persistence_status.log" <<'STATUS'
stage17_scope=ordinary_npc_shop_mysql_atomic_persistence_target_only
current_client_selector_verified=0
wire_gate_status=BLOCKED_NO_EXACT_CURRENT_CLIENT_AUTHORITY_IN_REPO
production_handler_wiring=OFF
legacy_or_historical_selector_promotion=OFF
gm_grant_changes=0
mysql_atomic_persistence=SHARED_DB_SINGLE_TRANSACTION
json_fallback_atomic_persistence=UNSUPPORTED_FAIL_CLOSED
split_mysql_db_atomic_persistence=UNSUPPORTED_FAIL_CLOSED
transaction_order=delete_bag_then_insert_bag_rows_then_delete_currency_then_insert_currency_then_commit
live_apply_and_client_publish=NOT_WIRED
targeted_test_package=internal/shopbuypersist
protocol_probe_runtime_dependency=NOT_SUBSTITUTED
STATUS

for spec in \
  'c28360b706f8a821aacb625af06b90ded69fef4c recovery/postbuild_files/cmd__protocol-probe__latest_client_shop_buy_atomic_persistence.go' \
  '0695e5038767dcee8ed46638ade7dbeb44b6ae84 recovery/postbuild_files/internal__shopbuypersist__persist.go' \
  '23643232fcf28e8926106085a970f8c5069b38fe recovery/postbuild_files/internal__shopbuypersist__persist_test.go'; do
  expected="${spec%% *}"
  path="${spec#* }"
  actual="$(git hash-object "$path")"
  echo "stage17_input_blob path=$path expected=$expected actual=$actual"
  test "$actual" = "$expected"
done

mkdir -p "$BUILD/internal/shopbuypersist"
cp recovery/postbuild_files/cmd__protocol-probe__latest_client_shop_buy_atomic_persistence.go "$BUILD/cmd/protocol-probe/latest_client_shop_buy_atomic_persistence.go"
cp recovery/postbuild_files/internal__shopbuypersist__persist.go "$BUILD/internal/shopbuypersist/persist.go"
cp recovery/postbuild_files/internal__shopbuypersist__persist_test.go "$BUILD/internal/shopbuypersist/persist_test.go"
gofmt -w "$BUILD/cmd/protocol-probe/latest_client_shop_buy_atomic_persistence.go" "$BUILD/internal/shopbuypersist"

(cd "$BUILD" && find . -type f ! -name 'ci.real.mod' ! -name 'ci.real.sum' ! -name 'protocol-probe' -printf '%P\0' | sort -z | xargs -0 sha256sum > "$ROOT/generated-manifest.txt")
sha256sum "$ROOT/generated-manifest.txt" > "$ROOT/generated-manifest.sha256"
source_files=$(find "$BUILD" -type f ! -name 'ci.real.mod' ! -name 'ci.real.sum' ! -name 'protocol-probe' | wc -l | tr -d ' ')
echo "stage17_source_files=$source_files"
echo "stage17_manifest_sha256=$(awk '{print $1}' "$ROOT/generated-manifest.sha256")"
if [ "$source_files" != "219" ]; then
  echo "unexpected Stage17 source count: $source_files" >&2
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
for x in shopbuyatomic shopbuyatomic_race build protocol_compile protocol_race_compile nonprotocol race vet windows; do
  code=$(cat "$ROOT/$x.exit")
  echo "$x=$code"
  if [ "$code" -ne 0 ]; then failed=1; fi
done
protocol=$(cat "$ROOT/protocol_runtime.exit")
echo "protocol_runtime=$protocol"
if [ "$protocol" -ne 0 ] && [ "$protocol" -ne 125 ]; then failed=1; fi
exit "$failed"
