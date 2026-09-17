# Stage43 — exact-current Prop display parser and SQL settlement gate (2026-09-17)

## Continuation / provenance

- Resume Stage42 commit `60dafec651ab8a29a78390bc4fccee88f53a8cda` on `kim8553/web` branch `stage37-assistant-handoff-20260917`. This checkpoint is NOT a reset to the older uploaded `九阴服务端.zip` or old resource files.
- Inspected actual current `bin64.zip` member `fxgamelogic.dll`, SHA256 `16cf49b652c39b63ba0767780cc39f32ea1e50bdc683da9d5e7b2697cf95c3e8`, PE x64 ImageBase `0x11000000`; inspected exact-current `share.package` payloads with whole-file SHA256 `shop.ini` `f42fce7631f7fe302d3fdedad60b12eaa20da12396de412ff0a84edcddc1a6d9` and `exchangeitem.ini` `ed931884d8a8bb19bad512b5eb571fe8ac97b783449144d4a4834eb050f0dee6`. File names inherit previously proven hash labels, not a newly decrypted PCK0 index.
- No client executable was run or edited; no private DLL, package or extracted INI committed.

## Native facts — **display/config parsing only**

- `FxGameLogic.dll` native `exchange_item_manager::InitCurExchangeData` starts at VA `0x11B15560` and requires exactly 11 top-level pipe tokens (`0x11B1561C..0x11B15625`). This is the S2C 557 exchange-form config, not a server purchase handler.
- `Type` (top-level index 0) goes through numeric conversion at `0x11B1568E..0x11B15695` into object DWORD at `+0x88`. `AddValue` (index 1) is copied as a string at `0x11B156D7..0x11B156F1` into `+0x90`. Neither setter proves the *gameplay meaning* or cost/grant direction of these fields.
- `Prop` (top-level index 10) starts at `0x11B164DC..0x11B1651C`; `0x11B16603` reads literal `;` from `0x12CBB388` and `0x11B1661A` splits the Prop field. Each element is split by literal `,` at `0x11B166C8..0x11B166D7` (delimiter address `0x12CB9528`). At `0x11B166DC..0x11B166E7`, fewer than two parts branch past this element; therefore **one final empty element after a trailing semicolon is skipped**, just as previously verified for Item. A value is converted along the `0x11B16870` call to `0x12A915C0` and written as a QWORD at `0x11B16875`; the name/value pair is passed to a native container routine at `0x11B1688A`. Avoid inferring signedness, balance source, debit direction, spending, reward, or settlement from this parsing path.
- The client also skips malformed intermediate short elements. The Go validator intentionally remains stricter on these: we have **not** established that silently skipping corrupt server-controlled price entries is safe. Only one **terminal** empty element is accepted.

## Exact-current data scope and bounded Go change

- Direct offline hash-authenticated audit of all referenced mode-3 definitions: **1,640** unique definitions author nonempty `Prop`, used by **3,594** mode-3 shop listing rows. None of the 1,640 current `Prop` strings ends with `;` or has an internal `;;`. `Prop` keys include `SchoolContribute` and `WGJobSkillPoint`; therefore `Prop` must not be equated with only a `CapitalType*` currency balance.
- Production `server/cmd/protocol-probe/latest_client_shop_exchange_config.go`, commit `8938f204e557889833db081ee88b1a02386722d2`: in `validateExchangePairList`, allow one final empty semicolon element for **Item OR Prop**, retaining the raw authored string. No item price arithmetic, purchase handler, MySQL store, DB schema, network selector, bind derivation, or gameplay settlement changed. Stage42's prior Item allowance remains intact. This is a **parser compatibility correction with no immediate newly unlocked Prop row**.
- Existing test `latest_client_shop_exchange_trailing_delimiter_test.go` was updated (commit `aa076c873ca4864062b6ad0abd3cbf2bdc4aedbc`) to test Prop terminal acceptance, intermediate/double-delimiter rejection, malformed numeric values and unchanged Item behavior. Exact-current private INI values for **all 1,640 referenced Prop definitions** were exercised against a locally extracted copy of the modified production validator with `go test -race`: PASS. This private test fixture was **not uploaded to GitHub**.
- [Actions isolated production-validator/race run 35232293955](https://github.com/kim8553/web/actions/runs/35232293955): SUCCESS. [Actions Go tests, vet, isolated 0x4F race and Windows amd64 cross-build run 35232293975](https://github.com/kim8553/web/actions/runs/35232293975): SUCCESS. These CI jobs do not contain the private current package and do NOT prove LIVE/E2E exchange.

## Verified server transaction blockers (do not enable purchases)

- `server/cmd/protocol-probe/latest_client_shop_exchange_contract.go` current `handleShopExchangeContract` 0x4F branch resolves exact-current mode-3 coordinates and condition observations, logs `native eligibility/cost/bind/commit path unresolved`, then returns **without material debit or item award**. This is intentional fail-closed behavior, not an accidental skipped award.
- `server/cmd/protocol-probe/zz_recovered_overlay.go` existing `mysqlBagStore.Save` independently begins an SQL transaction, deletes all rows for `role_id` in `role_bag_items`, reinserts its passed bag snapshot, and commits. It has no input representing an exchange request, material debit specification, result mapping, binding, currency/Prop debit, idempotency token, or other-store commit. **A transaction in this individual Save method is not an atomic multi-resource purchase transaction.** `mysqlBagStore.Load` returns `len(items) != 0`, so an empty saved bag is not distinguished from no loaded bag by that boolean; inspect callers and schema before attempting a fix.
- No SQL rollback/duplicate-request/concurrent buy test or MySQL-backed latest-client exchange LIVE/reconnect test was performed. Runtime `ShowBind`/`ExchangeBind` for `BindStatus=1` remains unresolved. `Type`/`AddValue` getter *use* and authoritative server cost/grant semantics remain unresolved.

## Next exact work

1. Trace native getter xrefs for `Type(+0x88)`, `AddValue(+0x90)`, and Prop container (`+0x150`) and the current Lua/UI usage; **do not** reinterpret client display paths as official server-side debits.
2. Map the actual Go actor inventory/Prop/currency ownership and persistence schema, demonstrate an immutable per-role snapshot and single SQL transaction or durable outbox strategy including replay idempotency, rollbacks and reconnect. Test independently before wiring any 0x4F production mutation.
3. Preserve old executable and all client binaries unchanged; do not invent token semantics, bind priority, quantity scaling or output item type.
