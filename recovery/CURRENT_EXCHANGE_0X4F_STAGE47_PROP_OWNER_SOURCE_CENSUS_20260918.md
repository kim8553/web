# Stage47 — current 0x4F Prop owner census (2026-09-18)

## Lineage and scope

Continue Stage46 `4dbfa66a56e4d4095803fbbf8ce0d53e5eb3b280` on `kim8553/web` branch `stage37-assistant-handoff-20260917`. Stage46 identified **37** referenced Prop identifiers in authenticated current mode3 exchange listings; `CapitalType1` and `CapitalType2` already have limited client-facing Go actor mappings, leaving **35** unresolved. This stage inspects the **actual current GitHub checkout**, not the original older RAR as a substitute for latest server code. No private client bytes, decoded Lua, packages, DLL/EXE or key are published.

## Executed source census

- Added [`tools/stage47_prop_owner_audit/audit.py`](https://github.com/kim8553/web/blob/58b525b7812e2e7b26c54ed32428013f2196fe57/tools/stage47_prop_owner_audit/audit.py) and [its checkout-based CI workflow](https://github.com/kim8553/web/blob/58b525b7812e2e7b26c54ed32428013f2196fe57/.github/workflows/jiuyin-stage47-prop-owner-census.yml).
- [GitHub Actions run 35239112053](https://github.com/kim8553/web/actions/runs/35239112053) checked out commit `58b525b7812e2e7b26c54ed32428013f2196fe57`. Job `current-server-census` **completed SUCCESS**, including `python3 -m py_compile`, exact-name scan, artifact upload, and scope disclosure. Scanner covered **167 Git-tracked `server/**/*.go` and `server/**/*.sql` files**, matching case-sensitive complete tokens and reporting locations only, not source lines.
- **33/35 names: zero literal whole-token hits in that scanned scope.** `SchoolContribute`: three hits, `WGJobSkillPoint`: one hit. **All four hits are exclusively in [an exchange pair-delimiter grammar test](https://github.com/kim8553/web/blob/58b525b7812e2e7b26c54ed32428013f2196fe57/server/cmd/protocol-probe/latest_client_shop_exchange_trailing_delimiter_test.go#L34-L55)**, not a production owner, debit, load/save, or replication handler.
- Positive controls: `CapitalType1` produced 20 scanned matches and `CapitalType2` produced 12, including [actual actor currency property frames](https://github.com/kim8553/web/blob/58b525b7812e2e7b26c54ed32428013f2196fe57/server/cmd/protocol-probe/zz_recovered_overlay.go#L4557-L4593). Thus the scanner does detect known production strings, though their hits alone still do not prove a valid exchange purchase.
- [Run artifact: 35-name exact source census with per-name counts and path:line matches](https://github.com/kim8553/web/actions/runs/35239112053/artifacts/10503814331). This is source discovery, **not** complete property-owner verification. Local pre-publication checks: Python syntax PASS and a synthetic Git-tracked file fixture verified exact positives, negative control and line tracking.

## Persistence cross-check and negative-finding limits

- Current [`server/internal/role/model.go`](https://github.com/kim8553/web/blob/58b525b7812e2e7b26c54ed32428013f2196fe57/server/internal/role/model.go) models Name, Appearance, Scene, Position; its generic Role repository is not a demonstrated 35-property exchange ledger.
- The inspected [`server/migrations` tree](https://github.com/kim8553/web/tree/58b525b7812e2e7b26c54ed32428013f2196fe57/server/migrations) contains `0001_normalize_role_repository.sql` and the runner/tests. [That migration](https://github.com/kim8553/web/blob/58b525b7812e2e7b26c54ed32428013f2196fe57/server/migrations/0001_normalize_role_repository.sql) handles account, role, appearance and location normalization, not a proven 35-Prop charge/award schema. These are **only findings for the inspected repository tree**, not proof about external databases or unseen official-server functionality.
- Exact text zero hits do **not** exclude dynamic property indices, aliases, generated or untracked files, separate services, private client resources, or server behavior not represented in this checkout. Never fabricate a per-role balance column, re-purpose an unrelated `playerProgress` field, or map arbitrary names to silver.
- Stage45 separately found bag JSON and currency JSON writes plus distinct MySQL bag/currency save transactions; Stage46 demonstrated Lua Item/Prop/Bind/VIP arithmetic is **UI display only**. Even `CapitalType1/2` cannot become spend-enabled without proven cost semantics, atomic Item+Prop+reward commit, bag capacity and binding, rollback, duplicate-request handling and durable client sync.

## Result and one next priority

**Unresolved end-to-end property owners: 35/35. Verified 35-name production exact-token owner matches in this bounded scan: 0/35. Actual 0x4F debit/grant: FAIL-CLOSED; no production handler, DB schema or client package changed.**

**Next single priority:** obtain the latest client indexed-property dictionary/native ordinal evidence for the 35 names and compare to real current Go `clientdata` property registry. For any exact match, follow its player state, durable per-role load/save and post-commit replication to identify a semantics-proven subset. Do not infer spendable balances from labels or turn purchases on without complete rollback/replay/atomicity evidence.

Validation boundary: GitHub checkout scan/CI **PASS**; full Go and Windows build, server boot, SQL transaction concurrency/rollback, client live buy, role persistence/reconnect **NOT RUN**.
