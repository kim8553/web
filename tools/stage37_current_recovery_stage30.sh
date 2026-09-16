#!/usr/bin/env bash
set -euo pipefail
ROOT="$(pwd)"

# Retain the verified Stage29 map/Lua source, Stage27 ordinary shop path, and
# fail-closed purchase gates. This A/B is OFF unless explicitly enabled.
bash tools/stage37_current_recovery_stage29.sh
PROBE="$ROOT/buildtree/cmd/protocol-probe"
printf '%s  %s\n' 'b29f012085f0cfaa511d609fcfa66152b88746a2ece4dd6931a6297ae252b009' "$PROBE/scene_lifecycle.go" | sha256sum -c -
printf '%s  %s\n' 'b6008176c804472e8e1b076195342fd4360e182efda55477efb587906c80d132' "$PROBE/latest_client_shop_view_contract.go" | sha256sum -c -
git apply --directory=buildtree --check recovery/stage30_shop_exchange_display_ab.patch
git apply --directory=buildtree recovery/stage30_shop_exchange_display_ab.patch
printf '%s  %s\n' '240cc47d133cbd1fc77a502d3da4d0386aedc1088ea125c2f3895f346ecfc380' "$PROBE/scene_lifecycle.go" | sha256sum -c -
printf '%s  %s\n' 'cd68f20ebadafa7ce13254d7f8758f489be6ac7445005d00bceb3fc690531573' "$PROBE/stage30_shop_exchange_display_ab.go" | sha256sum -c -
printf '%s  %s\n' '06f59b15f952428a28addc8f3f02e26aa3975c7ef79ee33f28fc6ad597dc1c29' "$PROBE/stage30_shop_exchange_display_ab_test.go" | sha256sum -c -
test -z "$(gofmt -l "$PROBE/scene_lifecycle.go" "$PROBE/stage30_shop_exchange_display_ab.go" "$PROBE/stage30_shop_exchange_display_ab_test.go")"
python3 - <<'PY'
from pathlib import Path
s = Path('buildtree/cmd/protocol-probe/scene_lifecycle.go').read_text()
assert 'os.Getenv("NINEYIN_SHOP_EXCHANGE_VIEW_AB") == "1"' in s
assert 'displayRows, exchangeStats := stage30ShopExchangeDisplayRows(items, exchangeAB)' in s
assert 'for _, item := range displayRows {' in s
assert 'mutation=unchanged_client_render=unverified' in s
m = Path('buildtree/cmd/protocol-probe/main.go').read_text()
assert m.count('stage29LegacyBorn02Position(') == 2
assert 'ignored early 0x0A while target scene is loading' in m
assert 'promoted later target-scene 0x0A' not in m
assert 'clientCustomExchangeFromShop' in Path('buildtree/cmd/protocol-probe/latest_client_shop_exchange_contract.go').read_text()
print('stage30_shop_ab_source_guard=PASS')
PY

# Run verbatim source helper and tests without resource-dependent package init.
# The object index helper is extracted unchanged from the exact SHA-guarded
# current view contract instead of inventing a replacement implementation.
UNIT="$RUNNER_TEMP/stage30-shop-display-unit"
mkdir -p "$UNIT"
cp "$PROBE/shop_catalog.go" "$PROBE/stage30_shop_exchange_display_ab.go" "$PROBE/stage30_shop_exchange_display_ab_test.go" "$UNIT/"
python3 - "$PROBE/latest_client_shop_view_contract.go" "$UNIT/stage30_view_stub.go" <<'PY'
from pathlib import Path
import sys
source = Path(sys.argv[1]).read_text()
assert 'const currentShopPageSize int32 = 500' in source
needle = 'func currentShopViewObjectIndex(item shopCatalogItem) (uint16, bool) {'
assert source.count(needle) == 1
start = source.index(needle)
brace = source.index('{', start)
depth = 0
end = None
for i in range(brace, len(source)):
    if source[i] == '{': depth += 1
    elif source[i] == '}':
        depth -= 1
        if depth == 0:
            end = i + 1
            break
assert end is not None
Path(sys.argv[2]).write_text('package main\n\nvar defaultModernShareRoot = ""\nconst currentShopPageSize int32 = 500\n\n' + source[start:end] + '\n')
PY
(cd "$UNIT" && GO111MODULE=off go test -count=1 -v shop_catalog.go stage30_shop_exchange_display_ab.go stage30_shop_exchange_display_ab_test.go stage30_view_stub.go > "$ROOT/stage30_unit.log" 2>&1 && GO111MODULE=off go test -race -count=1 shop_catalog.go stage30_shop_exchange_display_ab.go stage30_shop_exchange_display_ab_test.go stage30_view_stub.go > "$ROOT/stage30_unit_race.log" 2>&1)

cd "$ROOT/buildtree"
go test -modfile=ci.real.mod -c -o "$ROOT/stage30-protocol-probe.test" ./cmd/protocol-probe > "$ROOT/stage30_compile.log" 2>&1
go test -race -modfile=ci.real.mod -c -o "$ROOT/stage30-protocol-probe-race.test" ./cmd/protocol-probe > "$ROOT/stage30_race_compile.log" 2>&1
go vet -modfile=ci.real.mod ./cmd/protocol-probe > "$ROOT/stage30_vet.log" 2>&1
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -modfile=ci.real.mod -o "$ROOT/stage30-current-windows-amd64.exe" ./cmd/protocol-probe > "$ROOT/stage30_windows.log" 2>&1
sha256sum "$ROOT/stage30-current-windows-amd64.exe" > "$ROOT/stage30-current-windows-amd64.sha256"
file "$ROOT/stage30-current-windows-amd64.exe" > "$ROOT/stage30-current-windows-amd64.file.txt"
{
 echo 'source_basis=VERIFIED_9yin-go-server1.rar_STAGE29_PRESERVED'
 echo 'map_lua_stage29=UNCHANGED'
 echo 'ordinary_shop_default=STAGE27_UNCHANGED'
 echo 'exchange_display=OPT_IN_NINEYIN_SHOP_EXCHANGE_VIEW_AB_1'
 echo 'exchange_purchase_mutation=UNCHANGED_FAIL_CLOSED'
 echo 'exchange_missing_data_invalid_slot_collision_capacity=GUARDED'
 echo 'gm_web_item_grant=UNCHANGED_DEFERRED'
 echo 'isolated_source_tests=PASS'
 echo 'isolated_race=PASS'
 echo 'full_server_tests=COMPILE_ONLY_RESOURCE_DEPENDENT_INIT_NOT_RUN'
 echo 'full_server_race=COMPILE_ONLY'
 echo 'vet=PASS'
 echo 'windows_amd64_build=PASS'
 echo 'client_shop_render=UNVERIFIED'
 echo 'live_e2e=NOT_RUN'
} > "$ROOT/stage30_status.log"
cat "$ROOT/stage30_status.log"
cat "$ROOT/stage30-current-windows-amd64.sha256"
