#!/usr/bin/env bash
set -euo pipefail

ROOT="$(pwd)"
BUILD="$ROOT/buildtree"
HANDOFF_ZIP="$ROOT/JIUYIN_STAGE37_BUILDPROBE_HANDOFF_20260913.zip"
QUICKLZ='github.com/Hiroko103/go-quicklz@v0.0.0-20190115215310-59904abc50d0'
PATCH_SHA='50c20c9abaa91fb42cc885b3a8e81a5c8acee2a5bb09d80f53a1420772989c25'
MANIFEST_SHA='43522bed8e5ff9cd1712bab8ed85d03f9ee483ace199b0c458c439234c70ec31'
POST_MANIFEST_SHA='7632477cdc006efe7c95bbbdee58fd7ec8d79d9df26bdbf594b0a2e32f88c749'

cat JIUYIN_STAGE37_BUILDPROBE_HANDOFF_20260913.zip.part00 JIUYIN_STAGE37_BUILDPROBE_HANDOFF_20260913.zip.part01 > "$HANDOFF_ZIP"
echo '56a49fa0abb9b4aa6192a9e2d93f7e780c29e585cf11ccb3ff0cf9f398955fdb  JIUYIN_STAGE37_BUILDPROBE_HANDOFF_20260913.zip' | sha256sum -c -
rm -rf handoff "$BUILD"
unzip -q "$HANDOFF_ZIP" -d handoff
mkdir -p "$BUILD"
tar -xzf handoff/JIUYIN_STAGE37_BUILDPROBE_HANDOFF_20260913/buildprobe_global_merged4.tar.gz -C "$BUILD"
test -f "$BUILD/go.mod"

python3 tools/stage37_authority_compile_patch.py "$BUILD"
python3 tools/stage37_authority_test_patch.py "$BUILD"
python3 tools/stage37_authority_test_followup.py "$BUILD"
python3 tools/stage37_authority_runtime_patch.py "$BUILD"
base64 -d tools/stage37_authority_resource_test_patch.py.gz.b64 | gzip -dc > /tmp/stage37_authority_resource_test_patch.py
echo '2a6cbcf3e7c7b6087f0eecc24b09375abcd3559614bbafca12806a8c147aeeb8  /tmp/stage37_authority_resource_test_patch.py' | sha256sum -c -
python3 /tmp/stage37_authority_resource_test_patch.py "$BUILD"
base64 -d tools/stage37_authority_resource_test_followup.py.gz.b64 | gzip -dc > /tmp/stage37_authority_resource_test_followup.py
echo '826fe3feefbadffb3b6c8f75a1d5f74a19c956d38b2294c1ba98996c96fcfec7  /tmp/stage37_authority_resource_test_followup.py' | sha256sum -c -
python3 /tmp/stage37_authority_resource_test_followup.py "$BUILD"
python3 -m py_compile tools/stage37_authority_resource_test_final.py
python3 tools/stage37_authority_resource_test_final.py "$BUILD"
python3 -m py_compile tools/stage37_authority_yihua_exact_test_patch.py
python3 tools/stage37_authority_yihua_exact_test_patch.py "$BUILD"
find "$BUILD/cmd" "$BUILD/internal" -name '*.go' -print0 | xargs -0 gofmt -w

cat recovery/current_delta.patch.part* > current_delta.patch
test "$(sha256sum current_delta.patch | awk '{print $1}')" = "$PATCH_SHA"
(cd "$BUILD" && patch --batch --forward -p1 < "$ROOT/current_delta.patch")
test "$(find "$BUILD" -type f | wc -l | tr -d ' ')" = '192'
(cd "$BUILD" && find . -type f -printf '%P\0' | sort -z | xargs -0 sha256sum > "$ROOT/generated-manifest.txt")
test "$(sha256sum generated-manifest.txt | awk '{print $1}')" = "$MANIFEST_SHA"
test -f "$BUILD/migrations/0012_persist_runtime_bind_status.sql"
test -f "$BUILD/internal/exchangeplan/plan.go"
test -f "$BUILD/cmd/protocol-probe/latest_client_shop_exchange_preflight.go"
test -f "$BUILD/cmd/protocol-probe/latest_client_equip_view_ordinal_compat_test.go"

