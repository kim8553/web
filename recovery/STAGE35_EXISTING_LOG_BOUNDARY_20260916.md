# Stage37 Stage35 — available-log evidence boundary (2026-09-16)

## Authority and ancestry

- Checked branch `stage37-current-recovery-20260916` at `389fa018f9136aac4e3f89445d3e8cc430c01172` before recording this report. GitHub compare established `e5210e853a81f458a899e98806544c83946f664f` (Stage34) as the merge base, with three Stage35 commits ahead and zero behind. This is **not** a source reset.
- The user's freshly supplied **`9yin-go-server1.rar`** was read from a separate private working copy; SHA-256 `ddc2f6bc078660a40eedf077d43befd6402ed59478ccde987d44171c1d7aefa5`, 42,688,644 bytes, RAR5, 10,498 entries and 112 `.go` entries. Its exact hash matches `recovery/STAGE28_RAR1_BASELINE_AUDIT_20260916.md`. Neither `9yin-go-server.zip` nor an older JYZJ/V37 source was substituted.
- Extracted `resources/modern/share/trade/shop.ini` only in the private workspace; SHA-256 `ceb7a3a7e17cbc118ce5a7d3d979dea6120c50cdf6a5823697c05a5b1b9cf5dd`, identical to the Stage33 audited corpus. Original RAR and its internal files were not modified or uploaded.

## Previously existing logs examined, without publishing their content

| Provenance | Date within log | Bytes | Exact Stage32/Stage31 shop diagnostics |
| --- | --- | ---: | --- |
| RAR `logs/protocol-probe-live.log` | 2026-07-26 | 473,110 | 0 |
| RAR `logs/startup-20260726-185352.out.log` | 2026-07-26 | 1,142 | 0 |
| RAR `logs/startup-20260726-185352.err.log` | 2026-07-26 | 192 | 0 |
| RAR other three `*.log` files | empty | 0 each | 0 |
| Connected Google Drive `protocol-probe-live.log` (a separate file) | 2026-08-25 | 953 | 0 |

For every RAR log, a read-only text count of the exact prefixes `NPC shop menu diagnostic `, `shop service selected `, `shop display catalog `, `shop display preflight rejected `, `shop display frames `, `shop display capacity diagnostic `, and `shop exchange view AB ` returned zero. The Drive log was actually opened: it contains eight catalog/registry/store initialization lines and no shop selection or frame evidence. Its filename is the same as a RAR log, but its size/date/content differ; **they must not be conflated**. Search for the named `LIVE_RESULT`/Stage34 diagnostic ZIPs and exact `9yin-go-server1.rar` in connected Drive returned no matching accessible file; absence from search does not establish that no such file exists. The exact RAR is already supplied in the conversation, so re-uploading it is unnecessary.

Only counts, dates, and approved file hashes are recorded here. No raw logs, IP addresses, client binaries, RAR/resource bytes, account records, database dumps or character data have been committed.

## What these observations do and do not establish

1. Original-lineage and `shop.ini` corpus match the already audited Stage28/33 data: **PASS (hash equivalence)**. This does not prove an exhaustive development-history chain.
2. Stage34 source is preserved in ancestry; its script calls Stage32's recursive patch/build chain, checks Stage29 known `born02` position repair and Stage31 preflight, and applies only the bounded `PageCount` patch. This is script/commit-chain inspection, **not a fresh local Windows build**.
3. Previously recorded Stage34 Actions run `35093960024` (source commit `f6e08e586c748bd1355f5a41618735716658e318`) passed with Windows artifact `stage34-current-windows-amd64.exe` SHA-256 `a60774041ea671a0feaa5efafca5693d37ded958503c8bf25f78cc743c6c1c5d`. See `recovery/STAGE34_CI_VERIFICATION_20260916.md`. The EXE and diagnostic ZIP were **not downloaded or re-hashed in this investigation**. Earlier isolated test, race, vet and build results are historical, not rerun here.
4. Proven **offline data blockers**, not asserted live diagnoses: exact `Shop_GB_Yishiting` section missing while numbered suffixed sections exist (no established mapping); among 1,415 nonempty parsed shops, 980 have no ordinary-price rows; Stage30 exchange display is opt-in OFF; Stage34 changes only `Shop_special_001` from PageCount 1 to 2 and is not a general NPC purchase fix. See Stage32/33/34 reports.
5. Because the available logs predate the Stage32 diagnostic, this investigation cannot determine which shop a user selected on the latest client, whether a View61/item-add frame was written for that choice, whether the client rendered it, or whether a purchase changed bag/DB. **LIVE/E2E: NOT RUN / NOT ESTABLISHED.** A missing diagnostic line in an older build is not evidence that an NPC has no shop.

## Exact next evidence boundary

First analyze any existing **Stage34-or-newer** diagnostic logs containing a common NPC selection's `NPC shop menu diagnostic` -> `shop service selected` -> `shop display catalog` -> `shop display preflight rejected` **or** `shop display frames`, and check actual client-visible shop items separately. The Stage35 read-only tool `tools/stage35_shop_trace_audit.py` is present, with six synthetic tests and raw-error redaction, but has no post-Stage32 real diagnostic events in the examined logs to classify. If those logs remain unavailable, only the Stage34+ non-destructive kit's new local diagnostic run can cross this boundary. Do not guess `Shop_GB_Yishiting` suffix, enable exchange purchase, change currency/bag/DB behavior, or report a client render before evidence.

## Change and validation scope

**This commit adds only this sanitized evidence report.** It changes zero Go/Python code, zero workflows, zero game/GM behavior; it neither runs a build nor overwrites any user's PC server, DB, characters or resources. No LIVE/E2E test was executed. Future work must build from the *then-latest* HEAD, never reset to a historical SHA.