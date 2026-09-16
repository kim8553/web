#!/usr/bin/env bash
set -euo pipefail

ROOT="$(pwd)"
BUILD="$ROOT/buildtree"

# Reproduce the already-passed 207-file checkpoint first.
bash tools/stage37_current_recovery_stage6.sh

# Add the resource-independent transaction gate plus the protocol adapter. The
# adapter still does not invoke any mutation or persistence path.
mkdir -p "$BUILD/internal/exchangegate"
cp recovery/postbuild_files/internal__exchangegate__decision.go "$BUILD/internal/exchangegate/decision.go"
cp recovery/postbuild_files/internal__exchangegate__decision_test.go "$BUILD/internal/exchangegate/decision_test.go"
cp recovery/postbuild_files/cmd__protocol-probe__latest_client_shop_exchange_gate.go "$BUILD/cmd/protocol-probe/latest_client_shop_exchange_gate.go"
gofmt -w "$BUILD/internal/exchangegate" "$BUILD/cmd/protocol-probe/latest_client_shop_exchange_gate.go"

# Stage6 leaves CI-only module files and a Linux build output in buildtree.
test -f "$BUILD/ci.real.mod"
test -f "$BUILD/ci.real.sum"
test -f "$BUILD/protocol-probe"
mv "$BUILD/ci.real.mod" "$ROOT/ci.real.mod.stage7"
mv "$BUILD/ci.real.sum" "$ROOT/ci.real.sum.stage7"
rm "$BUILD/protocol-probe"
test "$(find "$BUILD" -type f | wc -l | tr -d ' ')" = '210'
(cd "$BUILD" && find . -type f -printf '%P\0' | sort -z | xargs -0 sha256sum > "$ROOT/generated-manifest.txt")
sha256sum "$ROOT/generated-manifest.txt" > "$ROOT/generated-manifest.sha256"
mv "$ROOT/ci.real.mod.stage7" "$BUILD/ci.real.mod"
mv "$ROOT/ci.real.sum.stage7" "$BUILD/ci.real.sum"

cd "$BUILD"
set +e
run_gate() {
  local name="$1"
  shift
  "$@" >"$ROOT/${name}.log" 2>&1
  echo $? >"$ROOT/${name}.exit"
}
run_gate exchangegate go test -modfile=ci.real.mod ./internal/exchangegate -count=1
run_gate exchangegate_race go test -modfile=ci.real.mod -race ./internal/exchangegate -count=1
run_gate exchangebinding go test -modfile=ci.real.mod ./internal/exchangebinding -count=1
run_gate exchangebinding_race go test -modfile=ci.real.mod -race ./internal/exchangebinding -count=1
run_gate exchangecommit go test -modfile=ci.real.mod ./internal/exchangecommit -count=1
run_gate exchangecommit_race go test -modfile=ci.real.mod -race ./internal/exchangecommit -count=1
run_gate planner go test -modfile=ci.real.mod ./internal/exchangeplan -count=1
run_gate migrations go test -modfile=ci.real.mod ./migrations -count=1
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
if [ -f resources/modern/share/skill/skill_new.ini ]; then
  go test -modfile=ci.real.mod ./cmd/protocol-probe -count=1 > "$ROOT/protocol_runtime.log" 2>&1
  echo $? > "$ROOT/protocol_runtime.exit"
else
  echo 'NOT RUN: exact-current resource corpus absent; no legacy substitution.' > "$ROOT/protocol_runtime.log"
  echo 125 > "$ROOT/protocol_runtime.exit"
fi

failed=0
for x in exchangegate exchangegate_race exchangebinding exchangebinding_race exchangecommit exchangecommit_race planner migrations build protocol_compile protocol_race_compile nonprotocol race vet windows; do
  code=$(cat "$ROOT/$x.exit")
  echo "$x=$code"
  if [ "$code" -ne 0 ]; then failed=1; fi
done
protocol=$(cat "$ROOT/protocol_runtime.exit")
echo "protocol_runtime=$protocol"
if [ "$protocol" -ne 0 ] && [ "$protocol" -ne 125 ]; then failed=1; fi
exit "$failed"
