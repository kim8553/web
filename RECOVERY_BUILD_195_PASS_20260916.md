# Stage37 current recovery — 195-file production build PASS

Authority branch: `stage37-current-recovery-20260916`

Verified workflow run: `35015805381` (run #22)
Verified branch commit: `24b71e5cce5594623d5d3fc62b58846ee037ba6a`
Conclusion: `success`

## Source integrity

- Reconstructed current source files: `195`
- Manifest SHA256: `7632477cdc006efe7c95bbbdee58fd7ec8d79d9df26bdbf594b0a2e32f88c749`
- Previous verified 192-file manifest remains the mandatory base: `43522bed8e5ff9cd1712bab8ed85d03f9ee483ace199b0c458c439234c70ec31`
- Post-build changes are applied only from 5 individually SHA-verified small patches plus 3 individually SHA-verified source templates.

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
- SHA256: `b5a2a1dc2a46608a4a28495034d0509f3713479875faa03d1f1495410fc33632`
- format: `PE32+ executable (console) x86-64, for MS Windows`
- Go: `go1.23.2`
- quicklz: `github.com/Hiroko103/go-quicklz v0.0.0-20190115215310-59904abc50d0`
- mysql: `github.com/go-sql-driver/mysql v1.9.3`
- x/text: `golang.org/x/text v0.28.0`

## New verified safety work in this 195-file tree

- `ARANGEITEM` no longer merges bound and unbound stacks with the same ConfigID.
- Read-only shop exchange preflight includes conservative inventory-capacity planning.
- Capacity calculation counts slots freed by the already-proven material deduction plan.
- It does not assume an output can merge into an existing stack.
- Unknown output static data or unsupported output container fails closed.

## Still intentionally blocked

C2S `0x4F` actual mutation remains OFF.

Reason: exact-current server rule for `ShowBind` and the final persisted result `BindStatus` precedence are not yet proven. `ExchangeBind` is treated only as the first-result material-derived bind preview. No material deduction, output creation, DB commit, or client mutation is enabled from this unresolved state.
