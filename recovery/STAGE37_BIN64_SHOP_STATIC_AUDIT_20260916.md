# Stage37 — uploaded bin64 shop static-anchor audit (2026-09-16)

## Authority / scope

- GitHub working branch `stage37-current-recovery-20260916` was checked at `6987b736c7536957a69b0c129c5ac4ad7621ce14` before this audit. Preserve Stage34–36 ancestry; do not reset or substitute `9yin-go-server.zip` or legacy JYZJ.server as server authority.
- Source: user-uploaded `bin64(2)(1).zip` copied privately and read-only, 74 ZIP entries, CRC test PASS. No client DLL or EXE was executed, patched or committed. No user PC, game executable, DB or characters were changed.
- `fxgame.exe`: 8,772,368 bytes; SHA-256 `c7f0366026452d525dc0c74b8499aac30fdb69189591420978e12c7094b5fde3`.
- `fxgamelogic.dll`: 43,327,760 bytes; SHA-256 `16cf49b652c39b63ba0767780cc39f32ea1e50bdc683da9d5e7b2697cf95c3e8`.
- `fxnet2.dll`: 6,496,528 bytes; SHA-256 `0443d9f401fdcc6a92a391869c898780ffac93399913dc9a82c7cac8cf6c3fde`.
- `fxcore.dll`: 4,902,672 bytes; SHA-256 `ce0da0f52a80db5033e177a59be868c44b0baeca4a71226b0ccb2aca76cae724`.
- The archive contains all four required exact basenames once each. File availability and their hashes do **not** establish that the binaries belong to one up-to-date patch set or match a user's currently running client.

## Narrow, independently checked machine-code evidence

The supplied `fxgamelogic.dll` is PE32+ x86-64 with image base `0x11000000`. File offsets were mapped to virtual addresses using its own PE section table. The audit uses RIP-relative `LEA/MOV` candidate discovery, followed by separate `objdump -d -M intel` inspection around the four addresses. These are *static string references*, **not** verified network handlers, packet selectors, arguments, or a successful rendered UI.

| Exact NUL-terminated string | File offset | Image virtual address | Text references verified in this binary |
| --- | --- | --- | --- |
| `form_stage_main\form_shop\form_shop` | `0x1d102d8` | `0x12d116d8` | `0x1174a5b5`, `0x1174aef4` |
| `open_shop` | `0x1ce5aa0` | `0x12ce6ea0` | `0x11401b90` |
| `on_open_shop_exchange_form` | `0x1de23b0` | `0x12de37b0` | `0x12a16d37` |
| `on_single_shop_info` | `0x1de1e70` | `0x12de3270` | `0x12a176f5` |

At both general-form references, the assembly loads the same form-path address and calls `0x11017963`; the function's full semantics are **not** established. The `open_shop` name is loaded as a separate string in another code region and passed onward; its binding or behavior is not proved. The exchange and single-shop callback names are stored in a separate contiguous registration-like region; their runtime dispatch and packet layout are not proved. These three categories must not be conflated simply because each string contains `shop`.

A plain ASCII search for `shop.ini` in this DLL returned two substrings, embedded in `share\Rule\AttributeMall\clone_shop.ini` and `ini\ui\championshop\championshop.ini`. This *does not* demonstrate absence of the normal NPC catalog: package contents may be compressed or referenced by other mechanisms. The connected Drive `lua.package` and `ini.package` have `PCK0` headers but were **not extracted or decrypted in this audit**; literal string absence in their packed bytes proves nothing.

## Reproducibility / test boundary

- New read-only tool: `tools/stage37_shop_binary_anchor_audit.py`. It consumes a ZIP by explicit path, validates four unique required members, checks PE32+ bounds, locates exact NUL-terminated string anchors and calculates RIP-relative text-reference addresses; it prints a restricted JSON summary and never extracts or executes an EXE/DLL.
- Synthetic instruction + malformed-PE self-test: `SELF_TEST_PASS`.
- Actual supplied ZIP: CRC `PASS`, four real anchors checked with independent expected-address regression `4/4 PASS`; all four file hashes emitted. The reference-discovery scan is not a complete disassembler or control-flow reconstruction.
- No Go gameplay changes, GM code changes, server compilation, Windows build or LIVE/E2E test in this stage. `packet_selector_verified=false`, `purchase_handler_verified=false`, `live_or_e2e_verified=false`.

## Next evidence boundary

Use already acquired client packages only after *actual, validated extraction* to inspect the exact ordinary shop UI Lua and resource path, or use a Stage34-or-newer live trace tying the same chosen NPC's `NPC shop menu diagnostic` → `shop service selected` → `shop display catalog` → `shop display frames`/preflight to the observed client screen. Retain exact RAR `shop.ini` authority. Do not guess `Shop_GB_Yishiting` suffix, enable exchange purchase or mutate currency, bag, DB or GM grant logic. Purchase E2E must follow display verification, not be inferred from this static anchor audit.
