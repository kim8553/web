#!/usr/bin/env bash
set -euo pipefail
ROOT="$(pwd)"
BUILD="$ROOT/buildtree"

# Preserve Stage21's verified fail-closed purchase route. Stage22 only replaces
# broad decoded-argument logging with passive, bounded current-wire evidence.
bash tools/stage37_current_recovery_stage21.sh
grep -Fx 'legacy_candidate_mutation_route=DISABLED_FAIL_CLOSED' "$ROOT/stage21_regular_shop_legacy_route_status.log"
grep -Fx 'safe_production_handler_wiring=OFF' "$ROOT/stage21_regular_shop_legacy_route_status.log"

for spec in \
  'recovery/postbuild_files/internal__shopwiretrace__trace.go' \
  'recovery/postbuild_files/internal__shopwiretrace__trace_test.go' \
  'recovery/postbuild_files/cmd__protocol-probe__latest_client_shop_wire_trace.go'; do
  test -f "$spec" || { echo "missing Stage22 source: $spec" >&2; exit 97; }
done
mkdir -p "$BUILD/internal/shopwiretrace"
cp recovery/postbuild_files/internal__shopwiretrace__trace.go "$BUILD/internal/shopwiretrace/trace.go"
cp recovery/postbuild_files/internal__shopwiretrace__trace_test.go "$BUILD/internal/shopwiretrace/trace_test.go"
cp recovery/postbuild_files/cmd__protocol-probe__latest_client_shop_wire_trace.go "$BUILD/cmd/protocol-probe/latest_client_shop_wire_trace.go"
python3 tools/stage37_current_recovery_stage22_wire_trace_patch.py
gofmt -w "$BUILD/internal/shopwiretrace" "$BUILD/cmd/protocol-probe/latest_client_shop_wire_trace.go" "$BUILD/cmd/protocol-probe/main.go"

python3 - <<'PY' > "$ROOT/stage22_shop_wire_trace_audit.log"
from pathlib import Path
main = Path('buildtree/cmd/protocol-probe/main.go').read_text()
overlay = Path('buildtree/cmd/protocol-probe/zz_recovered_overlay.go').read_text()
trace = Path('buildtree/cmd/protocol-probe/latest_client_shop_wire_trace.go').read_text()
if main.count('traceShopWireCustom(custom, conn.RemoteAddr().String())') != 2:
    raise SystemExit('Stage22 custom observer callsite count != 2')
if 'args=[%s]' in main and 'CustomSend opcode=' in main:
    raise SystemExit('Stage22 unredacted CustomSend argument logging remains')
if 'case 70:' not in main or 'handleShopBuyCustom(' not in main:
    raise SystemExit('Stage22 recovered selector dispatch drift')
start=overlay.index('func handleShopBuyCustom(')
end=overlay.index('\nfunc shopCatalogItems(',start)
handler=overlay[start:end]
for token in ['player.addGold(', 'player.addSilver(', 'player.addSilverCard(', 'player.addBagItem(', 'writeFrames(', 'persistBagEquip(', 'currencyStore.Save(']:
    if token in handler: raise SystemExit('Stage22 legacy mutation returned: '+token)
if 'return false, nil' not in handler: raise SystemExit('Stage22 legacy fail-closed return absent')
if 'NINEYIN_SHOP_WIRE_TRACE' not in trace or 'maxDetailedShopWireMessages' not in trace:
    raise SystemExit('Stage22 opt-in/bounded trace absent')
for token in ['player.', 'currencyStore', 'bagStore', 'writeFrames(', 'persistBagEquip(', 'shopbuygate.Bound']:
    if token in trace: raise SystemExit('Stage22 observer has mutation vocabulary: '+token)
