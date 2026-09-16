# Stage46 — newly uploaded BIN64 identity and private PCK0 index gate (2026-09-17 KST)

## Authority and scope

- Started from `stage37-current-recovery-20260916` HEAD `2665223e2dca0679d12f611e0808ca7675abd526`; keep the accumulated `server/` based on **`9yin-go-server1.rar`**, never substitute `9yin-go-server.zip` or historical JYZJ semantics.
- User's `bin64(2)(1)(1).zip` was materialized **privately** after a previous attachment-read blocker. Original ZIP: **146,467,400 bytes**, SHA-256 `167aeacadd22d551c252e8808a010a00080682de8d2b3fb5f21420ff4d75a306`, **74 ZIP members**, full `ZipFile.testzip()` **PASS**. No client binaries, unpacked resources or decrypted bytes were uploaded to GitHub.
- Independently read only original private Drive-derived `ini.package.bin` and `lua.package.bin`, previously SHA-confirmed in Stage45. These filenames denote temporary copies, not publicly committed game data.

## Identified binaries (case-insensitive archive member names)

| BIN64 member | bytes | SHA-256 |
| --- | ---: | --- |
| `fxgame.exe` | 8,772,368 | `c7f0366026452d525dc0c74b8499aac30fdb69189591420978e12c7094b5fde3` |
| `fxgamelogic.dll` | 43,327,760 | `16cf49b652c39b63ba0767780cc39f32ea1e50bdc683da9d5e7b2697cf95c3e8` |
| `fxnet2.dll` | 6,496,528 | `0443d9f401fdcc6a92a391869c898780ffac93399913dc9a82c7cac8cf6c3fde` |
| `fxcore.dll` | 4,902,672 | `ce0da0f52a80db5033e177a59be868c44b0baeca4a71226b0ccb2aca76cae724` |
| `fxres.exe` | 7,259,408 | `07ae76288148132995538488f12e2214fbecfdc0f18bdc2dd3189093e3e9fa9c` |

**Important**: this `fxres.exe` SHA-256 is *identical* to the previously audited Stage39 `fxres.exe`; all **13 Stage39 machine-instruction fingerprints** also matched in the new ZIP. It is not evidence of a newer decoder or that all components and packages are exactly from one patch build. `fxres.exe` can now be inspected without the previous Drive 403 download blocker.

## Additional same-ZIP resource topology evidence

- `fxpackage.dll` exists: **4,230,928 bytes**, SHA-256 `ac63e01378e5f46cd11c0f1f840f86bc32594545d7f13c807e81aadace5f1b47`. Its PE section table includes `.themida` and `.boot`; the export table advertises eight RVAs but `objdump` did not recover valid names. Static presence/packing is **not** proof that a particular export decrypts the original package index. Do not execute an unknown binary or infer its API by name alone.
- This ZIP's `packages.ini`: SHA-256 `667eb4f99e00359e36afb20f7e5b7ab13f73ea65133cf7ab3357c7e2f39d4687`; parsed with `configparser`, 52 named sections. Exact `[ini] File=res\\ini.package, Preload=1`, `[lua] File=res\\lua.package, Preload=0`, `[share] File=res\\share.package`.
- This ZIP's `fxres_packages.ini`: SHA-256 `cbf0039bfb255faeb0bfc497f0b25cc4ab4306992d2dfa75ceff7e1cf688ce26`; 27 named sections including `[ini] File=res\\ini.package` and `[share] File=res\\share.package`, **without `[lua]`**. This is a *package-path manifest*, **not a mapping from internal entry names to streams**, and cannot prove any particular `shop.ini` is present or which package owns it.
- A bounded scan found the literal `fxpackage.dll` once in `fxgame.exe`, but not in the four other identified binaries; this is not proof of how it is loaded or whether it contains the index decoder. `fxres.exe` imports did not reveal a static `FxPackage.dll` import in the inspected import table.

## Reproducible index-format incompatibility, not a decrypted index

