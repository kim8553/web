# Stage58 전체 로컬 서비스 테스트

기준 서버 계보는 사용자 제공 `9yin-go-server1.rar` -> 현재 `kim8553/web` Go 소스입니다.

Stage57은 `protocol-probe`만 실행해서 `127.0.0.1:19061`/GM 포트만 열었고, 원래 로컬 서버 구성이 요구하는 서버 목록 서비스 `127.0.0.1:4000`을 패키지에서 빠뜨렸습니다. Stage58은 원본 `server/启动本地九阴服务.bat` 계약에 맞춰 다음 세 서비스를 함께 올립니다.

- 서버 목록: `127.0.0.1:4000` (`loopback-lister.ps1`)
- 게임 서비스: `127.0.0.1:19061`
- GM UI: `127.0.0.1:19062`

실행 스크립트는 세 포트가 모두 LISTEN 상태가 된 경우에만 `STAGE58_READY=YES`를 출력합니다. 이것은 로컬 서비스 준비 상태만 의미하며 최신 Snail 클라이언트 LIVE/E2E 성공을 의미하지 않습니다.

## 실행

먼저 이전 Stage57 서버를 종료해서 19061/19062를 비웁니다. 그 다음:

```powershell
powershell -ExecutionPolicy Bypass -File .\run-stage58-live-services.ps1 -ServerRoot "E:\9yin-go-server1\9yin-go-server1"
```

또는 `RUN_STAGE58_LIVE.bat`를 실행하고 `9yin-go-server1` 루트 경로를 입력합니다.

`ServerRoot`에는 다음이 있어야 합니다.

- `go.mod`
- `resources\modern\share\skill\skill_new.ini`
- `data\roles.json`

포트 충돌 시 기존 프로세스를 자동 종료하지 않습니다.

`STAGE58_READY=YES`가 나온 뒤에만 최신 클라이언트 접속을 시도합니다. 접속 실패 시 `logs\protocol-probe-live.log`와 `logs\lister\request.bin`/`response.bin`이 진단 근거입니다.

Stage55/56/57에서 확인한 원본 런타임 누락 보조 리소스 처리와 Stage54 일반 상점 구매 `0x46` 저장 경로는 현재 소스 빌드에 그대로 포함됩니다. GM 지급과 exchange `0x4F`는 계속 보류입니다.
