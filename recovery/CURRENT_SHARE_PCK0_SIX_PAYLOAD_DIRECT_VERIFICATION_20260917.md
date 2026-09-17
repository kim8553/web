# Stage40 — direct latest `share.package` verification for 0x4F, 2026-09-17

## Authority and what was actually done

- Continue `kim8553/web` branch `stage37-assistant-handoff-20260917` from prior HEAD `b38f8143ee417f6b4d210b63a01af21c159c3e1d`. No original-server RAR, game binary, production Go source, resource or inventory was overwritten.
- Read the already downloaded, original **Google Drive `res/share.package`**, Drive file ID `1eoJ5ViSVdEY1xk8OKNcxmqHhcOmARQxh`, size **40,680,972 bytes**, SHA256 `200497852ba3a29279e51f01e2913b5f7260480f2a740a32ebd1680b869844b6`. Drive metadata rechecked the same file ID, name, size and parent `1GPzJ4RrC_TfcJLnH4sW41PB8HkNesOgQ`, modified `2026-09-12T12:41:54Z`. Do not treat this as a new patch beyond those actual file bytes.
- The examined PCK0 v15 primary header declares **9,726 entries** and a payload-start offset **790,911**. At precisely that offset, zlib decompression succeeded with checksum verification. The package index itself, including the original embedded filenames, is **not decrypted**. Stream scanning is deliberately *not* claimed to reconstruct the whole archive: 10,130 candidate streams were tested, 304 were rejected/overlarge, and zlib payload boundaries need not correspond one-to-one to header file entries.
- Instead of assuming an index filename, the actual decompressed **whole-file SHA256** of each selected stream was matched against the six separately documented `exactCurrent...SHA256` constants in `server/cmd/protocol-probe/latest_client_shop_condition_authority.go`. The names below are the prior source-file **hash labels**, not newly recovered index filenames. These are now independently proved to exist byte-for-byte in this exact current package.

## Six authenticated payloads (actual extracted bytes, not header-only guesses)

| Previously audited hash label | PCK0 absolute stream offset | Compressed bytes | Decompressed bytes | SHA256 |
|---|---:|---:|---:|---|
| `shop.ini` | 36877029 | 277336 | 2177326 | `f42fce7631f7fe302d3fdedad60b12eaa20da12396de412ff0a84edcddc1a6d9` |
| `exchangeitem.ini` | 28715362 | 82578 | 716215 | `ed931884d8a8bb19bad512b5eb571fe8ac97b783449144d4a4834eb050f0dee6` |
| `condition.ini` | 31562391 | 467209 | 4803460 | `b4fcbb6934617658af6f0e75f7e06734e1dd776b72d6fdf4b687643c1b778d3f` |
| `skill_maxlevel.ini` | 8676658 | 96955 | 1097891 | `d4a35d90b84921f733afd632b15558bebe4ccdf474cc9ef5d5b6bc636bd99829` |
| `condition_formula_a.ini` | 5604595 | 143740 | 785234 | `9a4053f5bd4ac0e3895fe42a44db8bbbf44a469e0cf6fe226571dc3e7fcd1b57` |
| `condition_formula_b.ini` | 9943317 | 143746 | 785220 | `272bdce8cf161a70dbaea5fb828e7c4905e0ffff3402073d9c1909c895af9862` |

Each selected decompressed byte sequence was rehashed after optional extraction into a *private local* directory; all six SHA256 values matched. `ini.package` was separately observed to contain a real valid zlib stream at its own primary payload start, but **no specific exchange INI is claimed from `ini.package`**. The exact latest resource blocker in the prior checkpoint is now resolved **for these six hashes only**.

## Direct content accounting — descriptive data, not transaction semantics

Parsed the hash-verified raw `shop.ini` **line by line** (do not deduplicate numeric row keys or replace multiple sections by an aggregate dictionary). Found **1,447 section headers**, **47,799 numeric listing rows**, **42,985 rows with price mode `3`**, of which **42,936** have nonzero ExchangeData, referring to **8,575 distinct ExchangeData IDs**. Every one of those 8,575 IDs has a corresponding section in the hash-verified `exchangeitem.ini` (10,585 sections). That file explicitly authors `BindStatus=1` in **131 distinct sections**; **388** nonzero mode-3 shop rows reference those sections. The 131 / 388 positive-binding rows must remain fail-closed: matching file hashes does not yield the missing runtime `ShowBind` or `ExchangeBind` values.

