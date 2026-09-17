# Stage53 Windows 테스트 서버

이 ZIP은 현재 `stage37-assistant-handoff-20260917` 브랜치의 `server/cmd/protocol-probe`를 Windows amd64로 빌드한 **테스트용 서버 EXE**다.

## 중요한 제한

- 기존 `9yin-game-native-menu.exe` 또는 `stage37-live-server.exe`를 자동으로 덮어쓰지 않는다.
- 이 패키지에는 사용자의 `resources`, MySQL 데이터, 클라이언트 파일, 비밀번호가 들어 있지 않다.
- 일반 NPC 상점 구매 0x46의 Stage50~53 변경은 포함되지만 **실제 사용자 DB/클라이언트 E2E 성공은 아직 미검증**이다.
- 일반 판매 0x47은 아직 구현되지 않았다.
- GM 아이템 지급과 교환 상점 0x4F는 보류 상태다.
- `skill_new.ini` 등 기존 서버가 요구하는 외부 데이터가 빠져 있으면 서버가 시작되지 않을 수 있다. 없는 파일을 임의 생성하지 않는다.

## 실행 전

1. 현재 `D:\9yin_server` 전체 또는 최소한 MySQL `data` 폴더를 복구 가능한 형태로 백업한다.
2. 기존 MySQL이 정상 실행되는지 확인한다.
3. 기존 `D:\9yin_server\resources` 폴더와 `mysql.env`를 그대로 둔다.
4. 이 ZIP은 **별도 폴더**에 푼다. 기존 서버 EXE 위에 덮어쓰지 않는다.

## 실행

PowerShell에서 ZIP을 푼 폴더로 이동한 다음:

```powershell
powershell -ExecutionPolicy Bypass -File .\run-stage53-test-server.ps1 -ServerRoot 'D:\9yin_server'
```

`mysql.env` 또는 현재 프로세스의 `NINEYIN_MYSQL_DSN`을 사용한다. DSN 값은 화면에 출력하지 않는다. 실행 작업 디렉터리는 `D:\9yin_server`로 맞추므로 기존 상대경로 리소스를 사용한다.

## 이번 테스트에서 먼저 볼 것

서버가 정상 부팅되어 클라이언트 접속까지 가능해진 뒤에만 일반 NPC 상점에서 **아이템 1개 구매**를 테스트한다. 구매 전/후 재화와 가방을 확인하고, 이후 게임을 완전히 종료했다가 재접속해서 아이템과 재화가 유지되는지 확인한다.

판매, GM 지급, 교환 시스템은 이번 테스트 대상이 아니다.

문제가 생기면 서버 콘솔/로그와 Stage53 DB 점검의 `stage53_report.json`을 제공하면 된다. 비밀번호나 `mysql.env` 자체는 공유하지 않는다.
