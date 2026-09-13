# Jiuyin Stage37 build status — latest handoff

Authority server EXE:
- file: `9yin-game-native-menu-current.exe`
- SHA256: `fee2df844b00e7df07508b9e67f64f0da9df20f00ace09befed790ef28b0d71c`
- size: `13,408,256 bytes`

Latest handoff ZIP:
- `JIUYIN_STAGE37_BUILDPROBE_HANDOFF_20260913.zip`
- SHA256: `56a49fa0abb9b4aa6192a9e2d93f7e780c29e585cf11ccb3ff0cf9f398955fdb`
- stored on `stage37` as `part00` + `part01`
- GitHub Actions reassembly/ZIP verification: PASS

Current reconstruction state:
- corrected function accounting: `739 / 739`
- authored signature audit: `720 MATCH / 0 MISMATCH`
- `main.handle`: MATCH
- Stage1~36 and Stage37 main.handle reconstruction: DO NOT RESTART
- full BUILD PASS: NOT YET ESTABLISHED
- LIVE/E2E: NOT RUN

GitHub build environment:
- local ChatGPT outbound DNS limitation is no longer the blocker
- GitHub Actions downloads real dependencies successfully
- exact build toolchain used in Actions: Go `1.26.5`
- authority-exact dependency versions used by the build probe:
  - `github.com/Hiroko103/go-quicklz v0.0.0-20190115215310-59904abc50d0`
  - `github.com/go-sql-driver/mysql v1.9.3`
  - `golang.org/x/text v0.28.0`
  - `filippo.io/edwards25519 v1.1.0`
  - `github.com/DATA-DOG/go-sqlmock v1.5.2` for tests/tooling

Latest GitHub branch checkpoint before this status update:
- branch: `stage37`
- commit: `f220a89b27efa79f718cbf15ee9d50b5c23faaf3`
- commit message: `Add third authority-backed compile-contract batch`
- latest real build workflow run: `34729310251` (run #5)
- workflow: `.github/workflows/jiuyin-stage37-real-build.yml`
- authority-backed patch script: `tools/stage37_authority_compile_patch.py`

Already closed compile-contract deltas — do not regress them:
1. NPC loader current 4-return contract; remove legacy `loadChengduNPCSpawns`
2. `playerActor` current DWARF has no `activeParry` field; remove legacy activeParry storage methods rather than inventing a field
3. exact ActorState fields `MaxQingGongPoint` / `MaxQingGongPointAdd`
4. current clientdata `WireInt64=4`, `Value.I64`, `Int64Value(int64)`
5. `S2CFacultyMessage` current type `int32`, value `191`
6. current openRoleStore uses no-op close closures; no authored `JSONRepository.Close`
7. `sceneNPCRegistry.taskSpawns` current field restored
8. transport menu helper/base restored; `reconcileViewportLocked` current two-return contract handled
9. `sendEntryScene(conn, scene, name)` current signature
10. legacy HeartBuddha/TaiJi shortcut workaround helpers removed
11. exact `effectsHaveKind` helper restored
12. skill switch current variadic message call and `WriteFrame` contract
13. current item/equip catalogs use `.byID`, not legacy `.items`
14. current JingMai record builders use fixed player object/owner constants rather than non-current ObjectID/OwnerID methods
15. `serverViewProperty.nest *serverViewNest` current field restored

Latest real `go build ./cmd/protocol-probe` remaining compiler errors from run `34729310251`:
1. `cmd/protocol-probe/zz_recovered_overlay.go`: imported `internal/entity` but not used
2. `zz_recovered_overlay.go:3750`: undefined `modernWuxueWuxingPath`
3. `zz_recovered_overlay.go:4289`: one variable assigned from `allJianghuQingGongSkillIDs`, but current function returns two values
4. `zz_recovered_overlay.go:6167` and `:6240`: undefined `horizontalDistanceXZ`
5. `zz_recovered_overlay.go:7764`: undefined `quicklz`
6. `cmd/protocol-probe/skill_combat.go`: imported `internal/world` but not used

IMPORTANT NEXT STEP:
- Continue from the six issue groups above only.
- For every fix, first verify current EXE DWARF/machine code/CALL/xref/source-line/static-data/source attribution.
- Do not fix unused imports by guess if they reveal an overlay source-attribution problem.
- Do not invent `modernWuxueWuxingPath`, `horizontalDistanceXZ`, quicklz usage/import, or the QingGong return handling without current authority evidence.
- Update `tools/stage37_authority_compile_patch.py`, let GitHub Actions rebuild with real dependencies, and repeat until `go build ./cmd/protocol-probe` exits 0.
- After Linux build PASS: run `go test`, `go vet`, race where supported, then Windows amd64 build and PE/SHA checks.
- Only after a real test package exists begin LIVE/E2E.
