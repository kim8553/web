# NPC 상점 구매: 생략된 0원 재화 필드와 DB 저장 검증 (2026-09-18)

## 출처 및 범위

- 수정 기준 GitHub `kim8553/web`, 브랜치 `stage37-restored-20260918`, 수정 전 HEAD `a8a26edfa22422a6a6d3e5d7874453cacfcebc42`. 정확한 커밋을 Actions `git archive`로 추출하여 SHA-256 검증한 작업 트리에서 분석·수정했다.
- 첨부 원본 `9yin-go-server1.rar` SHA-256: `ddc2f6bc078660a40eedf077d43befd6402ed59478ccde987d44171c1d7aefa5` (원본 보존).
- 첨부 최신 BIN64 ZIP SHA-256: `167aeacadd22d551c252e8808a010a00080682de8d2b3fb5f21420ff4d75a306`. 내부 `FxGameLogic.dll` SHA-256 `16cf49b652c39b63ba0767780cc39f32ea1e50bdc683da9d5e7b2697cf95c3e8`은 이전 `CURRENT_AUTHORITY.md`에 기록된 `3bfa3832c04291d9d7a44f6bc13cceb609aad55ba90e2ad937cc77221089ef96`과 다르다. 둘을 동일 DLL로 취급하지 않는다.
- Drive 별도 서버 경로 `9yin_server/resources/modern/share/trade/shop.ini` (ID `1qDCtz31_3vcYmshpF39yw82IqyRJJJTr`): 내려받은 원본 2,176,439 bytes, SHA-256 `ceb7a3a7e17cbc118ce5a7d3d979dea6120c50cdf6a5823697c05a5b1b9cf5dd`. PCK의 클라이언트 활성 스트림은 **미확정**이며 선택·교체하지 않았다.

## 기존 구매/SQL 경로에서 확인한 결함

`cmd/protocol-probe/zz_recovered_overlay.go`의 `currencySnapshot`은 `silver`, `gold`, `silver_card`, `silver_ticket` 네 필드가 있는 Go 구조체다. `mysqlCurrencyStore.Load`의 JSON 역직렬화는 생략된 정수 필드를 0으로 읽는다. `cmd/protocol-probe/shop_purchase_atomic.go`는 이 구조체를 네 필드 모두 포함한 JSON으로 직렬화하여 `internal/shopbuyatomic/checked.go`의 `SaveCheckedBag`에 넘긴다. 기존 저장 함수는 DB `role_currency.snapshot`과 구매 시작 스냅샷을 `map[string]int64`로 비교하면서 `reflect.DeepEqual`을 사용했다. 따라서 DB에 `{"silver":100}`이 들어 있고 실제 값이 변하지 않았어도 구매 시작 JSON `{"silver":100,"gold":0,"silver_card":0,"silver_ticket":0}`은 `ErrWalletChanged`로 거부됐다. **이 표현 호환성 결함은 실제 코드·격리 SQL mock에서 확인했으나, 사용자 PC DB에 생략형 행이 존재하는지는 알 수 없다.**

`sameWalletValues`는 저장된 키의 값이 모두 동일하며, 구매 시작 스냅샷에서만 존재하는 키의 값이 반드시 0일 때만 동등하다고 판단한다. 저장된 알 수 없는 키, 달라진 값, 누락된 비영(非零) 값은 계속 충돌로 거부하므로 임의 재화를 덮어쓰지 않는다. 역할 행 잠금, 가방 스냅샷 비교, 가방·재화 단일 트랜잭션, 실패 시 구매 성공 미전송 경로는 변경하지 않았다. 기존 NPC 판매 패킷이나 가격 규칙은 추가하지 않았다.

## 실행 검증 및 남은 한계

- 수정 **전** 새 회귀 `TestSaveCheckedBagAcceptsOmittedZeroWalletFields`: `ErrWalletChanged`로 실패함을 기록했다. 수정 **후** `internal/shopbuyatomic` 테스트, 비프로토콜 20개 패키지, 프로토콜 패키지 컴파일, `go vet ./...`, Windows amd64 PE32+ 빌드가 격리 작업 트리에서 성공했다. Go 모듈 의존성은 기존 CI 방식의 별도 `ci.real.mod` + 검증된 임시 vendor로 공급했으며 저장소의 `go.mod`를 변경하지 않았다.
- `checked_mysql_integration_test.go`의 기존 일회용 `shop_atomic_ci` 테스트에 생략형 재화로 구매한 뒤 별도 MySQL 연결에서 가방·재화 재조회하는 사례를 추가했다. **이 일회용 MySQL 테스트는 CI 결과를 확인하기 전까지 PASS로 간주하지 않는다.**
- Drive `nineyin` 폴더에 `role_bag_items.ibd`, `role_currency.ibd`가 있는 것은 확인했으나 `.ibd` 파일명만으로 열 형식, 인덱스, 트랜잭션 엔진, **사용자 PC 현재 스키마**는 알 수 없다. `tools/check-shop-db-columns-readonly.sql`은 컬럼 존재만 검사하므로 모든 DB 호환성을 증명하지 않는다. 사용자 `nineyin` DB 읽기/쓰기·초기화·자동 마이그레이션은 하지 않았다.
- 실제 클라이언트 접속·상점 구매·재접속 유지 LIVE/E2E, 판매 요청 형식, 클라이언트 PCK 활성 스트림 및 사용자 PC 실제 DB는 **미검증**이다. 이전 로그인·맵 진입 정상 ZIP도 수정하지 않았다.
