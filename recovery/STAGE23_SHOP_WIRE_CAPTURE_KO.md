# Stage23: 최신 클라이언트 일반 NPC 상점 구매 패킷 증거 수집

**중요:** Stage22 진단 EXE는 기존 서버 실행 파일을 백업한 후 동일한 리소스/DB 설정에서만 사용합니다. Stage23은 서버 기능을 켜거나 구매 성공을 보장하지 않습니다. 미검증 0x46의 돈/가방 변경은 차단 상태입니다. 상점 구매 버튼을 눌렀을 때 거절되거나 아이템이 들어오지 않는 것은 이 진단 단계에서 예상할 수 있습니다.

1. Stage22 Windows EXE의 SHA256 `51925a63087d1a43ec62450e9e9fd4bc47c323e8429f2cb7e5d0d47c4a7ac1aa`을 확인합니다. 서버 로그가 **파일**로 남는 현재 실행 방식을 유지합니다. 이번 수집에는 인자 상세 값이 필요 없으므로 `NINEYIN_SHOP_WIRE_TRACE=1`은 설정하지 않아도 됩니다.
2. 게임에 접속해 NPC 상점 화면을 띄운 뒤, 구매 버튼을 누르지 않은 상태에서 서버 로그 파일 경로를 확인합니다. PowerShell에서 다음 명령의 `-LogPath` 부분만 현재 실제 서버 로그 경로로 바꿔 실행합니다.
   ```powershell
   powershell -NoProfile -ExecutionPolicy Bypass -File .\tools\stage37_capture_shop_wire.ps1 -Phase baseline -LogPath 'D:\9yin_server\logs\YOUR_ACTUAL_SERVER_LOG.log' -Seconds 15
   ```
3. 동일한 상점 화면에서 다음 명령을 실행하고, 실행 후 15초 안에 **일반 NPC 상품 구매 버튼을 한 번** 누릅니다. mode3 교환 버튼이나 GM 지급은 사용하지 않습니다.
   ```powershell
   powershell -NoProfile -ExecutionPolicy Bypass -File .\tools\stage37_capture_shop_wire.ps1 -Phase purchase -LogPath 'D:\9yin_server\logs\YOUR_ACTUAL_SERVER_LOG.log' -Seconds 15
   ```
4. 생성된 `shop_wire_evidence\baseline.log`, `shop_wire_evidence\purchase.log`만 비교합니다. 수집기는 `SHOP_WIRE_OBSERVE` 중 opcode/selector/값 수/값 타입만 저장하고, 원격 IP·계정·문자열·원본 패킷을 파일에 포함하지 않습니다. 구매 로그가 0건이면 자동으로 오류를 냅니다.
   ```powershell
   python .\tools\stage37_shop_wire_diff.py --baseline .\shop_wire_evidence\baseline.log --purchase .\shop_wire_evidence\purchase.log --output .\shop_wire_evidence\candidate_report.json
   ```

결과 JSON의 `candidates`는 **관측 증가 후보**이지 selector가 증명됐다는 의미가 아닙니다. `selector_promotion_allowed=false`, `handler_enable_allowed=false`가 유지됩니다. 구매 전/후 실제 최신 클라이언트 동작·payload 레이아웃·서버 흐름과 별도 대조하기 전에는 production handler를 켜지 않습니다. 서버 stdout만 콘솔에 표시하고 로그 파일이 없다면, 다른 경로를 추측해 입력하지 말고 실제 저장 경로를 먼저 확인합니다.
