# Stage43 — skill_new.ini intake and resource boundary (2026-09-16)

## Authority and scope

- Source baseline: `stage37-current-recovery-20260916`, parent `ebac0676886deb7f6be959de5c5a7bec809ce143`.
- User-provided **local** `skill_new.ini` was inspected read-only. **Its contents were not added to this public GitHub repository or CI artifacts.** No GM web grant, NPC shop purchase, combat wire, or database mutation was modified.
- Upload provenance as the exact **current patched Snail client**: **NOT VERIFIED**. A matching filename is insufficient.

## Reproducible local intake results

Run locally: `python3 tools/stage43_skill_new_preflight.py /path/to/skill_new.ini`. The preflight outputs only aggregate statistics and a digest, not resource lines or identifiers.

| Metric | Observed |
| --- | --- |
| SHA-256 | `7d644fca3564515dd305a562e06c8cb1487fc5e68b652910e9f77d0a2c08c653` |
| Bytes | `1,687,076` |
| INI lines | `116,149` |
| Section headers / distinct sections | `17,562 / 17,562` |
| Duplicate section names | `0` |
| `script=SkillNormal` | `14,640` |
| `script=SkillLock` | `2,922` |
| Candidates under current `loadCombatSkillCatalog` script filter | `17,562` (not compiled/usable skill count) |
| Missing or nonnumeric `StaticData` in these candidates | `0` |
| NUL bytes / ignored malformed nonfield lines | `0 / 0` |
| Maximum raw line length | `39` bytes |
| UTF-8 validity | `false`; do **not** recode the original based on this fact. Current Go INI scanner is byte-tolerant for section/key storage. |
| Local Go scanner smoke (same loop and 4 MiB Scanner buffer as server) | `PASS` — 17,562 sections, no duplicates, same script counts |
| Local synthetic Python tests | `5 / 5 PASS` |
| GitHub synthetic-only CI | [run 35110885242](https://github.com/kim8553/web/actions/runs/35110885242) — `PASS` |

## What this does NOT establish

1. The current server `skill_catalog.go` loads **16** named INI inputs in `loadSkillResourceTables`: the supplied `skill_new.ini`, seven other skill definition tables (`skill_static.ini`, `skill_normal_varprop.ini`, `skill_lock_varprop.ini`, `skill_consume.ini`, `damage_calculate.ini`, `attack_hitshape.ini`, `attack_targetshape.ini`), five action tables (four `zhaoshi_player*.ini` variants and `zhaoshi_clone.ini`), and three buff tables (`buff_new.ini`, `buff_static.ini`, `buff_varprop.ini`). The remaining 15 were **not supplied or verified together** in this intake. The complete executable may need additional non-skill resources as well.
2. No full combat-skill compilation, exact-current client binary/resource correspondence, startup, LIVE, or E2E has been demonstrated by this intake.
3. The older supplied Go-server RAR lists a same-named file of `1,677,777` uncompressed bytes, a **different size**. This alone establishes the files are not byte-identical; the old RAR is **not** authority for current client behavior.
4. Stage42's ordinary shop purchase mutation remains **disabled** until its exact-current client wire and LIVE evidence are established. GM-web item grants remain on hold by user request.

## Next safe boundary

Continue NPC shop diagnosis against verified client resources and actual shop wire traces; validate the full skill-resource dependency set separately and privately before a resource-dependent server run. Never publish raw game assets or create fake `skill_new.ini`/action tables merely to make CI green.
