# Stage47 — 동일 fxres.exe의 인덱스 읽기 이후 레코드 검증 기계어 (2026-09-17 KST)

## 기준·보호 범위

- 시작 브랜치 `stage37-current-recovery-20260916` HEAD: `8ad31755b6abca9933de52910c5cbba512d99e9f`. 누적된 `server/`는 **`9yin-go-server1.rar` 기반**으로 그대로 유지. `9yin-go-server.zip`, 과거 JYZJ 동작과 혼합·롤백하지 않음.
- 사용자 ZIP `bin64(2)(1)(1).zip`의 `fxres.exe`와 연결된 비공개 Drive 원본 `res/ini.package`, `res/lua.package`를 로컬 읽기 전용으로 사용. EXE/DLL을 실행·수정하지 않았고 게임 바이너리·리소스·복호화 데이터·개인 정보는 GitHub/Actions로 보내지 않음.
- `fxres.exe` SHA-256 `07ae76288148132995538488f12e2214fbecfdc0f18bdc2dd3189093e3e9fa9c`는 Stage39/46 입력과 **동일**하며 기존 헤더 경로의 13개 명령 바이트도 일치.

## 새 기계어 근거 — 인덱스 읽기 이후 경로

`objdump -D -M intel`로 원본 `.text`를 확인하고 SHA 고정 프로브로 **19개 명령 바이트 지점 + `.rdata` 오류 문자열 5개**를 독립 대조했다.

| 실제 VA | 확인된 기계어 동작 | 해석 한계 |
| --- | --- | --- |
| `0x14000dc3b`–`0x14000dc48` | 인덱스 영역 읽기에 버퍼·파일 컨텍스트 전달 후 반환 검사 | 기존 Stage39 헤더 경계 재사용 |
| `0x14000dc99`–`0x14000dca1` | 파일 컨텍스트를 인수로 보호 섹션 ``.`s9`` 안의 `0x1402070b6` 직접 호출 | **복호화인지 여부·키·변환 방식 전부 미확인** |
| `0x14000dcb3`–`0x14000dcd8` | 각 레코드의 선두 `u16` 길이와 인덱스 끝 경계 검사 | 원본 변환 전 인덱스에 직접 적용 불가 |
| `0x14000dce4`–`0x14000dd02` | 레코드 길이−28, 오프셋 +27 이름 첫 바이트 비영(非零), 레코드 마지막 바이트 NUL, +25의 `u16`이 길이−28 미만인지 검사 | +25 필드의 완전한 의미 미확정 |
| `0x14000dd0c`–`0x14000dd55` | 길이만큼 포인터 이동·반복, 보관된 레코드 포인터+27을 이름 포인터로 사용 | 성공적으로 해독한 실제 레코드 없음 |

검증된 오류 문자열에는 `(CPackData::LoadFromFile)read file info size failed`, `file info size error`, `read file info failed`, `file name error`, `comment offset error`가 있다(공통 접두부 동일). 마지막 오류 메시지만으로 +25 필드의 형식이나 실제 index transform을 단정하지 않는다. **보호 호출의 내부 알고리즘을 알아냈다고 주장하지 않는다.**

원본 `ini.package` SHA-256 `6185812c6153b2a6da56071968dcd1254510a974df06a7e36a0229be2105779a`, 선언된 수 18,543; `lua.package` SHA-256 `283c8c245a3fb86af2a7e8c21c53b3de26c30432341dd590c37554adfce04e97`, 선언된 수 2,286. **두 수 모두 실제 파싱 개수가 아니다. 실제 인덱스 해독 0건, 이름을 확인한 패키지 멤버 0건.** Stage46에서 기존 평문 인덱스 파서가 이 두 원본 파일과 호환되지 않는다고 확인한 상태도 유지.

## 도구·테스트의 구분

- [`tools/stage47_fxres_postread_record_audit.py`](../tools/stage47_fxres_postread_record_audit.py)는 Stage46 읽기 전용 검증기를 재사용한다. ZIP CRC·원본 패키지 SHA·기존 기계어 13곳·신규 레코드 기계어 19곳·오류 문자열 5곳을 검증하며, **원본 인덱스를 복호화/파싱하거나 파일을 추출하지 않는다**. 실제 검사에 사용한 로컬 파일과 업로드된 GitHub blob SHA 모두 `bc0932f9d5b369951ce2dba1237b73fe136ccdf0` 일치.
- 로컬 Python 문법 검사 **PASS**. Stage46 합성 테스트 **PASS**. Stage47 합성 레코드 정상 2종·오류 9종 및 잘못된 EXE 거부 **PASS**. 실제 비공개 사용자 ZIP과 원본 두 패키지 검사는 **해시·ZIP CRC·기계어·문자열·헤더 동일성에 한해 PASS**. 합성 레코드는 현재 패키지에서 나온 것이 아님.
- [GitHub Actions Stage47 실행 35117575955](https://github.com/kim8553/web/actions/runs/35117575955) (`cacda9f208b4f674580bf575c43cbd7ec7bdc728`): **완료·SUCCESS 확인**. `synthetic-read-only` 작업과 개별 **Verify Python syntax**, **Run synthetic fixtures without private client files** 단계 모두 **SUCCESS**. GitHub CI에는 게임 파일이 없어 원본 인덱스 복호화나 상점 성공의 증거가 아님.
- Stage47에서 Go 테스트/race/vet/Windows 빌드·서버 부팅·게임 접속·NPC 상점 구매/가방/DB 저장의 **LIVE/E2E 전부 미실시**. `server/` 게임플레이·GM 아이템 지급 수정 없음, `PurchaseReady=false` 그대로 유지.

## 다음 증거 경계

동일한 클라이언트의 검증 가능한 기계어·기존 분석 산출물로 실제 인덱스 변환 및 **파일명 ↔ 데이터 스트림** 대응을 확인한 뒤, 원본의 실명 상점 INI/Lua를 비공개로 찾아 기존 언팩 리소스와 해시·섹션을 비교한다. 정확한 상점 ID·일반 구매 요청·화폐·가방 지속성은 별도 LIVE/패킷 근거를 얻기 전까지 확정하지 않으며, 키·XOR·패킷 번호를 추측해서 넣지 않는다.