- Authoritative earlier Stage39 machine code reads the 19-byte PCK0 primary header, declared count, and index end (`recovery/STAGE39_FXRES_PCK0_HEADER_CODE_AUDIT_20260916.md`). Do **not** interpret the primary record length 15 as an established game version. Machine code alone has **not** established semantics for the full u32 at file offset 6.
- The upstream public [JiuYinUnpackTool `src/pck.rs`](https://github.com/russell662/JiuYinUnpackTool/blob/main/src/pck.rs) reads a **u16 at offset 6 as `flags` and explicitly rejects nonzero flags**, and its *plain* entry parser expects a 27-byte fixed entry prefix. This is a statement about that tool, **not a proven interpretation of current client encryption**.
- Original `ini.package` (SHA-256 `6185812c6153b2a6da56071968dcd1254510a974df06a7e36a0229be2105779a`): offset-6 full u32 `0x33a10004`; low u16 `4`; naive plaintext first-record `length=8709` and declared data offset exceeds file bounds. Header declares **18,543 entries**, index `[19, 1,219,224)`; zero entries parsed.
- Original `lua.package` (SHA-256 `283c8c245a3fb86af2a7e8c21c53b3de26c30432341dd590c37554adfce04e97`): offset-6 full u32 `0x5e610004`; low u16 `4`; naive plaintext first-record `length=18757` and declared data offset exceeds file bounds. Header declares **2,286 entries**, index `[19, 187,177)`; zero entries parsed.
- Thus **the exact upstream plaintext-index parser must fail closed on both original packages**. The byte observations do not reveal the transform, key, name offsets, `shop.ini` provenance, or a valid current shop ID. Stage45's two valid *anonymous* first zlib streams remain anonymous; never label them named resources.
- Additional x64 disassembly of this same `fxres.exe`: around `0x14000dc9c`, execution contains a direct `call 0x1402070b6` into its nonstandard ``.`s9`` section, before the previously identified entry-loop at `0x14000dcaa`. The role of that call, including whether it decodes the index, is **unverified**. Do not derive XOR keys or algorithms from its location. No executable was run.
- Bounded case-insensitive literal scan of the five identified binaries found `shop.ini` twice in `fxgamelogic.dll`, in literal `share\\Rule\\AttributeMall\\clone_shop.ini` and `ini\\ui\\championshop\\championshop.ini`; neither is an established ordinary NPC shop catalog. Lack of a literal ASCII `trade\\shop.ini` is not evidence the client cannot construct it dynamically.

## Validation and state

- Committed Stage46 read-only probe [`tools/stage46_current_bin64_index_gate.py`](../tools/stage46_current_bin64_index_gate.py): Python syntax **PASS**; synthetic fixture tests for valid header / plaintext incompatibility plus seven invalid-header variants **PASS**; run against *actual private ZIP and both original package copies* **PASS** for ZIP CRC, five exact binary hashes, 13 machine-byte fingerprints, package hashes/header/unsupported-plaintext detection only. GitHub tool blob `0a4dbfa3ddd85c0bd363f3976bdbdae125144885` matched the locally tested Git blob hash. The probe emits metadata only and does not extract/decrypt or execute a game binary.
- New **Stage46 synthetic-only GitHub Actions** [run 35116772331](https://github.com/kim8553/web/actions/runs/35116772331) at commit `e7f8c9f51995906306461e58baf39b757fb37cc3`: **SUCCESS**; job `synthetic-read-only` and the Python syntax / synthetic fixture steps individually **SUCCESS**. CI has **no** private BIN64/INI/Lua data and cannot establish the actual package index was decoded. Stage45 CI success is separate earlier evidence.
- No changes to Go gameplay, GM web grants, shop catalog or readiness. No Stage46 Go build/race/vet, game login, NPC shop purchase, bag mutation, MySQL persistence or LIVE/E2E test. `PurchaseReady=false` remains fail-closed.

## Next grounded boundary

Investigate the *actual same-binary* index transformation and filename-to-stream association through validated execution paths or already-authorized verified code artifacts, especially the relation (if any) of protected/packed `fxres.exe` and `fxpackage.dll` routines. Do not force the legacy Rust parser or invent a key. Only after named package members can be independently located should private current-version shop resources be compared against Drive's previously unpacked `resources/modern/share/trade/shop.ini`, and only after ordinary shop request/response capture should Go purchase behavior be changed.
