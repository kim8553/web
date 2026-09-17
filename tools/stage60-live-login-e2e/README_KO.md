# Stage60 — 최신 Snail 클라이언트 LIVE 로그인 E2E

이 패키지는 `9yin-go-server1.rar` -> 현재 `kim8553/web` Go 서버 계보만 사용합니다.

## 현재 범위

Stage59의 JSON login identity adoption 코드를 그대로 포함한 Windows 서버를 사용해 실제 최신 Snail 클라이언트의 로그인 경로를 검증합니다.

이 패키지는 로그인 프로토콜을 새로 추측해 변경하지 않습니다. 먼저 실제 실패 지점을 수집합니다.

## 실행 전에

1. Stage58/59 또는 다른 Nine Yin 테스트 서버를 모두 종료합니다.
2. 이 ZIP을 별도 폴더에 풉니다.
3. 원본 `9yin-go-server1.rar`를 풀어둔 **실제 서버 루트**를 확인합니다.
   - 루트에는 `go.mod`
   - `data\roles.json`
   - `resources\modern\share\skill\skill_new.ini`
   가 있어야 합니다.
4. MySQL을 명시적으로 사용하지 않는다면 `NINEYIN_MYSQL_DSN`을 설정하지 않습니다.

`D:\9yin_server` 같은 과거 경로는 기본값으로 사용하지 않습니다.

## 실행

`RUN_STAGE60_LIVE.bat`

`ServerRoot:`가 나오면 실제 `9yin-go-server1` 루트를 입력합니다.

런처는 이전 서버를 재사용하지 않습니다. `4000`, `19061`, `19062` 중 하나라도 이미 사용 중이면 중단합니다.

JSON 모드에서는 시작 전에:

`data\stage60-backups\roles.before-stage60-login.<timestamp>.json`

백업을 만들고 원본과 SHA256이 같은지 검증합니다. 백업 실패 시 서버를 시작하지 않습니다.

## 첫 LIVE 시도

콘솔에 `STAGE60_READY=YES`가 나온 뒤에만 최신 Snail 클라이언트를 실행합니다.

순서:

1. server list
2. 기존 계정 로그인
3. 기존 캐릭터 목록
4. 기존 캐릭터 선택
5. scene 진입

첫 시도가 scene에 들어가거나 중간에 실패하면 Stage60 창으로 돌아와 Enter를 누릅니다.

런처는 이 시점의 서버 로그, 포트 상태, JSON hash를 저장합니다.

## 재접속

첫 시도가 **scene까지 성공했을 때만** 클라이언트를 정상 종료/접속 해제한 뒤 같은 계정/캐릭터로 한 번 더 접속해 scene까지 들어갑니다.

첫 시도가 실패했다면 반복 클릭하지 말고 두 번째 프롬프트에서는 바로 Enter를 눌러 결과를 마무리합니다.

## 결과

런처가 만든:

`STAGE60_LIVE_RESULT_<timestamp>.zip`

하나만 보내면 됩니다.

결과 ZIP에는 `data\roles.json` 원문이나 MySQL DSN을 넣지 않습니다. JSON 데이터는 SHA256 변화만 기록합니다.

서버 코드가 이미 남기는 실제 근거를 기준으로 다음 지점을 분리합니다.

- list request captured
- login opcode `0x02` decoded
- `ServerPlayerRoles(opcode=0x04, role_exists=true)` sent
- role chosen / scene init (`waiting for ClientReady`)
- ClientReady/activity evidence
- reconnect에서 동일 흐름 재현 여부

## PASS 기준

다음이 모두 확인되기 전에는 Stage60 LIVE LOGIN PASS가 아닙니다.

- Stage60 Windows 서버 boot
- 4000 list service
- 19061 game connection
- account login
- 필요한 경우 Stage59 JSON identity adoption
- role list
- 기존 캐릭터 선택
- scene 진입
- 재접속 후 account/role mapping 유지

`STAGE60_READY=YES`는 로컬 서비스 포트가 준비됐다는 뜻일 뿐 LIVE 로그인 성공을 뜻하지 않습니다.
