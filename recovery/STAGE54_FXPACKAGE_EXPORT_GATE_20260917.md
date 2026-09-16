# Stage54 — 현재 클라이언트 FxPackage.dll 진입점/비확정 경계 (2026-09-17 KST)

## 권위·보존
- 시작 브랜치 `kim8553/web:stage37-current-recovery-20260916`, 이전 HEAD `1b15095e1581f072149074bc16483b6f065a5333`. 원본 기준 `9yin-go-server1.rar`; 최신 `server/` 누적 수정 보존.
- 사용자 동일 BIN64 압축본에서 기존 검증한 `fxgame.exe` SHA-256 `c7f0366026452d525dc0c74b8499aac30fdb69189591420978e12c7094b5fde3` 및 `fxpackage.dll` SHA-256 `ac63e01378e5f46cd11c0f1f840f86bc32594545d7f13c807e81aadace5f1b47`를 로컬 읽기 전용 검사. 게임/EXE/DLL **실행하지 않음**, 공개 GitHub에 원본/복호화 키/추출 데이터 미업로드.

## 새로 확인한 실제 PE 및 기계어
- `FxPackage.dll`의 PE export directory에서 **이름·ordinal·함수 RVA 8쌍**을 직접 해석: `FxModule_GetEntCreator` `0x1c8a`; `FxModule_GetFuncCreator` `0x15d7`; `FxModule_GetIntCreator` `0x1aeb`; `FxModule_GetLogicCreator` `0x139d`; `FxModule_GetSpace` `0x1582`; `FxModule_GetType` `0x128a`; `FxModule_GetVersion` `0x17d5`; `FxModule_Init` `0x1389`. 모두 원시 섹션 이름이 8개 공백인 RVA `0x1000` 섹션에 매핑. DLL PE `AddressOfEntryPoint=0x6bd058`은 `.boot` 안(원시 파일 offset `0x3b258`). 첫 코드 섹션 앞 131,072바이트 Shannon entropy `7.9747 bits/byte` 관찰. **높은 엔트로피/보호 섹션/역어셈블 이상만으로 암호화 규격, 런타임 함수 몸통 또는 언팩 성공을 단정하지 않음.**
- 동일 EXE의 `FxPackage.dll;` 포함 세미콜론 구분 모듈 문자열은 존재하나 **그 문자열만으로 로드 완료 또는 해당 DLL이 특정 인덱스를 처리함을 입증하지 않음**.
- 실제 `fxgame.exe` `.text`: `0x14001d807`은 `PackFileSys\0`의 주소 `0x140065f60`을 `rdx`로 전달하며, `0x14001d816`은 `0x14005cc30`을 호출한다. 피호출자의 `0x14005cc80`–`0x14005ccae`에서 ASCII 대문자를 소문자로 바꾸고 문자를 비교하며 불일치 시 차이를 반환하는 **대소문자 무시 문자열 비교** 경로를 기계어 바이트 13개 위치에서 검증했다. 이는 `PCK0` 디코더의 증거가 아님. 해당 문자열 비교 호출 이외의 호출자/런타임 동작까지 입증하지 않음.
- 이전 Stage47 `fxres.exe`의 보호 영역 호출 역할 미상, Stage52 세 패키지의 원시 인덱스 비호환 및 Stage51 Lua 바이트코드 복원 결과 그대로 유지. **신규 인덱스 해독 0건, 원본 인덱스 경로 복원 0건, 전체 언팩 미완료**.

## 도구·테스트 경계
- `tools/stage54_fxpackage_export_gate.py`: EXE/DLL의 정확한 SHA-256 불일치 거부, PE32+ 구조와 범위 검증, 8 export RVA·진입 섹션·13 명령어 위치 대조, 식별 문자열 검사; JSON은 개인 바이너리 바이트나 키를 출력하지 않는다.
- 로컬 `python -m py_compile`: PASS; `--self-test`: 변형/잘못된 PE 3개와 틀린 SHA 1개 거부 PASS; 실제 동일 원본 EXE/DLL 감사 PASS(8 exports, 13 byte-site checks); 함수 진입 코드의 1바이트 변경 SHA 거부 PASS. 로컬 메타데이터 `/mnt/data/STAGE54_FXPACKAGE_BOUNDARY_PRIVATE_20260917.json` SHA-256 `397031d69fc0874d21438e845b19c489b1382b1444ddcc938fa32ec62fa287c3`.
- 게임 실행 중 로드 상태, 실제 보호 풀린 메모리 함수, 인덱스 버퍼 변환 호출은 **검증하지 못함**. 신규 Go build/race/vet 또는 아이템/상점 LIVE/E2E **실행하지 않음**. `PurchaseReady=false` 유지, `server/` 변경 없음.

## 실제 남은 경계
현재 사용자 Windows 게임 프로세스의 합법적인 런타임 진단 또는 동판본의 검증된 비보호 코드가 있어야 `FxPackage.dll` 실제 함수 몸통과 인덱스 버퍼 읽기/변환을 확인할 수 있다. 그 전에는 `FxPackage.dll` exports, `PackFileSys` 비교, 기존 `fxres.exe` 보호 호출을 인덱스 복호화 함수라고 이름 붙이거나 임의 키/알고리즘으로 언팩을 시도하지 않는다. 일반 구매/스킬/퀘스트 실게임 성공과 별개다.
