# Drive `share.package` 색인 경계·상점 두 스트림의 차이 검증 (2026-09-18)

## 출처와 변경 금지 경계

- 원본 서버 계보: 사용자 지정 `9yin-go-server1.rar`; 작업 브랜치: `stage37-restored-20260918`. 사용자 정상 로그인·맵 진입 ZIP, MySQL DB, 클라이언트 파일은 변경하지 않았다.
- 실제 입력: 연결된 Drive `res/share.package` (`https://drive.google.com/file/d/1eoJ5ViSVdEY1xk8OKNcxmqHhcOmARQxh/view`), 크기 40,680,972 바이트, SHA-256 `200497852ba3a29279e51f01e2913b5f7260480f2a740a32ebd1680b869844b6`.
- 최신으로 지정한 BIN64 ZIP 안의 `fxres.exe`: SHA-256 `07ae76288148132995538488f12e2214fbecfdc0f18bdc2dd3189093e3e9fa9c`; 이전 `tools/stage39_pck0_header_code_probe.py`의 13개 명령어 지문과 **이번 사용자 ZIP의 실제 exe 바이트**가 모두 일치했다. 지문 확인은 파일 인덱스 해독 성공이 아니다.

## 이번에 확정한 데이터 경계

- PCK0 파일 오프셋 `0x04`의 기본 레코드 길이 `15`, 엔트리 선언 수 `9726`, 색인 시작 `19`, 색인 끝 `790911` (배타적), 색인 크기 `790892` 바이트. 이전 Stage39의 `fxres.exe` 헤더 판독 코드와 일치한다.
- `fxres.exe`의 실제 `0x14000dcc0` / `0x14000dce7` / `0x14000dcf1`에서 항목 길이·파일명 NUL 검사가 관찰된다. `share.package`의 **색인 원본 바이트를 그대로** 적용하면 첫 항목 길이 후보 `45304`의 파일명 첫 바이트와 마지막 바이트가 NUL 검사를 통과하지 못한다. **색인 인코딩/해독 방식은 아직 모른다**. 이 결과만으로 암호 방식, 엔트리 배치, 파일명 또는 콘텐츠 우선순위를 정하지 않는다.
- 앞서 발견한 두 zlib 후보는 모두 색인 영역 바깥의 독립된 완전한 스트림이다. 스트림 시작/압축 종료(배타적)/해제 크기/해제 SHA-256:
  - `33744821` / `34022137` / `2177532` / `d2e058399d4016420846c8df1388c260181b1e81ceee9e51216ca80fcfd4787e`.
  - `36877029` / `37154365` / `2177326` / `f42fce7631f7fe302d3fdedad60b12eaa20da12396de412ff0a84edcddc1a6d9`.
- 두 스트림은 `[Shop_]` 섹션 이름 집합이 각각 **960개로 동일**하고, 내용 바이트가 다른 섹션은 **3개**, 동일 섹션은 **957개**이다. 서로 다른 섹션: `Shop_20190417_shopyl_001` (상품 목록 변경), `Shop_school_zhenghe_fc_sl035_zs_01` (네 상품의 페이지 1 칸 `12~15` 대 `24~27`), `Shop_zyb_sdhd` (상품 목록 변경). 두 스트림은 동일 바이트의 중복 파일이 아니다.

## 재현 도구·검증과 미확정 항목

- 새 읽기 전용 도구: `server/tools/audit_share_shop_pck.py`. 실제 파일의 SHA-256·PCK0 헤더·명시적으로 지정한 스트림의 zlib 종료/크기/해시·섹션별 차이를 출력한다. 출력의 `authoritative_shop_stream`은 **항상 null**이고 `shop_path_resolved_by_index`는 **false**이다. 임의 파일명 추정, 색인 복호화, 상품 자동 교체, 원본 리소스 쓰기는 하지 않는다. 전체 패키지 검색기가 아니라 **별도로 검증한 오프셋을 재감사하는 도구**다.
- 실제 입력 재현 예시: `python3 server/tools/audit_share_shop_pck.py --package /path/to/share.package --expected-sha256 200497852ba3a29279e51f01e2913b5f7260480f2a740a32ebd1680b869844b6 --candidate-offset 33744821 --candidate-offset 36877029`.
- 실제 Drive 사본에서 위 명령을 실행해 두 스트림의 경계와 해시, 섹션 차이 3개를 확인했다. 합성 self-test 3건은 정상적인 두 후보에서 자동 선택을 금지하고, 해시 불일치·색인 내부 오프셋·중복 후보·잘못된 PCK0 헤더·잘린 zlib 입력을 거부함을 확인한다.
- GitHub Actions `.github/workflows/stage37-share-shop-index-boundary.yml`은 공개 저장소에 원본 패키지를 게시하지 않고 합성 self-test만 실행한다. CI 성공을 실제 package의 색인 복원 또는 Windows 게임 실행 성공이라고 보고하지 않는다.

**미완료:** PCK 색인에서 `share/trade/shop.ini`의 정식 파일명/우선순위와 두 스트림 중 선택되는 바이트를 확정하지 못했다. 최신 BIN64 판매 요청 패킷·정산 규칙, 사용자 DB 컬럼 대조, NPC 구매/판매 실게임·재접속 E2E도 미검증이다. 따라서 `shop.ini`를 골라 덮어쓰거나 판매 핸들러를 추측해 구현하지 않는다. 다음 작업은 이번 BIN64의 패키지 색인 판독·파일 조회 코드에서 근거가 확보되는 경우에만 **정확한 경로→스트림 매핑**을 진행한다.
