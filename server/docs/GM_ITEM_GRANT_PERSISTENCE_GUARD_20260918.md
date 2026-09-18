# GM 아이템 지급: 가방 저장 실패 시 거짓 성공 방지 (2026-09-18)

## 확인한 결함과 수정 범위

`cmd/protocol-probe/main.go`의 `give_item` 일반 아이템·장비 지급 양쪽에서 이전 코드는 `player.addBagItem` 후 `bagStore.Save`가 오류를 반환해도 로그만 남기고 클라이언트 가방 프레임 및 성공 메시지 전송을 계속했다. 이는 저장되지 않은 지급을 사용자에게 성공한 것처럼 표시할 수 있다.

`gm_grant_persist.go`의 `grantBagItemDurably`는 기존 `addBagItem`, `bagSnapshot`, `bagStore.Save`, `restoreBag`를 이용한다. 저장 오류가 반환되면 지급 직전 액터 가방 스냅샷으로 복구하고 오류를 반환한다. 두 `give_item` 분기는 각각 GM 실패 메시지를 기록하고 즉시 반환하므로 **해당 저장 실패 경로**에서는 성공 프레임·메시지를 보내지 않는다. 기존 로그인·맵 진입·아이템 정의·클라이언트 패킷 형식은 변경하지 않았다.

## 근거와 검증

- 수정 전 `main.go` Git blob: `42a20f4ba3b02cafa4185cf13b7057b6af036078`; 수정 후 Git blob: `a2e196ce6d13318ed4ed54714abb12ce83bd808d`.
- 신규 회귀 테스트 `gm_grant_persist_test.go`는 일반 아이템·장비의 저장 오류/스냅샷 복구/저장 정상화 후 재시도, 스토어·역할 ID 누락 시 무변경을 검사한다.
- 격리 리소스 fixture에서 GM·상점 관련 선택적 Go 테스트, `go vet` 및 Windows amd64 Go 빌드 통과. 전체 `go test ./...`는 불완전한 fixture의 상점·내공·씬/NPC 등 리소스 부재로 `cmd/protocol-probe` 14개 테스트가 실패했으며 전체 PASS로 보고하지 않는다.
- GitHub Actions 임시 정확한 소스 패치 전송 실행 `35339384710` 성공, 실제 `main.go` 수정 커밋 `261d012793e829461e814b14d0151efbffebbda7`; 임시 workflow/patch는 성공 후 삭제했다.

## 남은 검증 및 안전 경계

사용자 PC MySQL, 스키마, 가방 데이터, 정상 로그인 ZIP 및 클라이언트 리소스는 수정하지 않았다. 이 패치는 `Save`가 **오류를 반환한 경우** 액터 상태를 복구하는 조치이며, 저장 후 응답 오류의 모호성까지 해결하는 분산 트랜잭션 보장은 아니다. 저장 성공 뒤 네트워크 프레임 전송 오류 또한 별도 검증이 필요하다. 현재 사용자 DB의 컬럼 호환성은 `tools/check-shop-db-columns-readonly.sql`을 해당 DB에 실행하지 않았으므로 미확인이다. NPC 일반 상점의 판매 요청·정산 규칙은 현재 검증된 소스/최신 클라이언트 근거가 부족하여 구현하지 않았다. 최신 Snail 클라이언트의 게임 내 지급·가방 표시·재접속 E2E 역시 미실행이다.

서버 원본 계보는 `9yin-go-server1.rar`, 최신 클라이언트 권위는 사용자 제공 BIN64와 동일 버전의 Drive `res` 리소스로 유지한다. 근거 없이 판매 패킷·효과·DB 구조를 추측하지 않는다.
