# NPC 상점 구매: DB의 NULL·빈 문자열·0 메타데이터 호환성 (2026-09-18)

## 소스 근거

- 기준 브랜치 `kim8553/web` / `stage37-restored-20260918`, 수정 전 HEAD `67b75098cb07baf2a307cc3a7a8d2472e5c388e8`.
- `server/cmd/protocol-probe/zz_recovered_overlay.go`의 `mysqlBagStore.Load`는 `name/equip_type`을 `sql.NullString.String`으로, `art_pack/hardiness/max_hardiness`를 `sql.NullInt64.Int64`의 `int32` 변환으로 로드한다. NULL과 각각 빈 문자열/0의 구분이 actor 메모리에서 사라진다.
- 같은 파일의 `nullableString`/`nullableInt32`와 `shop_purchase_atomic.go`의 `ordinaryShopBagRows`는 빈 문자열과 0을 INSERT 인자 `nil`로 변환한다. 기존 `internal/shopbuyatomic/checked.go`의 잠금 후 DB 행 비교는 `sql.NullString.Valid` 또는 `sql.NullInt64.Valid`만 검사하여 DB의 명시적 `''`/`0`을 actor의 `nil`과 다르게 보았다.

## 수정과 검증 경계

- `checkLockedBag`에서 이미 유실되는 표현 차이인 SQL NULL/빈 문자열 및 NULL/0만 동등하게 처리한다. 범위를 벗어나는 int32, 실제로 변경된 비어 있지 않은 이름·장비 유형·0이 아닌 수치, 수량·슬롯·ID 등은 이전대로 충돌 또는 오류로 거부한다. 구매 및 체크된 가방 이동의 공통 비교 함수만 수정했다. 패킷/가격/상품/DB 스키마를 변경하지 않는다.
- 새 회귀 `TestCheckedBagAcceptsLegacyExplicitEmptyMetadata`는 수정 전 구매·가방 이동 두 경우 모두 `ErrBagChanged`로 실패했고 수정 후 통과했다. `TestCheckedBagRejectsLegacyNonzeroMetadataChange` 및 기존 메타데이터 충돌 회귀는 실제 변경을 거부하는지 확인한다.
- 일회용 MySQL 통합 테스트 `TestSaveCheckedRealMySQLReconnectAndRollback`에는 DB가 명시적 빈 문자열/0을 가진 아이템에서 시작해 구매하고, 별도 연결에서 아이템·재화·NULL 정규화 결과를 재조회하는 사례를 추가했다. 이 테스트는 `JIUYIN_TEST_SHOP_MYSQL_CI=1` 및 DB 이름 `shop_atomic_ci`에서만 실제 DB를 변경하도록 제한한다.
- 격리 샌드박스에서 Go 1.23.2와 정확한 의존성 vendor를 사용한 `go test ./internal/shopbuyatomic`, `go vet ./...`, Windows amd64 빌드는 PASS였다. 일반 `go test ./...`는 `cmd/protocol-probe` 초기화에 필요한 사유 리소스 `resources/modern/share/skill/skill_new.ini` 부재로 해당 패키지만 FAIL; 나머지 패키지는 PASS. 전체 테스트 PASS로 보고하지 않는다.
- **격리 DB CI 검증 완료:** [Actions 35350547452](https://github.com/kim8553/web/actions/runs/35350547452)가 정확한 게시 코드 커밋 `3aa2e859db281114b727b9ab4bfd53678fb4d010` 및 Git tree `a6e312882043cc2b420da9bbf221597f8f3fb3b1`을 검증한 뒤 전용 MySQL 8.4.11 `shop_atomic_ci`에서 `TestSaveCheckedRealMySQLReconnectAndRollback`, 새 구매·가방 이동 성공 회귀, 메타데이터 변경 거부 회귀 및 관련 패키지 테스트를 실행하여 SUCCESS를 기록했다. 사용자 PC DB나 게임 내 실동작 테스트가 아니다.
- 사용자 PC의 `nineyin` DB는 접근/수정하지 않았다. 이 DB에 해당 구형 행이 실제 존재하는지, 현재 GitHub 빌드의 실제 게임 구매·판매 및 재접속 E2E 성공 여부는 **미검증**이다. 클라이언트 활성 PCK 스트림·NPC 판매 프로토콜은 여전히 미확정이다.
