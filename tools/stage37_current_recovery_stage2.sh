#!/usr/bin/env bash
set -euo pipefail

ROOT="$(pwd)"
BUILD="$ROOT/buildtree"
FINAL_MANIFEST_SHA='a96d33fe6fdc07449af7f1f757ac65a658ca7bf87606db88bd703456a367bcf7'

# First reproduce and fully verify the already-passed 195-file checkpoint.
bash tools/stage37_current_recovery_build.sh

# Add only the side-effect-free atomic staging layer from individually verified sources.
echo 'af2a8883df8d09478668b8edb8316ae0a73d6bfb9fa6121a2906d40c45f1d7ec  recovery/postbuild_files/internal__exchangeplan__replacement.go' | sha256sum -c -
echo '7b8b7ce49803c2ed0c9f9b493efac7f2c6b2f5f5a809c6b1c8f739c5685dfd7b  recovery/postbuild_files/internal__exchangeplan__replacement_test.go' | sha256sum -c -
echo '637cc24872a82832aee66621ffd5f3e71d65e0c4465a850bba79ef87e21180e1  recovery/postbuild_files/cmd__protocol-probe__latest_client_shop_exchange_stage.go' | sha256sum -c -
echo 'dcab7818e40b4ac8d833185d8ec2070a3424c166a73c2f70e405a8ea78263dc5  recovery/postbuild_files/cmd__protocol-probe__latest_client_shop_exchange_stage_test.go' | sha256sum -c -
cp recovery/postbuild_files/internal__exchangeplan__replacement.go "$BUILD/internal/exchangeplan/replacement.go"
cp recovery/postbuild_files/internal__exchangeplan__replacement_test.go "$BUILD/internal/exchangeplan/replacement_test.go"
cp recovery/postbuild_files/cmd__protocol-probe__latest_client_shop_exchange_stage.go "$BUILD/cmd/protocol-probe/latest_client_shop_exchange_stage.go"
cp recovery/postbuild_files/cmd__protocol-probe__latest_client_shop_exchange_stage_test.go "$BUILD/cmd/protocol-probe/latest_client_shop_exchange_stage_test.go"
find "$BUILD/cmd" "$BUILD/internal" -name '*.go' -print0 | xargs -0 gofmt -w

test "$(find "$BUILD" -type f | wc -l | tr -d ' ')" = '199'
(cd "$BUILD" && find . -type f -printf '%P\0' | sort -z | xargs -0 sha256sum > "$ROOT/generated-manifest.txt")
test "$(sha256sum generated-manifest.txt | awk '{print $1}')" = "$FINAL_MANIFEST_SHA"

# Re-run every production gate against the 199-file tree. Overwrite the 195-file
# evidence so the uploaded artifact always describes the final current tree.
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
for x in planner migrations build protocol_compile protocol_race_compile nonprotocol race vet windows; do
  code=$(cat "$ROOT/$x.exit")
  echo "$x=$code"
  if [ "$code" -ne 0 ]; then failed=1; fi
done
protocol=$(cat "$ROOT/protocol_runtime.exit")
echo "protocol_runtime=$protocol"
if [ "$protocol" -ne 0 ] && [ "$protocol" -ne 125 ]; then failed=1; fi
exit "$failed"
