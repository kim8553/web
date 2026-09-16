# Stage 27 — source-based observer

`cmd/protocol-probe/source_observer.go` adds opt-in, non-semantic live instrumentation.

Enable at runtime:

```powershell
$env:JIUYIN_SOURCE_OBSERVER = "1"
```

When disabled (default), it emits no observer logs.

Observed events:

- `OBS_RX_RAW` — successfully decoded application frame, after `ReadFrame` returns.
- `OBS_TX_RAW` — application frame immediately before the existing `WriteFrame` call.
- `OBS_CUSTOM_DECODE` / `OBS_CUSTOM_VALUE` — typed CustomSend values correlated with `rx_seq`.
- `OBS_C2S_211` — 211 summary, including V9 when present.
- `OBS_C2S_212` — 212 summary and flag value when structurally available; observe-only.

The observer does not mutate frame bytes, transport keys, game state, or handler return values.
The activity-envelope allowlist was extended to admit message 212 into the existing logging
block only; there is deliberately no 212 semantic switch case.

Verification currently completed in this sandbox:

- `gofmt`: PASS
- `git diff --check`: PASS
- Isolated Go tests for custom decoder + source observer: PASS
- Full `go test ./...`: NOT RUN TO PASS because this sandbox cannot retrieve three external Go
  modules (`go-sql-driver/mysql`, `x/text`, `go-sqlmock`).
- Windows amd64 full build of modified source: NOT YET VERIFIED.
- Live current-client E2E: NOT VERIFIED.
