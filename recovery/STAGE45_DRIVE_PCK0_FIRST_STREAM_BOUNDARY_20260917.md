# Stage45 — private Drive PCK0 first-stream boundary and NPC shop evidence (2026-09-17 KST)

## Authority, access and scope

- Parent baseline: `dd7f69401b8beeffaec5a4634d8c88d4f3078abe` on `stage37-current-recovery-20260916`; retain the cumulative `server/` tree based on `9yin-go-server1.rar`. No code rollback.
- The *private* Drive `res/ini.package` and `res/lua.package` were fetched into a temporary local execution environment. Both full-file SHA-256 digests match Stage44. Their bytes and extracted content are NOT in this repository, this report, or public Actions artifacts.
- The primary header interpretation is limited to the existing **Stage39** `fxres.exe` machine-code fingerprint (`recovery/STAGE39_FXRES_PCK0_HEADER_CODE_AUDIT_20260916.md`, `tools/stage39_pck0_header_code_probe.py`); the new probe does not extend that fingerprint to index decoding.
- Direct Drive fetch of the discovered `bin64/fxres.exe` was rejected by Drive with HTTP 403, `cannotDownloadAbusiveFile`. Its *existence* was verified; a fresh copy was not obtained or newly disassembled. Do not bypass the provider block, guess decryption or request repeat uploads before checking existing authorized evidence.

## Read-only package observations

| Observation | `res/ini.package` | `res/lua.package` |
|---|---|---|
| Byte count | 20,578,406 | 21,840,086 |
| Full SHA-256 | `6185812c6153b2a6da56071968dcd1254510a974df06a7e36a0229be2105779a` | `283c8c245a3fb86af2a7e8c21c53b3de26c30432341dd590c37554adfce04e97` |
| Primary record length | 15 | 15 |
| Index start offset | 19 | 19 |
| Index end (exclusive) / first data-stream offset | 1,219,224 | 187,177 |
| **Declared** index entry count, NOT parsed/verified | 18,543 | 2,286 |
| First **anonymous** zlib-stream compressed length | 105 | 1,342 |
| First anonymous stream decompressed length | 161 | 2,035 |
| SHA-256 of first decompressed anonymous stream | `7c8d1d6c65104c036d149155de103b3c231ec2935bd3b5b7e17b03161d5142d5` | `1d6daade801c4783baaba7dcdbeeb9247a59a10459e31d2b18ac9f8d226e63f5` |
| Prefix-only classification | INI-section-like (`[`) | Lua 5.1 bytecode-signature-like (`1b 4c 75 61 51`) |

A bounded zlib decoder reached EOF with a valid checksum for **only the first stream at the confirmed data boundary** in each package. These anonymous streams are **NOT named package members**: the raw index was not decrypted or parsed, no pathname or file-to-stream mapping was established, and neither stream is asserted to be `shop.ini`, `skill_new.ini` or any other particular resource. Prefix classification is not a full file-format validation. The declared count is not a count of successfully recovered files. No proprietary decompressed bytes are published.

## Reproducibility and tests

- Committed tool: `tools/stage45_pck0_first_stream_probe.py`. Bounded first stream to 8 MiB compressed/4 MiB decompressed and input package to 128 MiB. It fails closed for unexpected primary headers, out-of-range index, truncated or corrupt zlib stream, compressed-limit overflow and decompression-limit overflow. Reports offsets, lengths, hashes and prefix class only; writes no extracted files.
- Local Python syntax check: **PASS**. Synthetic fixture self-test: **PASS**, including valid INI/Lua prefixes and negative header, count, boundary, truncation, zlib checksum, decompression bomb and compressed-cap tests. Read-only execution on both private original packages: **PASS for the stated anonymous first-stream observations only**.
- Stage44 Actions succeeded previously, but that does not constitute a Stage45 CI result. New Stage45 Actions, Go test, race, vet, Windows amd64 build, server boot, game login, NPC shop LIVE/E2E: **NOT RUN / NOT VERIFIED** here. No server gameplay code was changed.

## NPC shop source-level boundary

- `server/cmd/protocol-probe/shop_catalog.go` opens the configured `resources/modern/share/trade/shop.ini` and matches the **exact** `[shopID]` section; it rejects absent/empty catalogs. Stage44 determined the Drive unpacked `shop.ini` is byte-identical to the legacy RAR file and lacks exact `[Shop_GB_Yishiting]`. Similar suffixes must not substitute.
- `server/cmd/protocol-probe/npc_service_audit.go` explicitly distinguishes `catalog_ready` from purchase `ready`, with `shopreadiness.FromCatalogError`; catalog presence alone does not authorize purchasing.
- `server/cmd/protocol-probe/latest_client_shop_exchange_gate.go` checks independent condition, property, materials, capacity, binding and persistence evidence. `latest_client_shop_wire_trace.go` is bounded *observation only*, does not establish an ordinary shop selector, mutate state or send replies.
- The current package's named shop resource, exact NPC shop ID and ordinary buy packet/currency contract remain **UNKNOWN**; no current-package-vs-unpacked shop SHA comparison is possible yet. Preserve fail-closed `ready=false`. **No GM web item-grant source or tests were touched.**

## Next evidence boundary

Recover actual index parsing and filename-to-stream mapping using *verified same-version client machine code/call paths* or previously retained authoritative client analysis, without guessing a key/XOR or publishing binaries. Then extract only relevant named current-package shop INI/Lua privately and compare hashes/sections against Drive's unpacked candidates. Trace NPC interaction -> shop menu -> ordinary buy request separately before making a server gameplay change. Do not label build success, static preflight or anonymous zlib success as NPC shop LIVE/E2E.
