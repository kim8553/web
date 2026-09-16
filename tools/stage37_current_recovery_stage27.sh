#!/usr/bin/env bash
set -euo pipefail
ROOT="$(pwd)"

# Reconstruct Stage23 exactly, keeping its unverified-purchase fail-closed gates.
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
TEST_SOURCE="$ROOT/buildtree/cmd/protocol-probe/stage27_shop_observability_test.go"
cp recovery/postbuild_files/cmd__protocol-probe__stage27_shop_observability_test.go "$TEST_SOURCE"
test -z "$(gofmt -l "$SOURCE" "$TEST_SOURCE")"

# New code observes the ordinary display filter but cannot authorize purchases.
python3 - <<'PY'
from pathlib import Path
s = Path('buildtree/cmd/protocol-probe/scene_lifecycle.go').read_text()
assert 'case 0, 1, 2:\n\t\tdefault:\n\t\t\tcontinue' in s
assert s.count('func (s *sceneLifecycle) openShopLocked(shopID string) error') == 1
assert 'client_render=unverified' in s
assert 'shop service selected npc_config=' in s
print('stage27_shop_observer_source_guard=PASS')
PY

# Full server package init requires exact-current skill_new.ini, absent in CI.
# Compile full-package tests without executing init, and run verbatim extracted
# pure observer tests without substituting old or invented client resources.
UNIT_DIR="$RUNNER_TEMP/stage27-verbatim-unit"
python3 tools/stage37_stage27_isolated_test.py "$SOURCE" "$TEST_SOURCE" "$UNIT_DIR"
(
  cd "$UNIT_DIR"
  GO111MODULE=off go test -count=1 -v stage27_isolated.go stage27_isolated_test.go > "$ROOT/stage27_isolated_test.log" 2>&1
  GO111MODULE=off go test -race -count=1 stage27_isolated.go stage27_isolated_test.go > "$ROOT/stage27_isolated_race.log" 2>&1
)
cd "$ROOT/buildtree"
go test -modfile=ci.real.mod -c -o "$ROOT/stage27-protocol-probe.test" ./cmd/protocol-probe > "$ROOT/stage27_protocol_test_compile.log" 2>&1
go test -race -modfile=ci.real.mod -c -o "$ROOT/stage27-protocol-probe-race.test" ./cmd/protocol-probe > "$ROOT/stage27_protocol_race_compile.log" 2>&1
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
  echo 'verbatim_isolated_regressions=PASS'
  echo 'verbatim_isolated_race=PASS'
  echo 'full_protocol_test_compile=PASS_RUNTIME_NOT_RUN'
  echo 'full_protocol_race_compile=PASS_RUNTIME_NOT_RUN'
  echo 'full_protocol_runtime=BLOCKED_EXACT_CURRENT_SKILL_RESOURCE_ABSENT'
  echo 'vet=PASS'
  echo 'windows_amd64_build=PASS'
  echo 'client_shop_render=UNVERIFIED'
  echo 'live_e2e=NOT_RUN'
} > "$ROOT/stage27_status.log"
cat "$ROOT/stage27_status.log"
cat "$ROOT/stage27-current-windows-amd64.sha256"
