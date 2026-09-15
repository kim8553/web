# Stage37 current recovery — 199-file production build PASS

Authority branch: `stage37-current-recovery-20260916`

Verified workflow run: `35016914610` (run #25)
Verified branch commit: `f8046bce01ac35a4204bb364f26aa5f12d53e8ff`
Conclusion: `success`

## Source integrity

- Current source files: `199`
- Manifest SHA256: `a96d33fe6fdc07449af7f1f757ac65a658ca7bf87606db88bd703456a367bcf7`
- 195-file base manifest remains independently verified: `7632477cdc006efe7c95bbbdee58fd7ec8d79d9df26bdbf594b0a2e32f88c749`

## Verification gates

All returned exit code 0:

- planner
- migrations
- production build
- protocol compile
- protocol race compile
- non-protocol tests
- race tests
- go vet
- Windows amd64 build

`protocol_runtime=125`: NOT RUN because the exact-current resource corpus is absent from the GitHub Actions runner. No legacy resource substitution is allowed.

LIVE/E2E: NOT RUN.

## Windows artifact

- file: `stage37-current-windows-amd64.exe`
- SHA256: `e466397bced9b1ad5acd9aae80ca86904bf6f33eee546aaf2d13b3f4a47d6695`
- format: `PE32+ executable (console) x86-64, for MS Windows`
- Go: `go1.23.2`

## New verified staging layer

- Pure `PlanAtomicReplacement` validates a proven material plan against an immutable inventory snapshot.
- No committable deduction/add prefix is returned when the whole output cannot fit.
- Result stack merge is not guessed.
- Final result BindStatus remains an explicit caller input; the planner does not infer unresolved current-server binding precedence.
- Shop exchange staging wrapper is side-effect free and does not touch player state, DB, or client frames.

## Still intentionally blocked

C2S `0x4F` actual mutation remains OFF. Exact-current `ShowBind` server calculation and final persisted result `BindStatus` precedence are still unproven.
