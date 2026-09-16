# Stage37 Current Recovery — 207-file Production Build PASS

Authority branch: `stage37-current-recovery-20260916`

Verified GitHub Actions run: `35040738015`
Head commit: `3e5574c3df536dd2881176b5d4f6d271c0d7ba7d`
Artifact ID: `10424214915`
Artifact digest: `sha256:42fd346ab920a3df0b32f6b38b487de0c02cb88976891260add9f3cebf851771`

## Source integrity

- source files: `207`
- generated manifest SHA256: `e3476fd61908982f258c9420cd134b6516641a3367b895507932220e68b37e18`

## Gates

All of the following exited `0`:

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

`protocol_runtime=125` means **NOT RUN** because the exact-current `resources/modern/share/skill/skill_new.ini` corpus is absent. No legacy or synthetic resource substitute was used.

## Windows candidate

- file type: `PE32+ executable (console) x86-64, for MS Windows, 15 sections`
- SHA256: `f929223e953e74af9329dbd602f7dc613655a9d015610214cc0ee97836985665`

## Safety boundary

The final exchange result binding precedence / current server-side `ShowBind` rule is still unresolved. `internal/exchangebinding.Decision` remains zero-value unknown and `Require()` fails closed until an explicit authoritative `Bound()` or `Unbound()` decision exists.

C2S `0x4F` actual inventory mutation remains OFF. LIVE/E2E remains NOT RUN.
