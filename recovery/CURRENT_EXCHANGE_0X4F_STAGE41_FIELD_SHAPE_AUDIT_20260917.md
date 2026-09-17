# Stage41 — exact-current 0x4F ExchangeItem field-shape checkpoint (2026-09-17)

## Authority and continuation

- Continue `kim8553/web` branch `stage37-assistant-handoff-20260917`, starting from Stage40 commit `9cf3c80694d3bdd178d997510bcf14e4c6e0b874`. Do not reset to the original RAR, Stage37, V37/V46/JYZJ, or substitute gameplay semantics from the newly uploaded `九阴服务端.zip`.
- Directly used mounted original `res/share.package` bytes (40,680,972 bytes, SHA256 `200497852ba3a29279e51f01e2913b5f7260480f2a740a32ebd1680b869844b6`). Extracted two zlib streams at the previously verified Stage40 offsets and **verified each entire decompressed file digest**, with zlib end-of-stream/checksum checks: `shop.ini` at offset 36,877,029, 2,177,326 bytes, SHA256 `f42fce7631f7fe302d3fdedad60b12eaa20da12396de412ff0a84edcddc1a6d9`; `exchangeitem.ini` at offset 28,715,362, 716,215 bytes, SHA256 `ed931884d8a8bb19bad512b5eb571fe8ac97b783449144d4a4834eb050f0dee6`. These labels come from already audited exact SHA256 references, **not** a newly recovered encrypted PCK0 index filename.
- Analyzed these exact bytes offline using `tools/stage41_exchange_authority_shape_audit.py`, which refuses other file hashes, preserves duplicate numeric row keys while walking each shop section line by line, and fails closed on duplicate/missing exchange sections. It is **read-only; not a live exchange handler, material allocator, or settlement implementation**. No private package bytes or extracted INIs were checked into GitHub.

## Audited structural counts (NOT purchase semantics)

| Authored shape among referenced ExchangeData | Unique definitions | Referencing nonzero mode-3 shop rows |
|---|---:|---:|
| `Item` only | 6,931 | 39,319 |
| `Prop` only | 1,502 | 3,270 |
| both `Item` and `Prop` | 138 | 324 |
| neither | 4 | 23 |
| **Total** | **8,575** | **42,936** |

The authenticated `shop.ini` contains 47,799 numeric listing rows, of which 42,985 have price mode 3 and 42,936 reference nonzero ExchangeData. The authenticated `ExchangeItem.ini` contains 10,585 unique sections; all 8,575 referenced IDs exist. Exactly 131 referenced definitions (`388` shop rows) author `BindStatus=1` and remain blocked pending authoritative `ShowBind` / `ExchangeBind`. `Type` is **absent** in 8,049 referenced definitions (41,353 rows), `Type=1` in 2 (2), `Type=2` in 41 (41), `Type=3` in 483 (1,540). `AddValue` is authored in 43 referenced definitions (43 rows). Neither `Type` nor `AddValue` may be ignored in an all-items settlement approximation. `Prop` includes identifiers other than `CapitalType*` (e.g., `SchoolContribute`, `WGJobSkillPoint`), so its name alone is **not** proof of currency or a universal debit API.

## Four explicit non-Item/non-Prop cases

- ExchangeData `1050` (one row) and `1051` (one row): `Type=1`, `AddValue=10000000` / `15000000`, `Condition=17216`, `ConditionType=0`, neither `Item` nor `Prop`.
- ExchangeData `15311` (one row): `Condition=204307`, neither `Item` nor `Prop`.
- ExchangeData `7395` (20 rows): **empty section**. An empty section is not a free-purchase authorization, nor evidence that those 20 shop results should be granted.

These are exact authored text observations, not verified native meanings of `Type`, `AddValue`, or any runtime response.

## Newly isolated syntax mismatch: 10 mode-3 rows

Exactly 10 referenced ExchangeData definitions have a trailing semicolon in `Item`, producing an empty final token after splitting on `;`: `11564`, `14087`, `14088`, `14089`, `14286`, `14287`, `14288`, `14322`, `14323`, `14324`. Each is referenced by one nonzero mode-3 shop row; no referenced `Prop` pair-list has the same malformed-token shape. The **present Go implementation** `latest_client_shop_exchange_config.go:validateExchangePairList` rejects a trailing empty entry (`len(parts) < 2`); therefore those 10 rows cannot currently pass that Go full-definition validation. This audit does not establish whether the latest native `FxGameLogic.dll` parser accepts trailing delimiters; **do not patch the validator by guessing**. Confirm its native parser tokenization and explicit tests before changing acceptance, especially before any debit/grant.

## Tests performed and exact limits

- Local synthetic `--self-test`: PASS for repeated shop row keys, Item-only/Prop-only/neither shape, trailing semicolon detection, duplicate exchange section and duplicate-field rejection.
- Local full audit against the **actual hash-authenticated current extracted INI bytes**: PASS; counts above. No gameplay was run.
- `.github/workflows/jiuyin-stage41-exchange-shape-audit.yml` verifies the synthetic test and agrees with the two production Go SHA256 constants. The real private 40 MB package is **not** in GitHub Actions, so Actions does not prove actual-package extraction or runtime behavior.
- Production Go server and client binary **unchanged**. No item debit, reward, binding, database transaction, view/replication, packet send, client LIVE, or reconnect E2E tested or activated.

## Next strict actions

1. Trace current `FxGameLogic.dll`/Lua and authorized native observations for `Item` versus `Prop` usage, `Type=1/2/3`, `AddValue`, exact Count multiplication, and the trailing-`;` parser behavior. Never infer item inputs/outputs merely from the word `Item` or shop row result identifier.
2. Before mutation, establish binding precedence and actual material consumption, including the 131 `BindStatus=1` definitions / 388 shop rows, and determine if native `ShowBind` and `ExchangeBind` have an authoritative source.
3. Verify lossless inventory schema and single-transaction per-role serialization/rollback/idempotency, write insufficiency/full-bag/duplicate/failed-commit/reconnect tests, then verify latest-client success/failure response and bag replication. Existing `mysqlBagStore.Save` only rewrites a bag snapshot; it does not already provide a proven multi-resource atomic exchange.
