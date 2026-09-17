# Stage53 — Windows 실제 DB 읽기 전용 점검 패키지

이 패키지는 **게임 서버가 아니라 DB 스키마 진단 도구**입니다. 사용자 PC의 현재 `nineyin` MySQL DB에 원격으로 연결하거나 직접 열람한 결과가 아닙니다. 사전 점검 프로그램은 SELECT/메타데이터 조회만 수행하며 마이그레이션, CREATE/ALTER/DELETE/INSERT/UPDATE, 아이템 거래, GM 지급, 교환 상점을 실행하지 않습니다.

## 준비

1. 기존 `nineyin` 데이터의 **정상 복구 가능한 별도 백업을 먼저 확인**하세요. MySQL을 실행 중인 상태에서 data 디렉터리를 단순 복사하는 것을 안전한 일관성 백업으로 취급하지 마세요. 사전 점검 자체는 DB를 수정하지 않습니다.
2. 최신 Stage53 GitHub Actions 검증에서 받은 ZIP의 `retail-schema-preflight.exe`와 `run-stage53-preflight.ps1`을 같은 폴더에 둡니다. 원본 게임 클라이언트 폴더나 실행 중인 서버 EXE를 덮어쓰지 마세요.
3. 현재 MySQL이 정상 실행 중인지 확인합니다. `mysql.env`는 서버 폴더 또는 그 아래 `server` 폴더에 있어야 합니다. 환경변수 `NINEYIN_MYSQL_DSN`이 이미 설정된 경우 이를 우선 사용합니다. **DSN·비밀번호·mysql.env 파일은 채팅이나 공개 GitHub로 전송하지 마세요.**

## 실행 (Windows PowerShell)

패키지를 `D:\9yin_server\stage53-check`에 풀었다면:

```powershell
cd D:\9yin_server\stage53-check
powershell -NoProfile -File .\run-stage53-preflight.ps1 -ServerRoot 'D:\9yin_server'
```

위 경로는 **예시**입니다. 실제 설치 위치에 맞춰 변경하세요. 실행 정책이 차단하면 무조건 해제하지 말고 내려받은 스크립트 내용을 확인한 뒤 신뢰할 수 있는 경우에만 실행하세요.

## 결과 처리

- `PASS`는 실행한 DB의 **필수 테이블/컬럼·InnoDB·마이그레이션 체크섬·역할별 재화 행·재화 JSON 형식** 사전 조건을 통과했다는 뜻입니다. 구매 성공을 뜻하지 않습니다.
- `BLOCKED`는 필요한 조건을 충족하지 못했거나 접속/검사가 실패했다는 뜻입니다. 기존 DB를 자동으로 수정하지 않습니다.
- 실행 폴더에 `stage53_report.json`이 생성되면 **그 JSON만 이 대화에 첨부**해 주세요. 로그 전체, `mysql.env`, DB 덤프, 계정 정보는 보내지 마세요. 보고서는 DB 이름 및 집계/스키마 결과만 포함하도록 설계됐지만 보내기 전에 내용을 확인하세요.
- 결과가 없으면 프로그램 실행을 다시 반복하기 전에 MySQL의 실행 여부와 로컬 `mysql.env` 위치만 확인하세요. 비밀번호는 공유하지 마세요.

## 검증 범위와 남은 작업

CI는 안전한 폐기용 MySQL fixture와 빌드를 검증할 뿐, 실제 사용자의 PC·DB에는 접근하지 않습니다. 이 패키지는 기존 서버를 구동하지 않고 게임에 로그인하지 않으므로 아이템 구매(0x46)·가방 화면·재접속·판매(0x47)를 검증하지 않습니다. 기존 `skill_new.ini` 부팅 의존성도 해결하지 않습니다. GM 아이템 지급 및 교환 상점(0x4F)은 사용자 요청에 따라 보류 중입니다.