# Apply the post-build current changes only from individually verified small files.
echo 'fec8dfd13c50269ed567abddc96111e29b60e126a83bf7d1cf47e60ae54b2cb2  recovery/postbuild_patches/latest_client_shop_exchange_contract.patch' | sha256sum -c -
echo '17783a557fb394ae7ba05375310d89b2d48eb1eb665f5a315f1e318d36829bdb  recovery/postbuild_patches/latest_client_shop_exchange_preflight.patch' | sha256sum -c -
echo 'caff9f44addaf30e2f267fc6795eb91d5d69cf77d58a59b3cdd7208326e37fcf  recovery/postbuild_patches/latest_client_shop_exchange_preflight_test.patch' | sha256sum -c -
echo 'f3d9fda229ea672554a717ab13f974decef6882b131f87e28cf8fc7fc75a2d5a  recovery/postbuild_patches/main.patch' | sha256sum -c -
echo '0afc54e0db16b6e57453e4ac378846ee44ed031634a77304b2b99f97ddc04d6e  recovery/postbuild_patches/zz_recovered_overlay.patch' | sha256sum -c -
echo '9e13e76e914007916b853e0f1dcc9a8da668edded1556677d82e26c1c723eaa7  recovery/postbuild_files/cmd__protocol-probe__latest_client_bag_bind_arrange_test.go' | sha256sum -c -
echo '712de6d1477b8750f6f4f1942d814a9c4afbd8cf367d5748b1097e9b59b20757  recovery/postbuild_files/internal__exchangeplan__capacity.go' | sha256sum -c -
echo '51d73aa02af50182a23f0d0c6ea271c49d1c12a148bd4050322e36ac60bab0b1  recovery/postbuild_files/internal__exchangeplan__capacity_test.go' | sha256sum -c -
for p in \
  recovery/postbuild_patches/latest_client_shop_exchange_contract.patch \
  recovery/postbuild_patches/latest_client_shop_exchange_preflight.patch \
  recovery/postbuild_patches/latest_client_shop_exchange_preflight_test.patch \
  recovery/postbuild_patches/main.patch \
  recovery/postbuild_patches/zz_recovered_overlay.patch; do
  (cd "$BUILD" && patch --batch --forward -p1 < "$ROOT/$p")
done
cp recovery/postbuild_files/cmd__protocol-probe__latest_client_bag_bind_arrange_test.go "$BUILD/cmd/protocol-probe/latest_client_bag_bind_arrange_test.go"
cp recovery/postbuild_files/internal__exchangeplan__capacity.go "$BUILD/internal/exchangeplan/capacity.go"
cp recovery/postbuild_files/internal__exchangeplan__capacity_test.go "$BUILD/internal/exchangeplan/capacity_test.go"
find "$BUILD/cmd" "$BUILD/internal" -name '*.go' -print0 | xargs -0 gofmt -w

test "$(find "$BUILD" -type f | wc -l | tr -d ' ')" = '195'
(cd "$BUILD" && find . -type f -printf '%P\0' | sort -z | xargs -0 sha256sum > "$ROOT/generated-manifest.txt")
test "$(sha256sum generated-manifest.txt | awk '{print $1}')" = "$POST_MANIFEST_SHA"
test -f "$BUILD/internal/exchangeplan/capacity.go"
test -f "$BUILD/internal/exchangeplan/capacity_test.go"
test -f "$BUILD/cmd/protocol-probe/latest_client_bag_bind_arrange_test.go"

cd "$BUILD"
cp go.mod ci.real.mod
cp go.sum ci.real.sum
go mod edit -modfile=ci.real.mod -dropreplace=github.com/go-sql-driver/mysql
go mod edit -modfile=ci.real.mod -dropreplace=golang.org/x/text
go mod edit -modfile=ci.real.mod -dropreplace=filippo.io/edwards25519
go mod edit -modfile=ci.real.mod -dropreplace=github.com/DATA-DOG/go-sqlmock
go mod edit -modfile=ci.real.mod -dropreplace=github.com/Hiroko103/go-quicklz
go mod edit -modfile=ci.real.mod -require="$QUICKLZ"
go mod download -modfile=ci.real.mod "$QUICKLZ"
grep -F 'github.com/Hiroko103/go-quicklz v0.0.0-20190115215310-59904abc50d0 h1:' ci.real.sum
go mod download -modfile=ci.real.mod
go list -modfile=ci.real.mod -m all > "$ROOT/modules.txt"

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
