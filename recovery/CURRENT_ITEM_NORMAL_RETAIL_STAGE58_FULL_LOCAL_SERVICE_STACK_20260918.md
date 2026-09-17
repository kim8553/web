# Stage58 — full local service stack restoration

Date: 2026-09-18

## Basis

- Server lineage: user-supplied `9yin-go-server1.rar` -> current `kim8553/web` Go source.
- Working branch: `stage37-assistant-handoff-20260917`.
- Build source HEAD: `b42704557946ebd51f89438e1092d971e3f456a9`.
- Latest-client LIVE/E2E remains unverified until a user client actually connects.

## Corrected mistake

Stage57 launched only the `protocol-probe` game/GM process. Treating `protocol probe listening on 127.0.0.1:19061` as proof that the complete local client-facing service stack was open was incorrect.

The repository's own runtime launcher `server/启动本地九阴服务.bat` establishes the required local composition:

- server list: `127.0.0.1:4000` via `loopback-lister.ps1`
- game: `127.0.0.1:19061`
- GM: `127.0.0.1:19062`

The Stage37 LIVE/E2E checkpoint documents the same three-port package contract.

## Stage58 package

Package: `STAGE58_9YIN_GO_SERVER1_FULL_LOCAL_SERVICES_WINDOWS_20260918.zip`

Inner package SHA256:

`f44f4efab806a5ae8683c1050a2e3bf37c1fe78c90db2134e7c3218fd21b27e0`

Server EXE SHA256:

`ebc20779784703edfe7dd3d6b34df051d39e32b3877ff5f67475bcb87265ddd4`

Contents include:

- `stage58-live-server.exe`
- unchanged runtime `loopback-lister.ps1`
- `run-stage58-live-services.ps1`
- `RUN_STAGE58_LIVE.bat`
- `README_KO.md`
- `BUILD_INFO.txt`
- `SHA256SUMS.txt`

## Verification

GitHub Actions run `35258639807`: SUCCESS.

Verified on Windows runner:

- original three-port local-service contract: PASS
- Stage58 PowerShell syntax/contract: PASS
- lister copied to a path containing spaces, launched, listened, accepted a request, returned the server-list response targeting `127.0.0.1:19061`, and wrote capture output to a spaced path: PASS
- Stage54 normal-buy source retained: PASS
- Stage55/56/57 startup compatibility source retained: PASS
- `go test -c ./cmd/protocol-probe`: PASS
- Windows `go build ./cmd/protocol-probe`: PASS
- package creation/upload: PASS

The server launcher deliberately does not pass a path-valued `-log-file` argument through `Start-Process`; it uses the server's existing `NINEYIN_SERVER_ROOT` default log path `<ServerRoot>/artifacts/local/logs/protocol-probe-live.log`.

## Runtime gate

The launcher prints `STAGE58_READY=YES` only when ports 4000, 19061 and 19062 are all listening. It does not kill conflicting processes automatically.

`STAGE58_READY=YES` proves only that the local service endpoints are listening. It does not prove latest-client login, role select, scene entry, NPC shop purchase, or persistence.

Next LIVE step after `STAGE58_READY=YES`: launch the current Snail client and verify that a server-list request reaches the lister and that the client then connects to port 19061. Only after client entry succeeds should normal NPC buy `0x46` be tested.
