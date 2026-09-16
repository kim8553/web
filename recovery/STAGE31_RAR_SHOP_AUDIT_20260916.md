# Stage31 — read-only NPC shop route and frame-preflight handoff (2026-09-16)

## Source authority / preservation

- User-selected baseline attachment: `9yin-go-server1.rar`, SHA-256 `ddc2f6bc078660a40eedf077d43befd6402ed59478ccde987d44171c1d7aefa5`.
- Verified RAR-contained `resources/modern/share/trade/shop.ini`, SHA-256 `ceb7a3a7e17cbc118ce5a7d3d979dea6120c50cdf6a5823697c05a5b1b9cf5dd`.
- Preserve Stage29 born02 coordinate and Lua ordinal compatibility; Stage30 exchange display remains **opt-in** (`NINEYIN_SHOP_EXCHANGE_VIEW_AB=1`). No changes to purchase mutation, currency, bag, DB, GM web grant, quest or combat.
- Original RAR is not overwritten or uploaded to GitHub. Stage31 source patch targets the verified Stage30 reconstructed Go source.

## Direct RAR `shop.ini` audit (offline data, **not** LIVE)

- 1,447 shop sections; 47,818 parseable numeric-page merchandise rows; no failures in the scanned numeric price/position fields.
- 1,012 sections contain zero ordinary-mode (0,1,2) merchandise rows, **including empty sections**; they will list no ordinary items on the unchanged default display path. This alone does not prove a specific live NPC has this shop ID.
- `Shop_nf_01001` has 103 ordinary rows; `shop_jjman_01` has 102. Server View61 currently declares `Capacity=100`, **but the number of permitted object-add frames and relationship to page/object identifiers have not been proven by client LIVE tests**. Stage31 must NOT invent a hard 100-row cutoff. Offline Go tests retain and validate all 103/102 rows.
- One section contains colliding combined ordinary/exchange grid indexes; 50 sections have over 100 combined eligible rows. Stage30 experimental exchange extension declines such uncertain cases without changing the default ordinary listing.

## Verified server path and diagnosed boundary

`NPC instance → modernNPCServices/effectiveNPCBusinessValue (creator.ShopID before template.ShopID) → selected service log (NPC config + ShopID + source) → shopCatalogItems(shop.ini, ShopID) → ordinary/default or explicitly enabled Stage30 exchange A/B selection → latestClientShopViewProperties and latestClientShopItemProperties → View61 create/add frames`.

Stage31 ensures **all selected item frame indexes are checked for validity and duplicates, and every frame is encoded, before the first view-create frame is sent**. On a validation/encode failure, it logs a preflight rejection and sends no shop view. A transport write error after sending the create frame can still leave a partial delivery and is not claimed fixed. Shop purchases remain fail-closed until an exact verified current-client purchase selector and persistence tests exist.

## Test and evidence boundaries

- Five standalone verbatim helper tests, including permissive 103-row authored fixture, invalid index, duplicate index, encode failure and preserved ordering; normal/race tests run without missing resource-dependent main-package initialization.
- Additional **local-only** tests read exact RAR `shop.ini` and check the two real 103/102 ordinary-row sections and a 25-row exchange-only section. No guessed client packets or substitute client resources.
- GitHub Actions should compile the full server test binary, compile race test binary, run `go vet`, build Windows amd64 EXE, and publish SHA-256; these are **not** live game acceptance tests. Check latest workflow run result; do not report PASS before it completes.
- Next LIVE evidence boundary: captured `shop service selected npc_config=… shop=…`, `shop display catalog …`, `shop display frames …`, plus client shop rendering and purchase traces. Without matching live logs, cannot attribute any user's NPC issue to a specific ShopID or assert the shop works.
