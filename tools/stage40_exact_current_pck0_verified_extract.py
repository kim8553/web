#!/usr/bin/env python3
"""Read-only, hash-pinned, filename-blind recovery of six *verified* current
share.package payloads. Does not decrypt PCK0's index or infer unseen names,
server exchange rules, bind values, or permissions. No production server use.
"""
import argparse
import hashlib
import json
from pathlib import Path
import struct
import zlib

PACKAGE_SHA256 = "200497852ba3a29279e51f01e2913b5f7260480f2a740a32ebd1680b869844b6"
# Names are mappings from previously audited source-file hashes, NOT recovered
# package-index filenames. Only byte-for-byte SHA256 matches are accepted.
EXPECTED = {
    "shop.ini": "f42fce7631f7fe302d3fdedad60b12eaa20da12396de412ff0a84edcddc1a6d9",
    "exchangeitem.ini": "ed931884d8a8bb19bad512b5eb571fe8ac97b783449144d4a4834eb050f0dee6",
    "condition.ini": "b4fcbb6934617658af6f0e75f7e06734e1dd776b72d6fdf4b687643c1b778d3f",
    "skill_maxlevel.ini": "d4a35d90b84921f733afd632b15558bebe4ccdf474cc9ef5d5b6bc636bd99829",
    "condition_formula_a.ini": "9a4053f5bd4ac0e3895fe42a44db8bbbf44a469e0cf6fe226571dc3e7fcd1b57",
    "condition_formula_b.ini": "272bdce8cf161a70dbaea5fb828e7c4905e0ffff3402073d9c1909c895af9862",
}
MAX_PACKAGE_BYTES = 64 * 1024 * 1024
MAX_STREAM_INPUT = 2 * 1024 * 1024
MAX_STREAM_OUTPUT = 24 * 1024 * 1024


def recover(package: bytes, package_sha256: str, targets: dict[str, str]):
    """Return exactly authenticated streams and minimal metadata or raise.

    Both compression and Adler32 are checked by zlib; a header resemblance
    alone is never treated as a valid stream. Unmatched streams are discarded.
    """
    if not 19 <= len(package) <= MAX_PACKAGE_BYTES:
        raise ValueError("package size outside verified offline bounds")
    got = hashlib.sha256(package).hexdigest()
    if got != package_sha256:
        raise ValueError("original package SHA256 mismatch")
    if package[:4] != b"PCK0" or struct.unpack_from("<H", package, 4)[0] != 15 or package[18] != 0:
        raise ValueError("not the examined PCK0 version-15 primary header")
    declared = struct.unpack_from("<I", package, 10)[0]
    data_start = struct.unpack_from("<I", package, 14)[0]
    if not 0 < declared <= 1000000 or not 19 <= data_start < len(package):
        raise ValueError("invalid primary header count or data start")
    if not targets or len(set(targets.values())) != len(targets):
        raise ValueError("targets must have distinct expected hashes")
    by_sha = {digest: name for name, digest in targets.items()}
    found = {}
    pos = data_start
    checked = invalid = 0
    while pos < len(package) - 1:
        candidate = package.find(b"\x78", pos)
        if candidate < 0 or candidate + 1 >= len(package):
            break
        flg = package[candidate + 1]
        if ((0x7800 | flg) % 31) != 0 or (flg & 0x20):
            pos = candidate + 1
            continue
        checked += 1
        window = memoryview(package)[candidate:candidate + min(MAX_STREAM_INPUT, len(package) - candidate)]
        decoder = zlib.decompressobj()
        try:
            payload = decoder.decompress(window, MAX_STREAM_OUTPUT + 1)
        except zlib.error:
            invalid += 1
            pos = candidate + 2
            continue
        if not decoder.eof or len(payload) > MAX_STREAM_OUTPUT:
            invalid += 1
            pos = candidate + 2
            continue
        consumed = len(window) - len(decoder.unconsumed_tail) - len(decoder.unused_data)
        if consumed <= 0:
            invalid += 1
            pos = candidate + 2
            continue
        digest = hashlib.sha256(payload).hexdigest()
        name = by_sha.get(digest)
        if name:
            if name in found:
                raise ValueError("ambiguous duplicate authenticated payload: " + name)
            found[name] = ({"offset": candidate, "compressed_bytes": consumed,
                            "uncompressed_bytes": len(payload), "sha256": digest}, payload)
        pos = candidate + consumed
    missing = sorted(set(targets) - set(found))
    if missing:
        raise ValueError("missing exact-hash payloads: " + ", ".join(missing))
    report = {"package_sha256": got, "header_entry_count": declared,
              "index_end_from_header": data_start, "zlib_candidates_checked": checked,
              "invalid_or_oversized_candidates": invalid,
              "index_filenames_recovered": False, "full_archive_extracted": False,
              "verified_payloads": {name: found[name][0] for name in sorted(found)}}
    return report, {name: content for name, (_metadata, content) in found.items()}


