# Stage49 — 원본 share.package 상점 관련 익명 스트림 내용 대조 (2026-09-17 KST)

## 기준·보존·안전

- 시작 `stage37-current-recovery-20260916` HEAD `a49bfb0c1f3e8f978d14cef1e5305387ae6eb92c`. 기존 `9yin-go-server1.rar` 기반 `server/`를 유지하며 구형 `9yin-go-server.zip` 또는 JYZJ로 복귀하지 않음.
- 기존 BIN64의 `packages.ini`와 `fxres_packages.ini` 모두 **`res\\share.package`를 패키지 경로로 명시**한다(Stage46 보고서). 따라서 Stage48에서 `ini.package`/`lua.package`만 비교한 결과는 `share.package` 안의 상점 리소스를 배제하지 못한다.
- 연결된 *비공개* Drive 원본 `res/share.package`를 별도 로컬 사본으로 읽음: **40,680,972 bytes**, SHA-256 `200497852ba3a29279e51f01e2913b5f7260480f2a740a32ebd1680b869844b6`. 기본 PCK0 헤더 길이 15, 선언 엔트리 **9,726**, 인덱스 시작 19, 끝 790,911 바이트. 인덱스 복호화는 하지 않았음.
- 소스 도구 [`tools/stage49_share_shop_evidence.py`](../tools/stage49_share_shop_evidence.py) Git blob SHA `dbcda3d88ac5e0340bf6d57994c549309c1574da`는 실제 원본과 별도 Drive 원본 5개에 실행해 성공한 로컬 도구의 `git hash-object` 결과와 일치. 원본 패키지·EXE·비공개 게임 데이터는 **공개 GitHub 또는 CI에 업로드하지 않음**. EXE/DLL을 실행하거나 패치하지 않음.

## share.package 압축 데이터의 실제 검증 범위

- 원본 인덱스 끝 오프셋 **790,911** 이후 데이터 구간의 일반 zlib 스트림을 Stage48의 제한된 해제기로 재검사: **15,382개 후보 위치 테스트, 완전한 Adler-32 검증 익명 스트림 9,817개, 해제 후 총 204,283,707바이트**. 선언 엔트리 수(9,726)와 스트림 수(9,817)는 다르며, **파일 9,817개를 언팩했다는 뜻이 아님**.
- 인덱스 레코드 복호화 **0건**, 원본 내부 파일명·경로 검증 **0건**. 아래 명칭은 **별도로 확보한 Drive 파일과의 콘텐츠 대조 레이블**이며 실제 패키지 인덱스의 이름을 읽은 것이 아니다.

## 상점 관련 익명 스트림과 독립된 Drive 원본 대조

| 별도 Drive 파일/내용 분류 | 원본 압축 스트림 시작 | 검증 결과 |
| --- | ---: | --- |
| `shopsingle.ini` | 13,782,597 | Drive 원본 전체와 크기 3,693 bytes 및 SHA-256 완전 일치 |
| `clonestore.ini` | 3,812,575 | Drive 원본 전체와 크기 1,066,765 bytes 및 SHA-256 완전 일치 |
| `repairaddbag.ini` | 24,207,113 | Drive 원본 전체와 크기 577 bytes 및 SHA-256 완전 일치 |
| `shopfunc.ini` 유사본 | 25,407,276 | Drive 원본 전체 116,512 bytes가 스트림 116,751 bytes의 **완전한 앞부분**; 뒤에 239 bytes 추가. 전체 파일 SHA는 서로 다름 |
| `shop.ini` 유사본 A | 33,744,821 | 2,177,532 bytes, Drive 원본과 길이 24 이상 고유 줄 **43,425개 정확히 일치** / 원본 고유 줄 43,626개 |
| `shop.ini` 유사본 B | 36,877,029 | 2,177,326 bytes, 같은 기준 **43,420개 정확히 일치** / 원본 고유 줄 43,626개 |
| NPC 관련 데이터 후보 | 25,088,049 | 원문 `Shop_GB_Yishiting` 문자열 **35회**; 어떤 원본 파일명인지는 미확인 |

- 두 `shop.ini` 유사본과 기존 Drive 원본 모두에서 **정확한 `[Shop_GB_Yishiting]` 섹션은 없음**. 두 유사본 각각에는 정확한 `[Shop_GB_Yishiting_1]`부터 `[Shop_GB_Yishiting_5]`까지의 섹션이 각 1개. NPC 관련 데이터 후보는 접미사 없는 `Shop_GB_Yishiting`을 참조한다. **NPC 기본 ID → 어떤 접미사/페이지를 선택하는지의 클라이언트 규칙은 여전히 알 수 없음**. `_1`을 단순 기본값으로 채택하거나 서버에 임의로 덮어쓰지 않음.
- 근접한 shop 카탈로그 두 스트림 중 어느 것이 실제 클라이언트의 적용 파일·버전·패치 우선순위인지도 미확인. SHA/공통 줄 수는 콘텐츠 관계 근거이지 원래 파일명/패치 우선순위 근거가 아니다.
- 기존 Go `server/cmd/protocol-probe/shop_catalog.go`의 `loadShopCatalogSection`은 **정확히 `[` + shopID + `]`**와 일치하는 섹션만 선택한다. 따라서 기본 ID를 넣으면 위 후보의 접미사 섹션으로 *자동 연결된다고 주장할 수 없다*. `PurchaseReady=false` fail-closed 유지, 구매 규격·통화·가방/DB 계약 추측 금지.

## 비공개 증거 묶음과 검사

- 대화 내 비공개 생성 파일 `/mnt/data/STAGE49_SHARE_SHOP_EVIDENCE_PRIVATE_20260917.zip`: **622,629 bytes**, SHA-256 `dd2fd71461344a06b5610272a9f6ce283b1013cddc4fb9a86f22f82dec2c5a72`. 익명 스트림 **7개** + `manifest.json` + `README_KO.txt` 포함. ZIP CRC **PASS**, 각 7개 출력 바이트 재판독 **PASS**. 임시 증거용 자료이며 게임/서버 폴더에 덮어쓰거나 공개 저장소에 올리지 말 것.
- 로컬 원본/참조파일 실행: **PASS**. Python syntax / 합성 자체 테스트: **PASS**. GitHub Actions [Stage49 run 35120092265](https://github.com/kim8553/web/actions/runs/35120092265) 전체 및 `synthetic-only` job **SUCCESS**; Stage48 해제기와 Stage49 검증기의 문법 및 합성 테스트 각 단계 **SUCCESS**. 공개 CI는 게임 패키지, ZIP, 독립 Drive 원본을 읽지 않았으므로 실제 패키지 검증은 로컬 결과와 명확히 분리.
- **Go gameplay/GM 지급 코드 변경 없음; 이 단계 Go build/race/vet, 클라이언트 접속, NPC 메뉴/구매/가방/DB LIVE/E2E 미실시.**

## 다음 확인점

실제 바이너리 근거로 (1) share.package **인덱스 해독 및 파일명-오프셋 대응**, (2) NPC `Shop_GB_Yishiting` 기본 식별자에서 `_1`~`_5` 섹션으로의 **선택·페이지 변환 경로**, (3) 동일 패키지 안 카탈로그 A/B의 **패치 우선순위**를 분리해서 검증할 것. 그 후에만 정확한 일반 구매 패킷·금액·가방/DB 변화를 계측하고 서버 코드를 수정할 수 있다.
