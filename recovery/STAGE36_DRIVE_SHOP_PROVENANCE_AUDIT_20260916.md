# Stage37 Stage36 — Drive res / JYZJ.server / bin64 상점 근거 대조 (2026-09-16)

## 기준 / 범위
- 이 조사 시작 당시 작업 브랜치 HEAD: `eae28417ba265bde6ecba6f467e51a6cf928604e` (Stage35). Stage34까지의 기존 패치·재구성 계보를 되돌리거나 재적용하지 않았다.
- 권위 있는 서버 원본은 사용자가 제공한 `9yin-go-server1.rar`; RAR 내부 `resources/modern/share/trade/shop.ini`의 SHA-256은 `ceb7a3a7e17cbc118ce5a7d3d979dea6120c50cdf6a5823697c05a5b1b9cf5dd`로 Stage33 보고서와 일치한다. 잘못된 `9yin-go-server.zip`은 사용하지 않았다.
- Google Drive의 `res` 패키지, 구형 `JYZJ.server/Res/Trade/Shop` XML 및 `bin64` 목록·클라이언트 로그를 **읽기 전용**으로 확인했다. Drive 폴더 구조와 파일명은 실제 조회한 항목만 기록한다. 원본 자료·내부 로그·DLL/EXE·개인 데이터는 공개 저장소에 업로드하지 않았다.

## 신규 확정 결과
1. 구형 JYZJ XML `Shop.xml` (56,269 bytes, SHA-256 `1e6f4f7465262ae385f40d6b299ce4b990a9bd6a5e4acd73e47783a5f447028f`)에서 `Property/@ID` 76개를 파싱했다. `ShopItem.xml` (472,012 bytes, SHA-256 `d6431391b5ecff782edcf2d390367cdf6b36ae8df8ae4483d4de026ba395d96e`)의 상품 `Property`는 2,755개다. 해당 XML은 GB2312 선언을 GB18030 호환 디코딩한 **레거시 비교 자료**이며 최신 프로토콜 명세가 아니다.
2. RAR `shop.ini`에서 후보 구역 이름 1,447개를 추출해 구형 `Shop.xml`의 76개 ID와 **대소문자까지 정확히 비교한 교집합은 0개**였다. 이 결과는 상점 ID 집합 간 비교이지 원본 서버의 전체 기능 호환성 판정이 아니다. 따라서 구형 JYZJ의 상점 상품·가격·화폐 필드를 Stage37에 자동 이식하지 않는다.
3. `Shop_GB_Yishiting`의 정확히 일치하는 구역은 RAR `shop.ini`와 구형 XML 양쪽에 없다. RAR에는 `_1`~`_5` 접미사 후보 5개가 있다. **올바른 매핑은 알 수 없으며** 자동 치환하지 않는다.
4. 검사 도중 RAR `shop.ini`에 중첩 여는 대괄호를 가진 후보 구역 헤더 1개를 확인했다. 보고된 1,447 후보 구역 숫자와 정합성을 유지하기 위해 검사용 스크립트는 인쇄 가능한 ASCII 구역 헤더를 세고 중첩 대괄호 수를 별도 출력한다. 이 한 건의 실제 게임 영향은 확인되지 않았으므로 파서나 리소스를 수정하지 않았다.
5. Drive `res/lua.package` (21,840,086 bytes, SHA-256 `283c8c245a3fb86af2a7e8c21c53b3de26c30432341dd590c37554adfce04e97`) 및 `res/ini.package` (20,578,406 bytes, SHA-256 `6185812c6153b2a6da56071968dcd1254510a974df06a7e36a0229be2105779a`)를 별도의 사본으로 확보했다. 두 파일의 처음 네 바이트는 `PCK0`이다. **해제·Lua 디코딩·클라이언트 버전 일치 확인은 수행하지 않았다.** 압축된 바이트에서 특정 문자열이 보이지 않는 것을 식별자 부재의 증거로 해석하지 않는다.
6. Drive `bin64` 폴더 목록에서 `fxgame.exe`, `fxgamelogic.dll`, `fxnet2.dll`, `fxcore.dll`을 확인했다. 그러나 `fxgame.exe`와 `fxgamelogic.dll`의 실제 다운로드는 Google Drive가 `cannotDownloadAbusiveFile` (403)로 차단했다. 이는 제공자 다운로드 차단 사실만 의미하며 바이너리가 악성이라고 독립 판정한 것이 아니다. **기계어·RTTI·해시·서로 같은 패치 버전인지는 검증 불가**. 우회 다운로드를 시도하지 않았다.
7. Drive `bin64/trace.log` (12,379 bytes, SHA-256 `a2466b6b7a3467a3dfed5281312c1bc95d0e7f1097e6eeb3e8d7350c9feaf318`)를 GB18030으로 읽어 177개 줄을 확인했으며 `shop`/`npc` 문자열 검색은 각 0건이었다. 이 **클라이언트 로그는** Stage32 이후 서버 상점 프레임 로그가 아니며 구매·표시 성공/실패 근거로 사용할 수 없다. 실제 내용은 공개하지 않았다.

## 신규 검사 도구와 실행 결과
- `tools/stage36_shop_authority_guard.py`: 원본 경로를 명시하는 읽기 전용 입력 2개, 상점 ID 지정 인자, 합성 self-test, 정량 JSON 출력. 자동 매핑·구형 XML 권위 승격·구매 성공 판정은 명시적으로 모두 `false`이며 원본을 변경하지 않는다.
- 로컬 `--self-test`: `SELF_TEST_PASS` (합성 3개 INI 구역/2개 XML ID, 교집합 1개, 접미사 2개, 형식 오류 fail-closed 등 검증).
- 원본 RAR `shop.ini` + Drive XML: 후보 구역 1,447, 비정상 중첩 대괄호 후보 1, 구형 XML 상점 76, 정확한 교집합 0, 질의 상점의 정확한 구역 0, 접미사 후보 5. 전체 Go 테스트·race·vet·Windows 빌드는 **이번 작업에서 수행하지 않았다**. 이 검사 도구는 게임 실행 코드와 무관하다.

## 다음 근거 경계
- Google Drive의 이전 구형 `JYZJ.server`는 상점 패킷/화폐 구현의 권위가 아니다. 클라이언트 최신 `res`의 해당 상점 Lua/INI를 실제로 해제하여 정확한 참조를 확인할 수 있기 전까지 접미사를 선택할 수 없다.
- 최신 클라이언트 `bin64`의 실제 바이너리 분석은 다운로드 차단 때문에 현재 불가하다. 이미 확보된 Stage12 근거·현재 소스·패치 보고서를 계속 활용한다.
- 라이브 표시 검증은 Stage34 이후 **실제 사용 중인** 서버에서 같은 NPC 선택의 `NPC shop menu diagnostic` → `shop service selected` → `shop display catalog` → `shop display frames` (또는 preflight error) 및 클라이언트 실제 화면 기록이 있어야 가능하다. 기존 Stage35에서 확인한 로그에는 그 이벤트가 없다.
- GM 웹페이지 지급, 교환 구매·가격·화폐·가방·DB, 사용자 PC 실행 파일/DB/캐릭터·원본 Drive 파일은 **변경 없음**. LIVE/E2E **NOT RUN / NOT ESTABLISHED**.
