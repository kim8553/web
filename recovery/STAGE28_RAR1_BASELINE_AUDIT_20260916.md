# Stage28 — 사용자 기준 `9yin-go-server1.rar` 확인 및 상점 파서 회귀 재현

기준 자료는 사용자가 잘못 올렸다고 정정한 ZIP이 아니라 `9yin-go-server1.rar`이다. 원본 RAR·기존 Stage37 소스·MySQL·실행 파일을 덮어쓰지 않는다. GM 웹페이지 아이템 지급은 사용자 요청대로 보류한다.

## 검증된 원본 및 공통 소스

- RAR SHA-256: `ddc2f6bc078660a40eedf077d43befd6402ed59478ccde987d44171c1d7aefa5`, RAR5, 항목 10,498개.
- RAR의 Go 파일 112개 모두 Stage26 검증 소스의 같은 경로에 존재. 바이트 동일 44개, 내용 차이 68개, Stage26에만 존재하는 Go 파일 88개, RAR에만 존재하는 Go 파일 0개.
- `README.md`, `go.sum`, `migrations/0001_normalize_role_repository.sql`은 바이트 동일. `main.go`, `scene_lifecycle.go`, `shop_catalog.go`는 서로 다름. 공통 기반의 증거는 있으나 정확한 Git 커밋 계보 및 변경의 완전 승계를 입증한 것은 아님.
- RAR 루트 `9yin-game-native-menu.exe` SHA-256 `34ce3c018cf7ab838578f8e15ee3709ebfa2a8106d183a531d5e78e3e056ca0b`; RAR의 `build/9yin-game-native-menu.exe` SHA-256 `7acc63911cfa36ca5ac6f99d9e9c079ae5f80e9066252f7be5dad47f3cbe89d2`. 두 실행 파일은 서로 다르며 현재 PC의 실행 파일은 미확인.

## RAR 원본 데이터로 실제 재현한 상점 오류

RAR `resources/modern/share/trade/shop.ini` SHA-256: `ceb7a3a7e17cbc118ce5a7d3d979dea6120c50cdf6a5823697c05a5b1b9cf5dd`.

독립적인 Go 단위 검사에서 RAR 원본 `cmd/protocol-probe/shop_catalog.go`와 Stage27 대응 파일 각각에 **동일한 업로드 RAR의 shop.ini**를 주었다.

- `Shop_menpai_xtc_01`: RAR 원본 파서 `invalid page=0 position=0` 실패, Stage27 파서 3개 상품·3페이지·첫 페이지 0으로 성공.
- `Shop_schoolmaster_xtb_01`: 원본 25개 상품이나 첫 페이지 1, Stage27은 25개 상품·첫 페이지 0.
- `Shop_ittaketwo_item`: 원본 `invalid page=0 position=0` 실패, Stage27은 2개 상품·첫 페이지 0으로 성공.

Stage37의 상점 파서 변경을 보존할 직접적인 이유다. 하지만 이것은 **카탈로그 로딩 검사**일 뿐 인게임 상품 표시·구매 성공이 아니다. 원본 RAR의 옛 상점 구매 핸들러를 Stage37에 무조건 복사하지 않는다. 정확한 최신 구매 패킷 미확인 상태에서는 Stage37의 fail-closed 구매 경로를 유지한다.

## 실행 검증의 범위

- RAR의 `internal/protocol/...`, `internal/transport/...`, `internal/npcfunc/...`은 RAR의 프로토콜 테스트 fixture를 사용한 `go test`와 `go test -race` 통과.
- RAR 전체 테스트는 이 로컬 환경의 필수 Go 외부 모듈 캐시 부족으로 미수행.
- Stage27 [GitHub Actions run 35080922642](https://github.com/kim8553/web/actions/runs/35080922642)는 성공하였으나 전체 프로토콜 테스트 실행에 필요한 정확한 최신 클라이언트 리소스가 없어 그 부분은 컴파일 검증만 수행. LIVE/E2E 미실행.

## 다음 단계

RAR을 원본 증거로 보존하고 Stage37의 68개 변경 파일·88개 추가 파일 가운데 NPC 상점, 가방, 화폐, DB 관련 변경을 근거별로 비교한다. 이후 실제 최신 클라이언트 버전의 NPC ShopID → `shop.ini` → Shop View 및 구매 C2S를 검증한다. Windows 빌드·실게임 동작은 별개로 판정한다.

**개인 데이터 보호:** 이 GitHub 문서에는 RAR 원문·캐릭터 데이터·비밀번호·리소스 본문을 업로드하지 않았다. 정확한 원본 파일 SHA·비개인 소스 비교 수치 및 익명화된 시험 결과만 기록했다.
