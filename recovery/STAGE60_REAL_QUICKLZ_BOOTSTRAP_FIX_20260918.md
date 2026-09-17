# Stage60 — real QuickLZ bootstrap correction (2026-09-18 KST)

## Authority / scope
- Base: user-provided `9yin-go-server1.rar` lineage in `kim8553/web`, branch `stage37-assistant-handoff-20260917`.
- Newest Snail client EXE/DLL and matching resources govern protocol semantics. No historical V37/V46/JYZJ/V2 behavior is borrowed.
- Stage60 remains the active gate. Ordinary NPC BUY **and SELL** are the next item-system objective; GM grants and Exchange remain **HOLD**.
- Do not repeatedly request user LIVE tests. Distinguish static evidence, test, build, service boot and actual scene/reconnect E2E.

## Evidence from previous Stage60 LIVE result
The user PC's `STAGE60_LIVE_RESULT_20260918_052614_143.zip` shows successful login and role-list transfer, followed by PlayerEntry and first scene bootstrap packets. The limited diagnostic gate deliberately transmitted only the first three of five spawn frames, then withheld all post-spawn traffic; no ClientReady was reached. It is **not** a valid full-scene compatibility test, nor proof that any specific early frame is the sole reason the client closed.

The emitted second frame began `28 01 00 00 11 0B 5C 7A 00 ...` (opcode `0x28`, followed by raw player entity ID). The source `buildSnapshot608Frame` calls `quicklz.New(...).Compress(...)` and prefixes its result with `0x28`, so this raw prefix exposed an issue in the resolved compressor.

## Confirmed defect
Before this fix, `server/go.mod` replaced `github.com/Hiroko103/go-quicklz` with `./_builddeps/quicklz`. That local `quicklz.go` is only a no-op copy: its `Compress` is `return copy(*dst,*src),nil`. It does **not** generate a QuickLZ header or compressed bytes, even though the caller labeled the raw buffer as a compressed `0x28` frame. This is an actual source/wire-format defect independently of any claim that fixing it alone will make a complete scene load.

## Implemented fix and regression
- Removed the local QuickLZ replacement and pinned the actual upstream module `github.com/Hiroko103/go-quicklz v0.0.0-20190115215310-59904abc50d0` in `server/go.mod`.
- Pinned the upstream module's checksums in `server/go.sum` from Windows GitHub Actions' verified module download.
- Added `server/internal/stage60diag/quicklz_wire_test.go`: the test rejects raw-identity copies, requires a QuickLZ header and verifies decompression round-trip. It fails with the old identity-only shim and passes with upstream QuickLZ.
- Added `.github/workflows/jiuyin-stage60-quicklz-verified.yml`: Windows dependency resolution (assert no Replace), targeted regression, server test-binary compilation, Windows executable build and build-only artifact. Workflow run `35275461261` passed all steps; `go version -m` on the EXE confirms the pinned upstream QuickLZ dependency rather than a local replacement. Windows EXE SHA256 from that run: `450381ab974b422f4e52d20160140c197bcc5fe09ea61b744dd6a51f3f3f09a8`.

## Not yet proven / next gate
- No user-machine LIVE scene success or reconnect persistence has been established with the newly corrected compressor.
- The prior 3-frame gate intentionally omitted required frames and must not be used to infer which later scene message is invalid; avoid arbitrary opcode, field and order changes.
- Complete existing autonomous source/static/Windows verification, then request a single un-gated full-entry LIVE test only when needed; do not present build-only binary as a completed game server.
- Do not work on GM grant or Exchange. Continue BUY **and SELL** plus the entire item lifecycle after Stage60 scene entry is stable.
