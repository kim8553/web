# Stage37 Current Recovery — Stage9 Typed-nil Persistence Gate PASS

Authority branch: `stage37-current-recovery-20260916`

Verified GitHub Actions run: `35042407891`
Head commit: `83a8ef5cd74a7ea7caa80e033d578f5beb948fb5`
Artifact ID: `10426171086`
Artifact ZIP digest: `sha256:b4e18ba3e109e4d0307b0995c377550346813b17fe3a6e70bacbbccccf260134`

## Verified gates from completed job log

All exited `0`:
- `exchangegate`
- `exchangegate_race`
- `exchangebinding`
- `exchangebinding_race`
- `exchangecommit`
- `exchangecommit_race`
- `planner`
- `migrations`
- `build`
- `protocol_compile`
- `protocol_race_compile`
- `nonprotocol`
- `race`
- `vet`
- `windows`

`protocol_runtime=125` = NOT RUN because the exact-current skill resource corpus is absent. No legacy/synthetic resource substitute was used.

## Persistence hardening

`currentShopExchangePersistenceReady` now requires both a non-zero role ID and a persistence target that is neither a nil interface nor a typed-nil reference hidden inside an interface. The previous weak inline `bagStore != nil` readiness expression is rejected by Stage9 static verification.

The real 0x4F handler remains dry-run only. Direct staging/commit calls remain statically forbidden in the handler. Actual inventory mutation remains OFF. LIVE/E2E remains NOT RUN.

The manifest and Windows EXE SHA files are in the uploaded artifact. Because the local container timed out and this project switches to GitHub-only fallback after a container timeout, those inner artifact hashes were not re-read locally in this checkpoint; the next workflow stage prints them directly into the GitHub Actions log.
