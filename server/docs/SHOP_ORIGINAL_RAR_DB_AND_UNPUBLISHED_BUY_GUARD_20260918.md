# 원본 RAR DB 자료 감사 및 NPC 구매의 미게시 상품 좌표 거부 (2026-09-18)

## 확인한 입력과 범위

- 원본: 사용자 제공 `9yin-go-server1.rar`와 동일 해시인 작업 첨부 `9yin-go-server1_2.rar`, SHA-256 `ddc2f6bc078660a40eedf077d43befd6402ed59478ccde987d44171c1d7aefa5`. 이름 대신 파일 바이트를 기준으로 삼았다.
- 최신 첨부 BIN64 ZIP SHA-256 `167aeacadd22d551c252e8808a010a00080682de8d2b3fb5f21420ff4d75a306`: `fxgame.exe`=`c7f0366026452d525dc0c74b8499aac30fdb69189591420978e12c7094b5fde3`, `fxgamelogic.dll`=`16cf49b652c39b63ba0767780cc39f32ea1e50bdc683da9d5e7b2697cf95c3e8`, `fxnet2.dll`=`0443d9f401fdcc6a92a391869c898780ffac93399913dc9a82c7cac8cf6c3fde`, `fxcore.dll`=`ce0da0f52a80db5033e177a59be868c44b0baeca4a71226b0ccb2aca76cae724`. 정적 파일 식별만 수행했다. 현재 DLL에서 보이는 범용 `OnSellClick` 등의 문자열만으로 일반 NPC 판매의 송신 패킷이나 정산 규칙은 확정하지 않는다.
- Drive의 지정 `res` 폴더를 다시 직접 목록 조회하여 `share.package`, `lua.package`, `lua64.package`, `ini.package`의 존재와 파일 ID를 확인했다. 기존 `SHOP_SERVER_LOOSE_VS_CLIENT_PCK_THREE_SOURCE_AUDIT_20260918.md`의 PCK 두 후보 스트림 어느 것이 클라이언트 활성 `shop.ini`인지는 여전히 미확정이다. 원본 리소스나 패키지에 쓰지 않았다.

## 원본 RAR의 DB 구성 — 실제 압축 목차 및 파일 읽기 결과

- RAR의 비디렉터리 파일 9,785개를 검사했다. `.sql` 1개 (`migrations/0001_normalize_role_repository.sql`, 6,725 bytes)와 `migrations/runner.go`, `internal/role/mysql_repository.go` 등 코드가 있다. `.sql.gz`, `.ibd`, `.frm`, `.myd`, `.myi`, `.sqlite`, `.sqlite3`, `.db`, `.dump`, `.bak`, `.mdb`, `.db3` 확장자의 파일은 목차에서 발견되지 않았다.
- 원본 RAR의 `migrations/0001_normalize_role_repository.sql`을 실제 압축 해제하여 현행 GitHub 소스의 동일 경로와 비교했다. 둘 다 SHA-256 `b5cc5a2f62c05e4efd4d340f86747b589640d24e8fdd4b2193827a027f7df31d`로 **바이트 동일**하다. `internal/role/mysql_repository.go`와 `cmd/protocol-probe/shop_catalog.go`는 원본과 현재 파일 해시가 달라, 원본을 현재 코드 위에 덮어쓰지 않는다.
- 이는 RAR에 DB **접속·마이그레이션 코드가 있다는 사실**이지 RAR에 사용자 캐릭터·아이템의 실제 DB 덤프가 들어 있다거나 사용자 PC `nineyin` 스키마와 호환된다는 증거가 아니다. 원본 RAR, 사용자 DB, 구형 `D:\\9yin_server` DB를 서로 동일시하지 않는다.

## 수정 전 재현 및 최소 수정

- `cmd/protocol-probe/scene_lifecycle.go`의 `openShopLocked`는 모드 0·1·2의 상품을 현재 클라이언트에 게시하기 전에 `currentShopViewObjectIndex(item)`이 유효한지 검사한다. 이 함수는 페이지당 500칸 및 16비트 object index 제약을 구현한다.
- 반면 `zz_recovered_overlay.go`의 `handleShopBuyCustom`이 사용하는 `currentShopListing`은 같은 상품의 게시 가능 여부를 검사하지 않았다. 실제 함수만 분리한 수정 전 회귀에서 `page=0,pos=501` (position=500) 및 `page=131,pos=36` (object index 65536)이 화면에는 게시될 수 없는데 구매 조회에서 선택되어 FAIL을 재현했다. 이는 잘못된 좌표의 상품이 DB 구매 경로에 도달할 수 있다는 **서버 소스 결함**이며, 현재 클라이언트가 그런 패킷을 보낸다는 주장은 아니다.
- `latest_client_shop_view_contract.go`의 `currentShopListing`에 **기존 `currentShopViewObjectIndex` 검사만 재사용**하고 미게시 상품을 `nil`로 거부한다. 새 패킷·가격·판매 로직·데이터베이스 스키마·게임 리소스는 건드리지 않는다. `latest_client_shop_view_contract_test.go`에는 최종 유효 좌표와 두 가지 범위 초과 좌표 회귀를 추가했다.

## 검증 범위와 미완료

- 수정 전 격리된 **실제 production 함수** 회귀 FAIL 2건 → 수정 후 같은 회귀 PASS. 수정된 실제 `cmd/protocol-probe` 패키지의 `TestCurrentShopListingRejectsUnpublishedCoordinates` PASS (원본 RAR 스킬 리소스를 sandbox에만 풀어 초기화).
- sandbox의 격리 vendor 복사본과 일시적인 Go 모듈 설정 조정으로 `go test ./internal/... ./migrations/...` PASS, `go vet ./...` PASS, Windows amd64 `go build ./cmd/protocol-probe` PASS. 일시적인 `go.mod` 편집은 복원했고 vendor 및 리소스는 Git에 포함하지 않는다.
- `go test ./...` 전체는 **FAIL**: 원본 RAR의 creator manifest를 읽는 기존 scene registry 테스트 세 개가 `declares no XML creator`로 실패했다. 이 테스트들의 해당 resource/version 대응은 별도 조사 필요; 전체 테스트 PASS라고 보고하지 않는다. 사용자 PC의 MySQL, 실제 게임 구매/판매·가방 UI·재접속 LIVE/E2E는 NOT RUN.
- 일반 NPC **판매**는 별도 처리 경로와 최신 클라이언트 요청 필드·정산 근거가 아직 확인되지 않았다. 기존 `SellPrice` 속성이나 DLL 문자열을 판매 요청으로 단정하여 핸들러를 구현하지 않는다. GM 웹 지급과 mode3 Exchange는 기존 보류 상태를 유지한다.
