#!/usr/bin/env bash
set -euo pipefail
ROOT="$(pwd)"

# Reconstruct Stage23 unmodified, including its 0x46 fail-closed checks.
bash tools/stage37_current_recovery_stage23.sh
SOURCE="$ROOT/buildtree/cmd/protocol-probe/scene_lifecycle.go"
BASE_SHA=0af96878e7e3aa7c84fdb275d96b62e238e432bc3ff447c2c99944544f0e0666
PATCHED_SHA=b29f012085f0cfaa511d609fcfa66152b88746a2ece4dd6931a6297ae252b009
printf '%s  %s\n' "$BASE_SHA" "$SOURCE" | sha256sum -c -
test "$(awk '{print $1}' "$ROOT/generated-manifest.sha256")" = bc2aeaf77ce9d9b198d725356b976599063b12c3324fc548e41298c8c7e46d9a
grep -Fx 'safe_production_handler_wiring=OFF' "$ROOT/stage22_shop_wire_trace_status.log"
grep -Fx 'legacy_candidate_mutation_route=DISABLED_FAIL_CLOSED' "$ROOT/stage22_shop_wire_trace_status.log"

git apply --directory=buildtree --check recovery/stage27_shop_display_diagnostics.patch
git apply --directory=buildtree recovery/stage27_shop_display_diagnostics.patch
printf '%s  %s\n' "$PATCHED_SHA" "$SOURCE" | sha256sum -c -
cp recovery/postbuild_files/cmd__protocol-probe__stage27_shop_observability_test.go "$ROOT/buildtree/cmd/protocol-probe/stage27_shop_observability_test.go"
test -z "$(gofmt -l "$SOURCE" "$ROOT/buildtree/cmd/protocol-probe/stage27_shop_observability_test.go")"

# The new code observes the already existing ordinary display filter; it must
# not enable mode-3 exchange item delivery, any buy selector or state mutation.
python3 - <<'PY'
from pathlib import Path
s = Path('buildtree/cmd/protocol-probe/scene_lifecycle.go').read_text()
assert 'case 0, 1, 2:\n\t\tdefault:\n\t\t\tcontinue' in s
assert s.count('func (s *sceneLifecycle) openShopLocked(shopID string) error') == 1
assert 'client_render=unverified' in s
assert 'shop service selected npc_config=' in s
print('stage27_shop_observer_source_guard=PASS')
PY

cd "$ROOT/buildtree"
go test -modfile=ci.real.mod ./cmd/protocol-probe -run '^TestStage27ShopDisplaySummary' -count=1 > "$ROOT/stage27_targeted_test.log" 2>&1
go test -modfile=ci.real.mod -race ./cmd/protocol-probe -run '^TestStage27ShopDisplaySummary' -count=1 > "$ROOT/stage27_targeted_race.log" 2>&1
go vet -modfile=ci.real.mod ./cmd/protocol-probe > "$ROOT/stage27_vet.log" 2>&1
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -modfile=ci.real.mod -o "$ROOT/stage27-current-windows-amd64.exe" ./cmd/protocol-probe > "$ROOT/stage27_windows.log" 2>&1
sha256sum "$ROOT/stage27-current-windows-amd64.exe" > "$ROOT/stage27-current-windows-amd64.sha256"
file "$ROOT/stage27-current-windows-amd64.exe" > "$ROOT/stage27-current-windows-amd64.file.txt"
{
  echo 'stage27_scope=read_only_npc_shop_display_observability'
  echo 'stage23_baseline_manifest=UNCHANGED'
  echo 'ordinary_shop_modes=0,1,2_UNCHANGED'
  echo 'exchange_mode_3=SKIPPED_UNCHANGED'
  echo 'purchase_selector=UNVERIFIED_FAIL_CLOSED'
  echo 'selected_role_currency=UNMODIFIED'
  echo 'targeted_regressions=PASS'
  echo 'targeted_race=PASS'
  echo 'vet=PASS'
  echo 'windows_amd64_build=PASS'
  echo 'client_shop_render=UNVERIFIED'
  echo 'live_e2e=NOT_RUN'
} > "$ROOT/stage27_status.log"
cat "$ROOT/stage27_status.log"
cat "$ROOT/stage27-current-windows-amd64.sha256"
