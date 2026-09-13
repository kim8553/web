# Stage37 first LIVE/E2E package checkpoint

Created after the final Stage37 static/build/test checkpoint on 2026-09-13.

## Build source

- authority-chain real-build commit: `f7ce5e74af7d18836974a806c07a6fa740b38146`
- `JIUYIN Stage37 Real Build Probe` run: `34735092558` (run #14), SUCCESS
- Windows amd64 PE SHA256: `8e12f8cd44667e65c349e2af8e0bf2ccd0168613e2af773af4f0a0335b5378ab`
- PE: `PE32+ executable (console) x86-64, 16 sections`
- Go: `1.26.5`

## Generated package

- file: `JIUYIN_STAGE37_LIVE_E2E_20260913.zip`
- package SHA256: `a0c9193a72a8e921086f69c12a5350ff2cfbe63db57aba7dbf8ad4dc91813fc6`
- the binary is deliberately named `stage37-live-server.exe`; it does not overwrite the authority EXE.

Package contents:
- `stage37-live-server.exe`
- `RUN_STAGE37_LIVE.bat`
- `loopback-lister.ps1`
- `COLLECT_STAGE37_LIVE_RESULT.bat`
- `COLLECT_STAGE37_LIVE_RESULT.ps1`
- `LIVE_TEST_CHECKLIST.txt`
- `README_STAGE37_LIVE_KO.txt`
- `MANIFEST_SHA256.txt`
- mutable `logs/` directory

The ZIP was re-extracted after creation and every static payload entry passed its internal SHA256 manifest.

## Runtime root contract

The package is intended to be placed directly under the existing server root, or used with `NINEYIN_SERVER_ROOT` explicitly set.

The launcher requires and SHA-checks:
- `resources/modern/share/skill/skill_new.ini`
- expected SHA256: `c762e7401c5c1bd4ead6d46b7544b983dd6224bbcfee7516e3bb25975db967c0`

It also SHA-checks the Stage37 test PE before starting.

Ports:
- server list: `127.0.0.1:4000`
- game: `127.0.0.1:19061`
- local GM UI: `127.0.0.1:19062`

No process is killed automatically on a port conflict.

## Persistence mode

- If `NINEYIN_MYSQL_DSN` is set, the server uses MySQL and runs the current migrations.
- Otherwise it uses the current JSON fallback under `<server-root>/data`.
- The result collector records only whether MySQL mode was enabled; it never records the DSN value or password.

## LIVE boundary

This package has **not** yet been validated by an actual latest Snail client session.

Do not report LIVE success until a user-run session has verified at minimum:
- server list/login/role select
- scene entry
- NPC visibility/talk/service/shop
- bag/item grant/shop purchase
- movement/viewport
- learned skill action/cooldown/buff behavior
- QingGong
- logout/relogin persistence

After the run, use `COLLECT_STAGE37_LIVE_RESULT.bat` and analyze the produced `STAGE37_LIVE_RESULT_*.zip` against the exact client actions/timestamps.
