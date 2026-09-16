# Stage32 — 9yin-go-server1 RAR NPC 상점 참조·선택 진단

## 권위와 보존 원칙
- 사용자 제공 기준: `9yin-go-server1.rar` (SHA-256 `ddc2f6bc078660a40eedf077d43befd6402ed59478ccde987d44171c1d7aefa5`). 이전의 다른 ZIP으로 되돌리지 않는다.
- 해당 RAR의 `resources/modern/share/trade/shop.ini` SHA-256: `ceb7a3a7e17cbc118ce5a7d3d979dea6120c50cdf6a5823697c05a5b1b9cf5dd`.
- Stage31까지의 좌표·Lua, 상점 프레임 사전검사, 구매 fail-closed를 유지한다. GM 웹페이지 아이템 지급은 보류한다.

## 오프라인 데이터 검증 (실제 NPC 등장 또는 클라이언트 화면 검증 아님)
- RAR의 `npc/npcconfig/**/*.txt` 템플릿 표에서 `ID`·`ShopID` 헤더가 있는 행을 읽고, 0/공백이 아닌 `ShopID`를 `shop.ini`의 **정확히 일치하는** 구역 이름과 비교했다.
- `shop.ini` 구역: 1,447개. ShopID 지정 템플릿 행: 1,328개. 고유 참조 ID: 767개.
- 정확히 일치하는 구역이 없는 참조 ID: **7개, 템플릿 행 41개**. 그중 `Shop_GB_Yishiting`은 35개 행이다. 나머지 ID는 `shop_xlsbz_1`, `testskill_shop_021`, `testskill_shop_022`, `testskill_shop_023`, `testskill_shop_024`, `testskill_shop_025`이다.
- `shop.ini`에는 `Shop_GB_Yishiting_1`부터 `_5` 구역이 존재하지만 **상점 기본 ID가 어느 접미사를 가리키는지 확인한 근거가 없다. 접미사를 임의 선택해 고치지 않는다.**
- 위 수치는 템플릿 레코드만의 비교 결과이다. NPC creator 오버라이드, 실제 resolved NPC, 맵 배치 수, 플레이어가 선택한 NPC 또는 라이브 오류 건수로 해석하지 않는다.

## 기존 소스에서 관찰한 경로와 Stage32 변경
1. `modernNPCServices`는 `creator.ShopID`를 먼저, 없으면 `template.ShopID`를 사용해 상점 서비스를 구성한다.
2. `playableNPCServices`는 상점 구역을 파싱하지 못하면 해당 상점 메뉴를 제외한다. 기존 코드는 제외 사유를 NPC 선택 시 출력하지 않아 재현 분석을 어렵게 했다.
3. Stage32는 **NPC object request에서**, 메뉴 필터보다 먼저, *기존 resolved service의 정확한 ID*로 `shop.ini`를 읽어 NPC object/scene/config, ShopID, source, 데이터 오류, 전체·일반·교환 상품 행 수를 로그에 기록한다. 읽기 전용이며 상점 ID 자동 수정, 상품/프레임/구매/가방/화폐/DB 변경은 없다.
4. 이 진단 로그는 `default_shop_render=unverified`를 명시한다. 데이터가 읽힌다는 사실로 클라이언트 렌더링이나 구매 성공을 주장하지 않는다.

## 검증 경계
- 로컬 독립 테스트 5건(누락 구역·접미사 자동매핑 방지·일반/교환 분리·빈/손상 구역·정확한 RAR `shop.ini`) 및 race 검사는 통과했다. CI에는 원본 RAR 파일이 없어 RAR 전용 테스트를 건너뛰도록 명시되어 있다.
- GitHub Stage32 검사: `tools/stage37_current_recovery_stage32.sh`; Actions workflow `.github/workflows/stage37-stage32-npc-shop-selection-diagnostic.yml`.
- Windows 빌드/Go vet/전체 패키지 **테스트 컴파일** 및 race **컴파일**은 해당 Actions 실행 결과로만 확정한다. 전체 패키지 런타임 테스트와 게임 내 LIVE/E2E는 별개이다.
- 실제 다음 증거: 사용자 테스트에서 선택한 NPC의 `NPC shop menu diagnostic ...`, `shop service selected ...`, `shop display catalog ...`, `shop display frames ...`를 같은 NPC/ShopID로 연결해 비교. 구매 요청·DB 저장 검증은 현재 미실시.
