# Stage44 — Type/AddValue topology and transaction preconditions (2026-09-17)

## Provenance and unchanged production behavior

- Continue `kim8553/web` branch `stage37-assistant-handoff-20260917` from Stage43 HEAD `48e68ed6940d747714eb20670e979a2d7f434ab2`. Do **not** substitute the older `九阴服务端.zip`, older RAR server, V37/V46/JYZJ, or their resources for the exact-current Snail client.
- Local inputs were the already hash-verified current `shop.ini` SHA256 `f42fce7631f7fe302d3fdedad60b12eaa20da12396de412ff0a84edcddc1a6d9`, current `exchangeitem.ini` SHA256 `ed931884d8a8bb19bad512b5eb571fe8ac97b783449144d4a4834eb050f0dee6`, and read-only current `FxGameLogic.dll` SHA256 `16cf49b652c39b63ba0767780cc39f32ea1e50bdc683da9d5e7b2697cf95c3e8`. Private file bytes are not committed. Authenticated file labels follow the Stage40 known-content extraction; the encrypted package index filename is not newly proven.
- No production Go server file, purchase handler, MySQL table, executable, client binary or Lua file was changed by Stage44. Purchase 0x4F remains fail-closed, and LIVE/E2E/reconnect were NOT RUN.

## Native evidence: shape and binding names, NOT server settlement

- The Stage43 native `exchange_item_manager::InitCurExchangeData` at VA `0x11B15560` reads 11 S2C-557 config fields; numeric `Type` is written at `0x11B15695` to object DWORD `+0x88`, and authored `AddValue` string is copied at `0x11B156E9..0x11B156F1` to `+0x90`. The actual current DLL also contains adjacent method metadata names `GetType` (file offset `0x1d3cf18`, VA `0x12d3e318`) and `GetAddValue` (file offset `0x1d3cf58`, VA `0x12d3e358`); these names alone do not prove any Lua call site or the getters' gameplay usage. No full getter-xref/official server handler has yet been recovered.
- The client-side form/config read, native metadata, and current INI values are **not evidence** that `Type=1` is a currency, `Type=2` is a recipe, `Type=3` is a property, or that `AddValue` means debit/reward. Do not encode those guesses.

## Exact-current authoritative cross-matrix (offline read only)

- Mode-3 shop rows: **42,985**; `ExchangeData=0` rows: **49**; nonzero referenced rows: **42,936**; distinct referenced definitions: **8,575**.
- Definitions by authored `Type` and presence of `Item` / `Prop`:

| Type authored | Item only | Prop only | Item + Prop | Neither | Total definitions |
|---|---:|---:|---:|---:|---:|
| missing | 6,746 | 1,173 | 128 | 2 | 8,049 |
| 1 | 0 | 0 | 0 | 2 | 2 |
| 2 | 41 | 0 | 0 | 0 | 41 |
| 3 | 144 | 329 | 10 | 0 | 483 |
| **Total** | **6,931** | **1,502** | **138** | **4** | **8,575** |

- Consequently **154 Type=3 definitions contain an Item field**, ten of them alongside Prop. Type cannot safely be treated as a simple Item-vs-Prop debit selector.
- `AddValue` occurs in exactly 43 referenced definitions: `Type=1` with authored values `10000000` and `15000000` (one each), and `Type=2` with authored `2000` (41 definitions). No referenced `Type=3` definition has AddValue. This is data correlation only, not payment direction or gameplay semantics.
- Exactly four referenced definitions have neither Item nor Prop: **1050**, **1051** (both author Type=1 and AddValue), **15311** (condition-only), **7395** (empty, referenced by 20 listings). Do not make an empty ingredient list automatically free or grant anything.
- Authored nonzero BindStatus affects **131** definitions / **388** shop rows and retains its unresolved runtime `ShowBind` / `ExchangeBind` gate.
- Prop contains **37 distinct authored property names** among the referenced definitions; `SchoolContribute`, `SchoolDanceTotalScore`, and `WGJobSkillPoint` occur, so a settlement path handling only `CapitalType*` is insufficient. This is a string census, not confirmation of balance storage or debit behavior.

## Concrete tooling and checks

- Added `tools/stage44_exchange_special_field_audit.py`: SHA256-gates both exact-current inputs, counts mode3 rows and unique definition shapes separately, rejects duplicate exchange definitions/keys and broken or nonnumeric references, and emits JSON. It never modifies the provided files and never guesses settlement semantics.
- Added `tools/test_stage44_exchange_special_field_audit.py`: eight tests with synthetic data only, including cross-matrix weighting, SHA mismatch, malformed/missing refs, duplicates and unknown Type handling.
- Local actual-private-data assertions passed for exact row/definition totals, all Type matrix cells, 43 AddValue definitions, four neither/unsupported definitions, and 131/388 binding cases. Local `python -m unittest -v` passed 8/8.
- GitHub CI [.github/workflows/jiuyin-stage44-special-field-audit.yml](../.github/workflows/jiuyin-stage44-special-field-audit.yml) executes ONLY the synthetic tests; [Actions 35233597826](https://github.com/kim8553/web/actions/runs/35233597826) completed **SUCCESS**. GitHub Actions did **not** receive either private current INI, run the game or test SQL purchases. A Stage44 Windows build is **NOT RUN** because this stage did not change Go server production source; Stage43's separate cross-build result remains separately recorded.

## Transaction gates and exact next work

1. Stage43 confirmed production `handleShopExchangeContract` resolves 0x4F mode3 and condition observations, then exits with no debit/grant; retain this fail-closed behavior.
2. Stage43 confirmed `mysqlBagStore.Save` has a transaction limited to deleting/reinserting `role_bag_items` and committing that snapshot, **not** one atomic transaction across ingredient inventory, Prop/currency and reward. `mysqlBagStore.Load` reports `len(items)!=0` rather than explicit empty-snapshot existence, so investigate callers before modifying it. The absence of `NINEYIN_MYSQL_DSN` makes `openGameDataDB` return a nil DB; do not imply durable MySQL persistence merely because a SQL driver is linked.
3. Next: recover actual authoritative semantics of nonempty `Item`/`Prop` alongside Type/AddValue (including combination cases), source of player-owned values, binding decision and multiply-by-Count rules. Then design a role-serialized immutable snapshot, one SQL `BeginTx`/rollback for all relevant stores, explicit idempotency, committed-result replication and reconnect checks. No `0x4F` mutation until each prerequisite has independently supported evidence and tests.
