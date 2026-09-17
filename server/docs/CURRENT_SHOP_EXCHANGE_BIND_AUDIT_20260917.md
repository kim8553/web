# Current-client shop exchange: binding/UI and fail-closed checkpoint

Scope: `9yin-go-server1.rar`-derived cumulative Go server on `stage37-assistant-handoff-20260917`. This note records verified current-client/resource findings and deliberately separates them from unresolved production-server semantics and LIVE/E2E results. The raw client, decoded Lua, XOR key and private `res` assets remain outside GitHub.

## Verified client-side call chain

- The private current `res/lua64.package` was authenticated against the retained client package before extracting bytecode locally. Its decoded `lua64/form_stage_main/form_shop/form_shop.lua`, function at source lines 967–987, closes any existing exchange form, resolves `get_view_item(viewid, bindindex)`, returns when the item is invalid, and otherwise invokes `custom_sender.custom_request_shop_exchange_form(viewid, bindindex, shopid, page, pos)`. These are **five function arguments**; the CustomSend wire additionally includes selector `0x40`.
- The exact current `lua64/custom_handler.lua` handler `on_open_shop_exchange_form` at source lines 8769–8789 requires at least six payload values, converts `(viewid, bindindex, shopid, page, pos, config_str)` to integer/integer/string/integer/integer/string, and dispatches to `form_stage_main/form_shop/form_exchange.show_form`.
- The exact current `lua64/form_stage_main/form_shop/form_exchange.lua` `show_form` at source lines 699–728 requires a valid view item **and** a valid `exchange_item_manager`, stores the first five arguments on the form, calls `exchange_item_manager.InitCurExchangeData(config_str)` with the sixth, and shows the form. This is evidence for the display call chain, **not** for the unresolved server decision that produces all fields of `config_str`.
- The same exchange Lua at source lines 261–347 fetches `GetCurBindStatus`, `GetCurExchangeBind`, and `GetCurShowBind` separately. It displays the first binding label when `BindStatus > 0`, replaces it with a second label when `ExchangeBind > 0`, and hides the label when `ShowBind == 0`; it also hides the label in the branch where `BindStatus <= 0`. These are **UI comparison and display rules only**, not by themselves a server binding-derivation formula.
- The exchange Lua's buy-button handler at lines 81–92 issues `custom_sender.custom_exchange_item(shopid, page, pos, amount)` when the count is positive; the wire selector is `0x4f` and the `ExchangeData` definition is **not** sent by the client. The server must resolve the listing authoritatively.

## Verified current-resource material binding semantics

A retained exact-current localized text resource independently closes the material-derived part of the exchange binding rule:

- if **any material actually used by an exchange is bound**, the produced item is bound;
- bound material is consumed **before unbound material** by the normal exchange behavior;
- for a multi-result exchange, the UI warning/preview refers to the **first result's binding state**.

These semantics corroborate the already reconstructed side-effect-free `internal/exchangeplan.PlanBatch` behavior in the current recovery delta: it chooses bound stacks before unbound stacks and records `MaterialDerivedBound` independently for each produced result. They also support the existing first-result `ExchangeBind` preview. This closes only the material-inheritance side of binding; it does **not** prove that an all-unbound material plan must produce an unbound result, because an authored/base result binding rule may still apply.

No private localized text, decoded resource payload, XOR key or client binary is committed here; only the verified semantic conclusion is recorded.

## Existing Go contract and safety boundary

- `cmd/protocol-probe/latest_client_shop_exchange_contract.go` parses the exact `0x40` and `0x4f` shapes and deliberately stops both requests without sending a success frame or committing a purchase; `0x45` condition-detail replies are implemented separately as S2C 508, subject to available authoritative resources and the evaluator's fail-closed handling.
- `cmd/protocol-probe/latest_client_shop_exchange_config.go` has a typed S2C 557 encoder and the independently recovered 11-field `InitCurExchangeData` order `Type|AddValue|BindStatus|Item|ShowBind|ExchangeBind|ConditionType|Condition|Condition2|Filters|Prop`. The current `ExchangeItem.ini` does **not** author `ShowBind`/`ExchangeBind`; the Go helper requires those values from independently proven rules.
- The current recovery planner already implements the now-resource-backed bound-first material selection and per-result material-derived binding marker. No exchange mutation gate is changed by this documentation update.
- Regression coverage for the existing fail-closed contract remains: valid `0x40`/`0x4f` requests emit **zero frames** while unresolved; separately, encoding S2C 557 must round-trip through the project's typed custom-message parser as selector plus exactly six correctly typed and ordered arguments. A passing test is **not** proof that the client received a frame in LIVE play.

## Outstanding evidence (do not infer)

1. Source-backed **server-authoritative precedence for the final result `BindStatus` when no bound material is consumed**, including whether/how the authored `ExchangeItem.ini` `BindStatus` participates. The newly proven material rule establishes `bound material used => bound result`; it does not establish the converse.
2. Exact server-authoritative derivation of `ShowBind`. Current Lua proves only how this value affects presentation. Do not substitute `BindStatus`, `ExchangeBind`, or a constant without evidence.
3. Remaining server-side pricing/conditions, inventory and currency mutation, atomic DB commit/rollback, notification/bag replication and error behavior for `0x4f`. Client-provided `shopid,page,pos,amount` cannot be treated as trusted cost or item identifiers.
4. An actual running-server trace of `0x40` request → S2C 557 → client form, and a separate end-to-end purchase with relog persistence. These remain **NOT RUN**. Keep the mutation gate fail-closed until the unresolved semantics above are closed.

## Reproduction boundary

Private local resources can support protocol tests, but the public GitHub CI deliberately reports protocol runtime `NOT_RUN` when private INIs are absent. Earlier private-local protocol tests for this recovery line were static/regression evidence only and do not establish LIVE parity. Keep client binaries, decompiled/decoded Lua, keys, resource files and user account data out of the public repository.
