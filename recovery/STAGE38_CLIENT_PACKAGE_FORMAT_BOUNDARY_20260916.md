# Stage37 Stage38 — client package manifest / PCK0 compatibility boundary (2026-09-16)

## Authority and preservation

- Working branch `stage37-current-recovery-20260916` HEAD checked at `2c666eb65e6092d3604ba01ff63951af38cd28cb` before this pass. Stage34–37 lineage retained; no reset, no use of `9yin-go-server.zip` as the original source.
- Inputs were the user's uploaded `bin64(2)(1).zip` (read-only) and copies of `res/lua.package` and `res/ini.package` already obtained from the connected Google Drive. The original `9yin-go-server1.rar` remains the server baseline. No proprietary binary, package, raw log, credential, or account/character record was committed.
- Scope is exact manifest/header provenance. This is **not** unpacking, decrypting, a full disassembly, a packet trace, a server patch, or a game-client test.

## Confirmed client manifest and available files

- The uploaded ZIP contains `packages.ini` and `fxres_packages.ini` once each, plus `fxpackage.dll` and `fxres.exe` once each. Parsing both configuration files confirms exact package paths `res\\lua.package`, `res\\lua64.package`, `res\\ini.package`, `res\\gui.package`, and `res\\share.package`. The shared `ini`, `gui`, and `share` entries agree between the two lists; `fxres_packages.ini` need not list `lua`.
- Connected Drive `res` folder has the named `lua64.package`, `gui.package`, and `share.package` as separate items; no content from these three was downloaded in this pass. `gui.package` is listed as 1,287,874,519 bytes. This metadata does not prove the internal path `form_stage_main\\form_shop\\form_shop` exists within it.
- `fxpackage.dll`: 4,230,928 bytes, SHA-256 `ac63e01378e5f46cd11c0f1f840f86bc32594545d7f13c807e81aadace5f1b47`; literal ASCII `PCK0` occurrences: 0. `fxres.exe`: 7,259,408 bytes, SHA-256 `07ae76288148132995538488f12e2214fbecfdc0f18bdc2dd3189093e3e9fa9c`; literal `PCK0` occurrences: 1. These are string counts, **not** identification of the package parser or evidence that a DLL lacks one.

## Direct header comparison; no inferred decoding

| Input | SHA-256 | First 8 bytes | u16 at 0x04 / 0x06 | Contents extracted |
| --- | --- | --- | --- | --- |
| `res/lua.package` | `283c8c245a3fb86af2a7e8c21c53b3de26c30432341dd590c37554adfce04e97` | `50 43 4b 30 0f 00 04 00` | 15 / 4 | NO |
| `res/ini.package` | `6185812c6153b2a6da56071968dcd1254510a974df06a7e36a0229be2105779a` | `50 43 4b 30 0f 00 04 00` | 15 / 4 | NO |

- The published [Ersanio/wushu-utils `Unpacker.cs`](https://github.com/Ersanio/wushu-utils/blob/main/Wushu.Utils.Package/Unpacker.cs) checks a fixed first 10 bytes `PCK0 0f 00 00 00 00 00`; **both input packages fail this exact check** at offset 0x06. It would be incorrect to claim extraction based on this legacy decoder.
- The separately published [Ekey/WC2.PACKAGE.Tool `PackageUnpack.cs`](https://github.com/Ekey/WC2.PACKAGE.Tool/blob/main/WC2.Unpacker/WC2.Unpacker/FileSystem/Package/PackageUnpack.cs) requires its own package format's version 20 and patch version 4; the observed input u16 at offset 0x04 is 15. That code was written for a different game and is **not** adopted as an Age of Wushu specification. The `4` at input offset 0x06 is only a measured u16, not a demonstrated matching WC2 patch-version field.
- Bytes after the observed header prefix have not been interpreted as valid record counts, encryption keys, compression methods, or file offsets. Neither `Shop_GB_Yishiting` nor any ordinary-shop Lua path is certified present/absent inside these still-packed files.

## Read-only code and validation

- Added `tools/stage38_package_compatibility_probe.py`: reject duplicate/missing ZIP manifest/binary names; parse bounded INI files; compare listed package paths; hash two package copies; compare exact PCK0 header to the two published legacy assumptions without unpacking, extracting, executing, or altering any file. It reports explicit `false` for extraction, client shop Lua identification, NPC mapping, patch-coherence, and LIVE/E2E.
- Local synthetic test `--self-test`: `SELF_TEST_PASS` (realistic v15/4 fixture; corrupted magic rejected; legacy-header match control; manifest-path fixture; duplicate config rejected). `python -m py_compile`: PASS. Actual uploaded ZIP + two Drive package copies: read-only preflight PASS, hashes and paths as above. Verified the committed tool's Git blob SHA against `git hash-object` of the locally tested file: `d0af1a7a77b8f2eae55e6f3bbed8127e7f2b9c1a`.
- The previously verified ZIP CRC and four `fxgamelogic.dll` shop string references remain Stage37 static findings, not dynamic shop success. **No Go tests, race/vet, Windows build, server/GM change or LIVE/E2E in Stage38.**

## Next concrete evidence boundary

1. Obtain an actually compatible, validated reader for **these specific v15/4 package bytes** (e.g., independently review original client's package loader / an available first-party matching unpacker) before interpreting file listings or shop Lua. Do not execute unknown downloaded tools or force unrelated formats. The private Drive already contains the relevant packages; avoid requesting repeat uploads.
2. Alternatively, inspect a real Stage34-or-newer same-NPC server trace from `NPC shop menu diagnostic` through `shop display catalog` to `shop display frames`/preflight and the actual client-visible screen. This is the remaining direct way to establish the current display failure before purchase tests.
3. Do not automatically choose `Shop_GB_Yishiting_1`...`_5`, import JYZJ XML price or currency mappings, enable exchange purchase, or edit GM grant, DB, bag or the user's PC. Display, purchase and reconnect persistence remain unverified.
