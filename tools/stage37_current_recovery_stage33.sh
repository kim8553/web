#!/usr/bin/env bash
set -euo pipefail
ROOT="$(pwd)"
# Reconstruct exactly the RAR-descendant Stage32 source, including Stage29
# map/Lua preservation, Stage30 opt-in exchange display, Stage31 preflight,
# and Stage32 read-only NPC ShopID diagnostics. No resource/client uploads.
bash tools/stage37_current_recovery_stage32.sh
PROBE="$ROOT/buildtree/cmd/protocol-probe"
printf '%s  %s\n' '5e58a9185019617cada73eb7a64d3762f1596e3021be8cb28e30f0599ace9db3' "$PROBE/scene_lifecycle.go" | sha256sum -c -
printf '%s  %s\n' 'aee89c22932f0211081ef745b050eb2faaa817871a32ce03b19615d21f6dfcdd' "$PROBE/stage32_shop_menu_diagnostic.go" | sha256sum -c -
cd "$ROOT/buildtree"
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -modfile=ci.real.mod -o "$ROOT/stage33-linux-audit-probe" ./cmd/protocol-probe > "$ROOT/stage33_linux_build.log" 2>&1
sha256sum "$ROOT/stage33-linux-audit-probe" > "$ROOT/stage33-linux-audit-probe.sha256"
file "$ROOT/stage33-linux-audit-probe" > "$ROOT/stage33-linux-audit-probe.file.txt"
go version -m "$ROOT/stage33-linux-audit-probe" > "$ROOT/stage33_linux_buildinfo.txt"
{
 echo 'source_basis=VERIFIED_9yin-go-server1.rar_STAGE32_PRESERVED'
 echo 'linux_audit_binary=BUILD_PASS'
 echo 'resources=NOT_UPLOADED_TO_GITHUB'
 echo 'offline_audit=NOT_RUN_IN_CI_REQUIRES_USER_PROVIDED_RAR_RESOURCES'
 echo 'npc_shop_purchase=UNCHANGED_FAIL_CLOSED'
 echo 'gm_web_item_grant=UNCHANGED_DEFERRED'
 echo 'live_client_e2e=NOT_RUN'
} > "$ROOT/stage33_status.log"
cat "$ROOT/stage33_status.log"
cat "$ROOT/stage33-linux-audit-probe.sha256"
