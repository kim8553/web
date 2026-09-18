구음진경 NPC 구매: MySQL 읽기 전용 호환성 감사 키트 (2026-09-19)

기준: kim8553/web stage37-restored-20260918, Go 구매 코드 기반 7d2e6ab2f39679c107f56117c75718b0b30e7b76.
로그인 확인 기준: JIUYIN_STAGE37_LEGACY_RICH_BAG_AB_TEST_20260914(1).zip.
이 키트는 서버 설치·구매 실행 프로그램이 아닙니다. SQL 3개 모두 SELECT만 수행하며 서버/DB/클라이언트를 수정하지 않습니다.

현재 검증된 사실: 게임 구매는 같은 MySQL의 가방과 재화 트랜잭션이 필요합니다. 현재 사용자 PC의 DB 내용과 구조는 직접 확인하지 못했습니다.
기존 01은 테이블 엔진, 필수 17개 컬럼, PK와 인덱스만 표시합니다. 02는 별도 구매 INSERT에서 빠진 필수 컬럼 및 NULL 금지 컬럼을 확인합니다. 03은 빌드 기준 마이그레이션 버전 1의 SHA-256과 기록된 장부를 대조합니다.

사용자가 준비가 되었을 때만 본인이 기존 MySQL monitor에서 수행하는 선택적 절차:
1) 기존 서버와 DB 백업 보존. 본인이 실제 이용하는 DB가 nineyin인지 먼저 확인하고 MySQL 클라이언트에서 해당 DB를 선택합니다. 비밀번호나 DSN을 채팅에 보내지 마세요.
2) SELECT DATABASE(); 결과가 의도한 DB인지 확인합니다. DB 이름이 다르거나 NULL이면 중지합니다.
3) SOURCE C:/절대경로/01_SHOP_SCHEMA_METADATA_ONLY.sql;
4) 01 결과의 schema_migrations가 실제 BASE TABLE일 때만 SOURCE C:/절대경로/03_MIGRATION_LEDGER_IF_PRESENT_ONLY.sql;
5) SOURCE C:/절대경로/02_SHOP_INSERT_CONSTRAINTS_ONLY.sql;
   (SOURCE는 MySQL 클라이언트 명령으로, SQL 파일을 읽습니다. 경로는 본인의 압축 해제 위치에 맞게 바꾸세요.)
6) 결과 중 BLOCKER_*, MISSING_*, CHECKSUM_MISMATCH, EXTRA_VERSION*, REVIEW_*가 있으면 즉시 구매 실게임 검증을 보류합니다. 모두 없더라도 DB 행의 JSON 유효성, FK/트리거/제약, 현재 클라이언트, 성공 ZIP과의 동등성, 실제 구매 가능 여부는 입증되지 않습니다.

절대 실행하지 말 것: migrations/0001_normalize_role_repository.sql 원문, mysql 초기화/복원 명령, NINEYIN_ALLOW_SCHEMA_MIGRATIONS=YES, 구매 EXE를 기존 정상 서버 폴더에 덮어쓰기.
메타데이터 결과만 필요합니다. 로그인 암호, DSN, 계정/캐릭터 행, role_currency.snapshot 값 또는 전체 DB 덤프는 업로드하지 마세요.

검증 한계: MySQL CLI/실제 사용자 DB는 이 작업 환경에 없어 세 SQL의 실 DB 실행은 NOT RUN입니다. 01~03은 정적 검사만 수행했습니다. 사용자 승인 없이는 DB 수정이나 구매 테스트 실행을 진행하지 않습니다.
