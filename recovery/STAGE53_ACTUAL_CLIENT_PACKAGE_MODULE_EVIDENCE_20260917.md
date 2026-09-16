# Stage53 — 실제 최신 클라이언트 패키지 모듈 근거 경계 (2026-09-17 KST)

## 범위 및 출발점

- 시작 HEAD `f0a3ad7d4550e04e0f4b043b2fd441e496033bd3`, 브랜치 `stage37-current-recovery-20260916`. `9yin-go-server1.rar` 기반 누적 `server/`는 보존하며 구버전 서버로 복귀하지 않는다.
- 이전 Stage47의 `fxres.exe` 읽기 경로·Stage48–51 부분 데이터 및 Lua 복구·Stage52 공개 언팩 도구 비호환성 결과를 **다시 분석하지 않고**, 이전에 확인되지 않았던 **실제 게임 실행 파일과 패키지 DLL의 관계**만 한정 조사했다.
- 입력은 기존 사용자의 비공개 BIN64 ZIP(74개 멤버, ZIP CRC PASS) 속 같은 파일 세 개를 로컬에서 읽기 전용으로 검사. 공개 저장소에는 사용자 EXE/DLL·패키지·복호화 키·추출 리소스 바이트를 올리지 않았다. EXE/DLL을 실행하지 않았다.

## SHA-256으로 고정한 실물 관찰

| 이름 | 크기(bytes) | SHA-256 | 관찰 |
|---|---:|---|---|
| `fxgame.exe` | 8,772,368 | `c7f0366026452d525dc0c74b8499aac30fdb69189591420978e12c7094b5fde3` | NUL 종료 모듈 목록 문자열에서 `FxPackage.dll;` 항목이 정확히 1회 등장하며 전후 항목은 `FxModelAdv.dll;` 및 `FxGnugo.dll;`. 이 문자열 자체는 **런타임 로드·호출을 증명하지 않음**. |
| `fxpackage.dll` | 4,230,928 | `ac63e01378e5f46cd11c0f1f840f86bc32594545d7f13c807e81aadace5f1b47` | PE 섹션 12개 중 이름 `.edata`, `.themida`, `.boot` 존재. `.edata` 원시 바이트의 NUL 종료 `FxModule_*` 이름은 정확히 8개: `FxModule_GetEntCreator`, `FxModule_GetFuncCreator`, `FxModule_GetIntCreator`, `FxModule_GetLogicCreator`, `FxModule_GetSpace`, `FxModule_GetType`, `FxModule_GetVersion`, `FxModule_Init`. 이 이름들은 인덱스 해독 API를 명시하지 않으며 보호된 구현의 의미를 밝혀주지 않는다. |
| `fxres.exe` | 7,259,408 | `07ae76288148132995538488f12e2214fbecfdc0f18bdc2dd3189093e3e9fa9c` | 이전 Stage47의 별도 유틸리티 파일. `CPackData::LoadFromFile` 진단 문자열과 `PCK0` 비교 코드가 존재. 별도 실행 파일의 파서가 게임 내에서 그대로 사용되는지 **확인되지 않음**. |

## Stage47 보호 호출에 대한 엄격한 정정

- 실제 `fxres.exe` `.text` 기계어를 다시 확인: `0x14000dc3b: mov rdx,rdi` → `0x14000dc3e: mov rcx,rsi` → `0x14000dc41: call 0x14000d050`(인덱스 크기만큼 읽는 경로). 이후 `0x14000dc99: mov rcx,rsi` → `0x14000dc9c: call 0x1402070b6`(보호 섹션). 5개 위치의 명령어 바이트와 원본 SHA 모두 일치.
- `rdi`는 앞서 읽기 대상으로 전달된 데이터 버퍼, `rsi`는 열린 파일 관련 호출에서 이어지는 변수라는 문맥을 확인했지만, **보호 호출에는 버퍼 `rdi`가 명시적인 첫 인수 `rcx`로 전달되지 않는다**. 다른 인수·전역·간접 효과를 알 수 없으므로 그 호출을 **인덱스 복호화 함수 또는 단순 close 함수 어느 쪽으로도 확정하지 않는다**. 이전 기록의 `opaque_call_role_known=false`를 유지한다.
- 위 기계어는 **별도 `fxres.exe`**의 것이다. `fxgame.exe`나 `fxpackage.dll` 내부에 똑같은 처리 경로가 존재한다는 증거로 오용하지 않는다. `.themida` 섹션 명칭과 정적 보호 상태만으로 실제 패키지 포맷 변환·키·알고리즘을 추론하지 않는다.

## 결과·중지 경계

- 이번 신규 확인: `fxgame.exe` 모듈 목록의 `FxPackage.dll` 항목, 동일 BIN64의 실제 DLL, 해당 DLL의 공개 PE 메타데이터 및 Stage47 호출 인수 경계. **원본 패키지 인덱스 해독 0건, 원래 파일명 복원 0건, 전체 언팩 미완료**.
- 누적 Stage48–51에서 확보한 익명 zlib 및 Lua 2,294개 바이트코드 복구 결과는 그대로 유지. `server/` 소스 변경 0건, 신규 Go build/race/vet/LIVE/E2E 미실행, NPC 구매 `PurchaseReady=false` 유지.
- 다음 직접 증거는 **실제 `fxgame.exe`가 런타임에 호출하는 동판본 `FxPackage.dll`의 인덱스 해석 경로**(검증 가능한 기계어·정상 파일 목록 또는 실측 호출 추적)다. 모듈 목록의 문자열만으로 해당 경로가 확인됐다고 말하지 않는다. 보호된 정적 파일만으로 이 연결을 증명하지 못하면 즉시 `확실하지 않음`으로 기록하고, 기존 `fxres.exe`의 불명 호출을 반복해서 해독기라고 명명하지 않는다.
