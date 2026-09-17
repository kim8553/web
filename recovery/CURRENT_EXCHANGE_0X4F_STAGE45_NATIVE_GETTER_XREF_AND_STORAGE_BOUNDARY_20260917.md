# Stage45 — current Type/AddValue getter registration and 0x4F storage boundary (2026-09-17)

## Scope and durable base

- Resumed `kim8553/web` `stage37-assistant-handoff-20260917` from Stage44 `95478c5e9d6809b90c3009d94b56e75c974da29e`; retained all intervening work. This report follows the Stage45 source-regression and CI commits `4ec4e400d4ebc9499f15331ca9fc57c36d5b7dde`, `d9d99a966887fe53c994b108b0b614aea17c66a6`.
- The Google Drive original `9yin-go-server1.rar` is source lineage only. Current `res` packages and `bin64.zip` are client authority. Original private bytes, extracted client binaries, Lua and INI are not added to the public repository.
- Production Go purchase handler, MySQL migrations and schema, original RAR, DLL, and packages: **UNCHANGED**. 0x4F material debit / reward: **DISABLED / FAIL-CLOSED**.

## Direct file evidence (actual read-only bytes)

| Input | Google Drive file ID | Actual SHA256 |
|---|---|---|
| original `9yin-go-server1.rar` | `1JkVemCp5s1giUks4QK-CmY0vhpNN4v0S` | `ddc2f6bc078660a40eedf077d43befd6402ed59478ccde987d44171c1d7aefa5` |
| exact-current `bin64.zip` | `13KBBJ9nEQjPZngrI_Ald-u15zlV1MKnS` | `0bfa5c66c874f6fdd78cb343d7c1ebff29490c2f5d3e7e10be45012bc0f3b2ec` |
| ZIP member `fxgamelogic.dll` | within above ZIP | `16cf49b652c39b63ba0767780cc39f32ea1e50bdc683da9d5e7b2697cf95c3e8` |
| current `share.package` | `1eoJ5ViSVdEY1xk8OKNcxmqHhcOmARQxh` | `200497852ba3a29279e51f01e2913b5f7260480f2a740a32ebd1680b869844b6` |
| current `lua64.package` | `1NWh8NSeN-mQ2RdeJi9yCkX40uuDBW7j6` | `700cb8c888374ed3ceb0f03b940e3f148620519f82352499f7957991184677d2` |
| current `lua.package` | `1jMfaMUsM5EhLo4hG-zo_3OamgpHVoMt6` | `283c8c245a3fb86af2a7e8c21c53b3de26c30432341dd590c37554adfce04e97` |

`share.package` PCK0 v15, offline zlib payload scan with zlib checksum and exact pinned SHA256 proof (no client execution):

| Payload label from Stage40 known-content fingerprint | Zlib stream file offset | Compressed bytes | Uncompressed bytes | SHA256 |
|---|---:|---:|---:|---|
| `exchangeitem.ini` | `28715362` | `82578` | `716215` | `ed931884d8a8bb19bad512b5eb571fe8ac97b783449144d4a4834eb050f0dee6` |
| `shop.ini` | `36877029` | `277336` | `2177326` | `f42fce7631f7fe302d3fdedad60b12eaa20da12396de412ff0a84edcddc1a6d9` |

Source labels above are matched **by previously verified complete payload SHA256**, not independently recovered from the encrypted PCK0 index. This stage did not claim complete archive extraction or reconstruct unknown filenames. Read-only method: scan from the header's data-start for zlib-compatible headers, decompress with `zlib.decompressobj()`, require `eof`, bounded output, checksum validity, and match complete payload digest. There were 9,826 valid candidate zlib streams in this bounded scan; only these two requested payload digests are reported.

## Newly inspected current DLL registration path: evidence limited to exposure

All virtual addresses here are for the SHA256-pinned current `fxgamelogic.dll` (PE x64 ImageBase `0x11000000`). `objdump -D -M intel` static disassembly, **not** a run-time trace:

