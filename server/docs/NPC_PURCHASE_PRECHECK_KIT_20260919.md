# NPC 구매 실게임 테스트 준비: 격리 사전 점검 패키지 (2026-09-19)

## 정확한 출처와 결과

- 사용자 지정 원본 계보 `9yin-go-server1.rar`; 실제 작업 소스: `kim8553/web`, `stage37-restored-20260918`, **빌드 기준 커밋** `7d2e6ab2f39679c107f56117c75718b0b30e7b76`, **검증된 Git 트리** `b59b563f0a6d39c68a1dd8eabe19cc7f9f86a196`.
- Windows amd64 EXE: `stage37-purchase-test-7d2e6ab.exe`, SHA-256 `405775f2b8dce6f5945e84ee97ada9bc058be696b2da851bd6dd066205baaae0`. 현 작업 샌드박스에서 해당 트리로 빌드했고 `PE32+ x86-64` 형식을 확인했다.
- 사전 점검 ZIP: `JIUYIN_NPC_PURCHASE_PRECHECK_7d2e6ab_20260919.zip`, SHA-256 `f507e43eac7300e6b485b3eb154b12c30701d4237dea30cb8cb3efa3283278c3`, 크기 54,673,504바이트. 사용자에게 비공개 대화 파일로 전달한 것으로, **공개 GitHub에는 바이너리·리소스를 올리지 않는다**. ZIP 내부 파일 9,668개 SHA-256 매니페스트 모두 일치, ZIP CRC 검증 통과. ZIP에는 실행 BAT가 없으며 `CHECK_PACKAGE_ONLY.bat`는 리소스·EXE 점검만 한다.
- 동일 소스 트리에서 Go 1.23.2 / vendored 실제 의존성 및 외부 임시 `verify.mod`: `go test ./... -count=1` **PASS**, `go vet ./...` **PASS**, `GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -trimpath ./cmd/protocol-probe` **PASS**. 원본 `server/go.mod` 및 게임 동작 소스는 변경하지 않았다. 별도 Linux 빌드를 동일 리소스 후보로 실행하여 격리 JSON 모드 TCP 게임/GM 포트 응답과 GM HTTP 200 (10,535바이트) 확인. **JSON 모드 구매는 테스트하지 않았으며 구매 저장 코드가 JSON 모드를 의도적으로 거부한다.**

## 리소스 및 버전 경계

- 소스 스냅샷에 있던 `resources` 약 316 MiB에 연결된 사용자 Google Drive `playerweapon` INI 16개 (출처 폴더 `https://drive.google.com/drive/folders/1EQVXknAw8CStJ4skVkp7OsIYvJ3DYotq`), `stringname.idres` (SHA-256 `e6a4f1f5493e97903b0e306c83f10ca8c03afff43c4ef0acef5b7e216c183aaa`), `drop_table.json` (SHA-256 `1772db61dcfa37f907e0759d9ccb07ab865aaaa8cdd1109b38ff265d69d9975b`)를 **별도 패키지에만** 복사했다. 파일별 Drive ID·SHA-256은 ZIP의 `SOURCE_AND_RESOURCE_PROVENANCE.json`에 기록했다.
- `verify-runtime-resources.ps1`에 열거된 필수 파일 22개와 디렉터리 3개는 로컬 정적 확인을 통과했다. PowerShell 스크립트 자체는 Windows에서 실행하지 않았고, 이 검사로 최신 클라이언트 리소스와의 버전 일치를 증명하지 못한다. 예: 번들 `skill_new.ini` SHA-256은 `8028418122d024ca3467e675939110a0ec7c4fc686f1a285c2df8af4f60e3`.
- 과거 로그인·맵 진입 성공 ZIP `JIUYIN_STAGE37_LEGACY_RICH_BAG_AB_TEST_20260914(1).zip`은 이번 대화의 탑재 작업공간·연결 파일/Drive 검색에서 확보되지 않았으므로 새 EXE/리소스와의 동등성은 **미검증**.

## 구매 테스트 차단 조건과 다음 단계

- `shop_purchase_atomic.go:persistOrdinaryShopPurchase`는 가방·재화를 **동일 MySQL DB** 트랜잭션으로 저장하도록 요구하고 JSON 저장소를 거부한다. 따라서 JSON 모드에서 구매가 성공한다고 주장하거나 실제 구매 검증용으로 안내하지 않는다.
- 사용자의 MySQL `schema_migrations`·`role_bag_items`·`role_currency` 구조가 최신 실행 파일과 호환되는지 아직 확인하지 않았다. `NINEYIN_ALLOW_SCHEMA_MIGRATIONS=YES`를 설정하거나 사용자 DB를 자동 변경·복제·초기화하지 않는다. 먼저 **읽기 전용 메타데이터/마이그레이션 장부 검증**과 안전한 별도 DB 준비 경계를 확인한다.
- 최신 Snail BIN64의 실제 구매 요청과 가방/재화 표시 동작, Windows 실행, 로그인, 구매, 로그아웃/재접속 **LIVE/E2E=NOT RUN**. NPC 판매 패킷·정산은 여전히 미확정. GM 웹 지급, mode3 Exchange 보류 유지.
- **다음 작업 지점:** 과거 성공 ZIP의 실제 입력이 확보되면 실행환경 차이를 비교하고, 현재 MySQL에 대해 파괴적 작업 없는 호환성 감사 또는 승인된 별도 테스트 DB로 구매 E2E를 준비한다. 사전 점검 ZIP을 기존 정상 서버 폴더에 덮어쓰지 않는다.