def self_test():
    first = b"[shop]\r\n0=reward,1,3,0,0,0,0,8\r\n"
    second = b"[8]\r\nItem=required_item,1\r\n"
    stream_a, stream_b = zlib.compress(first), zlib.compress(second)
    header = b"PCK0" + struct.pack("<HIII", 15, 0x33a10004, 2, 64) + b"\x00"
    package = header + b"\xaa" * (64 - len(header)) + stream_a + b"\x00\x00random" + stream_b
    pinned = hashlib.sha256(package).hexdigest()
    expected = {"shop.ini": hashlib.sha256(first).hexdigest(),
                "exchangeitem.ini": hashlib.sha256(second).hexdigest()}
    report, recovered = recover(package, pinned, expected)
    assert recovered == {"shop.ini": first, "exchangeitem.ini": second}
    assert len(report["verified_payloads"]) == 2
    assert not report["index_filenames_recovered"] and not report["full_archive_extracted"]
    corrupted_stream = bytearray(package)
    corrupted_stream[64 + len(stream_a) - 1] ^= 1  # break zlib Adler32, not just a filename
    corrupted_stream = bytes(corrupted_stream)
    invalid_end = bytearray(package)
    struct.pack_into("<I", invalid_end, 14, len(package) + 1)
    invalid_end = bytes(invalid_end)
    cases = [
        (corrupted_stream, hashlib.sha256(corrupted_stream).hexdigest(), expected),
        (invalid_end, hashlib.sha256(invalid_end).hexdigest(), expected),
        (package, "0" * 64, expected),
        (b"PACK" + package[4:], hashlib.sha256(b"PACK" + package[4:]).hexdigest(), expected),
        (package, pinned, {"shop.ini": "0" * 64}),
        (package + stream_a, hashlib.sha256(package + stream_a).hexdigest(), expected),
    ]
    for content, digest, target_map in cases:
        try:
            recover(content, digest, target_map)
        except ValueError:
            pass
        else:
            raise AssertionError("tampered, absent or duplicate proof was accepted")
    print("SELF_TEST_PASS: authenticated bytes; rejects bad checksum, header bounds, wrong package, missing and duplicate stream")


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--package", type=Path, help="private exact-current res/share.package")
    parser.add_argument("--output-dir", type=Path, help="optional NEW local directory; never overwrite")
    parser.add_argument("--self-test", action="store_true")
    args = parser.parse_args()
    if args.self_test:
        self_test()
        return
    if args.package is None or not args.package.is_file():
        parser.error("--package must name the actual private share.package file")
    report, recovered = recover(args.package.read_bytes(), PACKAGE_SHA256, EXPECTED)
    if args.output_dir is not None:
        args.output_dir.mkdir(exist_ok=False)
        for name, payload in recovered.items():
            (args.output_dir / name).write_bytes(payload)
    print(json.dumps(report, indent=2, ensure_ascii=False, sort_keys=True))


if __name__ == "__main__":
    main()
