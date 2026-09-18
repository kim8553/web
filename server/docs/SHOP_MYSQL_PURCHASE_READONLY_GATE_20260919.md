# NPC 구매 실게임 전 MySQL 구조 감사 (2026-09-19)

## 기준과 확인한 사실

- 사용자 제공 서버 원본 계보: `9yin-go-server1.rar`. 기존 접속 성공 기준: `JIUYIN_STAGE37_LEGACY_RICH_BAG_AB_TEST_20260914(1).zip` (해시 `88ab1992d4dfcedb7ecc03304dc7906da279eaf5d70565c71c0d7bdbf12fc0a5`).
- 실행 후보 EXE는 복원 GitHub `kim8553/web`의 `7d2e6ab2f39679c107f56117c75718b0b30e7b76` 소스로 빌드한 `stage37-purchase-test-7d2e6ab.exe`다. 서버는 Go 바이너리만으로 구매 데이터를 저장하지 않으며, `persistOrdinaryShopPurchase`는 하나의 MySQL DB에 연결된 bag/currency store가 아니면 구매를 거부한다.
- 구매 원자성 구현 `internal/shopbuyatomic/checked.go:saveChecked`는 `roles` 행을 잠그고 `role_currency.snapshot`·기존 `role_bag_items`를 검사한 후 가방 12개 컬럼과 재화 2개 컬럼에 INSERT한다. 사용자 DB의 스키마를 추측하면 데이터 손상이나 구매 실패로 이어질 수 있다.
- Go `migrations/runner.go`의 `//go:embed *.sql` 및 `Embedded()`에 따르면 버전 1 체크섬은 `migrations/0001_normalize_role_repository.sql` **전체 바이트** SHA-256 `b5cc5a2f62c05e4efd4d340f86747b589640d24e8fdd4b2193827a027f7df31d`다. `Runner.VerifyApplied`는 읽기 전용으로 ledger 버전 수와 체크섬을 정확히 일치시켜야 하며, 장부가 없거나 다르면 자동 변경이 비활성인 서버 실행을 거부한다.

## 이번 작업에서 추가한 검사

1. `01_SHOP_SCHEMA_METADATA_ONLY.sql`: 기존 GitHub `server/tools/check-shop-db-columns-readonly.sql`을 수정 없이 복사. DB명, 네 테이블 엔진, 필수 17개 컬럼, PK, 인덱스를 조회한다.
2. `02_SHOP_INSERT_CONSTRAINTS_ONLY.sql`: 구매가 NULL을 넣을 수 있는 `name/equip_type/art_pack/hardiness/max_hardiness`의 NULL 허용 여부, 구매 INSERT의 컬럼 목록 밖에 있는 NOT NULL·기본값 없는 컬럼을 조회한다. `BLOCKER_*`는 구조상 구매 실패 위험을 나타낸다. 추가 제약이나 트리거는 모두 확인하지 못한다.
3. `03_MIGRATION_LEDGER_IF_PRESENT_ONLY.sql`: **01에서 `schema_migrations` 존재를 확인한 경우에만** 버전 1 체크섬 비교 및 추가 버전을 조회한다. ledger가 없다면 01의 MISSING_TABLE이 증거이고 03을 실행하지 않는다. 마이그레이션 SQL 본문을 실행하는 파일이 아니다.
4. `validate_readonly_kit.py`와 4개 오프라인 회귀 테스트: 위 SQL에서 SELECT 이외 구문과 대표적인 위험한 SELECT 변형을 거부하도록 정적 검사한다. SQL 본문의 정적 검사일 뿐 DB 권한·서버 실행 결과를 대신하지 못한다.

## 안전한 판정 경계

- **실사용 DB를 연결해 감사를 실행하지 않았다**: 샌드박스에는 사용자의 Windows MySQL에 접근할 자격/연결이 없다. DB schema_migrations 여부, 컬럼의 실제 정의, 로그인 성공 패키지의 DB 상태 모두 **UNKNOWN**.
- 따라서 `TEST READY`, `DB PASS`, 구매·재접속 성공 등은 선언하지 않는다. 모든 SQL은 SELECT만 포함하며 조회할 DB는 사용자가 확인 후 선택해야 한다. DB 변경/자동 migration 플래그 활성화/덤프 수집은 금지한다.
- 01~03 결과를 확인하고 이상 항목을 해결할 계획과 별도 테스트 DB의 안전한 범위가 정해진 후에만 Windows/최신 클라이언트 로그인→상점 열기→구매→가방·재화→재접속 LIVE를 요청할 수 있다. 이미 접속되는 서버나 리소스는 덮어쓰지 않는다.

## 검증 결과

- `python validate_readonly_kit.py migrations/0001_normalize_role_repository.sql` → 예상 체크섬 MATCH, SELECT 구문 수 `5+2+2=9`, 정적 검사 PASS.
- `python -m unittest -v test_validate_readonly_kit.py` → **4 tests PASS**. MySQL 실제 서버/사용자 DB 감사 **NOT RUN**. 구매 LIVE/E2E **NOT RUN**.