- `GetType` name pointer VA `0x12D3E318` is referenced by function-registration initialization at `0x11B138B2` and another initialization at `0x12C242BD`. Corresponding code pointer `0x11034E2D` is written at `0x11B138C0` / `0x12C242D2`; the nearby callable wrapper is at `0x11B13750`. At that wrapper `0x11B13789..0x11B13791`, an indirect vtable `+0x98` call obtains a value, which is passed to conversion/storage at `0x11B137A5..0x11B137A9`. Do not interpret the wrapper as a direct official server payment implementation.
- `GetAddValue` name pointer VA `0x12D3E358` is referenced at `0x11B13A62` and `0x12C246CD`; corresponding code pointer `0x11066248` is written at `0x11B13A70` / `0x12C246E2`. Adjacent wrapper at `0x11B13910` uses an indirect vtable `+0x98` call at `0x11B13950` before string-handling call at `0x11B13961`.
- Previous Stage43/44 exact-current evidence establishes `exchange_item_manager::InitCurExchangeData` VA `0x11B15560`, `Type` parsed into object `+0x88` at `0x11B15695`, and `AddValue` copied to object `+0x90` at `0x11B156E9..0x11B156F1` (11-field S2C 557 *form/config*).
- This establishes native metadata/registration and an indirect getter-exposure path, **not** a complete call graph, an actual Lua getter invocation, a server-side charge, a reward operation, or `AddValue` direction.

Lua-specific limitation: authenticated `lua64.package` PCK0 v15 scan found 2,293 valid bounded zlib payloads; `lua.package` scan found 2,295. The scanned payloads start with `\x1bLuaQ` (Lua 5.1 bytecode), and the bounded raw-byte substring search found **zero** `GetCurType`, `GetAddValue`, `GetCurAddValue`, `custom_exchange_item`, `ExchangeData`, `form_exchange` or other selected plaintext tokens in those payloads. A zero **plaintext** hit in binary bytecode is **NOT** evidence of absence of a Lua call site: decoded Lua constant tables/file index and any additional encoding still require verification. No Lua file name, function location, or getter semantics is asserted based on this scan.

## Original RAR vs current Go handler

- Original RAR5: 42,688,644 bytes; 10,498 archive entries; read-only `libarchive` extraction of original Go/SQL source (114 extracted source/module files in the working sandbox). Examples actually present: `cmd/protocol-probe/main.go`, `player_actor.go`, `shop_catalog.go`, `custom_c2s.go`, `scene_lifecycle.go`, `internal/role/mysql_repository.go`, `migrations/0001_normalize_role_repository.sql`. Do **not** assert the archive has all historical source or lacks all hidden binary behavior.
- Original `cmd/protocol-probe/shop_catalog.go` parses a basic `capitalType` field; its `scene_lifecycle.go:openShopLocked` populates View 61 price/display properties and sends `serverViewAdd`, with no exchange debit/grant in that function. Exact search of the extracted original Go source found no implementation named `handleShopExchangeContract`, `parseShopExchangeBuyRequest`, `GetAddValue` or an exact-current 0x4F purchase settlement. The original display/retail path is not a verified latest-protocol exchange purchase.
- Current `server/cmd/protocol-probe/latest_client_shop_exchange_contract.go` `parseShopExchangeBuyRequest` requires selector + `(ShopID string, Page int32, Position int32, Count int32)`, 5 typed values total; ConfigID, ExchangeData, price, and an idempotency key are **not** in that decoded wire struct. This does not prove that no unrelated session key exists elsewhere; do not add a fictional packet field.
- Current `handleShopExchangeContract` re-resolves listing coordinates and authenticated current condition information, logs `blocked: native eligibility/cost/bind/commit path unresolved`, and returns `true,nil` without material debit or grant. The condition evaluator is an observation rather than authoritative purchase admission.
- Current `latest_client_shop_exchange_config.go` validates config grammar and encodes display/config 557; it does not determine purchase settlement. Original exchange resource semantics must not overwrite newer client behavior.

## Actual state and transaction gates

