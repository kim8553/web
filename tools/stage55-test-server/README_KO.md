# Stage55 테스트 서버

이 패키지는 사용자 제공 `9yin-go-server1.rar`에서 이어진 현재 Go 서버 소스 계보를 기준으로 빌드합니다.

Stage54 실기동에서 확인된 차단점은 `resources/modern/share/ini/effect/playerweapon` 디렉터리 부재였습니다. 현재 코드에서 해당 디렉터리에서 채우는 `weaponModels` / `weaponHeld` 맵은 로드 후 실제 읽기 사용처가 없었고, 원본 `9yin-go-server1.rar`에도 해당 디렉터리가 없습니다. Stage55는 이 디렉터리가 **없을 때만** 빈 보조 맵으로 진행하며, 다른 I/O 오류는 숨기지 않습니다.

실행:

```powershell
powershell -ExecutionPolicy Bypass -File .\run-stage55-test-server.ps1 -ServerRoot "실제 9yin-go-server1 루트"
```

`ServerRoot`에는 `go.mod`와 `resources\modern\share\skill\skill_new.ini`가 존재하는 실제 `9yin-go-server1` 루트를 지정하세요.

현재 범위: 일반 NPC 구매 `0x46` 우선. 판매 `0x47` 미구현. GM 지급 및 교환 `0x4F` 보류.
