#!/usr/bin/env bash
set -euo pipefail

ROOT="$(pwd)"
BUILD="$ROOT/buildtree"
CONTRACT="$BUILD/cmd/protocol-probe/latest_client_shop_exchange_contract.go"

# Reproduce the already-passed Stage9 typed-nil persistence checkpoint first.
bash tools/stage37_current_recovery_stage9.sh

# Add a resource-independent activation-order coordinator. It remains dormant:
# the real 0x4F handler is still dry-run only and cannot call staging/commit.
mkdir -p "$BUILD/internal/exchangeexecute"
cp recovery/postbuild_files/internal__exchangeexecute__execute.go "$BUILD/internal/exchangeexecute/execute.go"
cp recovery/postbuild_files/internal__exchangeexecute__execute_test.go "$BUILD/internal/exchangeexecute/execute_test.go"
gofmt -w "$BUILD/internal/exchangeexecute"

# Hard safety boundary remains in force.
if grep -q 'commitShopExchangeReplacementPersistenceFirst' "$CONTRACT"; then
  echo 'forbidden: 0x4F handler wired to persistence commit' >&2
  exit 91
fi
if grep -q 'stageShopExchangeAtomicReplacement' "$CONTRACT"; then
  echo 'forbidden: 0x4F handler wired to atomic mutation staging' >&2
  exit 92
fi
if grep -q 'exchangeexecute\.PersistFirst' "$CONTRACT"; then
  echo 'forbidden: 0x4F handler wired to dormant execution coordinator' >&2
  exit 93
fi

# Stage9 leaves CI-only module files and a Linux build output in buildtree.
test -f "$BUILD/ci.real.mod"
test -f "$BUILD/ci.real.sum"
test -f "$BUILD/protocol-probe"
mv "$BUILD/ci.real.mod" "$ROOT/ci.real.mod.stage10"
mv "$BUILD/ci.real.sum" "$ROOT/ci.real.sum.stage10"
rm "$BUILD/protocol-probe"
test "$(find "$BUILD" -type f | wc -l | tr -d ' ')" = '212'
(cd "$BUILD" && find . -type f -printf '%P\0' | sort -z | xargs -0 sha256sum > "$ROOT/generated-manifest.txt")
sha256sum "$ROOT/generated-manifest.txt" > "$ROOT/generated-manifest.sha256"
mv "$ROOT/ci.real.mod.stage10" "$BUILD/ci.real.mod"
mv "$ROOT/ci.real.sum.stage10" "$BUILD/ci.real.sum"

cd "$BUILD"
set +e
run_gate() {
  local name="$1"
  shift
  "$@" >"$ROOT/${name}.log" 2>&1
  echo $? >"$ROOT/${name}.exit"
}
run_gate exchangeexecute go test -modfile=ci.real.mod ./internal/exchangeexecute -count=1
run_gate exchangeexecute_race go test -modfile=ci.real.mod -race ./internal/exchangeexecute -count=1
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
for x in exchangeexecute exchangeexecute_race exchangegate exchangegate_race exchangebinding exchangebinding_race exchangecommit exchangecommit_race planner migrations build protocol_compile protocol_race_compile nonprotocol race vet windows; do
  code=$(cat "$ROOT/$x.exit")
  echo "$x=$code"
  if [ "$code" -ne 0 ]; then failed=1; fi
done
protocol=$(cat "$ROOT/protocol_runtime.exit")
echo "protocol_runtime=$protocol"
if [ "$protocol" -ne 0 ] && [ "$protocol" -ne 125 ]; then failed=1; fi

# Print immutable evidence directly into Actions logs so GitHub-only fallback can
# verify hashes without reopening the artifact in a local container.
echo "source_files=$(find "$BUILD" -type f ! -name 'ci.real.mod' ! -name 'ci.real.sum' ! -name 'protocol-probe' | wc -l | tr -d ' ')"
echo "manifest_sha256=$(awk '{print $1}' "$ROOT/generated-manifest.sha256")"
if [ -f "$ROOT/stage37-current-windows-amd64.sha256" ]; then
  echo "windows_sha256=$(awk '{print $1}' "$ROOT/stage37-current-windows-amd64.sha256")"
  echo "windows_file=$(cat "$ROOT/stage37-current-windows-amd64.file.txt")"
fi
exit "$failed"
