# 로그인 성공 기준 ZIP과 NPC 구매 사전 점검 ZIP 비교 (2026-09-19)

## 확인한 입력 및 경계
- 사용자 확인: `JIUYIN_STAGE37_LEGACY_RICH_BAG_AB_TEST_20260914(1).zip`을 사용한 서버에서 로그인·맵 진입 성공. 이는 사용자 경험 사실이며, 현재 샌드박스에서 Windows/클라이언트 LIVE를 재실행한 결과는 아니다.
- 원본 ZIP SHA-256: `88ab1992d4dfcedb7ecc03304dc7906da279eaf5d70565c71c0d7bdbf12fc0a5`; 17개 ZIP 엔트리, 16개 파일 해시 매니페스트 일치, ZIP CRC 오류 없음.
- 사전 점검 ZIP: `JIUYIN_NPC_PURCHASE_PRECHECK_7d2e6ab_20260919.zip`; 소스 기준 GitHub `7d2e6ab2f39679c107f56117c75718b0b30e7b76`. 이것은 구매 LIVE 완료 패키지가 아니다.

## 직접 확인된 차이
| 검사 대상 | 로그인 기준 ZIP | 새 사전 점검 ZIP |
| --- | --- | --- |
| Windows EXE SHA-256 | `06ea9364d306127868ed10c3ab19dac16ca9e4fc3f5091b59e50d10daeab7995` | `405775f2b8dce6f5945e84ee97ada9bc058be696b2da851bd6dd066205baaae0` |
| Go 버전 (EXE build info) | `go1.26.5` | `go1.23.2` |
| Go VCS revision (EXE build info) | `ddbbbcc082fdea2e4304fabad1dfaeea8ca7eb63` (`vcs.modified=true`) | `53e6dee3f671c81a65fcb47662e0f5e837bedd74` (`vcs.modified=true`); 신규 ZIP에 기록된 별도 GitHub 소스 기준 `7d2e6ab...`과 동일한 VCS 문자열이 아니므로 추정으로 합치지 않는다. |
| `skill_new.ini` SHA-256 | `7d644fca3564515dd305a562e06c8cb1487fc5e68b652910e9f77d0a2c08c653` | `8028418122d024ca3467e675939110a0ec7c4fc686f1a285c2df8af4f60e3` |
| `skill_new.ini` 길이/섹션 수 | 1,687,076바이트/17,562 섹션 | 1,677,777바이트/17,466 섹션 |
| 실행 범위 | 기존 `D:\JIUYIN_STAGE37_FULL_SERVER_20260913\JIUYIN_STAGE37_FULL_SERVER_20260913` 작업 디렉터리에 의존하는 `RUN_COMPAT_AB_ONLY.bat`와 EXE 포함. 전체 기존 서버 폴더나 DB 덤프는 ZIP에 없음. | 분리된 `resources/`와 EXE, 읽기 전용 `CHECK_PACKAGE_ONLY.bat`; 게임 서버 실행 BAT나 MySQL 접속 정보 없음. |

새 사전 점검 `skill_new.ini`는 로그인 기준 `skill_new.ini`의 정확한 **바이트 접두부**이며, 로그인 기준에는 그 뒤에 9,299바이트와 96개 추가 섹션이 있다. 이 차이가 실제 로그인·구매에 미치는 영향은 확인되지 않았으며 새 파일로 교체할 근거도 없다.

로그인 기준 `RUN_COMPAT_AB_ONLY.bat`는 EXE 해시를 검사하고 게임/GM 포트 19061/19062 및 lister 4000을 확인한다. 단, 기존 서버 폴더의 `resources\modern\share\skill\skill_new.ini`가 다르면 백업 후 사용자 확인 없이 해당 파일을 덮어쓸 수 있다. 이 비교에서는 BAT/EXE를 실행하지 않았고 실제 DB·기존 서버 파일을 수정하지 않았다.

## 다음 안전한 작업
1. 사용자 확인된 성공 EXE와 기존 서버 폴더 및 원본 `skill_new.ini`를 보존한다. 새 리소스를 정상 서버에 합쳐 넣지 않는다.
2. 현재 서버 소스의 MySQL 구매 저장 요건을 기준으로 기존 DB의 테이블/PK/열/마이그레이션 상태를 **읽기 전용**으로 확인한다. 스키마 변경·DB 초기화·자동 마이그레이션은 하지 않는다.
3. 호환성 검증 후 별도 테스트 환경과 명시적으로 선택한 DB에서 신규 EXE의 로그인→NPC 상점 열기→구매→가방/재화 반영→재접속을 검증한다. 이전 성공 ZIP으로 구매 성공이 증명된 것은 아니다.

**검증 범위:** ZIP 원본 해시/내부 매니페스트/CRC, EXE build info, 배포 스크립트 정적 검사, 양쪽 `skill_new.ini` 바이트 대조. 실제 Windows 게임 LIVE/E2E와 MySQL DB 적합성은 여전히 미검증.
