# Stage33: complete RAR shop catalog offline audit (2026-09-16)

Authority: user-supplied `9yin-go-server1.rar` resources, **not** the older `9yin-go-server.zip`; source at Stage32 on branch `stage37-current-recovery-20260916`. Only aggregate results are committed. No RAR resources, character data, client binaries, or user logs are uploaded to this public repository.

`resources/modern/share/trade/shop.ini` SHA-256: `ceb7a3a7e17cbc118ce5a7d3d979dea6120c50cdf6a5823697c05a5b1b9cf5dd`.

The exact Stage32 `loadShopCatalogSection`, Stage30 `stage30ShopExchangeDisplayRows`, Stage31 `stage31PrepareShopItemFrames` coordinate gate, and unmodified current-client `currentShopViewObjectIndex` were executed on every shop section in the RAR. Only a dummy frame callback was used for Stage31: this is a coordinate/duplicate test, **not** a client packet encoding or game-rendering test.

| Measure | Result |
| --- | ---: |
| Exact shop sections | 1,447 |
| Sections parsed with at least one item | 1,415 |
| Sections with no items | 32 |
| Other section parser errors | 0 |
| Authored item rows | 47,818 |
| Ordinary price modes 0/1/2 rows | 4,814 |
| Mode 3 exchange rows | 43,004 |
| Mode 3 rows eligible for opt-in view according to current gate | 42,955 |
| Non-ordinary-only parsed shops | 980 |
| Stage31 default coordinate/duplicate failures | 0 |
| Stage31 opt-in coordinate/duplicate failures on selected rows | 0 |
| Stage30 opt-in shops blocked by its unverified total-row capacity assumption | 49 |
| Stage30 opt-in shops blocked by an object-index collision | 1 |
| Stage30 opt-in shops with at least one exchange row selected | 1,076 |
| Stage30 opt-in exchange rows selected | 37,072 |
| Stage30 opt-in eligible exchange rows still excluded | 5,883 |

Interpretation: the Stage30 opt-in display gate does not cover all legitimate authored exchange rows, including some exchange-only shops; its 100-total-row check is a **conservative experiment**, not a proven current-client capacity rule. Do not silently delete products, infer a larger View capacity, or enable unproven exchange purchases. A rendered shop and purchase completion remain unverified.

The template-only Stage32 audit separately found 41 rows with 7 exact ShopIDs not present in `shop.ini`. That does not establish runtime impact because NPC creator overrides, scene membership and actual selection remain to be resolved. Stage33 CI produces a Linux executable of the exact Stage32 server source to run the server's existing `-npc-service-audit-scene` with the user's RAR resources locally, without uploading those resources to GitHub.

Unchanged: Stage29 map/Lua compatibility, Stage30 opt-in only, Stage31 full-frame preflight, Stage32 read-only NPC logs, fail-closed shop purchase, GM web grant deferred. No full live game E2E result is claimed.
