구음진경(Age of Wushu / 九阴真经) 최신 Snail 클라이언트 + exact current Go 서버 EXE 기반 복원 작업을 이전 채팅의 Stage37 최신 GitHub real-build 상태 그대로 이어서 진행해.

이 작업은 처음부터 다시 시작하는 작업이 아니다.
Stage1~36은 이미 완료됐고, Stage37의 `main.handle` 복원 및 signature 감사도 완료됐다.
반드시 GitHub 저장소 `kim8553/web`의 `stage37` 브랜치를 최신 기준으로 사용해.

먼저 반드시 다음을 확인해:

1. `stage37/STAGE37_STATUS.md`
2. `.github/workflows/jiuyin-stage37-real-build.yml`
3. `tools/stage37_authority_compile_patch.py`
4. 최신 GitHub Actions `JIUYIN Stage37 Real Build Probe` run
5. `JIUYIN_STAGE37_BUILDPROBE_HANDOFF_20260913.zip.part00`
6. `JIUYIN_STAGE37_BUILDPROBE_HANDOFF_20260913.zip.part01`

Handoff ZIP 원본 SHA256:
`56a49fa0abb9b4aa6192a9e2d93f7e780c29e585cf11ccb3ff0cf9f398955fdb`

Authority server EXE:
`9yin-game-native-menu-current.exe`
SHA256:
`fee2df844b00e7df07508b9e67f64f0da9df20f00ace09befed790ef28b0d71c`
size:
`13,408,256 bytes`

현재 확정 상태:
- corrected function accounting = `739/739`
- authored signature audit = `720 MATCH / 0 MISMATCH`
- `main.handle` = MATCH
- Stage1~36 재분석 금지
- `main.handle` 처음부터 재분석 금지
- BUILD PASS = 아직 아님
- LIVE/E2E = NOT RUN

네트워크 문제는 해결됐다.
GitHub Actions에서 Go `1.26.5`와 실제 외부 모듈 다운로드가 성공한다.
따라서 더 이상 로컬 ChatGPT 컨테이너의 DNS 제한을 BUILD blocker로 취급하지 마.

latest real-build checkpoint before status-doc commit:
- branch: `stage37`
- commit: `f220a89b27efa79f718cbf15ee9d50b5c23faaf3`
- workflow run: `34729310251` (run #5)
- result: real dependency download PASS, `go build ./cmd/protocol-probe` compile stage reached, build FAIL due to remaining source-contract deltas

이미 authority 근거로 닫힌 compile-contract 수정은 되돌리지 마:
- NPC loader 4-return current contract
- legacy `loadChengduNPCSpawns` 제거
- `playerActor`에 activeParry field를 임의로 추가하지 않음; current DWARF에 없어서 legacy storage methods 제거
- `MaxQingGongPoint` / `MaxQingGongPointAdd`
- `WireInt64=4`, `Value.I64`, `Int64Value(int64)`
- `S2CFacultyMessage int32 = 191`
- JSONRepository Close 추측 구현 금지; current openRoleStore no-op close closures 유지
- `sceneNPCRegistry.taskSpawns`
- transport menu current helper/base
- `reconcileViewportLocked` two-return contract
- `sendEntryScene(conn, scene, name)`
- old HeartBuddha/TaiJi shortcut workaround 제거
- current `effectsHaveKind`
- skill switch variadic custom-message + `WriteFrame`
- item/equip catalog `.byID`
- JingMai record fixed player object/owner constants
- `serverViewProperty.nest`

지금 바로 이어갈 최신 remaining compile errors는 정확히 다음 6개 issue group이다:

1. `cmd/protocol-probe/zz_recovered_overlay.go`: `internal/entity` unused import
2. `zz_recovered_overlay.go:3750`: `modernWuxueWuxingPath` undefined
3. `zz_recovered_overlay.go:4289`: `allJianghuQingGongSkillIDs`가 2 values를 반환하는데 1 variable로 받고 있음
4. `zz_recovered_overlay.go:6167`, `6240`: `horizontalDistanceXZ` undefined
5. `zz_recovered_overlay.go:7764`: `quicklz` undefined
6. `cmd/protocol-probe/skill_combat.go`: `internal/world` unused import

중요 작업 원칙:
- 컴파일 오류만 보고 추측 수정 금지.
- 각 항목은 current EXE DWARF / 기계어 / CALL / xref / source line / PE static data / Go type info로 먼저 확인.
- unused import도 단순 삭제 전에 overlay source-attribution 문제인지 확인.
- `modernWuxueWuxingPath`, `horizontalDistanceXZ`, quicklz import/호출, QingGong 2-return 처리 형태를 임의 작성하지 마.
- authority 근거가 확보된 수정만 `tools/stage37_authority_compile_patch.py`에 추가.
- 수정 후 GitHub Actions real build를 다시 실행/자동 실행하고 로그를 확인.
- `go build ./cmd/protocol-probe` exit 0이 나올 때까지 반복.

BUILD가 닫힌 다음 순서:
1. Linux real dependency build PASS
2. `go test`
3. `go vet`
4. race 가능한 범위
5. Windows amd64 build
6. PE 확인
7. SHA256 기록
8. 최초 LIVE/E2E 테스트 패키지 생성

STATIC 성공과 LIVE 성공을 절대 혼동하지 마.
실제 최신 Snail client로 테스트하기 전에는 해결됐다고 단정하지 마.

설명만 하지 말고 `STAGE37_STATUS.md`와 최신 Actions 로그를 확인한 뒤 remaining compile error #1부터 바로 실제 분석을 이어서 진행해.
