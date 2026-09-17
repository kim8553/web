# Stage56 테스트 서버

기준 서버 계보: 사용자 제공 `9yin-go-server1.rar` -> 현재 `kim8553/web` Go 소스.

이번 변경은 `resources/modern/text/stringname.idres`가 없는 원본 런타임에서 서버 전체가 종료되는 문제만 다룹니다. 이 파일은 현재 코드에서 GM 카탈로그 표시명 보조 용도로만 사용되므로, 파일이 없으면 빈 이름 맵으로 계속 부팅합니다. 다른 읽기 오류는 숨기지 않습니다.

## 실행

```powershell
powershell -ExecutionPolicy Bypass -File .\run-stage56-test-server.ps1 -ServerRoot "E:\9yin-go-server1\9yin-go-server1"
```

`ServerRoot`는 실제 `9yin-go-server1` 루트이며 `go.mod`와 `resources\modern\share\skill\skill_new.ini`가 있어야 합니다.

이번 ZIP에는 Snail 클라이언트/리소스 데이터가 포함되지 않습니다. 먼저 서버 부팅 로그를 확인하고, 서버가 계속 실행 중일 때만 게임 접속 테스트로 넘어갑니다.
