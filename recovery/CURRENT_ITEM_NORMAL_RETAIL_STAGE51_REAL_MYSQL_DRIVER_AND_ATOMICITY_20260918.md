# Stage51 — 실제 MySQL 드라이버 복구 및 일반 구매 저장 검증 (2026-09-18)

## 기준과 범위

Stage50 인계 `636bd74d7ccb25a368e32edf6387770321bba7c2`, `kim8553/web` 브랜치 `stage37-assistant-handoff-20260917`에서 이어서 진행. 이번 단계는 **일반 NPC 상점 구매 0x46의 MySQL 저장 경로**만 검증한다. GM 지급과 교환 구매 0x4F는 사용자 요청대로 보류하고 변경하지 않았으며, 일반 판매 0x47의 가격이나 처리 규칙은 추정하지 않았다. 최신 클라이언트 EXE/DLL/패키지, 디코딩한 Lua 및 비공개 키는 공개 저장소에 업로드하지 않았다.

## 선행 차단 원인: 실제 SQL 드라이버가 없는 빌드

Stage50의 `server/go.mod`는 `github.com/go-sql-driver/mysql v1.9.3`을 요구하는 듯 보였지만 `replace github.com/go-sql-driver/mysql => ./_builddeps/mysql`로 덮어쓰고 있었다. 해당 대체 모듈에는 MySQL TCP 인증, `database/sql` 드라이버 등록 또는 쿼리 실행 구현이 없고 `ParseDSN`/`FormatDSN`/오류형만 존재했다. 종속 라이브러리 `filippo.io/edwards25519`도 빈 대체 모듈로 연결되어 있었다. 그러므로 이전 `go build` 통과를 MySQL 사용 가능의 근거로 삼을 수 없었다. `openGameDataDB`의 실제 `sql.Open("mysql", normalized)`/`PingContext`에는 정상 드라이버가 필요하다.

공식 `github.com/go-sql-driver/mysql v1.9.3` 및 `filippo.io/edwards25519 v1.1.0`을 Go 모듈 캐시에서 내려받아 대체 모듈 경로 두 개를 제거하고 컴파일 및 실제 DB 검증 후에만 `server/go.mod` 수정 사항을 게시했다. 최종 수정 커밋: [`9f4a4b72178e4aa825ab36f6e997dbc72679a320`](https://github.com/kim8553/web/commit/9f4a4b72178e4aa825ab36f6e997dbc72679a320). `go.sum`은 기존 공식 모듈 체크섬을 포함하고 있어 변경하지 않았다. 다른 `x/text` 등 로컬 대체 모듈은 이번 단계에 수정하지 않았다.

## 실제 DB 테스트 설계 및 결과

`server/cmd/protocol-probe/latest_client_normal_shop_mysql_real_test.go`는 `mysql_integration` 태그와 환경변수로 명시적으로 활성화된다. 반드시 `NINEYIN_STAGE51_DISPOSABLE=YES_DROP_STAGE51_TABLES`이고 연결된 DB 이름이 **정확하게** `jiuyin_stage51`인 경우에만 테스트용 테이블을 만들거나 삭제한다. 이 테스트는 기존 MySQL 가방·재화 저장 코드가 사용하는 열을 모사한 별도의 **폐기용 테이블**을 사용한다. 실제 사용자 DB나 본서버 마이그레이션을 실행하지 않았다.

[Stage51 GitHub Actions 실행 35244755660](https://github.com/kim8553/web/actions/runs/35244755660) `real-mysql-atomic` 작업 **SUCCESS**, 독립 MySQL 8.0.46 InnoDB 컨테이너 사용. 공식 드라이버 연결/Ping 후 7개 실DB 하위 테스트 모두 PASS:

1. 구매 성공 후 독립 SQL 연결에서 아이템 추가와 잔액을 재조회.
2. 저장된 재화가 예상과 다를 때 구매와 모든 변경 거부.
3. 저장된 가방이 예상과 다를 때 구매와 모든 변경 거부.
4. 가방 INSERT에 중복 슬롯 오류를 발생시키고 이전 가방·잔액이 모두 유지되는지 확인.
5. 재화 UPDATE에 DB 트리거로 오류를 발생시키고 이전 가방·잔액이 모두 유지되는지 확인.
6. 재화 행이 누락되었을 때 변경 없이 거부.
7. 동일 이전 상태로 동시 구매 2개를 발생시켜 정확히 한 트랜잭션만 커밋되고 하나는 거부되는지 확인.

Stage49 표준 라이브러리 가짜 SQL 드라이버 회귀 테스트 8개도 `-race`로 전부 PASS. 실서버 `go test -c` 및 Linux/Windows amd64 `go build` 모두 PASS. CI는 테스트 후 공식 드라이버 수정 사항만 `server/go.mod`에 커밋·푸시했다. 마지막 GitHub 브랜치 파일을 다시 읽어 두 공식 모듈의 `replace` 삭제를 확인했다. GitHub Actions의 임시 MySQL 컨테이너는 종료·삭제되었다.

## 검증 경계와 다음 작업

**검증된 것:** 공식 Go MySQL 드라이버 종속성 및 독립 InnoDB 테스트 테이블에서 Stage49 원자적 가방+재화 저장 함수의 정상 커밋, SQL 오류 롤백, 경합 거부, 독립 재조회. Stage50 구매 라우팅 코드는 유지.

**검증되지 않은 것:** 사용자 Windows 서버 부팅, 사용자 `nineyin` DB의 실제 테이블/열·마이그레이션 적합성, 실제 게임에서 NPC 상점 열기→결제→가방 화면 반영, 종료 및 재접속 후 동일 아이템 유지, 다른 저장 경로와의 동시성 및 재전송 멱등성, 모든 아이템·수량, JSON 통합 저장, 일반 판매 0x47. 테스트용 테이블은 실제 사용자 SQL 스키마와 동일함이 입증되지 않았으며, DB 트랜잭션 PASS만으로 게임 구매 PASS를 주장하면 안 된다. 기존 `skill_new.ini` 초기화 의존도 본 작업에서 우회하지 않았다. **지금 기존 게임 서버에 접속해 구매하라고 안내하지 않는다.**

**다음 단일 우선순위:** 읽기 전용으로 실제 서버 스키마·초기 저장 경로(`role_currency` 기존 행, `role_bag_items` 필수 열, 초기화/migration, JSON 선택 여부)를 검증해 사용자 DB를 파괴하지 않는 재현 가능한 테스트 환경을 마련하고, 해당 환경에서 서버 부팅 및 실구매·재접속을 검증한다. 이어서 근거 있는 일반 판매 0x47을 구현한다. GM 지급과 교환 0x4F는 계속 보류한다.
