# Stage37 Current Recovery — Stage15 Status

Date: 2026-09-16

## Authority

- Branch: `stage37-current-recovery-20260916`
- Stage15 workflow run: `35048282955`
- Stage15 workflow conclusion: `SUCCESS`
- Verification artifact: `10428196039`
- Artifact digest: `sha256:ee4aa1b4e07af4e6b1a9351062ca1f40b9e9ac902f27010ea4828b5be8371ffb`
- Canonical reconstructed production source: 212 files
- Canonical source manifest SHA256: `936a075a8f0baf477e253c789cf8e756436b576c67629b56901826afb04e1a44`

## Regular NPC shop wire evidence gate

Stage15 is read-only with respect to production gameplay behavior. It does not touch the deferred GM grant route and does not enable mode3 mutation.

The canonical server currently has exactly one non-test `handleShopBuyCustom` declaration at reconstructed `cmd/protocol-probe/zz_recovered_overlay.go` and its server-side candidate contract is:

- selector candidate: `0x46`
- payload candidate: `shopid:string, page:int32, pos:int32, amount:int32`

This is **not** promoted to an exact-current client contract. Stage15 found no exact-current client authority asset inside the canonical GitHub repository and therefore records:

- `current_client_selector_verified=0`
- `wire_gate_status=BLOCKED_NO_EXACT_CURRENT_CLIENT_AUTHORITY_IN_REPO`

Historical/legacy selectors must not be substituted for this missing proof.

## Existing ordinary-shop mutation topology

Stage15 confirms the existing handler currently performs all of the following:

- live currency mutation
- live bag mutation
- client frame write
- bag persistence
- currency persistence

It also confirms:

- client frames are written before both bag and currency persistence
- bag and currency persistence are separate in the handler

Therefore this handler is not yet accepted as the final atomic ordinary-shop transaction path. Production mutation ordering is intentionally unchanged until the exact-current purchase wire is proven.

## Verification matrix

- regular shop wire gate: PASS as a read-only audit (`0`)
- regular shop topology audit: PASS (`0`)
- binding constructor audit: PASS (`0`)
- planner UNIT: PASS (`0`)
- migrations UNIT: PASS (`0`)
- exchangebinding UNIT/RACE: PASS (`0` / `0`)
- exchangecommit UNIT/RACE: PASS (`0` / `0`)
- exchangeexecute UNIT/RACE: PASS (`0` / `0`)
- exchangegate UNIT/RACE: PASS (`0` / `0`)
- non-protocol test suite: PASS (`0`)
- full non-protocol RACE: PASS (`0`)
- vet: PASS (`0`)
- native build: PASS (`0`)
- protocol compile: PASS (`0`)
- protocol race compile: PASS (`0`)
- Windows amd64 build: PASS (`0`)
- protocol runtime: **NOT RUN** (`125`) because the exact-current resource corpus is absent; no legacy resource substitution is allowed
- LIVE/E2E: **NOT RUN**

## Windows build identity for run 35048282955

- format: PE32+ executable (console), x86-64, Windows
- Go: `go1.23.2`
- SHA256: `8ba8d52e15c6b41de0698cc8a11862fbef42b64f6bdbd1fe7ec1bbdc9dbcd5b9`

## Unresolved / next exact work

1. Prove the ordinary NPC shop purchase selector and payload from exact-current client evidence (`form_shop.lua` / sender path and, if necessary, `FxGameLogic.dll` CustomSend xrefs). Do not infer it from the server literal `0x46`.
2. Only after that proof, replace the current `handleShopBuyCustom` live-first sequence with a staged bag+currency persistence-first atomic transaction and add rollback/failure tests.
3. Keep actual mode3 `0x4F` mutation OFF until final persisted `BindStatus` precedence and the exact `ShowBind` server rule are proven.
4. GM grant remains deferred and must not be changed.
