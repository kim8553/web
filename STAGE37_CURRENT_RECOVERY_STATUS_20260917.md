# Stage37 current recovery status — 2026-09-17

## Authority

- Working branch: `stage37-assistant-handoff-20260917`.
- Status immediately before this checkpoint file: `9aedbabc4bd11f45150c5457829196937d02887e`.
- Preserve the cumulative `9yin-go-server1.rar` lineage and the current `server/` recovery work. Do not substitute legacy JYZJ/V37/V46 semantics for current-client evidence.

## Exchange binding evidence closed in this continuation

- Exact-current exchange Lua continues to treat `BindStatus`, `ExchangeBind`, and `ShowBind` as separate values in the 11-field exchange configuration string.
- Exact-current localized resource evidence establishes the one-way material rule: if an exchange actually consumes at least one bound material, the produced result is bound.
- The same resource evidence establishes bound-material-first consumption and first-result binding preview semantics for multi-result exchange UI.
- Existing reconstructed `exchangeplan.PlanBatch` already matches that evidence: it consumes bound stacks before unbound stacks and records per-result `MaterialDerivedBound`.
- Staging `internal/exchangebinding` now exposes `FromMaterialDerivedBound(bool)`: `true` becomes authoritative `Bound`; `false` deliberately remains unknown. This prevents an unsupported converse from turning all-unbound material consumption into an assumed unbound final result.
- Focused Go unit test for the staging decision package: PASS locally. This is a unit-test result, not GitHub CI, LIVE, or E2E proof.

## Still unresolved — keep fail-closed

- The final persisted result `BindStatus` when no bound material was consumed is **unknown**. In particular, the precedence/role of authored `ExchangeItem.ini BindStatus` in the persisted output has not been proven.
- Exact server-authoritative derivation of `ShowBind` is **unknown**. Current Lua proves presentation behavior only.
- Therefore no `BindStatus OR ExchangeBind`, constant `ShowBind`, or equivalent guessed formula is authorized.
- Actual exchange mutation, inventory/currency commit, DB persistence, replication/error behavior, and relog verification remain gated until the missing semantics are proven.
- LIVE `0x40 -> S2C 557 -> exchange form` and end-to-end `0x4f` purchase/relog tests remain **NOT RUN** for this checkpoint.

## Commits in this continuation

- `79d37247f6f64091c5eb745446954ff340629d8e`: document exact-current material-derived binding boundary.
- `4d24ed00c6282cddcb6bc5584e04afba9853190d`: add fail-closed staging helper for the proven one-way rule.
- `9aedbabc4bd11f45150c5457829196937d02887e`: add unit tests proving `true -> Bound`, `false -> Unknown`.

## Next evidence target

Recover source-backed or exact-binary/resource-backed evidence for the relationship between authored `ExchangeItem.ini BindStatus`, material-derived `ExchangeBind`, and the final persisted item `BindStatus`. Until that relationship is proven, the production exchange mutation gate remains closed. `ShowBind` is a separate unresolved UI-field derivation and must not be inferred from either binding value.
