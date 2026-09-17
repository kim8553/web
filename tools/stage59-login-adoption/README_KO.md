# Stage59 JSON 로그인 identity 재바인딩 LIVE 테스트

이 패키지는 `9yin-go-server1.rar` -> 현재 `kim8553/web` Go 서버 계보를 사용합니다.

## 이번 단계의 범위

Stage58 LIVE에서 최신 클라이언트는 4000 서버목록과 19061 게임 포트에 정상 연결했고 로그인 `0x02`도 정상 해석됐지만, 현재 account identity가 원본 JSON 저장소의 과거 `acct:v1:` identity와 달라 `51001 role: not found`로 종료됐습니다.

Stage59는 **native JSON 모드에서만**, 현재 로그인 키를 찾지 못했을 때 다음 조건을 모두 만족하는 과거 계정이 정확히 하나인 경우에만 identity를 재바인딩합니다.

- `acct:v1:` 형식
- active 계정
- 기존 캐릭터를 소유
- 아직 password verifier가 없음
- `local_default`가 아님

후보가 0개 또는 여러 개면 임의 선택하지 않고 실패합니다. MySQL 로그인 경로는 이 migration을 사용하지 않습니다.

## 중요한 데이터 변경

첫 성공 로그인은 `data\roles.json`의 계정 identity와 password verifier를 갱신할 수 있습니다. 런처는 JSON 모드에서 서버를 시작하기 전에 자동으로:

`data\stage59-backups\roles.before-stage59-login.<timestamp>.json`

백업을 만들고 SHA256이 원본과 같은지 확인합니다. 백업 실패 시 서버를 시작하지 않습니다.

## 실행

1. Stage58 등 이전 테스트 서버 창을 모두 종료해 19061/19062 포트를 비웁니다.
2. 이 ZIP을 별도 폴더에 풉니다.
3. `RUN_STAGE59_LIVE.bat`을 실행합니다.
4. `ServerRoot:`에 실제 `9yin-go-server1` 루트를 입력합니다. 예: `E:\9yin-go-server1\9yin-go-server1`
5. `STAGE59_READY=YES`와 4000/19061/19062가 모두 `True`인지 확인합니다.
6. 최신 Snail 클라이언트에서 같은 계정으로 로그인합니다.

## LIVE 성공 기준

서버 로그에 `sent ServerPlayerRoles(opcode=0x04, role_exists=true)`가 나오고, 클라이언트가 `登录......` 화면을 넘어 캐릭터 선택/진입 단계로 진행해야 합니다.

`STAGE59_READY=YES`는 포트가 열렸다는 뜻일 뿐 로그인 성공을 의미하지 않습니다.

로그 기본 위치:

`<ServerRoot>\artifacts\local\logs\protocol-probe-live.log`

로그인에 실패하면 게임을 반복 클릭하지 말고 서버 콘솔 마지막 줄과 위 로그를 보존해 주세요.
