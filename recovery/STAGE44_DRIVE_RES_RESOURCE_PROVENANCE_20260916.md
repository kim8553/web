# Stage44 — connected Google Drive `res` resource provenance audit (2026-09-16)

## Scope and authority

- Parent baseline: `2e89e2b6dc8a1d87ed9fee57e9cf8f4df512322e` on `stage37-current-recovery-20260916`.
- Drive's **original-client `res/`** exists and contains `ini.package`, `lua.package`, `lua64.package`, `share.package`, etc. Two separate Drive `resources/modern` folders hold **unpacked server-side INIs**. Do not confuse these locations or infer their versions from names.
- Exact current-client gameplay authority remains `fxgame.exe`, `FxGameLogic.dll`, `FxNet2.dll`, `fxcore.dll` and resource bytes demonstrated to match that client. The older Go-server RAR is comparison evidence, **not** current gameplay authority.
- All original game resource bytes and any credentials remain outside the public GitHub repository and GitHub Actions. No GM-web grant, shop purchase, inventory mutation, combat network, or DB code was modified in this stage.

## Confirmed source distinctions

1. The unpacked Drive `resources/modern/share/trade/shop.ini` is **byte-identical** to the previous `9yin-go-server1` RAR's `resources/modern/share/trade/shop.ini`: each is 2,176,439 bytes, SHA-256 `ceb7a3a7e17cbc118ce5a7d3d979dea6120c50cdf6a5823697c05a5b1b9cf5dd`. It has 1,447 section headers and **no exact `[Shop_GB_Yishiting]` header**. An unrelated suffix variant must not be silently substituted; this legacy-matching INI cannot establish the exact-current client's NPC shop mapping.
2. Drive `res/ini.package` is 20,578,406 bytes, SHA-256 `6185812c6153b2a6da56071968dcd1254510a974df06a7e36a0229be2105779a`; `res/lua.package` is 21,840,086 bytes, SHA-256 `283c8c245a3fb86af2a7e8c21c53b3de26c30432341dd590c37554adfce04e97`. In both local copies PCK0 header's u16 at offset 6 is `4`; the examined external Rust `JiuYinUnpackTool/src/pck.rs` treats nonzero flags as an unsupported encrypted-index variant. This is **secondary parser evidence**, not a verified client decryption routine. Stage39 still has **zero decoded index entries and zero extracted live package resources**.
3. There are **three distinct `skill_new.ini` candidates**. Their SHA-256 values and results when combined **only for read-only comparison** with the Drive July other-15-file set are:

| Candidate | Bytes | SHA-256 | Supported-script sections | Level-one preflight | Missing static | Missing level-one varprop | Missing player action |
| --- | ---: | --- | ---: | ---: | ---: | ---: | ---: |
| Drive July folder | 1,677,777 | `8028418122d024ca3467e675939110a0ec7c4fc686f1a285c2df8af4f60e3ca2` | 17,466 | 4,652 | 2 | 34 | 12,778 |
| Drive August folder | 1,677,596 | `5d729549b916b4b158b3c87d049391ee3082f2bae038233b6761c7ecada8f83e` | 17,466 | 4,652 | 0 | 34 | 12,780 |
| User-provided upload | 1,687,076 | `7d644fca3564515dd305a562e06c8cb1487fc5e68b652910e9f77d0a2c08c653` | 17,562 | 4,652 | 98 | 34 | 12,778 |

The user upload contains **96 additional section names** compared with the July candidate; all 17,466 shared section bodies matched the July candidate in a normalized, line-based comparison. **All 96 additional sections point to a StaticData ID absent from the July `skill_static.ini`.** The August candidate has the same 17,466 section names as July but two shared section bodies differ. This demonstrates a concrete mixed-version dependency; it does not prove which candidate is the exact current client file.

## All 16 named skill loader inputs located and read

Read the actual [`server/cmd/protocol-probe/skill_catalog.go`](../server/cmd/protocol-probe/skill_catalog.go) load list, not a guessed resource list. The user-supplied `skill_new.ini` plus 15 Drive files were downloaded privately: `skill_static.ini`, `skill_normal_varprop.ini`, `skill_lock_varprop.ini`, `skill_consume.ini`, `damage_calculate.ini`, `attack_hitshape.ini`, `attack_targetshape.ini`, `zhaoshi_player.ini`, `zhaoshi_player_2.ini`, `zhaoshi_player_dodge.ini`, `zhaoshi_player_parry.ini`, `zhaoshi_clone.ini`, `buff_new.ini`, `buff_static.ini`, `buff_varprop.ini`. Their total input size was **48,151,722 bytes**. Files from the same July Drive resource tree were used for the remaining 15 inputs; provenance against the current patched client is **not verified**.

A local, **read-only, Go Scanner-compatible** (4 MiB token maximum; section/key/value trimming and action-file override + clone fallback modeled from the server) preflight read all 16 files and found 17,562 supported-script section IDs and 4,652 candidates satisfying its level-one `StaticData` / `skill_static` / level-one varprop / player-action checks. The 16-file scan counted 9,918 merged action sections. Five source files contain repeated section **headers**; the current Go loader *merges* these, so repeated headers alone are **not an error**. The preflight matched a separately implemented local audit; four synthetic-only checks of the reusable read-only auditor passed locally. This preflight is **not** the full `compileCombatSkill` execution, executable-server startup, gameplay success, or proof that these mixed files correspond to the exact client.

## Safe next step

1. Obtain verified read-only index decoding/extraction for the exact-current `res/ini.package` / `res/lua.package` using primary client binary evidence and a bounded, safe tool. Never silently treat the July/August unpacked folders as the live client's unpacked bytes.
2. Compare the *extracted exact-current* `skill_new.ini`, corresponding static/varprop/action data, and ordinary NPC shop Lua/config by SHA-256 and documented per-section differences to the Drive candidates. Select a coherent verified set **before** any resource-dependent server startup or gameplay change.
3. Continue ordinary NPC shop wire / NPC shop-ID validation without inventing suffixes, prices, purchase packets, or mutations. `ready=false` remains honest until actual purchase E2E. GM web grants remain **on hold at the user's request**.

The raw assets, client binaries, account information, and private Drive download references have **not** been committed. Preserve all earlier Stage37–43 server fixes and source lineage; this audit does not replace them.
