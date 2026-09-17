# Stage60 autonomous verification checkpoint — 2026-09-18 KST

Authority: original `9yin-go-server1.rar` lineage, newest Snail client EXE/DLL and corresponding `res`/BIN64. GM item grant and Exchange HOLD. Next after Stage60: ordinary NPC BUY **and SELL**, followed by complete item use/equip/persistence.

## Proven source defect corrected in earlier commits
`server/go.mod` previously replaced real QuickLZ with an identity-copy stub while `buildSnapshot608Frame` labeled its raw bytes as compressed 0x28. The replacement was removed, actual upstream QuickLZ dependency pinned and a real roundtrip test added. Do not mistake this source defect proof for proof that it alone caused all client exits.

## New autonomous checks (no user's PC or saved role mutation)
- Archived GitHub branch server source using workflow `.github/workflows/jiuyin-stage60-source-offline-verify.yml` (run 35277241966), exported genuine Go dependencies for disconnected sandbox tests. Offline dependency copy removes the empty local sqlmock test shim; no gameplay/source `go.mod` change was committed for this test-only purpose.
- Worked in an isolated copy of the original RAR resources. On Linux, ~1,493 case aliases were added to that copy only because RAR file names are lowercase but INI references mixed case. Never transplant those symlinks to Windows or change the source filenames on the user's PC.
- `GOPROXY=off go test -mod=vendor -count=1 ./...`: exit 0, 17 test packages passed. Includes actual resources and the new `TestStage60ActualPlayerSpawnUnGatedQuickLZWire` committed to `server/cmd/protocol-probe/stage60_actual_bootstrap_wire_test.go`.
- Actual player spawn test calls `sendPlayerSpawn` with a synthetic player and NO diagnostic gate: five existing frames 0x10, 0x28, 0x10, 0x1F, 0x10. Real 0x28 QuickLZ compressed payload length 313 bytes, decompressed payload length 456, negotiated 23 properties; verifies QuickLZ header length, full decompress, entity identity, two transforms and property bytes. This validates our own encoder; it does not prove client accepts its semantics.
- Built Linux server with real vendored QuickLZ. Booted it with NO user account/role data; original resources copied to isolated directory; confirmed actual TCP LISTEN 127.0.0.1:19071, data catalogs loaded. Boot is NOT logged as a full client login/scene/reconnect E2E. No MySQL was connected.
- Windows CI workflow `.github/workflows/jiuyin-stage60-quicklz-verified.yml` now compiles the new actual-spawn test and builds the Windows executable; CI runner has no full user resources, therefore does not run the actual-spawn test itself. Windows workflow run 35277933469 is the run to check before declaring Windows PASS.

## Outstanding evidence gate
Client-side scene acceptance and actual ClientReady/reconnect persistence are **NOT RUN** with corrected QuickLZ. Prior gate3 LIVE diagnostic intentionally withheld required bootstrap frames, therefore cannot disprove a full un-gated scene. Continue examining newest client DLL evidence rather than guessing opcodes/field order. Do not ask the user for repeated live tests. Do not mark Stage60 DONE or item system COMPLETE until full LIVE E2E evidence exists.
