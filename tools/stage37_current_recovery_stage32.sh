#!/usr/bin/env bash
set -euo pipefail
ROOT="$(pwd)"
# Stage31 source and checks are the authority. Do not replace the RAR lineage.
bash tools/stage37_current_recovery_stage31.sh
PROBE="$ROOT/buildtree/cmd/protocol-probe"
printf '%s  %s\n' 'a0b2a4a0b9cffac0b18fc98a8cb5eaceae83d9edf260eb166468b99892546d70' "$PROBE/scene_lifecycle.go" | sha256sum -c -
printf '%s  %s\n' '3ed975fb54435b9612dcc88a2e058f4d81440ea4d4ebca9e481be4c6b7cbb4c9' "$PROBE/shop_catalog.go" | sha256sum -c -
git apply --directory=buildtree --check recovery/stage32_npc_shop_selection_diagnostic.patch
git apply --directory=buildtree recovery/stage32_npc_shop_selection_diagnostic.patch
printf '%s  %s\n' '5e58a9185019617cada73eb7a64d3762f1596e3021be8cb28e30f0599ace9db3' "$PROBE/scene_lifecycle.go" | sha256sum -c -
printf '%s  %s\n' 'aee89c22932f0211081ef745b050eb2faaa817871a32ce03b19615d21f6dfcdd' "$PROBE/stage32_shop_menu_diagnostic.go" | sha256sum -c -
printf '%s  %s\n' 'c71a095b5d7de15f64bf2ff4c697945f81f31096c253b09dcc4f326bd6050ef6' "$PROBE/stage32_shop_menu_diagnostic_test.go" | sha256sum -c -
test -z "$(gofmt -l "$PROBE/scene_lifecycle.go" "$PROBE/stage32_shop_menu_diagnostic.go" "$PROBE/stage32_shop_menu_diagnostic_test.go")"
python3 - <<'PY'
from pathlib import Path
s = Path('buildtree/cmd/protocol-probe/scene_lifecycle.go').read_text()
assert s.index('NPC shop menu diagnostic object=') < s.index('if entity.interaction != npcInteractionTalk {')
assert 'shop service selected npc_config=' in s
shop = s[s.index('func (s *sceneLifecycle) openShopLocked('):s.index('\ntype talkMenuItem struct {')]
assert 'stage31PrepareShopItemFrames(shopID, displayRows,' in shop
assert 'client_render=unverified' in shop
assert 'NINEYIN_SHOP_EXCHANGE_VIEW_AB' in shop
assert (Path('buildtree/cmd/protocol-probe/main.go').read_text().count('stage29LegacyBorn02Position(')) == 2
assert 'ordinary shop candidate selector=0x46 blocked' in Path('buildtree/cmd/protocol-probe/zz_recovered_overlay.go').read_text()
print('stage32_source_guard=PASS_READ_ONLY_NPC_SHOP_SELECTION')
PY
UNIT="$RUNNER_TEMP/stage32-npc-shop-unit"
mkdir -p "$UNIT"
cp "$PROBE/shop_catalog.go" "$PROBE/stage32_shop_menu_diagnostic.go" "$PROBE/stage32_shop_menu_diagnostic_test.go" "$UNIT/"
printf 'package main\nconst defaultModernShareRoot = ""\n' > "$UNIT/stub.go"
(
 cd "$UNIT"
 GO111MODULE=off go test -count=1 -v *.go > "$ROOT/stage32_unit.log" 2>&1
 GO111MODULE=off go test -race -count=1 *.go > "$ROOT/stage32_unit_race.log" 2>&1
)
cd "$ROOT/buildtree"
go test -modfile=ci.real.mod -c -o "$ROOT/stage32-protocol-probe.test" ./cmd/protocol-probe > "$ROOT/stage32_compile.log" 2>&1
go test -race -modfile=ci.real.mod -c -o "$ROOT/stage32-protocol-probe-race.test" ./cmd/protocol-probe > "$ROOT/stage32_race_compile.log" 2>&1
go vet -modfile=ci.real.mod ./cmd/protocol-probe > "$ROOT/stage32_vet.log" 2>&1
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -modfile=ci.real.mod -o "$ROOT/stage32-current-windows-amd64.exe" ./cmd/protocol-probe > "$ROOT/stage32_windows.log" 2>&1
sha256sum "$ROOT/stage32-current-windows-amd64.exe" > "$ROOT/stage32-current-windows-amd64.sha256"
file "$ROOT/stage32-current-windows-amd64.exe" > "$ROOT/stage32-current-windows-amd64.file.txt"
{
 echo 'source_basis=VERIFIED_9yin-go-server1.rar_STAGE31_PRESERVED'
 echo 'npc_shop_menu=EXACT_SELECTED_SHOP_ID_AND_CATALOG_COUNTS_LOGGED'
 echo 'missing_catalog_id=LOGGED_NO_GUESSED_SUFFIX'
 echo 'npc_shop_selection=UNMODIFIED'
 echo 'shop_display=STAGE31_UNMODIFIED'
 echo 'shop_exchange_display=STAGE30_OPT_IN_UNMODIFIED'
 echo 'shop_purchase_currency_bag_DB=UNCHANGED_FAIL_CLOSED'
 echo 'map_lua_stage29=UNCHANGED'
 echo 'gm_web_item_grant=UNCHANGED_DEFERRED'
 echo 'isolated_tests_and_race=PASS_EXACT_RAR_CASE_SKIPPED_IN_CI'
 echo 'full_server_tests=COMPILE_ONLY_RESOURCE_DEPENDENT_INIT_NOT_RUN'
 echo 'full_server_race=COMPILE_ONLY'
 echo 'vet=PASS'
 echo 'windows_amd64_build=PASS'
 echo 'client_shop_render=UNVERIFIED'
 echo 'live_e2e=NOT_RUN'
} > "$ROOT/stage32_status.log"
cat "$ROOT/stage32_status.log"
cat "$ROOT/stage32-current-windows-amd64.sha256"
