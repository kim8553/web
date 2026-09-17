# Stage54 — 9yin-go-server1 JSON retail test server

Date: 2026-09-18 KST

## Authority / baseline correction

- Editable server baseline is the user-supplied `9yin-go-server1.rar` lineage, represented by the current Go source on branch `stage37-assistant-handoff-20260917`.
- Current Snail client binaries/resources remain protocol/gameplay authority.
- Historical `D:\9yin_server` layout is not assumed by the Stage54 launcher.
- Runtime root is explicit through `-ServerRoot` / `NINEYIN_SERVER_ROOT` and must point at the extracted/current `9yin-go-server1` runtime containing its real resources.
- No fake `skill_new.ini` is created or committed.

## Normal shop buy 0x46 storage correction

Stage50 had made the normal purchase path require MySQL. That was too narrow for the original `9yin-go-server1` runtime because the server also has native JSON-backed bag/currency stores when `NINEYIN_MYSQL_DSN` is unset.

Stage54 keeps the existing MySQL transactional path and adds a guarded JSON paired persistence path:

- bag JSON replacement is written through temp-file + fsync + rename;
- currency JSON replacement is then written;
- if the currency replacement fails during a normal process error, the bag file is restored to its exact pre-operation JSON snapshot;
- in-memory JSON store state is changed only after both replacements succeed;
- this is operation-level all-or-rollback, not a claim of crash/power-loss atomicity across two files.

## Verification

GitHub Actions run: `35251234937`

Result: **SUCCESS**.

Verified on Windows runner:

- corrected runtime-boundary/launcher checks: PASS;
- isolated exact JSON paired-persistence tests: PASS;
- JSON success persistence test: PASS;
- JSON second-write failure / bag rollback test: PASS;
- full `protocol-probe` test-binary compile (`go test -c`): PASS;
- full Windows server build (`go build`): PASS;
- package creation/upload: PASS.

The isolated JSON test deliberately avoids executing unrelated package init because the public GitHub repository does not contain the user's proprietary/current runtime resources. Full runtime init must use the real resources from the user's `9yin-go-server1` runtime.

The four Stage54 Go source blobs in the passing CI workspace exactly match the GitHub blobs after gofmt:

- `latest_client_normal_shop_atomic_buy.go`: `edcfe3a821177ded5cb215bb25243665171e5736`
- `latest_client_normal_shop_json_atomic.go`: `0aeff888f22888de881ca3bca3d1bb9491e95d2c`
- `latest_client_normal_shop_json_atomic_isolated_test_support.go`: `27896b238df2fae7da0cf7657ebb28efbc53c0e2`
- `latest_client_normal_shop_json_atomic_test.go`: `78c9a181ee66473469ff8114393a4c9e09533d5c`

## Test package

Package filename:

`STAGE54_9YIN_GO_SERVER1_TEST_SERVER_WINDOWS_20260918.zip`

Package SHA256:

`bbb5a89cba1d1dfdaeef1bafad0e199b1388c7b9a95c43f5ed42e8f4876bd187`

Server EXE inside package:

`stage54-test-server.exe`

Server EXE SHA256:

`7555668f15c1a784b6760b28e93b51df0eda452a87fe36d3abc02eb5a108de14`

The package does not contain client resources, DB data, credentials, or proprietary runtime resources.

## Not yet proven

- User-machine live server boot with the user's extracted/current `9yin-go-server1` runtime: NOT RUN.
- Current-client login and normal NPC purchase E2E: NOT RUN.
- Bag/currency display after live purchase: NOT RUN.
- Reconnect persistence after live purchase: NOT RUN.
- Power-loss atomicity of the two-file JSON path: NOT CLAIMED.
- Normal sale `0x47`: NOT IMPLEMENTED.
- GM grant: DEFERRED/HOLD.
- Exchange `0x4F`: DEFERRED/HOLD.

## Next gate

Run the Stage54 test server against the user's extracted/current `9yin-go-server1` runtime root. Do not point it at an old V37/V46/JYZJ or historical server tree. First verify clean boot, then current-client login, then exactly one ordinary NPC-shop purchase, bag/currency update, full client/server restart, and reconnect persistence.
