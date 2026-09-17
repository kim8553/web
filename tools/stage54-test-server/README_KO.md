# Stage54 Windows 테스트 서버 — 9yin-go-server1 기준

이 패키지는 사용자 제공 `9yin-go-server1.rar`에서 이어진 현재 GitHub Go 소스(`server/cmd/protocol-probe`)를 Windows x64로 빌드한 테스트 서버다.

## 기준

- 서버 구현 기반: `9yin-go-server1.rar` → 현재 GitHub 브랜치의 진화된 Go 소스.
- 최신 Snail 클라이언트/DLL/Lua/resources는 프로토콜·동작 권위 자료다.
- `D:\9yin_server`, V37/V45/V46/JYZJ 등 과거 별도 서버 폴더를 실행 기반으로 가정하지 않는다.
- 이 ZIP에는 저작권/개인 런타임 리소스를 복제하지 않는다. 원본/현재 `9yin-go-server1` 런타임의 resources를 사용한다.

## Stage54의 중요한 수정

Stage50에서는 일반 NPC 구매 0x46의 안전 저장을 MySQL에만 연결해 JSON 모드 구매를 차단했다. Stage54는 `9yin-go-server1`의 기본 JSON 저장에서도 **가방 + 재화 저장을 하나의 잠금 구간에서 처리하고, 두 번째 파일 저장 실패 시 첫 번째 가방 파일을 되돌리는 경로**를 추가했다.

따라서 `NINEYIN_MYSQL_DSN`이 없는 원본형 실행 환경에서도 일반 구매 테스트 자체는 가능하다. 단, JSON 두 파일 방식은 전원 차단/OS 크래시까지 보장하는 DB 트랜잭션과 동일한 내구성이라고 주장하지 않는다.

## 준비

1. 사용자 PC에서 `9yin-go-server1.rar`를 풀어 둔 **현재 서버 루트**를 사용한다.
2. 그 루트에 `go.mod`와 `resources\modern\share\skill\skill_new.ini`가 실제로 있어야 한다.
3. 기존 실행 파일은 덮어쓰지 않는다. 이 Stage54 ZIP은 별도 폴더에 푼다.

## 실행

PowerShell에서 Stage54 ZIP을 푼 폴더로 이동한 뒤:

```powershell
powershell -ExecutionPolicy Bypass -File .\run-stage54-test-server.ps1 -ServerRoot "C:\실제경로\9yin-go-server1"
```

`C:\실제경로\9yin-go-server1` 부분만 사용자 PC의 실제 압축 해제 경로로 바꾼다. 특정 드라이브나 과거 서버 폴더를 전제로 하지 않는다.

- `NINEYIN_MYSQL_DSN`이 설정되어 있으면 현재 MySQL 저장 경로를 사용한다.
- 설정되어 있지 않으면 JSON 저장 모드로 실행한다.
- 어느 경우에도 이 런처는 MySQL 설치 위치나 비밀번호를 추측하지 않는다.

## 이번 테스트 순서

서버가 정상 부팅되고 클라이언트가 접속된 뒤:

1. 일반 NPC 상점을 연다.
2. 아이템 **1개**를 구매한다.
3. 가방에 생기는지와 재화가 차감되는지 확인한다.
4. 게임을 완전히 종료한다.
5. 다시 접속하여 아이템/재화가 유지되는지 확인한다.

아직 일반 판매 0x47은 구현되지 않았다. GM 아이템 지급과 교환 0x4F도 계속 보류다.

서버가 부팅되지 않으면 콘솔/`artifacts\local\logs\protocol-probe-live.log`를 제공하면 된다. JSON 모드의 경우 `data\bag_items.json`, `data\currency.json` 전체를 보내지 말고 오류 로그부터 제공한다.
