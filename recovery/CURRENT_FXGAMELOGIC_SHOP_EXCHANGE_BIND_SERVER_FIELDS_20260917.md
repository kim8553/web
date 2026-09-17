# Current FxGameLogic shop exchange bind server-field proof — 2026-09-17

## Authority

Directly inspected current user-provided `bin64(2)(1)(1).zip`.

- `fxgamelogic.dll`
  - size: `43,327,760` bytes
  - SHA256: `16cf49b652c39b63ba0767780cc39f32ea1e50bdc683da9d5e7b2697cf95c3e8`
  - PE32+ x86-64
  - ImageBase: `0x11000000`
- `fxgame.exe`
  - SHA256: `c7f0366026452d525dc0c74b8499aac30fdb69189591420978e12c7094b5fde3`

This note records direct current-binary evidence only. It does not promote old JYZJ/V37/V46 behavior to current authority.

## Current native entity and getter fields

Current `FxGameLogic.dll` contains the exact strings:

- `exchange_item_manager`
- `InitCurExchangeData`
- `ClearCurExchangeData`
- `GetCurBindStatus`
- `GetCurExchangeBind`
- `GetCurShowBind`

The registered getter wrappers read these object fields:

- `GetCurBindStatus` -> `dword [object + 0xD8]`
- `GetCurExchangeBind` -> `dword [object + 0xDC]`
- `GetCurShowBind` -> `dword [object + 0xE0]`

Representative current instructions:

```text
11b09a6e: mov edx,dword ptr [rdi+0xd8]
11b09b8e: mov edx,dword ptr [rdi+0xdc]
11b09cae: mov edx,dword ptr [rdi+0xe0]
```

## InitCurExchangeData is an 11-field parser

The Lua-visible `InitCurExchangeData` wrapper reaches the current implementation at `0x11B15560`.

The implementation:

1. clears current exchange state,
2. splits the received config string on `|`,
3. requires exactly 11 fields (`cmp ... , 0xB`),
4. parses those fields into the manager state.

Direct current instructions for the three binding-related fields:

```text
11b15732: call 0x110477ad
11b15737: mov dword ptr [r12+0xd8],eax   ; token 2 = BindStatus

11b15be7: call 0x110477ad
11b15bec: mov dword ptr [r12+0xe0],eax   ; token 4 = ShowBind

11b15c30: call 0x110477ad
11b15c35: mov dword ptr [r12+0xdc],eax   ; token 5 = ExchangeBind
```

This matches the recovered exact-current wire grammar:

```text
Type|AddValue|BindStatus|Item|ShowBind|ExchangeBind|ConditionType|Condition|Condition2|Filters|Prop
```

## Proven conclusion

`ShowBind` and `ExchangeBind` are not derived by the current client manager after reception. They are runtime values supplied in the S2C 557 config string and parsed verbatim by `InitCurExchangeData`.

Therefore the authoritative derivation of `ShowBind` and `ExchangeBind` must be recovered from server behavior/source or an authoritative LIVE S2C 557 capture, not from a missing client-side calculation routine.

The current client display contract remains:

- `BindStatus <= 0`: binding preview hidden.
- `BindStatus > 0` and `ShowBind == 0`: binding preview hidden.
- otherwise the preview starts as unbound and switches to bound when `ExchangeBind > 0`.

## Production consequence

The current production partial activation is intentionally unchanged:

- authenticated mode3 definitions with `BindStatus <= 0` may send S2C 557 because the unresolved preview fields are not consumed by the current client branch;
- `BindStatus > 0` remains fail-closed until server-authoritative `ShowBind` / `ExchangeBind` derivation is proven;
- C2S `0x4F` exchange mutation remains fail-closed until final persisted output binding is authoritative.

Latest resource audit at this checkpoint:

- valid current mode3 shop rows: `42,936`
- `BindStatus <= 0`: `42,548` (`99.10%`)
- `BindStatus > 0`: `388` (`0.90%`)
- referenced unique ExchangeData: `8,575`
- safe unique `BindStatus <= 0`: `8,444` (`98.47%`)

## Fail-closed boundary

Current text/resource evidence proves only this binding direction:

`at least one actually consumed bound material -> produced result bound`

It does not yet prove the converse or complete precedence against any authored/base output binding rule. Existing recovery code must therefore continue treating the no-bound-material case as Unknown rather than automatically Unbound.
