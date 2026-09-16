#!/usr/bin/env bash
set -euo pipefail
ROOT="$(pwd)"

# Keep the verified Stage27 NPC-shop observer and its fail-closed purchase gates.
bash tools/stage37_current_recovery_stage27.sh
PROBE="$ROOT/buildtree/cmd/protocol-probe"
printf '%s  %s\n' '4bc4bea45c73c63fddbcd05f6f0bfbe36a5a7bae8235d966ed2621b51208ec59' "$PROBE/main.go" | sha256sum -c -
printf '%s  %s\n' '59baeadb4352484e31302751498b4c4c474dd6a9aaf52c1b685b4fce06cf3f7a' "$PROBE/latest_client_player_property_ordinal_compat.go" | sha256sum -c -
printf '%s  %s\n' '78788365b3bcce090bff747e9307ff49d2d89740c38871a945176b78636bfae6' "$PROBE/scene_transition.go" | sha256sum -c -
git apply --directory=buildtree --check recovery/stage29_legacy_born02_preservation.patch
git apply --directory=buildtree recovery/stage29_legacy_born02_preservation.patch
printf '%s  %s\n' 'ddbea23a7db5151f54a55ba1af7ad6652b2395309b954dbe707cb3117363feb3' "$PROBE/main.go" | sha256sum -c -
printf '%s  %s\n' 'c3184ad8861455730c10121b5926ddc75503a2c3b1cf5a588eecd3ab917c614f' "$PROBE/stage29_legacy_born02_position.go" | sha256sum -c -
printf '%s  %s\n' '5fa1421658a17a4ff37f18de9c8d14bb6ea4d90d831785a3c60f5e4b6cd42456' "$PROBE/stage29_legacy_born02_position_test.go" | sha256sum -c -
test -z "$(gofmt -l "$PROBE/main.go" "$PROBE/stage29_legacy_born02_position.go" "$PROBE/stage29_legacy_born02_position_test.go")"

python3 - <<'PY'
from pathlib import Path
p = Path('buildtree/cmd/protocol-probe')
m = (p / 'main.go').read_text()
assert m.count('stage29LegacyBorn02Position(') == 2
assert m.index('stage29LegacyBorn02Position(activeRole.Location.Scene') < m.index('sendPlayerSpawn(link, player')
assert 'stage29LegacyBorn02Position(runtime.activeRole.Location.Scene, position)' in m
assert 'latestClientPlayerWireProperties(rawProperties)' in (p/'player_actor.go').read_text()
assert 'latestClientSceneObjectWireProperties(properties)' in (p/'object_property.go').read_text()
assert 'ignored early 0x0A while target scene is loading' in m
assert 'promoted later target-scene 0x0A' not in m
assert 'client_render=unverified' in (p/'scene_lifecycle.go').read_text()
print('stage29_map_lua_source_guard=PASS')
PY

# Exercise verbatim generated helper and test using the exact role model, with
# no network, fake client resources, or server package init.
UNIT="$RUNNER_TEMP/stage29-map-lua-unit"
mkdir -p "$UNIT/internal/role" "$UNIT/cmd/protocol-probe"
printf 'module github.com/local/9yin-go-server\n\ngo 1.23.0\n' > "$UNIT/go.mod"
cp "$ROOT/buildtree/internal/role/model.go" "$UNIT/internal/role/model.go"
cp "$PROBE/stage29_legacy_born02_position.go" "$PROBE/stage29_legacy_born02_position_test.go" "$UNIT/cmd/protocol-probe/"
(cd "$UNIT" && go test -count=1 -v ./cmd/protocol-probe > "$ROOT/stage29_unit.log" 2>&1 && go test -race -count=1 ./cmd/protocol-probe > "$ROOT/stage29_unit_race.log" 2>&1)

cd "$ROOT/buildtree"
go test -modfile=ci.real.mod -c -o "$ROOT/stage29-protocol-probe.test" ./cmd/protocol-probe > "$ROOT/stage29_compile.log" 2>&1
go test -race -modfile=ci.real.mod -c -o "$ROOT/stage29-protocol-probe-race.test" ./cmd/protocol-probe > "$ROOT/stage29_race_compile.log" 2>&1
go vet -modfile=ci.real.mod ./cmd/protocol-probe > "$ROOT/stage29_vet.log" 2>&1
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -modfile=ci.real.mod -o "$ROOT/stage29-current-windows-amd64.exe" ./cmd/protocol-probe > "$ROOT/stage29_windows.log" 2>&1
sha256sum "$ROOT/stage29-current-windows-amd64.exe" > "$ROOT/stage29-current-windows-amd64.sha256"
file "$ROOT/stage29-current-windows-amd64.exe" > "$ROOT/stage29-current-windows-amd64.file.txt"
{
 echo 'source_basis=VERIFIED_9yin-go-server1.rar_DESCENDANT_STAGE27'
 echo 'player_ordinal_lua_fix=PREEXISTING_PRESERVED'
 echo 'legacy_born02_initial_and_ready=SHARED_LIVE_EVIDENCED_REMAP'
 echo 'scene_reentry_and_late_0x0A=UNCHANGED_NO_DISPROVEN_PROMOTION'
 echo 'npc_shop_observer=STAGE27_PRESERVED'
 echo 'gm_web_item_grant=UNCHANGED_DEFERRED'
 echo 'isolated_helper_and_race=PASS'
 echo 'full_server_tests=COMPILE_ONLY_RESOURCE_DEPENDENT_INIT_NOT_RUN'
 echo 'full_server_race=COMPILE_ONLY'
 echo 'vet=PASS'
 echo 'windows_amd64_build=PASS'
 echo 'live_e2e=NOT_RUN'
} > "$ROOT/stage29_status.log"
cat "$ROOT/stage29_status.log"
cat "$ROOT/stage29-current-windows-amd64.sha256"
