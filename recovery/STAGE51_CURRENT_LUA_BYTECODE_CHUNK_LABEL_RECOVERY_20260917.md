# Stage51 — 최신 Snail Lua 바이트코드 복원 및 NPC 상점 호출 근거 (2026-09-17 KST)

## 기준 및 작업 범위

- 작업 시작 HEAD: `f3ec810cfa46be58e5e113e2c1ae9bef30eaab76`, 브랜치 `stage37-current-recovery-20260916`. `9yin-go-server1.rar` 기반 현재 `server/` 수정 누적을 보존. 구형 `9yin-go-server.zip` 또는 JYZJ로 복귀하거나 덮어쓰지 않음.
- Stage48에서 비공개로 추출·검증한 `lua.package`의 익명 zlib 스트림 ZIP(원본 패키지 SHA-256 `283c8c245a3fb86af2a7e8c21c53b3de26c30432341dd590c37554adfce04e97`)과 업로드된 동일 `fxgame.exe` SHA-256 `c7f0366026452d525dc0c74b8499aac30fdb69189591420978e12c7094b5fde3`를 **로컬 읽기 전용**으로 사용. 실제 EXE를 실행하지 않음.
- 원본 `fxgame.exe`에서 알려진 16-byte 서명 앵커의 단일 존재와 523-byte 이어지는 NUL-종결 문자열을 직접 확인. 그 키의 SHA-256은 `d557b96634798bb398f81f77e2c800c0157efda43ec484b03e741cc2ceae9931`; 키 원문과 EXE·패키지·복원 데이터는 **공개 GitHub/Actions에 업로드하지 않음**. [공개 기존 Lua 필드 단위 XOR 알고리즘](https://github.com/russell662/JiuYinUnpackTool/blob/main/src/lua.rs)을 검토한 후 실제 바이트코드를 엄격한 구조 검증으로 확인.

## 실제 원본 개인 데이터 검증 결과

- Stage48 `lua` 익명 스트림 **2,295개** 중 Lua 5.1 header, 함수/상수/명령어 타입과 전체 스트림 길이 소비를 만족하는 **2,294개**를 필드별 XOR로 해독. 1개는 Lua header가 없는 비-Lua 데이터(`skin ...`로 시작); 버리거나 Lua로 위장하지 않고 manifest에 제외 이유·offset만 기록.
- 해독된 Lua chunk **2,294/2,294개**에서 컴파일러에 내장된 주 함수 `source` 문자열이 모두 `@G:\\Version\\01_Client\\lua\\... .lua` 형태임을 확인. 이 문자열은 **Lua 컴파일 시 포함된 chunkname**이며 PCK0 패키지의 인덱스 파일명/경로를 직접 복원한 증거가 아님. 안전한 상대경로만 출력 ZIP 항목명으로 사용하고 원래 같은 chunkname인 **9개는 중복을 보존**함. 표준 Lua 5.1 바이트코드 `.luac`이며 읽기 쉬운 원본 Lua 소스 코드가 아님.
- 확인된 컴파일러 경로 예시: `form_stage_main/form_shop/form_shop.lua`, `form_stage_main/form_shop/form_single_shop.lua`, `form_stage_main/form_shop/form_trade_buy.lua`, `form_stage_main/form_guildbuilding/form_guild_func_jiguan_shop.lua`. 비-Lua 스트림, 패키지 인덱스 해독, 원래 PCK0 엔트리의 파일명/오프셋 대응, 전 패키지 언팩은 여전히 별개 미완료 작업.
- 일반 상점 `form_shop.lua`의 하위 함수(바이트코드 소스 행 **724–766**, **769–815**)에서 `GETGLOBAL nx_execute` → `LOADK custom_sender` → `LOADK custom_buy_item` → `MOVE` 인수 4회 → `CALL B=7`을 각각 확인. 따라서 클라이언트 Lua에서 `nx_execute("custom_sender", "custom_buy_item", arg0, arg1, arg2, arg3)` 형태의 호출은 근거 있음. 인수의 서버 필드 의미, 네트워크 opcode, 구매 승인 여부·실제 돈/가방/DB 처리 및 선택된 길드 상점 등급은 이 사실만으로 **확정할 수 없음**. `form_single_shop.lua`에는 별개의 `custom_single_shop_msg`, `SingleShop_ClientMsg_BuyItem` 문자열도 있으므로 일반 NPC 상점과 단일 상점 경로를 혼동하지 않음.
- 2,294개 Lua chunk의 복원된 문자열 상수에서 **정확한 `Shop_GB_Yishiting` 문자열은 0건**. 이는 Lua 문자열 검색 범위에 한정된 결과로, 다른 패키지·바이너리·런타임에 선택 규칙이 없다는 뜻은 아님.

## 로컬 산출물 및 검증 구분

- **비공개 ZIP**: `STAGE51_LUA_CHUNK_BYTECODE_PRIVATE_20260917.zip`, **12,507,396 bytes**, SHA-256 `2f82a625a9ca216af84a83cbfed4851ff23ead5b1bc3c1bec550d4338349b207`; ZIP CRC PASS, **2,294개** 바이트코드+`manifest.json`+`README_KO.txt` = 2,296 항목. `manifest`에 원본 익명 스트림 오프셋/SHA 및 복원 바이트 SHA 기록. 대표 Lua 4개는 CRC·길이·복원 SHA를 별도 재확인했음. 이 파일은 **공개 GitHub/CI/Actions 아티팩트에 올리지 않음**.
- 복원 도구 [`tools/stage51_lua_named_chunk_recovery.py`](../tools/stage51_lua_named_chunk_recovery.py): 공개 소스에는 실제 사용자 키 바이트·게임 EXE·추출 데이터가 포함되지 않음. Git blob SHA `07877c929fd6c4082d6c715f06711ebc4602892b`는 로컬에서 문법 및 synthetic 자체테스트 통과한 동일 코드의 `git hash-object`와 일치. SHA가 다른 EXE나 Stage48 ZIP은 거부, 필드 경계/상수 태그/opcode/정확한 EOF/출력 경로 순회 검사, 기존 출력 덮어쓰기 거부.
- 로컬 **실제 사용자 입력** 복원·ZIP CRC·표본 SHA 검증 PASS; 로컬 **합성 입력** 정상/손상/잘못된 키·경로 순회 거부 테스트 PASS. 같은 공개 코드를 다시 실행한 경우 두 ZIP 컨테이너의 SHA-256은 생성 시각의 ZIP 메타데이터로 인해 달랐지만, **2,296개 항목의 이름과 각 항목의 압축 해제 후 SHA-256은 전부 동일**했음. ZIP 바이트 단위 결정적 재생산을 주장하지 않음.
- GitHub Actions [Stage51 synthetic-only run 35123171525](https://github.com/kim8553/web/actions/runs/35123171525): Python 문법·합성 self-test·개인 입력 부재 확인 전부 SUCCESS. Actions는 실제 게임 바이너리나 전체 Lua 언팩을 검증하지 않음.
- Go gameplay/GM 구매 코드 변경 없음, Go build/race/vet 신규 미실행, 게임 로그인·NPC 구매·가방 반영·DB 저장/재접속 LIVE/E2E **미실행**. 기존 `PurchaseReady=false` 그대로 유지.

## 다음 증거 경계

1. 확인된 `form_shop.lua` 호출의 4개 인수 값을 Lua 바이트코드의 호출자/변수 흐름 및 **현재 클라이언트 바이너리**의 `custom_buy_item` 수신/직렬화에 맞춰 추적하여 메시지와 데이터 구조를 입증한다. `form_single_shop` 경로와 분리한다.
2. 실제 `share.package`·`ini.package`의 인덱스 변환을 별도 기계어 근거로 해석해 원본 엔트리 경로를 복원한다. 이번 embedded Lua `chunkname`을 패키지 인덱스 복원으로 과장하지 않는다.
3. 실제 길드 상점 등급 선택과 구매·가방·DB 트랜잭션을 독립적으로 입증하기 전에는 임의 `_1`~`_5` 매핑이나 구매 승인 코드를 넣지 않는다.