Concrete authored *configuration association*, not a claim about live debit/grant: shop section `Shop_menpai_xtc_01`, row key `1`, lists config `box_paiz_xtb_01`, amount `1`, mode `3`, ExchangeData `13093`; ExchangeItem section `13093` authors `Item=item_honor_school13,100;stuff_jiebang_01,1`. We **have not proven** from that text alone which bound/unbound items are consumed first, which final item binding is assigned, what Count multiplies, whether other conditions override the result, or which response and atomic DB transaction the official server uses. Do not turn this association into a settlement function.

Important exceptions found rather than silently inventing a zero-cost rule: four *referenced* ExchangeData definitions have neither `Item` nor `Prop`: IDs `1050`, `1051`, `15311`, `7395`. `1050` and `1051` author `AddValue`/`Type`; `15311` authors a condition; the section for `7395` is **entirely empty** and is referenced by **20** nonzero mode-3 rows. An empty section is not evidence of a free exchange or permission to grant. Do not blanket-reject all four without determining the native semantics either.

A separate **anonymous** authenticated zlib payload from the same package at byte offset `38258851`, decompressed size **20,899,623**, SHA256 `53dfeed238569978664b49fb7486855f4adc6c5b8a39d317fcac4d5d6a22a87c`, contains **76,879 INI section headers** and numerous item catalog-like definitions. Of **1,059 distinct `Item`-field identifier tokens** referenced by the 8,575 ExchangeData sections, **971** occur as sections in this one anonymous payload, and **40,656 / 42,936** mode-3 listing result identifiers occur there. This is a *limited content cross-reference*, not authentication of a particular `tool_item.ini` filename: its original package index name and other item/equipment catalogs are not verified. The 88 other tokens and other results are **not proven missing from the client**.

## Reproducible code and tests

- `tools/stage40_exact_current_pck0_verified_extract.py`: offline-only, hash-pinned, filename-blind stream verification. It verifies the entire source package SHA256 and PCK0 v15 header, bounds compressed/decompressed bytes, requires zlib end-of-stream/checksum and an exact SHA256 match, rejects absent/duplicated target streams, and never uses untrusted package filenames as filesystem paths. By default it **writes nothing**; optional `--output-dir` must be a new private local directory. Only the six authenticated outputs can be written. No package binaries or extracted copyrighted contents were committed.
- Exact local execution: `python3 tools/stage40_exact_current_pck0_verified_extract.py --self-test`, and `python3 tools/stage40_exact_current_pck0_verified_extract.py --package <private-local-share.package>`. Both completed successfully against respectively synthetic failure fixtures and the **real** package whose SHA is above. The optional output files' individual digests were also checked independently.
- `.github/workflows/jiuyin-stage40-current-pck0-proof.yml`: https://github.com/kim8553/web/actions/runs/35222522269 **SUCCESS**; synthetic wrong-package/header/checksum/missing/duplicate rejection and six expected hashes agreeing with actual Go source were tested. The private 40 MB package is **NOT** checked into GitHub, so Actions **did not** run a real-package extraction. The real-package proof is the separately executed, hash-verified local operation in this checkpoint.
- No production Go purchase code changed in Stage40. Production `0x4F` debit, grant, bag writes and currency writes remain **disabled**; Windows client LIVE, SQL atomicity, failure response and reconnect E2E remain **NOT RUN**.

## Next exact continuation — do not guess or enable items

Investigate the current-client/native definitions and actual authorized 557/0x4F observations for `Item`/`Prop` consumption type, `AddValue` behavior, purchase Count multiplication, result type/amount/binding and the 131 positive-binding definitions. Find definitive original index filename evidence for the anonymous item-catalog payload before identifying it as a particular INI. Then design an isolated, rollback-safe inventory/DB transaction and tests for insufficient materials, full bag, repeated request, write failure, reconnect, and postcommit client view replication. A Go build or authenticated INI digest alone proves none of those runtime effects.
