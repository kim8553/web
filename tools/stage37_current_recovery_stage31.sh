#!/usr/bin/env bash
set -euo pipefail
ROOT="$(pwd)"

# Preserve verified 9yin-go-server1.rar lineage, Stage29 map/Lua remap, Stage30
# opt-in exchange DISPLAY and the unverified-purchase fail-closed gates.
bash tools/stage37_current_recovery_stage30.sh
PROBE="$ROOT/buildtree/cmd/protocol-probe"
BASE_SHA=240cc47d133cbd1fc77a502d3da4d0386aedc1088ea125c2f3895f346ecfc380
printf '%s  %s\n' "$BASE_SHA" "$PROBE/scene_lifecycle.go" | sha256sum -c -
printf '%s  %s\n' 'b6008176c804472e8e1b076195342fd4360e182efda55477efb587906c80d132' "$PROBE/latest_client_shop_view_contract.go" | sha256sum -c -
printf '%s  %s\n' '7a406c6d9ac78999a7a31b5a79a489196c1bc3a63c4e5640e5e42b9c3220bbe0' recovery/stage31_shop_frame_preflight.patch | sha256sum -c -
git apply --directory=buildtree --check recovery/stage31_shop_frame_preflight.patch
git apply --directory=buildtree recovery/stage31_shop_frame_preflight.patch
printf '%s  %s\n' '37140cd2ea22f6d5c2b670818abd01a0716328815c15748970f02b30670435aa' "$PROBE/scene_lifecycle.go" | sha256sum -c -
printf '%s  %s\n' 'babc4d4d86ae1d451418e088cf0ce8ccabb39c1940c78051c6e5a2a0584d21d4' "$PROBE/stage31_shop_frame_preflight.go" | sha256sum -c -
printf '%s  %s\n' 'da1f538cee814eedcc34c76c523c664737b87379024f8fee2c2cc5323a60c44d' "$PROBE/stage31_shop_frame_preflight_test.go" | sha256sum -c -
test -z "$(gofmt -l "$PROBE/scene_lifecycle.go" "$PROBE/stage31_shop_frame_preflight.go" "$PROBE/stage31_shop_frame_preflight_test.go")"

python3 - <<'PY'
from pathlib import Path
p=Path('buildtree/cmd/protocol-probe')
s=(p/'scene_lifecycle.go').read_text()
shop=s[s.index('func (s *sceneLifecycle) openShopLocked('):s.index('\ntype talkMenuItem struct {')]
assert 'stage31PrepareShopItemFrames(shopID, displayRows,' in shop
assert shop.index('stage31PrepareShopItemFrames(') < shop.index('s.conn.WriteFrame(frame)')
assert 'for index, itemFrame := range itemFrames {' in shop
assert 'shop service selected npc_config=' in s
assert 'func playableNPCServices(' in s
assert 'client_render=unverified' in shop
assert 'os.Getenv("NINEYIN_SHOP_EXCHANGE_VIEW_AB") == "1"' in shop
assert (p/'main.go').read_text().count('stage29LegacyBorn02Position(') == 2
print('stage31_shop_preflight_source_guard=PASS')
PY

# Verbatim helper and tests; copy the exact, SHA-guarded object-index function
# from the current view contract, not a substituted or invented implementation.
UNIT="$RUNNER_TEMP/stage31-shop-preflight-unit"
mkdir -p "$UNIT"
cp "$PROBE/shop_catalog.go" "$PROBE/stage31_shop_frame_preflight.go" "$PROBE/stage31_shop_frame_preflight_test.go" "$UNIT/"
python3 - "$PROBE/latest_client_shop_view_contract.go" "$UNIT/stage31_exact_index.go" <<'PY'
from pathlib import Path
import sys
source=Path(sys.argv[1]).read_text()
assert 'const currentShopPageSize int32 = 500' in source
needle='func currentShopViewObjectIndex(item shopCatalogItem) (uint16, bool) {'
assert source.count(needle)==1
start=source.index(needle)
brace=source.index('{',start)
depth=0
end=None
for i in range(brace,len(source)):
    if source[i]=='{': depth+=1
    elif source[i]=='}':
        depth-=1
        if depth==0:
            end=i+1
            break
assert end is not None
Path(sys.argv[2]).write_text('package main\n\nconst defaultModernShareRoot = ""\nconst currentShopPageSize int32 = 500\n\n'+source[start:end]+'\n')
PY
(
 cd "$UNIT"
 GO111MODULE=off go test -count=1 -v *.go > "$ROOT/stage31_unit.log" 2>&1
 GO111MODULE=off go test -race -count=1 *.go > "$ROOT/stage31_unit_race.log" 2>&1
)

cd "$ROOT/buildtree"
go test -modfile=ci.real.mod -c -o "$ROOT/stage31-protocol-probe.test" ./cmd/protocol-probe > "$ROOT/stage31_compile.log" 2>&1
go test -race -modfile=ci.real.mod -c -o "$ROOT/stage31-protocol-probe-race.test" ./cmd/protocol-probe > "$ROOT/stage31_race_compile.log" 2>&1
go vet -modfile=ci.real.mod ./cmd/protocol-probe > "$ROOT/stage31_vet.log" 2>&1
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -modfile=ci.real.mod -o "$ROOT/stage31-current-windows-amd64.exe" ./cmd/protocol-probe > "$ROOT/stage31_windows.log" 2>&1
sha256sum "$ROOT/stage31-current-windows-amd64.exe" > "$ROOT/stage31-current-windows-amd64.sha256"
file "$ROOT/stage31-current-windows-amd64.exe" > "$ROOT/stage31-current-windows-amd64.file.txt"
{
 echo 'source_basis=VERIFIED_9yin-go-server1.rar_STAGE30_PRESERVED'
 echo 'shop_frames=VALIDATED_AND_ENCODED_BEFORE_VIEW_WRITE'
 echo 'ordinary_shop_default=UNCHANGED_EXCEPT_PREVENT_PARTIAL_VIEWS'
 echo 'exchange_display=STAGE30_OPT_IN_UNCHANGED'
 echo 'shop_purchase_currency_bag_DB=UNCHANGED_FAIL_CLOSED'
 echo 'map_lua_stage29=UNCHANGED'
 echo 'gm_web_item_grant=UNCHANGED_DEFERRED'
 echo 'verbatim_isolated_tests=PASS'
 echo 'verbatim_isolated_race=PASS'
 echo 'full_server_tests=COMPILE_ONLY_RESOURCE_DEPENDENT_INIT_NOT_RUN'
 echo 'full_server_race=COMPILE_ONLY'
 echo 'vet=PASS'
 echo 'windows_amd64_build=PASS'
 echo 'client_shop_render=UNVERIFIED'
 echo 'live_e2e=NOT_RUN'
} > "$ROOT/stage31_status.log"
cat "$ROOT/stage31_status.log"
cat "$ROOT/stage31-current-windows-amd64.sha256"
