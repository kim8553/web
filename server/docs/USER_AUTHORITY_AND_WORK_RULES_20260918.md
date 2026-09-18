# 구음진경 서버 복원: 사용자 확정 기준 및 검증 경계 (2026-09-18)

> 이 문서는 사용자가 현재 대화에서 명시한 작업 지시와 직접 조회한 원본의 출처를 기록한다. **작업 규칙을 GitHub에 저장하는 것이 실제 게임 기능 구현이나 LIVE 성공을 의미하지 않는다.** 현재 소스 상태는 해당 브랜치의 실제 HEAD를 매 작업 시작 시 다시 확인한다.

## 1. 서버 출처 및 변경 대상

- 서버 **원본/복원 계보**: 사용자 제공 `9yin-go-server1.rar`. 이번 대화에 탑재된 동일 내용의 아카이브 로컬 표시명은 `9yin-go-server1_2.rar`이며 SHA-256은 `ddc2f6bc078660a40eedf077d43befd6402ed59478ccde987d44171c1d7aefa5`. 기존 `docs/CURRENT_AUTHORITY.md`에 `9yin-go-server1(3).rar`로 기록된 SHA-256과 같다. **파일 표시명의 차이보다 검증한 해시를 우선**한다. 이는 Go 서버의 구현 출발점이지 Snail 공식 서버 코드라는 뜻은 아니다.
- 수정·검증·인계할 코드 저장소: `kim8553/web`, 현재 복원 브랜치 `stage37-restored-20260918`. 사용자 데이터베이스, 본인 PC의 리소스 폴더 또는 정상 접속되는 실행 파일을 GitHub 소스와 혼동하거나 임의로 덮어쓰지 않는다.
- 사용자가 정상 로그인 및 맵 진입을 확인한 보존 기준: `JIUYIN_STAGE37_LEGACY_RICH_BAG_AB_TEST_20260914(1).zip` SHA-256 `88ab1992d4dfcedb7ecc03304dc7906da279eaf5d70565c71c0d7bdbf12fc0a5`. 이 ZIP에는 `stage37-live-server.exe`, 실행 BAT, `CURRENT_CLIENT_RESOURCE/skill_new.ini`, 설명/테스트 자료가 있다. **이 ZIP의 성공을 현재 GitHub HEAD의 Windows LIVE 성공으로 전가하지 않으며, 로그인과 맵 진입을 처음부터 재구현하지 않는다.**

## 2. 최신 클라이언트: 사용자 첨부 BIN64를 우선하여 파일 단위로 식별

- 사용자 제공 `bin64(2)(1)(1)_2.zip` SHA-256: `167aeacadd22d551c252e8808a010a00080682de8d2b3fb5f21420ff4d75a306`.
- ZIP 내 실제 파일 해시: `fxgame.exe` = `c7f0366026452d525dc0c74b8499aac30fdb69189591420978e12c7094b5fde3`; `fxgamelogic.dll` = `16cf49b652c39b63ba0767780cc39f32ea1e50bdc683da9d5e7b2697cf95c3e8`; `fxnet2.dll` = `0443d9f401fdcc6a92a391869c898780ffac93399913dc9a82c7cac8cf6c3fde`; `fxcore.dll` = `ce0da0f52a80db5033e177a59be868c44b0baeca4a71226b0ccb2aca76cae724`.
- **해시 불일치 명시:** 기존 `docs/CURRENT_AUTHORITY.md`에는 `FxGameLogic.dll` = `3bfa3832c04291d9d7a44f6bc13cceb609aad55ba90e2ad937cc77221089ef96`로 되어 있어 첨부 ZIP의 값과 다르다. 기존 파일 내용을 몰래 고쳐 쓰거나 두 DLL을 동일하다고 가정하지 않는다. 이번 사용자 지정 BIN64의 실제 바이트를 최신 분석 입력으로 쓰고, 이 불일치가 동작에 미치는 영향은 미검증으로 남긴다.

## 3. Google Drive의 현재 리소스 입력

