# Stage37 current recovery — 201-file production build PASS

Authority branch: `stage37-current-recovery-20260916`

Verified workflow run: `35017481595` (run #26)
Verified branch commit: `47da6c49a369095b67fdeff0a6f10023b1491bd6`
Conclusion: `success`

## Source integrity

- Current source files: `201`
- Manifest SHA256: `8a9a2f763948f7fd2735969720a5dd7e81e8d1f55c67131d897fad106874d5b3`
- 199-file base manifest remains independently verified: `a96d33fe6fdc07449af7f1f757ac65a658ca7bf87606db88bd703456a367bcf7`

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

`protocol_runtime=125`: NOT RUN because exact-current resources are absent from the GitHub Actions runner. No legacy resource substitution is allowed.

LIVE/E2E: NOT RUN.

## Windows artifact

- file: `stage37-current-windows-amd64.exe`
- SHA256: `41c1da98885fc540407b0cb69deb01db9dde6d86580faab798343d3edadb017d`
- format: `PE32+ executable (console) x86-64, for MS Windows`
- Go: `go1.23.2`

## New verified stale-snapshot layer

- Staged deductions preserve expected container, slot, ConfigID, BindStatus and AmountBefore.
- Cloned-bag application revalidates all staged identities against a fresh snapshot.
- Input bag snapshot is never mutated.
- Output slot collisions and stale output MaxAmount are rejected.
- The helper still does not mutate player state, persistence, or client frames.

## Still intentionally blocked

C2S `0x4F` actual mutation remains OFF. Exact-current `ShowBind` server calculation and final persisted result `BindStatus` precedence remain unproven.
