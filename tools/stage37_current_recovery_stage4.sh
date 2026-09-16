#!/usr/bin/env bash
set -euo pipefail

ROOT="$(pwd)"
BUILD="$ROOT/buildtree"
FINAL_MANIFEST_SHA='f4763daa006f5bbc21854f9938ae21f557ff9a3bc64edc8e024d08c47c92bf7f'

# First reproduce and fully verify the already-passed 201-file checkpoint.
bash tools/stage37_current_recovery_stage3.sh

# Add only the dormant persistence-first commit helper and its tests. The helper
# is intentionally not wired into C2S 0x4F and emits no client frames.
echo '942974e92bb271eb0135c2f47e97590b1005d4d734ad8c74688c18a4fa97401e  recovery/postbuild_files/cmd__protocol-probe__latest_client_shop_exchange_commit.go' | sha256sum -c -
echo 'fef0d08a09a8a939c99e3d5fbb39299d3308a9ca78eb2b5b8a1cdc6e0b502bcb  recovery/postbuild_files/cmd__protocol-probe__latest_client_shop_exchange_commit_test.go' | sha256sum -c -
cp recovery/postbuild_files/cmd__protocol-probe__latest_client_shop_exchange_commit.go "$BUILD/cmd/protocol-probe/latest_client_shop_exchange_commit.go"
cp recovery/postbuild_files/cmd__protocol-probe__latest_client_shop_exchange_commit_test.go "$BUILD/cmd/protocol-probe/latest_client_shop_exchange_commit_test.go"
gofmt -w "$BUILD/cmd/protocol-probe/latest_client_shop_exchange_commit.go" "$BUILD/cmd/protocol-probe/latest_client_shop_exchange_commit_test.go"

# Stage3 leaves CI-only module files and the Linux build output in buildtree.
# Remove them from source-integrity accounting, then restore the modfiles for
# the final 203-file production gate run.
test -f "$BUILD/ci.real.mod"
test -f "$BUILD/ci.real.sum"
test -f "$BUILD/protocol-probe"
mv "$BUILD/ci.real.mod" "$ROOT/ci.real.mod.stage4"
mv "$BUILD/ci.real.sum" "$ROOT/ci.real.sum.stage4"
rm "$BUILD/protocol-probe"
test "$(find "$BUILD" -type f | wc -l | tr -d ' ')" = '203'
(cd "$BUILD" && find . -type f -printf '%P\0' | sort -z | xargs -0 sha256sum > "$ROOT/generated-manifest.txt")
test "$(sha256sum generated-manifest.txt | awk '{print $1}')" = "$FINAL_MANIFEST_SHA"
mv "$ROOT/ci.real.mod.stage4" "$BUILD/ci.real.mod"
mv "$ROOT/ci.real.sum.stage4" "$BUILD/ci.real.sum"

# Re-run every production gate against the final 203-file tree. The commit
# helper remains dormant. Its resource-independent tests are executed directly
# (normal and race) even when the full exact-current resource corpus is absent.
cd "$BUILD"
set +e
run_gate() {
  local name="$1"
  shift
  "$@" >"$ROOT/${name}.log" 2>&1
  echo $? >"$ROOT/${name}.exit"
}
run_gate planner go test -modfile=ci.real.mod ./internal/exchangeplan -count=1
run_gate migrations go test -modfile=ci.real.mod ./migrations -count=1
run_gate build go build -modfile=ci.real.mod ./cmd/protocol-probe
run_gate protocol_compile go test -modfile=ci.real.mod -c -o "$ROOT/protocol-probe.test" ./cmd/protocol-probe
run_gate protocol_race_compile go test -modfile=ci.real.mod -race -c -o "$ROOT/protocol-probe-race.test" ./cmd/protocol-probe
run_gate protocol_commit_targeted go test -modfile=ci.real.mod ./cmd/protocol-probe -run '^TestCommitShopExchangeReplacementPersistenceFirst' -count=1
run_gate protocol_commit_targeted_race go test -modfile=ci.real.mod -race ./cmd/protocol-probe -run '^TestCommitShopExchangeReplacementPersistenceFirst' -count=1
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
for x in planner migrations build protocol_compile protocol_race_compile protocol_commit_targeted protocol_commit_targeted_race nonprotocol race vet windows; do
  code=$(cat "$ROOT/$x.exit")
  echo "$x=$code"
  if [ "$code" -ne 0 ]; then failed=1; fi
done
protocol=$(cat "$ROOT/protocol_runtime.exit")
echo "protocol_runtime=$protocol"
if [ "$protocol" -ne 0 ] && [ "$protocol" -ne 125 ]; then failed=1; fi
exit "$failed"
