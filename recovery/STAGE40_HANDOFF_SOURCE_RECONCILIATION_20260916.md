# Stage37 Stage40 — 구축 프로젝트 인계 ZIP과 GitHub 생성 소스 대조 (2026-09-16)

## 기준 및 비파괴 원칙
- 프로젝트 Library에서 실제 2026-09-15 인계 ZIP `JIUYIN_STAGE37_NEW_CHAT_HANDOFF_20260915_BIND_LUA_CURRENT.zip`을 찾아서 원본 사본을 검사했다. 489,788 bytes, SHA-256 `e898008a0ec1aef39d16663835c9f479daed6e643c4c90252c5adafc3f4ffe1e`.
- ZIP CRC PASS. `HANDOFF_MANIFEST_SHA256.txt`의 **189/189 원본 파일 내용 SHA-256 PASS**. UTF-8 플래그가 없는 중국어 이름 `.bat` 두 개는 ZIP의 CP437 표기를 원래 UTF-8 파일명으로 되돌려 해시를 검증했다. 파일 내용을 변경한 것이 아니다.
- 검증된 `STAGE37_CURRENT_WORKTREE`는 파일 **186개**이며 Go 파일은 156개이다. 인계 ZIP·RAR·사용자 PC·MySQL·클라이언트 바이너리를 수정하거나 공개 GitHub에 원본을 올리지 않았다.
- 비교 기준 GitHub Actions run [35041631604](https://github.com/kim8553/web/actions/runs/35041631604), commit [`4b7c975d40cf6223807c0a5eeb71d1eefdb646bf`](https://github.com/kim8553/web/commit/4b7c975d40cf6223807c0a5eeb71d1eefdb646bf), artifact ID `10425965179`, artifact SHA-256 `8bf7282b957655c1ef42993b68cc3f8177f84190dc98a03f14c671cbdc80b37d`. 다운로드 사본 CRC PASS. `generated-manifest.txt` 해시 `99f1b40a4a042e51f8e4c36255710094910aa05ce80853920be26f39c3d3e2dd` 검증 후 210개 경로를 비교했다.

## 실제 파일별 대조 결과

| 항목 | 검증된 개수 |
| --- | ---: |
| 2026-09-15 인계 worktree 파일 | 186 |
| Actions 빌드210의 소스 매니페스트 파일 | 210 |
| 동일 경로 + 동일 SHA-256 | 173 |
| 동일 경로 + 서로 다른 SHA-256 | 13 |
| 인계 ZIP에만 있는 경로 | **0** |
| Actions 쪽에 새로 추가된 경로 | **24** |

13개 변경 경로:
- `cmd/protocol-probe/latest_client_bag_view_ordinal_compat.go`
- `cmd/protocol-probe/latest_client_bag_view_ordinal_compat_test.go`
- `cmd/protocol-probe/latest_client_current_bag_wire_test.go`
- `cmd/protocol-probe/latest_client_nonbag_view_ordinal_compat.go`
- `cmd/protocol-probe/latest_client_player_property_ordinal_compat_test.go`
- `cmd/protocol-probe/latest_client_shop_exchange_config.go`
- `cmd/protocol-probe/latest_client_shop_exchange_config_test.go`
- `cmd/protocol-probe/latest_client_shop_exchange_contract.go`
- `cmd/protocol-probe/main.go`
- `cmd/protocol-probe/main_test.go`
- `cmd/protocol-probe/messages.go`
- `cmd/protocol-probe/zz_recovered_overlay.go`
- `migrations/runner_test.go`

24개 추가 경로는 `internal/exchangebinding`, `internal/exchangecommit`, `internal/exchangegate`, `internal/exchangeplan`의 구현/테스트, `cmd/protocol-probe`의 exchange stage/preflight/apply/commit/gate/가방·장비 테스트 및 `migrations/0012_persist_runtime_bind_status.sql`로 구성된다. 누락 경로가 0이라는 것은 9월 15일 파일 이름이 후속 빌드210에서 빠지지 않았다는 뜻이다. **13개 변경의 의미론적 무손실, 최신 Stage34 코드까지의 단일 통합, 실제 클라이언트 작동까지 보증하지 않는다.**

## GitHub 소스 계보의 추가 경계
- 2026-09-16 Stage34 진단 빌드는 `tools/stage37_current_recovery_stage34.sh -> stage32.sh -> ...`를 거쳐 `9yin-go-server1.rar` 기반의 `buildtree`를 새로 만든다. 검증된 Stage34 artifact ID `10445717538`, ZIP SHA-256 `602742981c75e37af7a803c8bba9b05c45657c3a6a3f4a4402c0b3e5197c545d`, 상태 로그에는 `source_basis=VERIFIED_9yin-go-server1.rar_STAGE32_PRESERVED`라고 명시됐다.
- 따라서 Stage34 진단 EXE가 9월 15일 인계의 모든 후속 변경과 빌드210의 24개 추가 파일을 보존했는지는 **별도 확인 전에는 알 수 없다**. Stage34를 자동으로 canonical production 서버로 승격시키지 않는다.
- 최신 브랜치 루트는 Go production worktree를 직접 담지 않고 복구용 압축 조각·패치·CI 스크립트로 생성한다. GitHub에서 코드 빌드가 가능하다는 것과 최신 생성 Go 소스 전체가 직접 편집 가능한 Git 트리에 있다는 것은 다르다.

## 이번 단계 신규 도구·후속 감사
- `tools/stage40_handoff_source_compare.py`는 ZIP SHA256/CRC, 내부 189개 해시, 중국어 파일명 복원, Actions source manifest 자체 SHA256와 전체 경로/파일 해시를 **읽기 전용**으로 비교한다. 실제 입력 결과 위 숫자와 동일. 합성 self-test 7개 시나리오 PASS, `python -m py_compile` PASS; GitHub 업로드본 Git blob SHA는 로컬 `git hash-object`와 동일하다.
- [Stage40 Source Lineage Audit](https://github.com/kim8553/web/actions/workflows/stage37-stage40-source-lineage.yml)는 실제 Stage34 `buildtree`를 재구성한 뒤 빌드210 매니페스트와 경로·SHA256을 비교하도록 추가했다. baseline 210 artifact 다운로드 성공은 확인했지만 이 문서 작성 시점의 **최종 실행 결과는 아직 검증되지 않았다**. 누락이 있으면 실패시키고 자동 코드 승격·서버 교체는 하지 않는다.

## 완료로 주장하지 않는 것
- 이 단계에서 production Go 코드 병합·최신 통합 EXE 배포·새로운 전체 Go 빌드·실게임 LIVE/E2E는 **NOT RUN / NOT COMPLETE**.
- `0x4F` 실제 교환 mutation, NPC 일반 구매, GM 지급의 가방 표시·재접속 지속성을 성공 처리하지 않는다.
- 다음 정확한 경계는 Stage40 Actions 감사 결과에서 Stage34 대비 빌드210의 누락·변경 경로를 확인한 다음, 기존 9월 15일 기준본 및 후속 안전 패치를 잃지 않는 단일 Go 소스 트리를 구성하고 빌드·실게임 테스트하는 것이다.
