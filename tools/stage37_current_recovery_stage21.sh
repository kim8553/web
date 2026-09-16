#!/usr/bin/env bash
set -euo pipefail

ROOT="$(pwd)"
BUILD="$ROOT/buildtree"

# Stage21 removes the unsafe recovered legacy/candidate 0x46 ordinary-shop
# mutation route from the reconstructed production binary. Stage20's safe path
# remains dormant until exact-current selector authority is proven.
bash tools/stage37_current_recovery_stage20.sh

grep -F 'current_client_selector_verified=0' "$ROOT/stage20_regular_shop_publish_boundary_status.log"
grep -F 'wire_gate_status=BLOCKED_NO_EXACT_CURRENT_CLIENT_AUTHORITY_IN_REPO' "$ROOT/stage20_regular_shop_publish_boundary_status.log"
grep -F 'production_handler_wiring=OFF' "$ROOT/stage20_regular_shop_publish_boundary_status.log"

python3 tools/stage37_current_recovery_stage21_disable_unverified_legacy_shop_buy.py
gofmt -w "$BUILD/cmd/protocol-probe/zz_recovered_overlay.go"

python3 - <<'PY' > "$ROOT/stage21_legacy_shop_buy_audit.log"
from pathlib import Path
p = Path('buildtree/cmd/protocol-probe/zz_recovered_overlay.go')
s = p.read_text(encoding='utf-8')
a = s.index('func handleShopBuyCustom(')
b = s.index('\nfunc shopCatalogItems(', a)
body = s[a:b]
for token in [
    'player.addGold(', 'player.addSilver(', 'player.addSilverCard(',
    'player.addBagItem(', 'writeFrames(', 'persistBagEquip(',
    'currencyStore.Save(',
]:
    if token in body:
        raise SystemExit(f'forbidden legacy mutation token remains: {token}')
if 'candidate selector=0x46 blocked' not in body:
    raise SystemExit('Stage21 fail-closed diagnostic missing')
if 'custom.Values[0].Int32 != 0x46' not in body:
    raise SystemExit('Stage21 candidate selector guard missing')
if 'return false, nil' not in body:
    raise SystemExit('Stage21 non-authoritative return missing')
print('legacy_candidate_handler_audit=PASS')
print('candidate_selector=0x46')
print('legacy_live_currency_mutation=ABSENT')
print('legacy_live_bag_mutation=ABSENT')
print('legacy_pre_persist_client_publish=ABSENT')
print('legacy_separate_bag_currency_save=ABSENT')
print('candidate_authoritative_handling=NO')
PY

cat > "$ROOT/stage21_regular_shop_legacy_route_status.log" <<'STATUS'
stage21_scope=disable_unverified_legacy_ordinary_shop_mutation_route
current_client_selector_verified=0
wire_gate_status=BLOCKED_NO_EXACT_CURRENT_CLIENT_AUTHORITY_IN_REPO
safe_production_handler_wiring=OFF
safe_client_publish_callsite=OFF
legacy_candidate_selector=0x46
legacy_candidate_mutation_route=DISABLED_FAIL_CLOSED
legacy_candidate_authoritative_handling=NO
legacy_live_currency_mutation=ABSENT
legacy_live_bag_mutation=ABSENT
legacy_client_publish=ABSENT
legacy_bag_currency_persistence=ABSENT
stage20_safe_path=READY_DORMANT
selector_promotion=OFF
gm_grant_changes=0
STATUS

(cd "$BUILD" && find . -type f ! -name 'ci.real.mod' ! -name 'ci.real.sum' ! -name 'protocol-probe' -printf '%P\0' | sort -z | xargs -0 sha256sum > "$ROOT/generated-manifest.txt")
sha256sum "$ROOT/generated-manifest.txt" > "$ROOT/generated-manifest.sha256"
source_files=$(find "$BUILD" -type f ! -name 'ci.real.mod' ! -name 'ci.real.sum' ! -name 'protocol-probe' | wc -l | tr -d ' ')
echo "stage21_source_files=$source_files"
echo "stage21_manifest_sha256=$(awk '{print $1}' "$ROOT/generated-manifest.sha256")"
if [ "$source_files" != "228" ]; then
  echo "unexpected Stage21 source count: $source_files" >&2
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
echo 'legacy_candidate_mutation_route=DISABLED_FAIL_CLOSED'
echo 'safe_production_handler_wiring=OFF'
exit "$failed"