- Current `zz_recovered_overlay.go` includes a `playerActor` bag snapshot-to-client view path (`grantBagItems`), JSON `bagStore` at `data/bag_items.json` using `.tmp` then rename, and JSON `currencyStore` at `data/currency.json` using direct `os.WriteFile`. These are **distinct files and write operations**; renaming the bag file does not atomically commit currency or SQL state. Current `mysqlCurrencyStore.Save` has its own `BeginTx` and DELETE/INSERT of `role_currency` snapshot; Stage43 verified `mysqlBagStore.Save` separately transacts DELETE/INSERT for `role_bag_items` only. Separate `BeginTx` methods are not one all-resource transaction.
- Stage44 established missing `NINEYIN_MYSQL_DSN` causes `openGameDataDB` to return nil DB; MySQL code existing does **not** mean the running server always saves to MySQL.
- **Not fully mapped** in Stage45: all 37 authored Prop property names to actual player state owners/columns/load paths; distinct equipment and bag update transaction boundaries; every post-commit client replication event; JSON/MySQL authority selection; empty-bag MySQL Load `len(items) != 0` caller behavior. These gaps prevent an atomic purchase design from being labelled implemented.
- No verified debit multiplier for Count, precise Item/Prop combination semantics, Type/AddValue charge direction, bonus/reward identity, ShowBind/ExchangeBind derivation, bag capacity admission, durable replay identity, or one-transaction rollback/reconnect contract. Empty Item+Prop does not imply free award.

## Actual code and checks in this Stage45

1. Added `tools/stage45_exchange_guard/guard_test.go`: Go AST source-regression test against the real checked-out current `latest_client_shop_exchange_contract.go`. Checks the 4-field purchase request shape, 5-value wire length check, explicit fail-closed terminal return and a bounded existing-call allowlist. Synthetic negative fixtures attempt to introduce four unreviewed side-effect calls, a fictional `RequestID`, and a changed final refusal. Scope is **regression detection only**, not a proof that arbitrary future Go constructs are side-effect free.
2. Added `.github/workflows/jiuyin-stage45-exchange-failclosed-guard.yml`: checkout and `GO111MODULE=off go test -race -v` for the above test package, no private client bytes in runner. [Actions 35236402411](https://github.com/kim8553/web/actions/runs/35236402411) completed SUCCESS on commit `d9d99a966887fe53c994b108b0b614aea17c66a6`; `source-regression` job and test step reported SUCCESS. This verifies its bounded synthetic and current-source AST tests, **not** purchase success or runtime behavior.
3. Sandbox read-only binary inspection, PCK0 payload verification and original RAR code comparison were performed. No permanent server production changes. Full exact GitHub working tree could not be cloned directly in the sandbox because `git ls-remote` could not resolve `github.com`; precise current production files and commits were read through the connected GitHub API, and the newly committed source regression was executed by Actions against an actual checkout. No claim of local full-server tests.

## Verification matrix (Stage45 only)

| Check | Result |
|---|---|
| GitHub exact branch/HEAD and Stage44 handoff | PASS read-only API |
| Original RAR / bin64 / share / lua / lua64 byte SHA256 | PASS offline |
| Exact-current DLL disassembly / registration xrefs | PARTIAL, registration verified; getter use and settlement unresolved |
| Exact-current share payload hash extraction | PASS for two pinned payloads; index filename unverified |
| Current Lua plaintext-token scan | DONE; zero plaintext hits, bytecode semantics NOT VERIFIED |
| Stage45 Go AST synthetic/current-source unit checks | PASS in GitHub Actions |
| Stage45 `go test -race` source-regression job | PASS in GitHub Actions (same test scope; **not** a concurrent purchase test) |
| SQL multi-store transactions / rollback / replay | NOT RUN / NOT IMPLEMENTED |
| Windows 64-bit full server build | NOT RUN |
| Server boot / current-client login / live 0x4F purchase | NOT RUN |
| Database persistence / reconnect | NOT RUN |

**Next single priority:** recover a verifiable *actual current Lua getter call site* with its bytecode constant-table/name provenance and usage context for `GetType`/`GetAddValue` (or explicitly prove a directly relevant native caller). Then connect that evidence to Item/Prop/Count/binding without inferring server settlement from UI rendering. Preserve 0x4F fail-closed until all transaction gates independently pass.
