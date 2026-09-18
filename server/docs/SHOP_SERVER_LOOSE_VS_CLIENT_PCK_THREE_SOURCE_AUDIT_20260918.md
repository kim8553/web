# 서버 기본 `shop.ini`와 최신 클라이언트 PCK의 세 자료 대조 (2026-09-18)

## 범위 및 실제 출처

- 현재 서버 브랜치 `stage37-restored-20260918`의 `server/cmd/protocol-probe/runtime_paths.go`에서 `defaultModernShareRoot = runtimeProjectPath("resources", "modern", "share")`; `shop_catalog.go`에서 `defaultShopINIPath = filepath.Join(defaultModernShareRoot, "trade", "shop.ini")`임을 확인했다. 이는 서버의 *기본* 파일 경로이지 사용자 PC 실제 파일의 해시를 확인한 것은 아니다.
- Drive 파일 계층 `9yin_server/resources/modern/share/trade/shop.ini`: 파일 ID `1qDCtz31_3vcYmshpF39yw82IqyRJJJTr`, 원본 크기 **2,176,439 bytes**, SHA-256 `ceb7a3a7e17cbc118ce5a7d3d979dea6120c50cdf6a5823697c05a5b1b9cf5dd`. Drive 파일 수정 시각은 `2026-07-17T04:23:20Z`; 실제 패키지 작성 시각, 사용자 PC의 활성 리소스, 최신 패치 우선순위를 의미하지 않는다.
- 별도 Drive `res/share.package`: ID `1eoJ5ViSVdEY1xk8OKNcxmqHhcOmARQxh`, 크기 **40,680,972 bytes**, SHA-256 `200497852ba3a29279e51f01e2913b5f7260480f2a740a32ebd1680b869844b6`. 이전에 검증한 두 zlib 스트림은 오프셋 `33744821`(해제 SHA-256 `d2e058399d4016420846c8df1388c260181b1e81ceee9e51216ca80fcfd4787e`)과 `36877029`(해제 SHA-256 `f42fce7631f7fe302d3fdedad60b12eaa20da12396de412ff0a84edcddc1a6d9`)이다. **어느 스트림이 실제 클라이언트 파일 경로에 매핑되는지 확인하지 못했다.**

## 재현한 세 파일의 구체적인 차이

- 세 원본 모두 총 섹션 헤더 **1,447개**, 그중 대문자 `Shop_`으로 시작하는 섹션 **960개**이며 섹션 이름 집합이 동일하다. 각 전체 SHA-256은 서로 다르다.
- 서버 경로의 loose `shop.ini` vs 첫 PCK 스트림: 전체 섹션 **34개** 내용이 다르다(대문자 `Shop_` 8개, 기타 접두사 26개). loose vs 두 번째 PCK 스트림: **36개** 내용이 다르다(대문자 `Shop_` 10개, 기타 접두사 26개). 두 PCK 스트림 사이 차이는 이전 검증대로 **3개**다. 바이트 일치만으로 파일 우선순위를 정하지 않는다.
- 서버 경로 loose 파일의 `Shop_school_zhenghe_fc_sl035_zs_01`은 첫 스트림과 일치해 페이지 1 칸 12에 서로 다른 두 상품이 있는 중복 사례를 포함하며, 두 번째 스트림의 해당 섹션과 다르다. `Shop_special_001`의 명시 `PageInfo=Page1` 및 상품 행 키 `0`,`1`은 세 자료 모두 동일하다.
- GitHub에 게시된 현행 `shop_catalog.go`와 **동일한 Git blob `c29e650862cc29e4c1baaacd36ac55ed98a6693b`**를 사용하여 loose `shop.ini`의 전체 1,447개 섹션을 독립 Go 테스트로 조회한 결과: `PASS=1413`, `EMPTY=32`, `DUPLICATE=1` (`Shop_school_zhenghe_fc_sl035_zs_01`), `PAGE_OUTSIDE=1` (`Shop_special_001`). Go 테스트 자체 PASS. 두 문제 상점은 현행 로더가 명시적으로 거부한다. 이는 구매/판매 또는 LIVE 성공이 아니다.

## 공개된 재현 도구·검증 경계

- `server/tools/audit_shop_three_sources.py`는 기존 `audit_share_shop_pck.py`의 패키지 경계/스트림 검사에 연결하여 **실제 loose 및 패키지의 필수 SHA-256을 확인한 뒤** 전체 섹션의 바이트 차이를 보고한다. 후보 스트림 자동 선택·리소스 쓰기·PCK 색인 복호화·게임 실행 기능은 없다. 실제 Drive 사본 대조 결과 `1447 / 34 / 36`을 재현했다. 합성 self-test 2개는 후보가 loose 파일과 같더라도 우선순위를 임의 선택하지 않는 점과 입력 SHA/중복 오프셋 거부를 확인했다.
- `fxres.exe`의 헤더·일부 색인 직접 판독 코드와 현재 `share.package` 첫 색인 바이트의 불일치는 이전 문서 `SHOP_PCK_INDEX_BOUNDARY_AND_STREAM_DIFF_20260918.md`를 참조한다. **이것만으로 실제 게임 클라이언트가 사용하는 색인 처리 함수를 확정하지 않았다.**
- 사용자 DB, 정상 로그인·맵 진입 ZIP, 원본 loose `shop.ini`, `share.package`, BIN64는 수정하지 않았다. 사용자 PC의 실제 경로에 존재하는 파일의 해시, 패키지 경로→스트림 매핑, NPC 판매 요청·정산, 실게임 구매·재접속 E2E는 미검증이다. 이 셋 중 어떤 파일도 임의로 배포하거나 덮어쓰지 않는다.
