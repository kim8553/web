# Stage52 — 일반 NPC 상점 구매용 읽기 전용 DB 사전 점검 (2026-09-18)

## 기준·범위

Stage51 인계 커밋 `797152e506ed489690680c6543b4d16603f2b99f`에서 `kim8553/web`의 `stage37-assistant-handoff-20260917` 브랜치를 이어 작업했다. 이번 범위는 **일반 NPC 상점 구매 0x46의 DB 구조/초기 상태를 변경 없이 진단할 수 있는 도구**다. GM 지급, 교환 상점 0x4F는 보류 및 미수정, 일반 판매 0x47 미구현. 최신 클라이언트 바이너리, DLL, 리소스, 디코딩 Lua, 키 또는 사용자 DB 데이터는 GitHub에 게시하지 않았다.

## 코드에서 확인된 실제 선행 차단 조건

`server/migrations/0001_normalize_role_repository.sql`의 구형→M2 이행 절차는 `roles.account_key`를 전제로 한다. `server/migrations/runner.go`는 `schema_migrations` 테이블의 version/checksum을 확인하고 미적용 파일을 실행하므로, **현대식 `roles(role_id, account_id, ...)`가 이미 있으나 올바른 이행 기록이 없으면 시작 시 구형 열을 참조하는 마이그레이션이 다시 시도될 수 있다.** 이는 소스 경로에 근거한 조건부 위험이며 사용자 현재 DB를 직접 조사하여 확정한 결과가 아니다.

Stage50/51 구매 저장 코드의 SQL은 `role_bag_items`에서 `role_id,seq,slot,config_id,item_type,amount,view_id,name,equip_type,art_pack,hardiness,max_hardiness`를 읽고 쓴다. `role_currency`에는 각 역할의 `snapshot` JSON이 들어 있는 기존 행이 있어야 하며 두 테이블은 InnoDB여야 트랜잭션 원자성 테스트의 전제가 맞는다. 저장 함수는 누락된 재화 행을 자동으로 만들지 않는다.

## 추가한 점검 도구

`server/cmd/retail-schema-preflight/main.go`는 공식 MySQL 드라이버로 **환경변수 `NINEYIN_MYSQL_DSN`에 지정된 DB에 읽기 전용 질의만** 보낸다. 테이블 엔진·필수 컬럼·임베디드 SQL 이행 체크섬·전체 역할 대비 재화 행 누락 개수·재화 snapshot의 JSON 유효성을 검사한다. 결과에는 PASS/BLOCKED와 집계 수치만 출력하고 개별 역할, 암호, DSN 또는 재화 상세값을 출력하지 않는다. 누락된 구성·접속 실패·검사 오류는 성공으로 표시하지 않는다. 도구는 서버 실행·게임 구매·마이그레이션 수행·스키마 수정 명령이 아니다.

`server/cmd/retail-schema-preflight/main_test.go`는 누락 컬럼 및 체크섬 비교 단위 테스트를 제공한다. `integration_test.go`는 별도 `mysql_integration` 빌드 태그, 정확한 테스트 DB명 `jiuyin_stage52` 및 `NINEYIN_STAGE52_DISPOSABLE=YES_DROP_STAGE52_TABLES`가 있을 때만 **폐기용 DB**에 시험 테이블을 만들고 수정한다. 쓰기 동작은 이 격리 테스트 안에만 존재한다.

## 검증 이력과 결과

첫 [Stage52 Actions 실행 35246590476](https://github.com/kim8553/web/actions/runs/35246590476)은 실패했다. 정상 fixture를 점검기가 잘못 BLOCKED로 판단했으며, 이 실패를 성공으로 취급하지 않았다. `BLOB` 보관 JSON을 명시적으로 utf8mb4 문자로 변환해 검사하고 fixture의 JSON을 SQL 리터럴 문자열 대신 바인딩된 바이트로 넣도록 변경했다. 수정 커밋 [`0e2eeffe837c820f0c768b1448dcc029033685b8`](https://github.com/kim8553/web/commit/0e2eeffe837c820f0c768b1448dcc029033685b8)은 소스 `main.go`와 격리 테스트 `integration_test.go` 두 파일만 수정했고 구매 로직이나 DB 이행 SQL은 바꾸지 않았다.

수정 후 [Stage52 Actions 실행 35246833502](https://github.com/kim8553/web/actions/runs/35246833502)의 **schema-fixture 작업 SUCCESS**: 읽기 전용 생산 도구의 쓰기 SQL 금지 검사, Go 포맷 검사, 두 단위 테스트, 실제 폐기용 MySQL 8.0에서 완전한 fixture의 PASS 및 필수 가방 컬럼 누락·마이그레이션 행 누락·재화 행 미초기화·손상된 재화 JSON의 BLOCKED 사례, Linux·Windows amd64 점검 도구 및 실제 `protocol-probe` 서버의 컴파일을 검사했다. 이는 **테스트 fixture와 컴파일 결과이지 사용자 DB 및 게임 런타임 검증이 아니다**. CI fixture MySQL은 테스트 종료 시 제거된다.

## 확인되지 않은 것 및 다음 실제 작업

**사용자 Windows의 `nineyin` DB는 연결하거나 조회하지 않았다.** 해당 DB의 현재 스키마, 마이그레이션 ledger, `role_currency` 초기 행, `role_bag_items.max_hardiness` 보유 여부는 알 수 없다. 임의로 컬럼 추가·테이블 삭제·마이그레이션 기록 삽입·기존 데이터를 재작성하는 자동 수리 기능은 넣지 않았다. 점검 PASS여도 서버의 모든 의존 파일(`skill_new.ini` 포함), 상점 NPC 세션, 최신 클라이언트 가방 동기화, 재접속 지속성 또는 모든 아이템 구매 성공을 뜻하지 않는다. 다른 가방·재화 저장 처리의 동시 실행/중복 요청의 완전한 해결도 증명되지 않았다.

**다음 우선순위:** 사용자 DB의 백업 확인 후 이 읽기 전용 점검을 안전하게 실행할 수 있는 실제 Windows 서버 테스트 환경을 확보하고, 그 결과에 근거해 실제 스키마 차이만 좁게 수정한다. 이후 검증된 빌드로 NPC에서 아이템 1개 구매→가방/재화→종료/재접속까지 E2E 수행한다. 사용자에게 지금 기존 서버로 구매하거나 게임에 접속하라고 안내하지 않는다. 일반 판매 0x47은 별도 실제 가격 및 조건 검증 후 구현한다.
