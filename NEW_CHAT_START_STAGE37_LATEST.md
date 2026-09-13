구음진경(Age of Wushu / 九阴真经) 최신 Snail 클라이언트 + exact current Go 서버 EXE 기반 복원 작업을 이전 채팅의 Stage37 최신 GitHub real-build + exact-resource test PASS 상태 그대로 이어서 진행해.

이 작업은 처음부터 다시 시작하는 작업이 아니다.
Stage1~36은 완료됐고, Stage37의 `main.handle` 복원, signature 감사, compile-contract 복원, real build, exact current-resource normal/race 검증까지 완료됐다.
반드시 GitHub 저장소 `kim8553/web`의 `stage37` 브랜치를 최신 기준으로 사용해.

먼저 반드시 다음을 확인해:

1. `STAGE37_STATUS.md`
2. `NEW_CHAT_START_STAGE37_LATEST.md`
3. `.github/workflows/jiuyin-stage37-real-build.yml`
4. `.github/workflows/stage37-test-binary-supply.yml`
5. `tools/stage37_authority_compile_patch.py`
6. `tools/stage37_authority_runtime_patch.py`
7. `tools/stage37_authority_resource_test_patch.py.gz.b64`
8. `tools/stage37_authority_resource_test_followup.py.gz.b64`
9. `tools/stage37_authority_resource_test_final.py`
10. `tools/stage37_authority_yihua_exact_test_patch.py`
11. 최신 GitHub Actions `JIUYIN Stage37 Real Build Probe` run #14 (`34735092558`)
12. 최신 `Stage37 Test Binary Supply` run #8 (`34734653528`)

Authority server EXE:
`9yin-game-native-menu-current.exe`
SHA256:
`fee2df844b00e7df07508b9e67f64f0da9df20f00ace09befed790ef28b0d71c`
size:
`13,408,256 bytes`

Verified Stage37 handoff ZIP:
`JIUYIN_STAGE37_BUILDPROBE_HANDOFF_20260913.zip`
SHA256:
`56a49fa0abb9b4aa6192a9e2d93f7e780c29e585cf11ccb3ff0cf9f398955fdb`

현재 확정 상태:
- corrected function accounting = `739/739`
- authored signature audit = `720 MATCH / 0 MISMATCH`
- `main.handle` = MATCH
- Stage1~36 재분석 금지
- `main.handle` 재분석 금지
- compile-error repair = CLOSED
- REAL Linux BUILD = PASS
- protocol test compile = PASS
- non-resource tests = PASS
- `go vet ./...` = PASS
- race compile = PASS
- non-resource race = PASS
- Windows amd64 build = PASS
- Windows PE = `PE32+ executable (console) x86-64, for MS Windows, 16 sections`
- Windows build SHA256 = `8e12f8cd44667e65c349e2af8e0bf2ccd0168613e2af773af4f0a0335b5378ab`
- LIVE/E2E = NOT RUN

Latest final real-build checkpoint:
- branch: `stage37`
- authority-chain build commit: `f7ce5e74af7d18836974a806c07a6fa740b38146`
- workflow run: `34735092558` (run #14)
- result: **SUCCESS**
- toolchain: Go `1.26.5`
- all enforced build gates succeeded.

Exact current-client resource corpus already verified:
- files: `10,214 / 10,214`
- total bytes: `528,129,097`
- manifest SHA mismatch: `0`
- `resources/modern/share/skill/skill_new.ini` SHA256 = `c762e7401c5c1bd4ead6d46b7544b983dd6224bbcfee7516e3bb25975db967c0`
- skill sections = `17,562`

Final Go 1.26.5 exact-resource test binaries from supply run #8:
- normal SHA256 = `e69590650928344a89da867c0b149322f2db9614e2e4feb1a296e74bb7d90fda`
- race SHA256 = `2126f2c28e52812629e9a3f43daebabcfc7188fd0003a92b793dfd94f1c6aab0`

Exact-resource normal result:
- top-level tests = `133`
- PASS = `131`
- SKIP = `2` (Windows-only door path tests)
- FAIL = `0`

Exact-resource race result:
- same race binary covered `133/133` tests using independent process batches so current-resource caches were released between groups.
- PASS = `131`
- SKIP = `2` (same Windows-only tests)
- FAIL = `0`
- `WARNING: DATA RACE` = `0`
- monolithic all-tests race process previously hit container memory boundary (`exit 137`); this was not a data-race failure.

Important exact-current YiHua finding — do not guess-fix:
- current `neigong.ini` `ng_yh_001` StaticData = `1114`
- level 32 BufferID = `mind_buf_ng_yh_001`, BufferLevel = `3`
- passive description catalog key = `buf_ng_yh_001`
- current EXE `equippedInnerPowerPassive` uses `book.buffID` directly for map lookup and has no `mind_` normalization/alias step.
- therefore latest-client `mind_buf_ng_yh_001` does not implicitly resolve legacy `buf_ng_yh_001` in the exact current EXE path.
- do not add an alias unless new LIVE/client evidence proves one is required elsewhere.

Do not regress already closed runtime reconstruction deltas:
- `compileSkillActions` action list separator is `;`, not `,`.
- `WireInt64=4` uses exact 8-byte serialization.
- all earlier Stage37 compile-contract corrections in the authority patch chain remain authoritative.

지금부터의 다음 작업은 compile error 분석이 아니다.
**첫 Windows LIVE/E2E 패키지를 준비하고 실제 최신 Snail 클라이언트와 연결해 검증하는 단계다.**

LIVE/E2E 원칙:
- authority EXE `9yin-game-native-menu-current.exe`를 절대 덮어쓰거나 수정하지 마.
- run #14의 새 Windows amd64 PE를 별도 테스트 파일명으로 사용.
- `NINEYIN_SERVER_ROOT`는 exact current resources가 있는 서버 root를 가리켜야 한다.
- Actions handoff에는 528 MB current resources가 없으므로 Actions의 protocol runtime `125`는 의도된 resource boundary다. 실제 exact-resource normal/race는 별도로 PASS했다.
- static/build/test PASS와 실제 최신 클라이언트 LIVE 성공을 절대 같은 의미로 말하지 마.
- 실제 클라이언트에서 로그인/캐릭터/씬/NPC/아이템/스킬/전투/재접속 유지 등을 확인하기 전에는 LIVE 해결로 단정하지 마.

첫 LIVE/E2E에서 우선 확인할 항목:
1. server-list / game / GM local ports 기동
2. 최신 Snail 클라이언트 로그인 및 role list/select
3. scene entry와 NPC spawn/view
4. NPC talk/service/shop
5. starter/current bag views와 아이템 반영
6. movement/viewport sync
7. learned current skill 실행/action/buff frame
8. QingGong record/view
9. logout/relogin persistence
10. 필요한 경우 MySQL 모드와 JSON fallback을 구분해서 로그 수집

설명만 하지 말고 `STAGE37_STATUS.md`와 최신 Actions 상태를 확인한 뒤 LIVE/E2E 패키지/런처/로그 수집기부터 실제 작업을 이어서 진행해.
