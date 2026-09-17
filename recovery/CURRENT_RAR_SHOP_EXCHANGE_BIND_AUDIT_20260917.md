# Original `9yin-go-server1.rar` shop exchange bind audit — 2026-09-17

## Authority and scope

This audit treats `9yin-go-server1.rar` as the original Go-server lineage/provenance source only. It is **not** allowed to overwrite the cumulative current `server/` tree, and current Snail client binaries/resources remain authoritative for current wire/client behavior.

Uploaded RAR:

- file: `9yin-go-server1.rar`
- size: `42,688,644` bytes
- SHA256: `ddc2f6bc078660a40eedf077d43befd6402ed59478ccde987d44171c1d7aefa5`
- format: RAR5
- archive entries: `10,498`

The archive was opened read-only and inspected through libarchive. No archive content was written back to the RAR.

## Relevant original server paths

The RAR contains the original Go server source and resources, including:

- `cmd/protocol-probe/custom_c2s.go`
- `cmd/protocol-probe/shop_catalog.go`
- `cmd/protocol-probe/scene_lifecycle.go`
- `cmd/protocol-probe/main.go`
- `resources/modern/share/item/exchangeitem.ini`
- `resources/modern/share/trade/shop.ini`
- `logs/protocol-probe-live.log`
- root `9yin-game-native-menu.exe`
- `build/9yin-game-native-menu.exe`

## Source-code result

An exact search over the extracted original Go/SQL source found:

- `ShowBind`: no server implementation
- `ExchangeBind`: no server implementation
- no original exchange-form response builder matching the current 11-field contract

Original `shop_catalog.go` uses the older/basic shop-row topology (`configID`, amount, capital type, price, page, position). The current recovery files such as `latest_client_shop_exchange_contract.go` and `latest_client_shop_exchange_config.go` are later recovery work and do not exist in the original RAR.

Therefore the original Go source does **not** contain an authoritative formula that can simply be copied for current `ShowBind` / `ExchangeBind`.

## Original binary cross-check

Two Go server executables were extracted read-only:

### Root executable

- size: `11,062,784` bytes
- SHA256: `34ce3c018cf7ab838578f8e15ee3709ebfa2a8106d183a531d5e78e3e056ca0b`

### `build/` executable

- size: `11,043,840` bytes
- SHA256: `7acc63911cfa36ca5ac6f99d9e9c079ae5f80e9066252f7be5dad47f3cbe89d2`

Both are PE32+ Go binaries with Go symbols/DWARF. Relevant symbols include `main.(*sceneLifecycle).openShopLocked`, `main.defaultShopINIPath`, and `main.loadShopCatalogSection`. No Go symbol/string establishes a hidden `ShowBind` / `ExchangeBind` server derivation or a hidden exchange-form implementation absent from the source.

The original `logs/protocol-probe-live.log` also contains no usable captured 557 exchange-form response from which these runtime values can be recovered.

## Resource lineage result

Original RAR `ExchangeItem.ini`:

- size: `715,225` bytes
- SHA256: `cfe0d226f365f63523e2e7cd46b5b714f5c357cce3abfdccc41a6580ad543276`
- sections: `10,573`
- sections authoring `BindStatus`: `131`
- every authored `BindStatus` value: `1`
- authored `ShowBind`: `0`
- authored `ExchangeBind`: `0`

Current extracted `ExchangeItem.ini`:

- SHA256: `ed931884d8a8bb19bad512b5eb571fe8ac97b783449144d4a4834eb050f0dee6`
- sections: `10,585`
- `BindStatus=1` sections: `131`

The binding-sensitive subset is lineage-stable:

- shared `BindStatus=1` definitions: `131 / 131`
- original-only binding definitions: `0`
- current-only binding definitions: `0`
- changed shared binding definitions: `0`

Original RAR `shop.ini`:

- size: `2,176,439` bytes
- SHA256: `ceb7a3a7e17cbc118ce5a7d3d979dea6120c50cdf6a5823697c05a5b1b9cf5dd`
- numeric item rows: `47,818`
- valid mode-3 rows with ExchangeData: `42,955`
- unique mode-3 ExchangeData values: `8,583`
- rows referencing the 131 `BindStatus=1` definitions: `388`
- unique binding-sensitive definitions referenced: `131`

Current resources:

- numeric item rows: `47,799`
- valid mode-3 rows: `42,936`
- unique mode-3 ExchangeData values: `8,575`
- rows referencing the binding-sensitive definitions: `388`
- unique binding-sensitive definitions referenced: `131`

Thus the exact 131 definitions / 388 shop rows currently held fail-closed are not a recent client-patch artifact; they persist from the original server-resource lineage.

## Current-client facts that remain authoritative

Separate current `FxGameLogic.dll` machine-code analysis established that message 557's `config_str` carries exactly 11 fields:

`Type|AddValue|BindStatus|Item|ShowBind|ExchangeBind|ConditionType|Condition|Condition2|Filters|Prop`

`InitCurExchangeData` copies field 2 into current BindStatus, field 4 into current ShowBind, and field 5 into current ExchangeBind. The current DLL does not derive these two runtime fields inside that initialization path.

The current text resource independently states that if any consumed exchange material is bound, the resulting item is bound; bound materials are consumed first by default; and for multi-result exchange the UI only presents the first result's binding state.

## Exact boundary after this audit

Proven:

1. The current client expects `ShowBind` and `ExchangeBind` as server-supplied runtime fields.
2. The original RAR source does not contain their formula.
3. The original RAR Go binaries do not expose a hidden implementation that is absent from the source.
4. The 131 binding-sensitive exchange definitions and their 388 shop references are stable from the original RAR lineage to the current resources.
5. Bound material actually consumed => produced result is bound.

Still **not proven**:

- the exact server rule for `ShowBind`
- whether an all-unbound first-unit material plan is sufficient in every one of the 131 definitions to prove final unbound
- any additional authored/base-result binding precedence

## Production safety decision

Do not invent `ShowBind=1` or derive `ExchangeBind` from incomplete assumptions.

Keep the current production boundary:

- `BindStatus <= 0`: 557 form response may use the proven safe path
- `BindStatus > 0`: the 388 rows remain fail-closed until an authoritative runtime response or server implementation proves the missing fields
- C2S `0x4f` mutation remains fail-closed until binding plus persistence/replication semantics are proven end-to-end

## Next evidence target

The highest-value next artifact is an authoritative runtime 557 exchange-form response for one of the 131 `BindStatus=1` definitions. A diagnostic observer should record only the 11 decoded fields, especially `BindStatus`, `ShowBind`, and `ExchangeBind`, without mutating inventory or server state. Controlled captures should distinguish at least:

1. required materials available only as unbound
2. a bound material available/selected by bound-first consumption
3. multi-count exchange, to verify first-result preview semantics

Until such evidence exists, the 388-row fail-closed boundary is intentional.