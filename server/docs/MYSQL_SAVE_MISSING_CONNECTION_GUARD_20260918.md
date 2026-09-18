# MySQL 저장 연결 누락 시 거짓 성공 차단 (2026-09-18)

## 확정된 소스 결함

`cmd/protocol-probe/zz_recovered_overlay.go`에서 `(*mysqlBagStore).Save`와 `(*mysqlCurrencyStore).Save`는 저장소 수신자가 nil이거나 DB 연결이 nil일 때 `nil` 오류를 반환하고 있었다. 이는 실제 SQL을 실행하지 않았으면서 저장 성공으로 취급할 수 있다. 직전 수정된 `grantBagItemDurably`가 `Save` 오류에 의존하므로 GM 지급도 이 연결 누락 경로에서는 잘못 성공할 수 있었다.

수정: 두 MySQL 저장 함수가 DB 연결 누락 시 각각 `bag store: missing database connection`, `currency store: missing database connection` 오류를 반환한다. 기존 SQL/트랜잭션, 정상 DB 연결 경로, 패킷/로그인/맵 진입/아이템 구현은 변경하지 않는다. 저장 실패를 반환받는 기존 GM 지급 경로는 지급 전 액터 가방 상태로 복구한다.

- 수정 전 `zz_recovered_overlay.go` Git blob: `ec375f90d32198535ff6c9e10f1b54a45d6ebe99`.
- 수정 후 Git blob: `9329ab12bedba7c7089d959f59f12e1de7563641`.
- 실제 소스 변경 커밋: `8ea465fff5a66bc64ff86fd5d6f60d98eb708def`.
- 신규 테스트: `cmd/protocol-probe/mysql_store_missing_connection_test.go` (nil receiver/DB 연결 없음 각각 가방·재화, GM 지급 복구, 지연된 재화 저장과 가방 이동 저장 실패).

## 재현·검증 기록

- 수정 전 새 회귀 테스트 실행: `go test -modfile=ci.real.mod -count=1 -run '^(TestMySQLStoreMissingConnectionRejectsWrites|TestGMGrantMissingMySQLConnectionRestoresActor)$' ./cmd/protocol-probe` → FAIL. 연결 누락 저장 4개 사례와 GM 지급 1개 사례가 모두 거짓 성공을 재현함.
- 수정 후 위 테스트 및 지연 재화·가방 변경 검증, 기존 GM 지급 회귀 테스트, 일반 상점 관련 선택적 테스트 → PASS (`NINEYIN_SERVER_ROOT`로 제공된 부분 리소스 fixture를 사용).
- `go test -modfile=ci.real.mod -count=1 ./internal/shopbuyatomic ./migrations ./internal/launchsafety` → PASS. `go vet -modfile=ci.real.mod ./...` → PASS.
- `GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -buildvcs=false -trimpath -modfile=ci.real.mod ... ./cmd/protocol-probe` → Windows x86-64 PE 빌드 PASS. 해당 로컬 바이너리는 감사 전용이며 사용자 로그인 성공 ZIP을 대체하지 않는다. `-buildvcs=false`는 로컬 체크아웃의 Git 소유권 검사 때문에 지정했으며 배포/실게임 검증과 무관하다.
- 전체 `go test -modfile=ci.real.mod -count=1 ./...` → **FAIL: `cmd/protocol-probe`에서 14개 테스트 실패**. 제공된 부분 fixture에서 `share/trade/shop.ini`, `share/skill/neigong/neigong_static.ini`, `text/chineses/desc2.idres`, `share/skill/qinggong/qgskill.ini`, 일부 씬 NPC creator 폴더가 없어 발생한 오류 및 관련 서비스 기대값 차이가 포함된다. 전체 PASS라고 보고하지 않는다.

## 출처·남은 작업

- 구현 계보 원본 `9yin-go-server1.rar`, 현재 브랜치 `stage37-restored-20260918`, 최신 BIN64/동일 시점 `res`는 `USER_AUTHORITY_AND_WORK_RULES_20260918.md`를 따른다.
- Google Drive `res`의 `share.package`를 실제 읽기 전용으로 확보하여 SHA-256 `200497852ba3a29279e51f01e2913b5f7260480f2a740a32ebd1680b869844b6`을 검증했다. zlib 스트림 일부에서 NPC 테이블의 `Shop_yaopin_00100` 참조가 확인됐지만, 이는 상점 정의나 **NPC 판매 패킷/정산 규칙의 증거가 아니다**. `res` 전체를 해독했다고 주장하지 않는다.
- 사용자 PC의 `nineyin` DB 구조·실제 구매/판매·GM 지급·재접속·최신 클라이언트 LIVE/E2E는 이번 변경에서 실행하지 않았다. `tools/check-shop-db-columns-readonly.sql`의 실제 사용자 DB 실행 결과도 없다. 자동 스키마 수정, 사용자 리소스 덮어쓰기 및 근거 없는 판매 구현 금지.
