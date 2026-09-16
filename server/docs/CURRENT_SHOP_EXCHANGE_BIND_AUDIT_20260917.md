# Current-client shop exchange: binding/UI and fail-closed checkpoint

Scope: `9yin-go-server1.rar`-derived cumulative Go server on `stage37-assistant-handoff-20260917`, inspected at parent commit `3fabb1f0d875141f3441b6bb583298e2bbd75d33`. This note records **static current-client findings**, not production-server semantics or a successful in-game transaction. The raw client, decoded Lua, XOR key and private `res` assets remain outside GitHub.

## Verified client-side call chain

- The private current `res/lua64.package` was authenticated against the retained client package before extracting bytecode locally. Its decoded `lua64/form_stage_main/form_shop/form_shop.lua`, function at source lines 967–987, closes any existing exchange form, resolves `get_view_item(viewid, bindindex)`, returns when the item is invalid, and otherwise invokes `custom_sender.custom_request_shop_exchange_form(viewid, bindindex, shopid, page, pos)`. These are **five function arguments**; the CustomSend wire additionally includes selector `0x40`.
- The exact current `lua64/custom_handler.lua` handler `on_open_shop_exchange_form` at source lines 8769–8789 requires at least six payload values, converts `(viewid, bindindex, shopid, page, pos, config_str)` to integer/integer/string/integer/integer/string, and dispatches to `form_stage_main/form_shop/form_exchange.show_form`.
- The exact current `lua64/form_stage_main/form_shop/form_exchange.lua` `show_form` at source lines 699–728 requires a valid view item **and** a valid `exchange_item_manager`, stores the first five arguments on the form, calls `exchange_item_manager.InitCurExchangeData(config_str)` with the sixth, and shows the form. This is evidence for the display call chain, **not** for the unverified server decision that produces `config_str`.
- The same exchange Lua at source lines 261–347 fetches `GetCurBindStatus`, `GetCurExchangeBind`, and `GetCurShowBind` separately. It displays the first binding label when `BindStatus > 0`, replaces it with a second label when `ExchangeBind > 0`, and hides the label when `ShowBind == 0`; it also hides the label in the branch where `BindStatus <= 0`. These are **UI comparison and display rules only**, not a binding-derivation formula or proof of any server-side item state.
- The exchange Lua's buy-button handler at lines 81–92 issues `custom_sender.custom_exchange_item(shopid, page, pos, amount)` when the count is positive; the wire selector is `0x4f` and the `ExchangeData` definition is **not** sent by the client. The server must resolve the listing authoritatively.

## Existing Go contract and safety boundary

- `cmd/protocol-probe/latest_client_shop_exchange_contract.go` parses the exact `0x40` and `0x4f` shapes and deliberately stops both requests without sending a success frame or committing a purchase; `0x45` condition-detail replies are implemented separately as S2C 508, subject to available authoritative resources and the evaluator's fail-closed handling.
- `cmd/protocol-probe/latest_client_shop_exchange_config.go` has a typed S2C 557 encoder and the independently recovered 11-field `InitCurExchangeData` order `Type|AddValue|BindStatus|Item|ShowBind|ExchangeBind|ConditionType|Condition|Condition2|Filters|Prop`. The current `ExchangeItem.ini` does **not** author `ShowBind`/`ExchangeBind`; the Go helper requires those values from an independently proven rule.
- New regression coverage: valid `0x40`/`0x4f` requests emit **zero frames** while unresolved; separately, encoding S2C 557 must round-trip through the project's typed custom-message parser as selector plus exactly six correctly typed and ordered arguments. A passing test is **not** proof that the client received a frame in LIVE play.

## Outstanding evidence (do not infer)

1. Source-backed **server-authoritative** derivation of `ShowBind` and `ExchangeBind`, including interactions with `BindStatus` and item-bound state. A Lua presentation comparison is insufficient.
2. Server-side pricing/consumption order, conditions, inventory and currency mutation, atomic DB commit/rollback, and notification/bag replication for `0x4f`. Client-provided `shopid,page,pos,amount` cannot be treated as trusted cost or item identifiers.
3. An actual running-server trace of `0x40` request → S2C 557 → client form, and a separate end-to-end purchase with relog persistence. These remain **NOT RUN**. Do not change the fail-closed gates until (1) and (2) have evidence.

## Reproduction boundary

Private local resources can support protocol tests, but the public GitHub CI deliberately reports protocol runtime `NOT_RUN` when private INIs are absent. Private-local protocol test after this change (Go 1.23.2 with available cached dependencies and private fixtures): **199 PASS / 0 FAIL / 5 SKIP**; the five skips are two Windows-only door tests and three missing NPC creator-resource tests. Targeted new tests: **2 PASS / 0 FAIL**. This does not establish parity with the public CI toolchain or a live connection. Keep client binaries, decompiled/decoded Lua, keys, resource files and user account data out of the public repository.
