# NPC 구매 DB 구조: 읽기 전용 확인 도구 (2026-09-18)

## 근거 및 실행 범위

- 기준: `kim8553/web` 브랜치 `stage37-restored-20260918`, 작성 전 HEAD `223e3264d5a994c73ec86fd93b3f978dc09c7ed7`.
- 기존 `server/tools/check-shop-db-columns-readonly.sql`을 확장했다. `cmd/protocol-probe/zz_recovered_overlay.go`의 `mysqlBagStore.Load/Save`, `mysqlCurrencyStore.Load/Save`, `internal/shopbuyatomic/checked.go`의 역할 잠금·가방/재화 원자적 재작성, `migrations/runner.go`의 장부 스키마와 `0001_normalize_role_repository.sql`의 `roles` 선언에 근거한다. **기존 게임 로직·DB 마이그레이션은 수정하지 않는다.**
- 스크립트의 실행문은 `SELECT DATABASE()` 및 `information_schema.TABLES/COLUMNS/STATISTICS`에 대한 `SELECT` 다섯 개뿐이다. 계정·캐릭터·가방·재화의 실제 행이나 암호는 조회하지 않으며, `CREATE/ALTER/DROP/INSERT/UPDATE/DELETE`, 잠금, 마이그레이션, 자동 복구는 실행하지 않는다.

## 사용 방법 및 해석

이미 로그인된 MySQL 클라이언트에서 대상 DB를 **본인이 확인해 선택한 뒤** `SOURCE server/tools/check-shop-db-columns-readonly.sql;`로 실행할 수 있다. 또는 해당 저장소의 `server` 디렉터리를 기준으로 `mysql --database=nineyin --table < tools/check-shop-db-columns-readonly.sql`을 사용한다. 인증은 사용자 PC의 기존 안전한 설정을 사용하고 비밀번호/DSN 및 개인 데이터가 포함된 전체 DB 덤프는 공유하지 않는다. 대상 DB를 확인하지 않은 상태에서 실행하지 않는다.

출력은 순서대로 (1) 선택 DB, (2) 네 테이블의 존재·테이블 유형·트랜잭션 엔진, (3) 17개 필수 컬럼의 존재·실제 형식·부호·NULL 허용·문자열 길이와 인코딩, (4) 순서까지 확인한 실제 PRIMARY KEY와 기대 형태, (5) 관련 인덱스 목록이다. `roles` / `role_bag_items` / `role_currency`의 InnoDB 및 PK 구조는 기존 잠금·트랜잭션의 전제이며, `schema_migrations`는 기존 migration runner의 장부다.

`MATCH_REFERENCE`는 기존 migration 또는 격리 통합 테스트에 적힌 타입과 **참고 형식이 일치**한다는 뜻일 뿐 데이터 호환성 합격 판정이 아니다. `INTEGER_REVIEW_RANGE_AND_SIGN`은 Go의 정수 변환·값 범위를 검토해야 하고, `TEXT_REVIEW_CAPACITY_AND_CHARSET`은 ConfigID/이름의 잘림·문자셋을 검토해야 한다. `TEXT_JSON_VALIDITY_UNCHECKED`는 snapshot이 문자열로 저장되어도 실제 행의 JSON 유효성은 검사하지 않았다는 뜻이다. `MISSING_*`, `DIFFERENT_PRIMARY_KEY`, `REVIEW_NON_INNODB_ENGINE`, `REVIEW_TYPE_MISMATCH`는 먼저 구조 확인이 필요하다는 뜻이다. **이 보고서는 `모두 PASS`를 출력하지 않는다.**

이 SQL은 데이터 값·행별 JSON 유효성·컬럼 추가 제약(FK/CHECK/트리거)·실제 트랜잭션 충돌·마이그레이션 checksum 내용·사용자 PC의 현재 구조·게임 LIVE/E2E를 검증하지 않는다. `shop_atomic_ci`의 격리 테스트 테이블과 실제 `nineyin` 스키마가 같다는 가정도 하지 않는다. 결과를 보고 스키마가 다르더라도 **백업 감사와 명시적 승인 전에는 DB에 아무 변경도 하지 않는다.**
