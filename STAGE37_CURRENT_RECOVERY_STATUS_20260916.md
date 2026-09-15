# Stage37 Current Recovery Status — 2026-09-16

Branch: `stage37-current-recovery-20260916`
Build commit: `a74b7895701a90b8dd9c6974d63f82eb4e55df57`
Actions run: `35011073031`

## Current-source identity

- Reconstructed current source files: 192
- Current delta SHA256: `50c20c9abaa91fb42cc885b3a8e81a5c8acee2a5bb09d80f53a1420772989c25`
- Current manifest SHA256: `43522bed8e5ff9cd1712bab8ed85d03f9ee483ace199b0c458c439234c70ec31`

## Verified gates

- exchangeplan: PASS
- migrations: PASS
- production Go build: PASS
- protocol-probe test compile: PASS
- protocol-probe race compile: PASS
- non-protocol tests: PASS
- non-protocol race tests: PASS
- go vet: PASS
- Windows amd64 build: PASS

Protocol runtime tests in GitHub Actions: `NOT RUN (125)` because the exact-current resource corpus is intentionally not substituted with legacy/synthetic data.
LIVE/E2E: NOT RUN.

## Windows build

- File: `stage37-current-windows-amd64.exe`
- SHA256: `659a122eb78bdc3754efc1018b0a0ee3cfb10b718c2722360a30bd8b9da258ca`
- PE: `PE32+ executable (console) x86-64, for MS Windows, 15 sections`
- Go: `go1.23.2`

## Remaining transaction boundary

- 0x4F read-only preflight is implemented.
- Actual exchange mutation remains OFF.
- `ExchangeBind` is proven as first-result bind preview.
- Exact server rule for `ShowBind` remains unresolved.
- Final persisted result BindStatus precedence remains unresolved.
- No material deduction / result creation / DB commit / View mutation is enabled until those semantics are closed.
