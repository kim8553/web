# Stage50 — 동일 share.package의 길드 상점 등급별 자료 대조 (2026-09-17 KST)

## 기준·보존

- 시작 권위 브랜치 `stage37-current-recovery-20260916` HEAD `725ff4c605de8368d402deddf3e0e7a7e3b130e5`. `9yin-go-server1.rar`에서 누적 복구한 `server/` 코드는 유지했고 `9yin-go-server.zip`이나 옛 JYZJ로 바꾸지 않았다.
- 입력: 사용자가 연결한 원본 `share.package` SHA-256 `200497852ba3a29279e51f01e2913b5f7260480f2a740a32ebd1680b869844b6` 및 비공개 Stage49 증거 ZIP SHA-256 `dd2fd71461344a06b5610272a9f6ce283b1013cddc4fb9a86f22f82dec2c5a72`. 별도의 패키지 인덱스 해독/파일명 확인은 수행되지 않았다.
- 검증된 zlib 스트림 `[0]`~`[4]` 데이터는 원본 `share.package` 시작 오프셋 **20,626,686**, 끝 **20,626,866**에서 읽고 Adler-32와 해제 후 SHA-256 `22f04ef1355845f5140d14ce4255683ca7f427e6ac219ad8f617217a32f1e1db`를 검증했다. 원본 이름을 `shop.ini` 등으로 단정할 수 없다.

## NPC 원본 컬럼과 상점 연결의 한계

- Stage49의 **익명 NPC 표**는 GBK/GB18030로 읽었을 때 첫 줄에 108개 열 제목을 갖는다. **0-based 45번 열 `商店ID`(상점 ID)**, **102번 열 `所属帮会地块编号`(소속 길드 구획 번호)**임을 원본 바이트로 확인했다.
- 정확한 상점 값 `Shop_GB_Yishiting`을 갖는 서로 다른 NPC **35개**의 45번 열을 확인했다. 같은 행의 `1701`, `1702`, `101` 등은 **102번 열**의 데이터이므로 이를 `_1`~`_5` 선택값이나 길드 레벨이라고 해석할 근거가 없다.
- Stage49에서 확인된 두 익명 XML 스트림(시작 오프셋 5,433,478 / 29,282,573)에도 `Guildfuncnpc151`이라는 NPC ID가 각각 한 번 등장했다. XML 스트림의 원래 경로와 클라이언트 사용 여부는 미확인이다.

## 검증된 다섯 구간 간 콘텐츠 대응 (런타임 selector 아님)

| 익명 INI 구간 | 상점 후보 A/B의 섹션 | 양쪽 상품 수 | 상품 필드 처음 4개·표시 순서 일치 | A/B 섹션 바이트 일치 |
| --- | --- | ---: | --- | --- |
| `[0]` | `[Shop_GB_Yishiting_1]` | 3 | 참 | 참 |
| `[1]` | `[Shop_GB_Yishiting_2]` | 4 | 참 | 참 |
| `[2]` | `[Shop_GB_Yishiting_3]` | 5 | 참 | 참 |
| `[3]` | `[Shop_GB_Yishiting_4]` | 6 | 참 | 참 |
| `[4]` | `[Shop_GB_Yishiting_5]` | 7 | 참 | 참 |

- 정확한 검사는 구간별 각 상품의 첫 **4개 쉼표 필드**가 순서대로 동일한지, 등급 INI의 **6번째(0-based 5) 필드**와 상점 후보의 **7번째(0-based 6) 필드**가 순서대로 일치하는지, 각 후보 A/B의 섹션 블록 전체가 동일한지 확인한다. 이 값들의 게임 의미를 알 수 없는 칼럼에 억지로 부여하지 않는다.
- 후보 A와 B의 *전체 파일* SHA는 서로 다르지만, 이 특정 **다섯 섹션은 원본 바이트가 모두 동일**하다. 대응은 강한 **정적 콘텐츠 상관관계**이며, 게임이 실제로 `[0]→_1` 등의 선택을 실행한다는 독립 증거가 아니다.
- 패키지 3개(`share`·`lua`·`ini`)의 Stage48 방식으로 검증된 익명 zlib 스트림에 한정한 읽기 전용 리터럴 조사에서 대소문자 일치 `Shop_GB_Yishiting`은 `share.package`의 NPC 표 **35회**, 대형 상점 후보 A/B **각 5회**, 다른 두 패키지 **0회**였다. 이것은 Lua 암호화·미분류 데이터 또는 런타임 문자열 조합에 해당 이름이 없다는 증명이 아니다.

## 재현·검증·비공개 산출물

- 일반 검증 도구 [`tools/stage50_guild_shop_tier_correlation.py`](../tools/stage50_guild_shop_tier_correlation.py) Git blob SHA `327bde20ed0aca945a417fea626b08b35889c16c`는 로컬 Python 문법 검사·합성 self-test·실제 원본 SHA-pinned 검사에 성공한 소스와 같다. 잘못된 상품 필드나 NPC 열 이름은 합성 테스트에서 거부한다. `python tools/stage50_guild_shop_tier_correlation.py --self-test`로 비공개 파일 없이 테스트할 수 있다.
- 실제 로컬 실행: `python tools/stage50_guild_shop_tier_correlation.py --share <private/share.package> --stage49-private-zip <private/Stage49-evidence.zip>`. 출력은 등급별 개수·컬럼 번호·섹션 SHA만 담은 메타데이터이다.
- 별도 **비공개** `STAGE50_GUILD_SHOP_TIER_EVIDENCE_PRIVATE_20260917.zip` 1,392 bytes, SHA-256 `928e66e00219172ddccfc0492dc86d80cc87b6682a69393c48463f4efcf570e3`, 내부 3개 항목(tier 익명 원본 1개, 메타데이터, 한국어 README), ZIP CRC 및 tier 스트림 SHA 재검증 **PASS**. 개인 대화 첨부로만 제공하며 **공개 GitHub/Actions에 업로드하지 않음**.
- GitHub Actions [Stage50 합성 전용 실행 35121949450](https://github.com/kim8553/web/actions/runs/35121949450): Python 문법 검사와 합성 변형 거부 검사 모두 **SUCCESS**. 비공개 패키지는 CI에 없다. 실제 패키지 검사 성공은 별도의 로컬 SHA-pinned 실행 근거이고, CI 결과는 실제 클라이언트 런타임 성공이 아니다.

## 다음 증거 경계

**원본 인덱스의 이름–오프셋 복구, 또는 실제 현재 클라이언트의 NPC 상점 선택 경로/일반 구매 요청·응답 캡처가 필요하다.** 현재 서버 `shop_catalog.go`는 상점 ID를 정확한 섹션명으로 읽고 `shopreadiness.FromCatalogError`도 비슷한 이름으로 대체하지 않도록 한다. 35개의 기본 ID를 `_1`~`_5`로 추측 변환하거나 상품·화폐·가방 데이터를 임의 수정하지 않는다. `PurchaseReady=false` 유지. 이번 Stage50에서 `server/` gameplay·GM 지급 코드는 변경하지 않았고 Go build/race/vet, 게임 로그인, NPC 구매, 가방/DB LIVE/E2E는 실행하지 않았다.
