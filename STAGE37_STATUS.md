# Jiuyin Stage37 build status — 2026-09-13 final static/build checkpoint

## Authority

Authority server EXE:
- file: `9yin-game-native-menu-current.exe`
- SHA256: `fee2df844b00e7df07508b9e67f64f0da9df20f00ace09befed790ef28b0d71c`
- size: `13,408,256 bytes`

Verified Stage37 handoff:
- `JIUYIN_STAGE37_BUILDPROBE_HANDOFF_20260913.zip`
- SHA256: `56a49fa0abb9b4aa6192a9e2d93f7e780c29e585cf11ccb3ff0cf9f398955fdb`
- stored on `stage37` as `part00` + `part01`
- GitHub Actions reassembly/SHA verification: PASS

Do not restart Stage1~36 or `main.handle` analysis.

## Reconstruction accounting

- corrected function accounting: `739 / 739`
- authored signature audit: `720 MATCH / 0 MISMATCH`
- `main.handle`: MATCH
- compile-contract reconstruction: CLOSED
- production source runtime-contract deltas found during current-resource execution: CLOSED (`compileSkillActions` semicolon split and `WireInt64=4` serialization)

## Real Build — PASS

Latest final build checkpoint before this status update:
- branch: `stage37`
- commit: `f7ce5e74af7d18836974a806c07a6fa740b38146`
- workflow: `JIUYIN Stage37 Real Build Probe`
- run: `34735092558` (run #14)
- conclusion: **SUCCESS**
- toolchain: Go `1.26.5`

Run #14 enforced results:
- production `go build ./cmd/protocol-probe`: `0`
- protocol test compile: `0`
- non-resource tests: `0`
- `go vet ./...`: `0`
- protocol race compile: `0`
- non-resource race tests: `0`
- Windows `GOOS=windows GOARCH=amd64 CGO_ENABLED=0` build: `0`
- Actions protocol-runtime resource boundary: `125` only because the 528 MB current-client resource corpus is intentionally not stored in the handoff; this is not treated as a build failure.

Windows amd64 output from run #14:
- format: `PE32+ executable (console) x86-64, for MS Windows, 16 sections`
- Go version: `go1.26.5`
- SHA256: `8e12f8cd44667e65c349e2af8e0bf2ccd0168613e2af773af4f0a0335b5378ab`

Authority-exact external modules used:
- `github.com/Hiroko103/go-quicklz v0.0.0-20190115215310-59904abc50d0`
- `github.com/go-sql-driver/mysql v1.9.3`
- `golang.org/x/text v0.28.0`
- `filippo.io/edwards25519 v1.1.0`

## Exact current-client resource verification — PASS

Verified current-client resource corpus:
- files: `10,214 / 10,214`
- total bytes: `528,129,097`
- manifest SHA mismatches: `0`
- `resources/modern/share/skill/skill_new.ini` SHA256: `c762e7401c5c1bd4ead6d46b7544b983dd6224bbcfee7516e3bb25975db967c0`
- `skill_new.ini` sections: `17,562`

Final Go 1.26.5 resource-test binaries compiled by Actions run `34734653528`:
- normal SHA256: `e69590650928344a89da867c0b149322f2db9614e2e4feb1a296e74bb7d90fda`
- race SHA256: `2126f2c28e52812629e9a3f43daebabcfc7188fd0003a92b793dfd94f1c6aab0`

Exact-resource normal execution:
- discovered top-level tests: `133`
- PASS: `131`
- SKIP: `2` (`TestDoorPackageRouteUsesInstalledDoorData`, `TestDoorResourceFromModernSceneConfig`; Windows-only path behavior)
- FAIL: `0`

Exact-resource race execution:
- one monolithic race process exceeded the container memory boundary and was SIGKILLed (`137`); it did not report a data race before termination.
- the same race binary was then executed in independent process batches so caches were released between groups.
- test coverage across successful race processes: `133 / 133`
- PASS: `131`
- SKIP: `2` (same Windows-only tests)
- FAIL: `0`
- `WARNING: DATA RACE`: `0`

## Current-client YiHua key finding — exact mismatch, do not guess-fix

Current client `neigong.ini`:
- `ng_yh_001` StaticData = `1114`
- level 32 BufferID = `mind_buf_ng_yh_001`
- BufferLevel = `3`

The passive description loader compiles `desc_mind_buf_ng_yh_001_*_ngtips` under the key `buf_ng_yh_001`.
Current EXE `main.(*playerActor).equippedInnerPowerPassive` machine code performs direct `mapaccess1_faststr` using `book.buffID`, followed by `mapaccess2_fast32` using `book.buffLevel`; there is no `mind_` normalization/alias step.
Therefore current full-unlock/equip of the latest-client `mind_buf_ng_yh_001` route does **not** implicitly resolve the legacy `buf_ng_yh_001` passive profile. Do not add an alias unless a later LIVE/client requirement is explicitly supported by new authority evidence.

## Closed compile/runtime deltas — do not regress

The final authority patch chain includes, among other already-verified items:
- NPC loader current 4-return contract; legacy `loadChengduNPCSpawns` removed
- no invented `playerActor.activeParry` field
- exact `MaxQingGongPoint` / `MaxQingGongPointAdd`
- `WireInt64=4`, `Value.I64`, `Int64Value(int64)`, 8-byte serialization
- `S2CFacultyMessage int32 = 191`
- current openRoleStore no-op close closures
- `sceneNPCRegistry.taskSpawns`
- transport menu current helper/base and two-return viewport reconcile
- current `sendEntryScene(conn, scene, name)`
- removal of non-current HeartBuddha/TaiJi shortcut workarounds
- current `effectsHaveKind`
- skill switch variadic custom-message + `WriteFrame`
- item/equip catalogs use `.byID`
- JingMai fixed player identity record builders
- `serverViewProperty.nest`
- `modernWuxueWuxingPath` current initializer
- QingGong two-return call form
- exact `horizontalDistanceXZ`
- exact quicklz module/import/call route
- generated unused import corrections
- current `compileSkillActions` action lists split on `;` (not `,`)

## Current state / next boundary

- STATIC reconstruction: PASS
- REAL Linux build: PASS
- normal tests with exact current-client resources: PASS
- race coverage with exact current-client resources: PASS in independent process batches
- vet: PASS
- Windows amd64 build / PE / SHA: PASS
- **LIVE/E2E: NOT RUN**

Next work is no longer compile-error repair. Prepare the first Windows LIVE/E2E package using the run #14 PE without overwriting the authority EXE, point `NINEYIN_SERVER_ROOT` at a root containing the exact current resources, then validate actual latest Snail client login/scene/NPC/item/skill behavior. STATIC/BUILD PASS must not be reported as LIVE success.
