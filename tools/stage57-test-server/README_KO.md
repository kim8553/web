# Stage57 테스트 서버

기준 서버 계보: 사용자 제공 `9yin-go-server1.rar` -> 현재 `kim8553/web` Go 소스.

이번 변경은 원본 `9yin-go-server1` 런타임에 존재하지 않는 서버 전용 `resources/modern/share/item/drop_table.json` 때문에 전체 서버가 종료되는 문제만 다룹니다.

현재 서버 코드에서 이 JSON은 GiftBox/드롭 해석 경로에서만 사용되고, 일반 NPC 상점 구매 selector `0x46`에는 필요하지 않습니다. 따라서 파일이 **없는 경우에만** 빈 drop table로 계속 부팅합니다. 파일이 존재하지만 JSON 파싱에 실패하거나 다른 I/O 오류가 발생하면 계속 실패합니다. 없는 drop table을 임의 생성하거나 최신 Snail 드롭 의미를 추측하지 않습니다.

`drop_table.json`이 없는 동안 GiftBox/드롭 해석은 fail-closed 상태입니다. 일반 NPC 상점 구매 테스트 범위와 분리합니다.

## 실행

```powershell
powershell -ExecutionPolicy Bypass -File .\run-stage57-test-server.ps1 -ServerRoot "E:\9yin-go-server1\9yin-go-server1"
```

`ServerRoot`는 실제 `9yin-go-server1` 루트이며 `go.mod`와 `resources\modern\share\skill\skill_new.ini`가 있어야 합니다.

이번 ZIP에는 Snail 클라이언트/리소스 데이터가 포함되지 않습니다. 먼저 서버 부팅 로그를 확인하고, `protocol probe listening on 127.0.0.1:19061`까지 도달해 프로세스가 계속 살아 있을 때만 게임 접속 테스트로 넘어갑니다.
