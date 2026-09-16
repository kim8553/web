#!/usr/bin/env bash
set -euo pipefail

ROOT="$(pwd)"
BUILD="$ROOT/buildtree"

# Stage16 does not activate an ordinary-shop selector. It preserves Stage15's
# exact-current wire gate and adds a reusable fail-closed transaction-order
# coordinator that cannot run unless both wire authority and one atomic
# bag+currency persistence boundary are explicitly ready.
bash tools/stage37_current_recovery_stage15.sh

grep -F 'current_client_selector_verified=0' "$ROOT/regular_shop_wire_gate.log"
grep -F 'wire_gate_status=BLOCKED_NO_EXACT_CURRENT_CLIENT_AUTHORITY_IN_REPO' "$ROOT/regular_shop_wire_gate.log"

cat > "$ROOT/stage16_regular_shop_transaction_status.log" <<'STATUS'
stage16_scope=ordinary_npc_shop_transaction_order_only
current_client_selector_verified=0
wire_gate_status=BLOCKED_NO_EXACT_CURRENT_CLIENT_AUTHORITY_IN_REPO
production_handler_wiring=OFF
legacy_or_historical_selector_promotion=OFF
gm_grant_changes=0
required_order=wire_gate_require_then_pure_stage_then_atomic_bag_currency_persist_then_live_apply_then_client_publish
publish_failure_boundary=durable_state_already_committed_requires_resync_not_db_rollback
STATUS

# These four files are committed immutable Stage16 inputs. Verify their exact
# Git blob identities from the canonical commit rather than maintaining a
# second manually copied SHA256 list that can diverge on whitespace alone.
for spec in \
  '4cc5c8eaee4dd38d433740d13ab154697dc5b823 recovery/postbuild_files/internal__shopbuygate__decision.go' \
  '48b03db972d7369ee72ba4e81cc76336358a2f4d recovery/postbuild_files/internal__shopbuygate__decision_test.go' \
  '09b2032b580d3bc6ca17a316e979e55b4bf3d15e recovery/postbuild_files/internal__shopbuyexecute__execute.go' \
  '3dd7f1eb0f5d13f4fcefe1224adf905442937f59 recovery/postbuild_files/internal__shopbuyexecute__execute_test.go'; do
  expected="${spec%% *}"
  path="${spec#* }"
  actual="$(git hash-object "$path")"
  echo "stage16_input_blob path=$path expected=$expected actual=$actual"
  test "$actual" = "$expected"
done

mkdir -p "$BUILD/internal/shopbuygate" "$BUILD/internal/shopbuyexecute"
cp recovery/postbuild_files/internal__shopbuygate__decision.go "$BUILD/internal/shopbuygate/decision.go"
cp recovery/postbuild_files/internal__shopbuygate__decision_test.go "$BUILD/internal/shopbuygate/decision_test.go"
cp recovery/postbuild_files/internal__shopbuyexecute__execute.go "$BUILD/internal/shopbuyexecute/execute.go"
cp recovery/postbuild_files/internal__shopbuyexecute__execute_test.go "$BUILD/internal/shopbuyexecute/execute_test.go"
gofmt -w "$BUILD/internal/shopbuygate" "$BUILD/internal/shopbuyexecute"

(cd "$BUILD" && find . -type f ! -name 'ci.real.mod' ! -name 'ci.real.sum' ! -name 'protocol-probe' -printf '%P\0' | sort -z | xargs -0 sha256sum > "$ROOT/generated-manifest.txt")
sha256sum "$ROOT/generated-manifest.txt" > "$ROOT/generated-manifest.sha256"
source_files=$(find "$BUILD" -type f ! -name 'ci.real.mod' ! -name 'ci.real.sum' ! -name 'protocol-probe' | wc -l | tr -d ' ')
echo "stage16_source_files=$source_files"
echo "stage16_manifest_sha256=$(awk '{print $1}' "$ROOT/generated-manifest.sha256")"
if [ "$source_files" != "216" ]; then
  echo "unexpected Stage16 source count: $source_files" >&2
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

run_gate shopbuygate go test -modfile=ci.real.mod ./internal/shopbuygate -count=1
run_gate shopbuygate_race go test -modfile=ci.real.mod -race ./internal/shopbuygate -count=1
run_gate shopbuyexecute go test -modfile=ci.real.mod ./internal/shopbuyexecute -count=1
run_gate shopbuyexecute_race go test -modfile=ci.real.mod -race ./internal/shopbuyexecute -count=1
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
for x in shopbuygate shopbuygate_race shopbuyexecute shopbuyexecute_race build protocol_compile protocol_race_compile nonprotocol race vet windows; do
  code=$(cat "$ROOT/$x.exit")
  echo "$x=$code"
  if [ "$code" -ne 0 ]; then failed=1; fi
done
protocol=$(cat "$ROOT/protocol_runtime.exit")
echo "protocol_runtime=$protocol"
if [ "$protocol" -ne 0 ] && [ "$protocol" -ne 125 ]; then failed=1; fi
exit "$failed"
