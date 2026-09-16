#!/usr/bin/env bash
set -euo pipefail
ROOT="$(pwd)"
# Preserve verified RAR-descendant Stage32, Stage29 map/Lua, Stage30 exchange
# display (opt-in OFF), Stage31 preflight, and Stage32 selected-NPC logs.
bash tools/stage37_current_recovery_stage32.sh
PROBE="$ROOT/buildtree/cmd/protocol-probe"
printf '%s  %s\n' '3ed975fb54435b9612dcc88a2e058f4d81440ea4d4ebca9e481be4c6b7cbb4c9' "$PROBE/shop_catalog.go" | sha256sum -c -
printf '%s  %s\n' '5da04ccd051db7cc27a6205db789939a5f0d65a3bd34797a3c2fc73a616fb280' recovery/stage34_shop_pagecount_reconcile.patch | sha256sum -c -
printf '%s  %s\n' 'b2acf77323cb69a5459485443756b33af91766fdab6a7c1ddbc4ce152bb307c2' recovery/stage34_shop_pagecount_test.go | sha256sum -c -
git apply --directory=buildtree --check recovery/stage34_shop_pagecount_reconcile.patch
git apply --directory=buildtree recovery/stage34_shop_pagecount_reconcile.patch
cp recovery/stage34_shop_pagecount_test.go "$PROBE/stage34_shop_pagecount_test.go"
printf '%s  %s\n' 'b683cfcf00a1b6b2d3bf1b37aceb266f2555e82fa91a181bdeb9bc702c1a8e51' "$PROBE/shop_catalog.go" | sha256sum -c -
test -z "$(gofmt -l "$PROBE/shop_catalog.go" "$PROBE/stage34_shop_pagecount_test.go")"
python3 - <<'PY'
from pathlib import Path
p=Path('buildtree/cmd/protocol-probe')
s=(p/'shop_catalog.go').read_text()
assert 'authoredPageCount := maxPageKey + 1; authoredPageCount > pageCount' in s
assert 'shopCatalogItems(defaultShopINIPath, shopID)' in (p/'scene_lifecycle.go').read_text()
assert 'stage31PrepareShopItemFrames(shopID, displayRows,' in (p/'scene_lifecycle.go').read_text()
assert (p/'main.go').read_text().count('stage29LegacyBorn02Position(')==2
assert 'ordinary shop candidate selector=0x46 blocked' in (p/'zz_recovered_overlay.go').read_text()
print('stage34_source_guard=PASS_ONLY_AUTHORED_PAGECOUNT_RECONCILIATION')
PY
# The exact parser and exact regression file run in isolation, without CI-only
# private resources or resource-dependent package initialization.
UNIT="$RUNNER_TEMP/stage34-shop-pages-unit"
mkdir -p "$UNIT"
cp "$PROBE/shop_catalog.go" "$PROBE/stage34_shop_pagecount_test.go" "$UNIT/"
printf 'package main\nconst defaultModernShareRoot = ""\n' > "$UNIT/stub.go"
(
  cd "$UNIT"
  GO111MODULE=off go test -count=1 -v *.go > "$ROOT/stage34_unit.log" 2>&1
  GO111MODULE=off go test -race -count=1 *.go > "$ROOT/stage34_unit_race.log" 2>&1
)
cd "$ROOT/buildtree"
go test -modfile=ci.real.mod -c -o "$ROOT/stage34-protocol-probe.test" ./cmd/protocol-probe > "$ROOT/stage34_compile.log" 2>&1
go test -race -modfile=ci.real.mod -c -o "$ROOT/stage34-protocol-probe-race.test" ./cmd/protocol-probe > "$ROOT/stage34_race_compile.log" 2>&1
go vet -modfile=ci.real.mod ./cmd/protocol-probe > "$ROOT/stage34_vet.log" 2>&1
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -modfile=ci.real.mod -o "$ROOT/stage34-current-windows-amd64.exe" ./cmd/protocol-probe > "$ROOT/stage34_windows.log" 2>&1
sha256sum "$ROOT/stage34-current-windows-amd64.exe" > "$ROOT/stage34-current-windows-amd64.sha256"
file "$ROOT/stage34-current-windows-amd64.exe" > "$ROOT/stage34-current-windows-amd64.file.txt"
{
 echo 'source_basis=VERIFIED_9yin-go-server1.rar_STAGE32_PRESERVED'
 echo 'stage34_authored_pagecount_reconciliation=ENABLED_ONLY_WHEN_PAGEINFO_UNDERDECLARES_AUTHORED_PAGE'
 echo 'shop_special_001=ONE_KNOWN_RAR_INCONSISTENCY_EXACT_RAR_TESTED_OFFLINE_NOT_IN_CI'
 echo 'npc_shop_selection=STAGE32_UNCHANGED'
 echo 'ordinary_shop_purchase=UNCHANGED_FAIL_CLOSED_UNVERIFIED_WIRE'
 echo 'exchange_display=STAGE30_OPT_IN_OFF'
 echo 'map_lua_stage29=PRESERVED'
 echo 'gm_web_item_grant=UNCHANGED_DEFERRED'
 echo 'isolated_tests_and_race=PASS'
 echo 'full_server_tests=COMPILE_ONLY_RESOURCE_DEPENDENT_INIT_NOT_RUN'
 echo 'full_server_race=COMPILE_ONLY'
 echo 'vet=PASS'
 echo 'windows_amd64_build=PASS'
 echo 'client_shop_render=UNVERIFIED'
 echo 'live_e2e=NOT_RUN'
} > "$ROOT/stage34_status.log"
cat "$ROOT/stage34_status.log"
cat "$ROOT/stage34-current-windows-amd64.sha256"
