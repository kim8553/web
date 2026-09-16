# Stage37 Stage25 — city NPC ShopID → ordinary shop catalog (2026-09-16)

**Result: offline backup-data correlation only. Not an empty-shop fix, complete-server restore, exact-current source/resource match, Windows build, client selector promotion, currency grant, or LIVE/E2E pass.** Production server source, legacy 0x46 fail-closed gate, user DB, GM grant, and user PC have not been modified.

## Verified input provenance

- Both files were individually fetched from the connected `9yin_server` backup under `resources/modern/share/`; no `JYZJ.server/Res/ini/Npc/NpcFunc` file was substituted. In particular, finding a separate old `JYZJ.server` NPC-function XML in Drive **does not** authorize using its mappings for the current Snail client.
- `npc/npcconfig/worldnpc/city_commonnpc.txt`: 1,943,923 bytes; SHA256 `ec3fbfc178e34ff7691f899f89b0e998d07d6e8a0afe6153e1bbc9b3a975f9c5`. The third, ASCII schema header contains `ID` at field 0 and `ShopID` at field 90, out of 128 tab-delimited columns. This identifies authored table fields, **not** a verified runtime server lookup.
- `trade/shop.ini`: 2,176,439 bytes; SHA256 `ceb7a3a7e17cbc118ce5a7d3d979dea6120c50cdf6a5823697c05a5b1b9cf5dd`. Stage24 already established this differs from the Stage23 exchange-condition loader's expected exact-current `shop.ini` digest. All correlations below are **for these two backup files**, not a claim of current-client-compatible corpus.

## Read-only join results

- NPC table has 7,195 physical data lines and 128 columns. The analyzer excludes empty-ID rows (230 nonempty placeholder lines; wholly blank tab rows are ignored) and 3 identical duplicate IDs; 6,664 unique nonblank NPC IDs remain. No duplicate ID with conflicting ShopID was accepted.
- 208 unique NPC IDs have a nonempty authored `ShopID`, referencing 132 unique catalog section IDs. Of the 208 NPC mappings: **111** reference a catalog section containing price mode 0, 1, or 2; **95** reference a section with merchandise but no normal price-mode rows (typically mode 3); **2** reference an existing section with no numeric merchandise row. No referenced section was absent in these backup files.
- Concrete examples, *not the user's confirmed clicked NPC*: `FuncNpc00602 → Shop_weapon_00403` (45 mode-1 rows; normal item-add frames are possible in the inspected source model), `FuncNpc00645 → Shop_yinpiao_00100` (32 mode-3 rows; Stage23 `openShopLocked` would skip them), `WorldNpc01205 → Shop_card_guoqingjie` (0 numeric merchandise rows). These values are from the actual backup tables, not invented packets, fields, or runtime IDs.
- Stage23 reconstructed `scene_lifecycle.go:477` builds a shop view then emits item-add frames only for price modes 0/1/2. **Conditional finding:** if the user's NPC resolves to one of the 95 mode-3-only sections *and uses this exact handler*, this handler would emit zero merchandise add frames. Mode 3 may belong to a different shop service; DO NOT turn those rows into ordinary purchasable items or alter the mode filter without verified latest-client behavior.
- The user's particular clicked NPC ID, its live service selector/ShopID, actual running EXE SHA256, runtime resource digest, startup logs, view-frame writes, UI rendering, and selected role's currency snapshot remain unknown. Neither the 95 nor the 2 classifications is an established diagnosis for their session.

## Reproducibility and safeguards

- New pure-Python tool: `tools/stage37_npc_shop_join_audit.py --npc-table <city_commonnpc.txt> --shop-ini <shop.ini> [--npc-id <EXACT_ID>]`. Only reads specified files; outputs aggregate counts and optionally one explicit NPC ID. Fails closed on malformed section headers, row widths, conflicting duplicate NPC IDs, missing schema, and malformed shop row classification. It deliberately does not output entire NPC tables, character data, passwords, full resources, or any packet selector.
- Tested locally against the two downloaded backup files: hashes and counts above; eight synthetic unit tests passed. The GitHub Actions `Stage37 NPC Shop Join Offline Audit` run `35076206677` completed SUCCESS on commit `cf6922f2d0562979b2e6401e2d5e29b92f191b09`: syntax, committed fixture tests, and explicit offline-only/production-disabled assertions. GitHub Actions uses synthetic fixtures; the connected Drive files are not copied into the public repository or CI.
- No Windows executable was built or user/client LIVE/E2E session performed in Stage25. The Stage23 build artifact remains a **partial** package without independently verified complete resources and safe data separation. No complete-server ZIP is available to present.

## Next evidence boundary

1. Determine the **actual clicked NPC ID** from a validated server/NPC interaction trace only after a version-consistent full test package exists; or independently establish one exact world NPC/ShopID mapping in the reconstructed server source. Do not treat an offline city-table ID as runtime identity merely because it shares text.
2. Verify whether server NPC function loading uses this exact `city_commonnpc.txt` and how the ShopID reaches `openShopLocked`; capture read-only logs for catalog open/parse, selected ShopID, mode counts, view create, planned and successful item-add frames. Avoid personal data and preserve the existing fail-closed purchase route.
3. Reconcile exact-current executable/resource manifest, verify a complete server ZIP, and inspect selected role currency in read-only fashion. No DB reset, guessed silver injection, handler enable, or purchase-button request while the shop is empty.
