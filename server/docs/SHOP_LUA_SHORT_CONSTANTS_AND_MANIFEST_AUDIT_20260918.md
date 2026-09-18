# 일반 NPC 판매 조사: 최신 BIN64 패키지 등록과 LuaQ 단문 상수 (2026-09-18)

## 검증 입력

- GitHub 복원 브랜치 `stage37-restored-20260918`, 작업 시작 HEAD `a8d11f544e76d1e3cf08f3c55bb9ae5783c38b0a`. 원본은 사용자 지정 `9yin-go-server1.rar`; 다른 서버로 대체하지 않았다.
- 사용자 제공 최신 `bin64(2)(1)(1)_2.zip` SHA-256 `167aeacadd22d551c252e8808a010a00080682de8d2b3fb5f21420ff4d75a306`에서 파일을 실제 열어 확인했다. `packages.ini` SHA-256 `667eb4f99e00359e36afb20f7e5b7ab13f73ea65133cf7ab3357c7e2f39d4687`에는 `[lua] File=res\lua.package Preload=0`과 `[lua64] File=res\lua64.package Preload=0`가 **둘 다 등록**되어 있다. `fxres_packages.ini` SHA-256 `cbf0039bfb255faeb0bfc497f0b25cc4ab4306992d2dfa75ceff7e1cf688ce26`에는 Lua 항목이 없다. 등록/Preload 값만으로 런타임의 패키지 선택이나 로드 우선순위를 확정하지 않는다.
- 연결된 Google Drive `res`의 `lua.package` SHA-256 `283c8c245a3fb86af2a7e8c21c53b3de26c30432341dd590c37554adfce04e97`, `lua64.package` SHA-256 `700cb8c888374ed3ceb0f03b940e3f148620519f82352499f7957991184677d2`를 로컬 격리 사본에서 직접 검사했다. 원본 파일·Go 서버·사용자 PC DB는 수정하지 않았다.

## 기존 3개 구조 후보에서 추가로 확인한 사실

기존 [`SHOP_CLIENT_LUA_SALE_CANDIDATES_20260918.md`](SHOP_CLIENT_LUA_SALE_CANDIDATES_20260918.md)의 **압축 스트림 시작 오프셋**만 지정했다. 새 `tools/audit_client_lua_shop_short_constants.py`는 SHA-256 불일치 시 즉시 거부하고, 바이트 길이 12 이하인 LuaQ 문자열 상수 중 `abcd464fghfd` 실측 마스크로 해석했을 때 ASCII와 NUL 종료 조건을 충족하는 **사전 지정된 식별자**만 센다. 이 마스크는 관찰된 짧은 `ShopID`, `SellPrice0`, `SellPrice1`, `index`, `grid` 등의 바이트와 32/64비트 짝에서 확인한 범위에 한정된다. 더 긴 문자열, 원본 경로, 명령어·전송 함수를 복호화했다고 주장하지 않는다.

| 패키지 | zlib 시작 오프셋 | 관찰된 완전한 짧은 문자열 상수 (출현 횟수) |
|---|---:|---|
| lua | 974916 | `ShopID` 1, `SellPrice0` 1, `SellPrice1` 2, `do_shop` 2 |
| lua | 1017054 | `ShopID` 1, `SellPrice0` 1, `SellPrice1` 2, `do_shop` 1 |
| lua | 16702203 | `shopid` 3, `ShopID` 2, `ShopType` 1, `SellPrice0` 2, `SellPrice1` 3, `SellPrice2` 2, `SellLimit` 2, `SellCount` 2, `BuyLimit` 1, `bSell` 1, `IsShop` 1, `@ui_shop` 1, `item_obj` 2 |
| lua64 | 1482633 | lua 974916과 위 식별자·횟수 일치 |
| lua64 | 10430324 | lua 1017054와 위 식별자·횟수 일치 |
| lua64 | 18061242 | lua 16702203과 위 식별자·횟수 일치 |

세 번째 짝에는 일반 상점의 페이지 및 판매 관련 **표시·상태 이름을 시사하는 단문 상수**가 더 많이 있지만, 이는 실제 파일명이 `form_shop.lua`이거나 판매 이벤트가 어느 CustomSend/C2S ID를 전송한다는 증거가 **아니다**. `SellPrice*`는 상점 View 속성일 수 있고 실제 플레이어 아이템 매각의 정산식이라고 해석할 수 없다. 패키지 색인 경로, 해당 이벤트의 명령어/호출 관계 및 사용자 클라이언트 활성 버전은 계속 미확정이다.

## 검증과 후속

- 합성 fixture에 한정한 새 단문 검출 테스트 6개와 기존 후보 테스트 6개: **12 PASS**. 실제 두 패키지의 명시된 오프셋 3개씩 완전 구조 확인 및 allowlist 집계가 상호 일치. 기존 클라이언트/서버 바이너리는 공개 GitHub에 업로드하지 않았다.
- 이번 작업은 읽기 전용 분석 도구·테스트·문서·CI만 추가/수정한다. 일반 NPC **판매 핸들러/패킷/가격/재화/DB: NOT IMPLEMENTED BY THIS CHANGE**. 실제 게임 구매·판매/재접속 **LIVE/E2E: NOT RUN**. GM 웹 지급 및 mode3 Exchange 보류 유지.
- 다음 확인 지점: 현재 BIN64의 파일 색인/로더에 근거한 실제 **후보 스트림↔원본 경로** 매핑과 상점 버튼 이벤트↔전송 함수↔요청 필드 관계. 확보 불가하면 추측하지 않고 다른 확인 가능한 기존 구매/가방 결함을 별도 재현한 다음 수정한다.
