#!/usr/bin/env bash
set -euo pipefail

ROOT="$(pwd)"
BUILD="$ROOT/buildtree"

# Stage18 adds only pure ordinary-shop purchase staging. The exact-current
# ordinary-shop selector remains unverified, so no production handler, live
# mutation, persistence invocation, or client publish is wired here.
bash tools/stage37_current_recovery_stage17.sh

grep -F 'current_client_selector_verified=0' "$ROOT/stage17_regular_shop_atomic_persistence_status.log"
grep -F 'wire_gate_status=BLOCKED_NO_EXACT_CURRENT_CLIENT_AUTHORITY_IN_REPO' "$ROOT/stage17_regular_shop_atomic_persistence_status.log"
grep -F 'production_handler_wiring=OFF' "$ROOT/stage17_regular_shop_atomic_persistence_status.log"

cat > "$ROOT/stage18_regular_shop_purchase_staging_status.log" <<'STATUS'
stage18_scope=ordinary_npc_shop_pure_purchase_staging
current_client_selector_verified=0
wire_gate_status=BLOCKED_NO_EXACT_CURRENT_CLIENT_AUTHORITY_IN_REPO
production_handler_wiring=OFF
live_apply_wiring=OFF
client_publish_wiring=OFF
legacy_or_historical_selector_promotion=OFF
gm_grant_changes=0
atomic_persistence_target=READY_MYSQL_SHARED_DB_ONLY
pure_stage_currency_modes=0_GOLD_1_SILVER_2_SILVER_CARD
reward_amount_source=REQUEST_COUNT_MATCHES_RECOVERED_HANDLER
shop_listing_amount_usage=NOT_USED_MATCHES_RECOVERED_HANDLER
slot_rule=SMALLEST_POSITIVE_FREE_SLOT_WITHIN_REWARD_BAG_VIEW
positive_pre_authored_slot=PRESERVED_MATCHES_RECOVERED_ADD_BAG_ITEM
stacking_inferred=NO
capacity_inferred=NO
binding_inferred=NO
negative_unit_price=FAIL_CLOSED_SAFETY_GUARD
STATUS

for spec in \
  '45f3fc422939a3c570e73712cf1fb4736e8289b3 recovery/postbuild_files/internal__shopbuyplan__plan.go' \
  '30d997e106003ab963f409cf52384d7933232152 recovery/postbuild_files/internal__shopbuyplan__plan_test.go' \
  '40e0913c0ab483c444655eba7480dd518fe8143d recovery/postbuild_files/cmd__protocol-probe__latest_client_shop_buy_stage.go'; do
  expected="${spec%% *}"
  path="${spec#* }"
  actual="$(git hash-object "$path")"
  echo "stage18_input_blob path=$path expected=$expected actual=$actual"
  test "$actual" = "$expected"
done

mkdir -p "$BUILD/internal/shopbuyplan"
cp recovery/postbuild_files/internal__shopbuyplan__plan.go "$BUILD/internal/shopbuyplan/plan.go"
cp recovery/postbuild_files/internal__shopbuyplan__plan_test.go "$BUILD/internal/shopbuyplan/plan_test.go"
cp recovery/postbuild_files/cmd__protocol-probe__latest_client_shop_buy_stage.go "$BUILD/cmd/protocol-probe/latest_client_shop_buy_stage.go"
gofmt -w "$BUILD/internal/shopbuyplan" "$BUILD/cmd/protocol-probe/latest_client_shop_buy_stage.go"

(cd "$BUILD" && find . -type f ! -name 'ci.real.mod' ! -name 'ci.real.sum' ! -name 'protocol-probe' -printf '%P\0' | sort -z | xargs -0 sha256sum > "$ROOT/generated-manifest.txt")
sha256sum "$ROOT/generated-manifest.txt" > "$ROOT/generated-manifest.sha256"
source_files=$(find "$BUILD" -type f ! -name 'ci.real.mod' ! -name 'ci.real.sum' ! -name 'protocol-probe' | wc -l | tr -d ' ')
echo "stage18_source_files=$source_files"
echo "stage18_manifest_sha256=$(awk '{print $1}' "$ROOT/generated-manifest.sha256")"
if [ "$source_files" != "222" ]; then
  echo "unexpected Stage18 source count: $source_files" >&2
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
for x in shopbuyplan shopbuyplan_race shopbuyatomic shopbuyatomic_race build protocol_compile protocol_race_compile nonprotocol race vet windows; do
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
echo 'live_apply_wiring=OFF'
exit "$failed"