print('stage22_telemetry_audit=PASS')
print('ordinary_custom_observer_calls=1')
print('activity_custom_observer_calls=1')
print('raw_custom_arguments_logging=REMOVED')
print('default_trace=SELECTOR_AND_TYPE_SHAPE_ONLY')
print('detail_opt_in=NINEYIN_SHOP_WIRE_TRACE=1')
print('detail_budget_per_process=8192')
print('arbitrary_text=REDACTED_SHA256_PREFIX')
print('legacy_candidate_selector=0x46')
print('legacy_candidate_mutation=DISABLED_FAIL_CLOSED')
print('selector_authority=NOT_ESTABLISHED_BY_TELEMETRY_CODE')
PY

cat > "$ROOT/stage22_shop_wire_trace_status.log" <<'STATUS'
stage22_scope=read_only_current_custom_message_wire_observation
ordinary_custom_opcode=0x1E_or_0x27
activity_custom_opcode=0x0A
raw_decoded_argument_logs=REPLACED_WITH_REDACTED_TYPE_SHAPE
trace_default=TYPE_SHAPE_ONLY
trace_detail_opt_in=NINEYIN_SHOP_WIRE_TRACE=1
trace_detail_max_per_process=8192
unknown_selector_observation=ENABLED_WHEN_DECODED
authoritative_selector_verification=REQUIRES_EXACT_CURRENT_LIVE_CORRELATION
current_client_selector_verified=0
wire_gate_status=BLOCKED_NO_EXACT_CURRENT_CLIENT_AUTHORITY_IN_REPO
safe_production_handler_wiring=OFF
legacy_candidate_selector=0x46
legacy_candidate_mutation_route=DISABLED_FAIL_CLOSED
gm_grant_changes=0
STATUS

(cd "$BUILD" && find . -type f ! -name 'ci.real.mod' ! -name 'ci.real.sum' ! -name 'protocol-probe' -printf '%P\0' | sort -z | xargs -0 sha256sum > "$ROOT/generated-manifest.txt")
sha256sum "$ROOT/generated-manifest.txt" > "$ROOT/generated-manifest.sha256"
source_files=$(find "$BUILD" -type f ! -name 'ci.real.mod' ! -name 'ci.real.sum' ! -name 'protocol-probe' | wc -l | tr -d ' ')
echo "stage22_source_files=$source_files"
echo "stage22_manifest_sha256=$(awk '{print $1}' "$ROOT/generated-manifest.sha256")"
if [ "$source_files" != '231' ]; then echo "Stage22 unexpected source count: $source_files" >&2; exit 96; fi

cd "$BUILD"
set +e
run_gate() {
    local name="$1"; shift
    "$@" >"$ROOT/${name}.log" 2>&1
    echo $? >"$ROOT/${name}.exit"
}
run_gate shopwiretrace go test -modfile=ci.real.mod ./internal/shopwiretrace -count=1
run_gate shopwiretrace_race go test -modfile=ci.real.mod -race ./internal/shopwiretrace -count=1
run_gate shopbuypublish go test -modfile=ci.real.mod ./internal/shopbuypublish -count=1
run_gate shopbuypublish_race go test -modfile=ci.real.mod -race ./internal/shopbuypublish -count=1
run_gate shopbuycommit go test -modfile=ci.real.mod ./internal/shopbuycommit -count=1
run_gate shopbuycommit_race go test -modfile=ci.real.mod -race ./internal/shopbuycommit -count=1
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
for x in shopwiretrace shopwiretrace_race shopbuypublish shopbuypublish_race shopbuycommit shopbuycommit_race build protocol_compile protocol_race_compile nonprotocol race vet windows; do
    code=$(cat "$ROOT/$x.exit")
    echo "$x=$code"
    if [ "$code" -ne 0 ]; then failed=1; fi
done
protocol=$(cat "$ROOT/protocol_runtime.exit")
echo "protocol_runtime=$protocol"
if [ "$protocol" -ne 0 ] && [ "$protocol" -ne 125 ]; then failed=1; fi

echo 'current_client_selector_verified=0'
echo 'legacy_candidate_mutation_route=DISABLED_FAIL_CLOSED'
echo 'safe_production_handler_wiring=OFF'
exit "$failed"
