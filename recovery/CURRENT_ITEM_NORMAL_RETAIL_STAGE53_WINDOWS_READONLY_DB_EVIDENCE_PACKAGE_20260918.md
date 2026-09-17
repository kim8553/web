# Stage53 — 실제 Windows DB 점검을 위한 검증된 읽기 전용 실행 패키지 (2026-09-18)

## 작업 기준 및 범위

Stage52 최종 커밋 `3d17a63f72a3347361b2f7f0425e3081d8df02b5`, `kim8553/web` 브랜치 `stage37-assistant-handoff-20260917`부터 이어서 진행했다. 사용자 요청은 일반 아이템 및 NPC 상점 구매·판매 우선, GM 아이템 지급·교환 상점 0x4F 보류다. 이번 단계에서는 일반 구매 0x46의 **실제 사용자 DB에 대한 읽기 전용 사전 검사 실행 경로**만 준비했으며 거래 코드, 판매 0x47, 서버 마이그레이션 및 아이템 데이터는 변경하지 않았다.

## 실제 데이터 접근 경계

연결된 Google Drive에서 8월에 수정된 `nineyin` 폴더의 `role_bag_items.ibd`, `role_currency.ibd` 등 과거 테이블 파일의 존재는 확인했지만, 이것은 **현재 사용자 Windows에서 실행 중인 MySQL DB 연결도 아니고 9월 최신 DB의 스키마/행 상태도 아니다.** 직접적인 사용자 PC 연결, 현시점 백업의 복구 가능성, 현재 `nineyin` DB 검사 결과는 얻지 못했다. 구형 자료로 `max_hardiness` 등의 실제 현존 여부를 추측하거나 자동 DDL을 실행하지 않았다. 개인 DB/키/클라이언트 바이너리를 공개 GitHub에 업로드하지 않았다.

## 만들어서 확인한 결과물

- `tools/stage53-retail-db-check/run-stage53-preflight.ps1`: 로컬 서버 폴더의 `mysql.env` 또는 **이미 프로세스에 설정된** `NINEYIN_MYSQL_DSN`을 읽어 Stage52 점검 EXE를 실행한다. 비밀번호를 명령행에 실어 보내거나 stdout에 출력하지 않는다. 보고서가 유효한 JSON일 때 로컬 `stage53_report.json`만 작성하며, 접속 실패나 조건 불충족 시 BLOCKED로 종료한다. 게임 서버/클라이언트 실행·마이그레이션·DB 자동 수리는 하지 않는다.
- `tools/stage53-retail-db-check/README_KO.md`: 별도 백업 확인, 현재 MySQL 실행, 원본 게임 파일을 덮어쓰지 않는 압축 해제, 로컬 점검 명령 및 결과 중 **`stage53_report.json`만** 대화에 전달하는 절차. 비밀번호·`mysql.env`·데이터 덤프는 요청하지 않는다.
- `.github/workflows/jiuyin-stage53-windows-db-check-package.yml`: Go의 실제 공식 MySQL 드라이버를 이용하여 `retail-schema-preflight.exe`를 Windows amd64로 빌드하고 SHA256SUMS, PowerShell 스크립트 및 README를 4파일 ZIP으로 묶어 GitHub Actions artifact로 배포한다. Linux/Windows 서버 컴파일은 코드 확인일 뿐 실서버 실행이 아니다.

## 검증 및 실패 기록

첫 [Stage53 Actions 35247680456](https://github.com/kim8553/web/actions/runs/35247680456)은 Linux 빌드·ZIP 업로드는 성공했지만 Windows smoke 작업 종료코드가 실패했다. Windows 로그에서 `WINDOWS_NO_DSN_FAIL_CLOSED=PASS` 출력 직후 이전 테스트의 의도된 종료코드 2가 셸 단계에 남아 전체 작업을 실패하게 한 것이 확인됐다. 이를 전체 성공이라고 보고하지 않고 workflow를 수정했다.

수정 후 [Stage53 Actions 35247839668](https://github.com/kim8553/web/actions/runs/35247839668)의 **build-package + windows-smoke 작업 모두 SUCCESS**. Go 단위 테스트, 읽기 전용 생산 코드의 쓰기 SQL 금지 정적 검사, 실제 Windows EXE 빌드, Windows 환경에서 DB 연결 문자열이 없을 때 안전 종료, 로컬 `mysql.env` 테스트 값 로드/자격정보 미출력, Linux 및 Windows 서버 컴파일을 검증했다. **이 Windows 점검은 실제 사용자 MySQL에 접속한 검증이 아니다.** 테스트용 DB 성공/오류 시나리오는 앞선 Stage51·52의 격리 MySQL CI 근거에 한정된다.

최종 ZIP은 Stage53 두 번째 실행의 `stage53-windows-db-check` artifact **ID 10508570018**이며 SHA256SUMS.txt가 ZIP 내부 세 파일의 체크섬을 제공한다. ZIP에는 `retail-schema-preflight.exe`, `run-stage53-preflight.ps1`, `README_KO.md`, `SHA256SUMS.txt` 네 개 파일만 포함한다. GitHub Actions 자동 만료일 2026-10-01(UTC 기준). 최종 CI workflow 수정 커밋 `6d62422fe45773a8be15229977b1468a1548f883`.

## 상태와 다음 단계

**완료:** 사용자 PC용 읽기 전용 점검 실행 패키지 및 독립 CI Windows 실행/컴파일 검증.

**미완료:** 사용자 실제 `nineyin` DB의 백업 확인/점검 결과, 실제 스키마 수정 필요 여부, Windows 서버 실부팅, NPC 아이템 1개 구매 0x46→클라 가방·재화→재접속의 E2E, 일반 판매 0x47. 기존 `skill_new.ini` 의존성을 임의로 우회하지 않는다. GM 지급 및 교환 0x4F는 계속 보류.

**다음 검증 입력:** 사용자가 복구 가능한 DB 백업을 확인한 후 **그 PC에서만** ZIP의 읽기 전용 점검 도구를 실행하여 `stage53_report.json` 결과만 대화에 제공해야 한다. 보고서가 도착하기 전에는 사용자 DB가 PASS/BLOCKED인지 알 수 없으며 수정 SQL을 추측해서 실행하지 않는다. 게임 접속은 요구하지 않는다.
