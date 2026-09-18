# NPC 상점 개방 상태와 구매 ShopID의 서버측 일치 검증 (2026-09-18)

## 출처·시작점

- 서버 출발점: 사용자 제공 `9yin-go-server1.rar`; 편집 대상: `kim8553/web`, `stage37-restored-20260918`, 시작 GitHub HEAD `d90445542cf8dcc0ff70294da077387602e5f0ea`. 이전 구매 저장 및 클라이언트 View 검증은 보존한다.
- 최신 클라이언트 `lua.package` 및 `lua64.package`의 앞선 세 쌍 후보 LuaQ를 다시 읽었다. 12바이트 단문 마스크를 *잠정적으로* 루트 소스 문자열 앞부분에 적용하면 여섯 개 모두 `@G:\Version_`로 보인다(32비트 원본 루트 소스 문자열 크기 60/64/66, 64비트 짝은 62/66/68). 이 사실만으로 PCK 색인 엔트리 ↔ 압축 스트림 원본 Lua 파일명 또는 판매 이벤트 호출 관계가 **확정되지 않는다**. 판매 패킷·가격·필드는 계속 미확정이며 추측한 판매 핸들러는 만들지 않았다.

## 별도로 확인한 기존 Go 구매 코드 결함

- `scene_lifecycle.go`에서 NPC 상점 서비스 `markShop=0x1001`를 선택해야 `openShopLocked(service.value)`가 실행되어 View 61 상품이 발행된다.
- 반면 수정 전 `zz_recovered_overlay.go:handleShopBuyCustom`은 클라이언트가 보낸 ShopID로 바로 `shopCatalogItems(defaultShopINIPath, shopID)`를 조회했다. 동일 접속에서 NPC 서비스로 해당 상점을 열었는지 검사하지 않았다. 두 `main.go` 메시지 분기에서 해당 함수가 호출된다. 이는 소스 수준의 서버측 허용 상태 누락이다. 실제 공격 발생이나 최신 클라이언트가 비정상 ShopID를 보낸다는 주장은 아니다.

## 최소 변경

- `sceneLifecycle.activeShopID`는 해당 접속에서 `openShopLocked`가 모든 상품 프레임을 성공적으로 발행한 **후에만** 기록한다. 실패한/부분적인 열기는 이전 허용 상태를 비운다. 다른 장면 `begin` 및 연결 `close`도 해당 상태를 비운다.
- `ordinaryShopBuyAuthorized`는 nonempty ShopID 및 `!closed`, 서버가 열어 준 동일 ShopID만 허용한다. `handleShopBuyCustom`이 상품 조회/아이템 구성/재화 변경/DB 저장 **전에** 호출한다. `main.go`의 두 기존 구매 분기 모두 동일 연결의 `world`를 전달한다.
- 새로운 프로토콜 필드, 가격 계산, 판매 정산, DB 스키마/마이그레이션, GM 웹 지급, mode3 Exchange, 로그인/맵 초기화, 원본 RAR 및 패키지 자료는 변경하지 않는다.

## 재현과 검증

- 수정 전 새 테스트 두 개가 미구현 서버측 검사/함수 연결로 컴파일 **FAIL**; 수정 후 `TestOrdinaryShopBuyRequiresOpenedNPCShop`, `TestOrdinaryShopBuyCannotUseClientShopIDBeforeOpen` **PASS**. 첫 테스트는 샌드박스 임시 `shop.ini`로 실제 NPC 서비스 열기 함수, 다른 ShopID 거부, 실패한 재열기, 장면 전환과 종료를 검사한다. 둘째는 NPC 미개방 구매를 실제 핸들러에 전달하고 응답 프레임이 없는지 검사한다.
- 샌드박스 격리 vendor와 일시적 `go.mod` 수정(테스트 후 원상복구)으로 `go test ./cmd/protocol-probe -run 'Test.*(Shop|Buy|Currency|Bag|NPC)' -count=1` PASS, `go test ./internal/... ./migrations/...` PASS, `go vet ./...` PASS, Windows amd64 `go build ./cmd/protocol-probe` PASS.
- `go test ./...` 전체는 **FAIL**: 기존 장면 creator 리소스 관련 테스트 세 개(`TestCurrentCitySceneRegistryMaterializesModernCreatorCatalog`, `TestShenJiHuiCatalogSkipsMissingOptionalCreators`, `TestYanYuZhuangCatalogSkipsEmptyZeroAmountCreatorPlaceholder`). 전체 PASS라고 보고하지 않는다.
- 사용자 PC의 MySQL, 실제 게임 구매·판매/Bag View·재접속 LIVE/E2E: **NOT RUN**. 상점 닫기 이벤트/거리/시간 만료는 근거가 없어 이번 변경에서 추가하지 않았으므로, 이 검증은 **해당 접속의 마지막 성공 개방 ShopID + 장면/종료 초기화**로 한정된다.

## 다음 미완료

- 현재 BIN64의 검증된 PCK 색인/로더 근거로 후보 스트림의 완전 파일 경로와 NPC 판매 버튼 → 전송 호출 관계를 추적한다. 불확실한 C2S를 구현하지 않는다. 기존 전체 테스트 실패 3건은 리소스 버전 출처와 함께 별도 감사한다.
