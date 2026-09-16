# Stage37 Stage24 — normal NPC shop display: read-only offline audit

**Scope:** source-model analysis of Stage23 reconstructed Go code and one Drive backup file. This is **not** a production fix, an exact-current client resource, a Windows LIVE session, a selector confirmation, or an E2E purchase pass. No DB, user PC, production handler, GM grant code, or network packet IDs were modified.

## Grounded source chain

- Stage23 verification artifact `10435251412`, `regular_shop_audit.log`, records `cmd/protocol-probe/scene_lifecycle.go:477`, `openShopLocked(shopID)` calling `shopCatalogItems(defaultShopINIPath, shopID)`. It creates view `ID=61, Capacity=100`, then loops through parsed items. Only `priceMode` **0, 1, 2** execute `serverViewAdd(61, objectIndex, properties)`; other modes execute `continue` without item-add frames. It returns errors on catalog load failures, index errors, encode errors, or frame write failures. The existence of this function does not prove the user's session invoked it.
- `cmd/protocol-probe/shop_catalog.go:24`, `loadShopCatalogSection`, opens `resources/modern/share/trade/shop.ini`, matches the exact `[shopID]` section, parses rows and fails if the section is missing, empty, or a row is malformed. `shopCatalogItems` delegates to this loader without adding synthetic merchandise.
- `cmd/protocol-probe/latest_client_shop_view_contract.go:6` specifies `currentShopPageSize=500` and derives the object index as `page*500+position+1` (uint16 bounded). A **capacity 100 versus index >100** discrepancy is observable in source, but the client consequence of that discrepancy is **unverified**. Do not silently resize a view or claim that it caused the user's empty shop.
- The legacy `0x46` purchase candidate remains fail-closed. No selector or client packet layout is inferred from the listing audit.

## Drive backup resource inspected, not assumed to be exact-current

- Input: `resources/modern/share/trade/shop.ini` fetched from the connected `9yin_server` backup, file size `2,176,439` bytes, SHA256 `ceb7a3a7e17cbc118ce5a7d3d979dea6120c50cdf6a5823697c05a5b1b9cf5dd`. It differs from `exactCurrentShopINISHA256=f42fce7631f7fe302d3fdedad60b12eaa20da12396de412ff0a84edcddc1a6d9` in the exchange-condition authority loader. The mismatch is a version gap, not proof of the NPC shop failure.
- Offline parser sections: **1,447** distinct headers; **435** have at least one normal item row with priceMode 0/1/2; **980** have rows but *no* 0/1/2 item rows; **32** have no numeric item rows. These categories exhaust 1,447. Note: this does not establish which shop ID the user's NPC opens.
- Parsed rows: mode 1 = **4,120**, mode 2 = **694**, mode 3 = **43,004**; mode 0 = 0. Total **47,818** numeric rows. No malformed row was detected in the offline source-model scan. Largest raw line = 66 bytes. The backup file's 980 mode-3-only sections would yield **zero item-add frames if passed through this exact `openShopLocked` path**. That conditional is a concrete *candidate explanation*, not a verified diagnosis of the actual user's NPC.
- Of normal-mode rows, **1,806** would calculate a view object index > the declared capacity of 100; **0** calculate an index outside the uint16/page-position validity conditions. There were **0** sections with duplicate eligible object indexes. Whether the client accepts index >100 is unknown.

## Reproducible checks and limits

`python tools/stage37_shop_display_audit.py --shop-ini <absolute-path-to-shop.ini>` runs a **read-only** aggregate scan, prints input SHA256 and source-model counts, and does not output the full resource or any DB data. Optional `--shop-id <exact-ShopID>` analyzes one known ID; it does not discover an NPC ID or guess what the user clicked. `python -m unittest discover -s tests -p test_stage37_shop_display_audit.py -v` runs four synthetic-fixture tests. The offline tool was executed on the downloaded backup file and all four tests passed in the ChatGPT container. No Windows build or LIVE test was performed in Stage24.

## Remaining prerequisite before requesting client interaction

1. Correlate a **specific** actual NPC and its `shopID` to this catalog, and verify the executable SHA256 / server root / loaded shop.ini SHA256 in the actual runtime. Until then the empty-shop cause remains unknown.
2. Read-only instrument or capture shop open: matched section, parsed modes, planned versus successfully written item-add frames, failures, and selected role's loaded currency category without recording credentials. Confirm whether mode-3 is handled by some other service path; do not insert exchange rows into the ordinary listing without current-client evidence.
3. Complete a version-consistent resource + executable + safe launcher inventory and exclude private MySQL data / credentials from any redistributable ZIP. Do not call the Stage23 artifact a full server.
4. Only after the correct NPC displays goods and the test role has a verified funding condition, resume passive purchase-wire work. Do not request purchase button presses while the shop is blank.