- 연결된 Drive에서 확인한 [`res` 폴더](https://drive.google.com/drive/folders/1GPzJ4RrC_TfcJLnH4sW41PB8HkNesOgQ) ID: `1GPzJ4RrC_TfcJLnH4sW41PB8HkNesOgQ`.
- 폴더 직접 목록에서 확인한 데이터 아카이브: [`ini.package`](https://drive.google.com/file/d/1ae-QNOVdK6aIPusyLh0kABrfo6qsp4F7/view), [`share.package`](https://drive.google.com/file/d/1eoJ5ViSVdEY1xk8OKNcxmqHhcOmARQxh/view), [`lua.package`](https://drive.google.com/file/d/1jMfaMUsM5EhLo4hG-zo_3OamgpHVoMt6/view), [`lua64.package`](https://drive.google.com/file/d/1NWh8NSeN-mQ2RdeJi9yCkX40uuDBW7j6/view), [`map_path.package`](https://drive.google.com/file/d/1QdxdSQ_H1p8Vsq0YM1J76tmwL1Gdt1kh/view) 등. 조회 시 파일별 수정 일자가 동일하지 않으므로 이름만으로 동버전·완전성을 가정하지 않는다.
- 필요한 자료는 Drive `res`의 실제 파일을 **찾기 → 파일 ID/크기/해시 확인 → 포맷 검증 → 필요한 부분만 원본 보존·읽기 전용 추출 → 최신 BIN64와 대조**한다. 이전 INI 전수 감사는 따로 확보한 일부 리소스와 Go 로더에 한정되며 `res` 전체 package 분석 완료가 아니다. 임의의 더미 데이터나 구형 JYZJ/V37/V46 의미를 가져오지 않는다.
- 원본 RAR, 클라이언트 DLL/EXE, 대형 package 및 자격 증명은 공개 GitHub에 재업로드하지 않는다. 저장소에는 검증된 해시·출처·코드·테스트·비민감 분석 결과만 남긴다.

## 4. DB·로그인·기능 검증의 경계

- 사용자 확인 사실: **첨부 ZIP으로 로그인과 맵 진입이 된다.** 그 성공 경로와 기존 아이템/상점 기능을 보존한다. 최신 GitHub 빌드의 동일 결과는 사용자 Windows + MySQL + 최신 클라이언트 실게임 테스트 전에는 별개로 표시한다.
- MySQL `nineyin`의 과거 스키마/테이블스페이스 복구 작업은 존재하지만, **현재 사용자 PC의 스키마 및 마이그레이션 장부가 호환되는지는 확인되지 않았다.** 격리 테스트 성공을 사용자 DB 수정 완료라고 보고하지 않는다.
- DB 확인은 먼저 `SHOW CREATE TABLE`, `information_schema`, 마이그레이션 기록 등 **읽기 전용**으로 수행한다. 백업과 SQL 영향 검토, 명시적 승인 없이 스키마 재생성/마이그레이션을 실행하거나 `NINEYIN_ALLOW_SCHEMA_MIGRATIONS=YES`를 지시하지 않는다. 비밀번호·DSN을 커밋·보고하지 않는다.
- `go build`/CI 성공, 서버 포트/GM 응답, 데이터 로딩, 특정 단위 테스트, 로그인, 맵 진입, 아이템/스킬 E2E를 서로 다른 검증 단계로 기록한다. 추측한 필드·패킷·공식 스킬 효과를 구현하지 않는다.

## 5. 바로 이어갈 작업

1. 정상 접속되는 ZIP과 현재 `web` 소스의 이미 구현된 로그인/장면/가방 경로를 **보존**하고, 성공했던 DB 저장·재로그인 복원의 *테스트 근거*를 분리한다. 로그인/맵 진입을 또 처음부터 만드는 작업은 하지 않는다.
2. Drive `res`에서 실제 필요한 상점/아이템/스킬/동작 자료를 적극적으로 찾아 원본 출처·해시를 붙인다. 사용자 첨부 BIN64의 실행 코드·타입/호출 근거와 대조해 **확인된 누락 또는 결함만** 수정한다.
3. NPC 상점 구매·판매 → GM 지급/가방 동기화 → 아이템 사용·장착·재접속 저장 → 스킬 타게팅·시전·히트 순으로 기존 Go 코드를 점검한다. 이미 정상인 기능 재구현 금지. 각 수정은 재현 테스트, 실제 변경 커밋, 빌드/검증 결과를 함께 남기며 LIVE 미실행 시 미실행으로 표시한다.

추가 작업을 시작할 때 본 문서, `server/AGENTS.md`, 현재 브랜치 HEAD 및 `server/docs/CURRENT_AUTHORITY.md`의 **해시 차이**를 함께 읽고 근거를 검증할 것.