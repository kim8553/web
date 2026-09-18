#!/usr/bin/env python3
"""Read-only PCK0 header and explicitly nominated shop zlib-stream audit.

This tool NEVER treats an unindexed zlib hit as a named archive file or
chooses a preferred shop version. It does not decrypt the PCK file index.
Use only with locally supplied packages; it writes no extracted data.
"""

import argparse
import hashlib
import json
import mmap
from pathlib import Path
import re
import struct
import tempfile
import zlib

MAX_PACKAGE = 128 * 1024 * 1024
MAX_SHOP_STREAM = 6 * 1024 * 1024
CHUNK = 64 * 1024
SECTION = re.compile(rb"(?m)^\[([^\]\r\n]+)\]\r?$")


def pck_header(data):
    if len(data) < 19 or data[:4] != b"PCK0":
        raise ValueError("not a complete PCK0 header")
    record_length = struct.unpack_from("<H", data, 4)[0]
    if record_length != 15 or data[18] != 0:
        raise ValueError("unsupported primary header record")
    count, index_end = struct.unpack_from("<II", data, 10)
    if not 0 < count <= 0x100000 or not 19 <= index_end <= len(data):
        raise ValueError("invalid index count or bounds")
    # fxres.exe code at 0x14000dcc0/0x14000dce7/0x14000dcf1 checks
    # entry length, first filename byte, and terminating NUL. These bytes
    # describe only the RAW index view, not its decoded on-client view.
    first_length = struct.unpack_from("<H", data, 19)[0] if index_end >= 21 else 0
    first_directly_parseable = (
        first_length >= 28
        and 19 + first_length <= index_end
        and data[19 + 27] == 0
        and data[19 + first_length - 1] == 0
    )
    return {
        "declared_entry_count": count,
        "index_start": 19,
        "index_end_exclusive": index_end,
        "raw_first_entry_length": first_length,
        "raw_first_entry_directly_parseable": first_directly_parseable,
        "index_path_mapping_verified": False,
    }


def read_bounded_stream(data, offset, index_end):
    if not index_end <= offset < len(data):
        raise ValueError(f"candidate offset {offset} outside package data")
    decompressor = zlib.decompressobj()
    output = bytearray()
    cursor = offset
    while cursor < len(data):
        chunk = data[cursor:min(cursor + CHUNK, len(data))]
        try:
            part = decompressor.decompress(chunk, MAX_SHOP_STREAM + 1 - len(output))
        except zlib.error as exc:
            raise ValueError(f"candidate {offset}: invalid zlib stream") from exc
        output.extend(part)
        if len(output) > MAX_SHOP_STREAM:
            raise ValueError(f"candidate {offset}: decompressed size over limit")
        if decompressor.eof:
            cursor += len(chunk) - len(decompressor.unused_data)
            break
        if decompressor.unconsumed_tail:
            raise ValueError(f"candidate {offset}: decompression output limit")
        cursor += len(chunk)
    if not decompressor.eof:
        raise ValueError(f"candidate {offset}: truncated zlib stream")
    raw = bytes(output)
    if not raw.startswith(b"[Shop_"):
        raise ValueError(f"candidate {offset}: not an observed shop INI stream")
    sections = {}
    headers = list(SECTION.finditer(raw))
    for idx, match in enumerate(headers):
        name = match.group(1)
        if not name.startswith(b"Shop_"):
            continue
        if name in sections:
            raise ValueError(f"candidate {offset}: duplicate shop section header")
        stop = headers[idx + 1].start() if idx + 1 < len(headers) else len(raw)
        sections[name] = raw[match.start():stop]
    if not sections:
        raise ValueError(f"candidate {offset}: no shop sections")
    return {
        "offset": offset,
        "compressed_end_exclusive": cursor,
        "uncompressed_bytes": len(raw),
        "uncompressed_sha256": hashlib.sha256(raw).hexdigest(),
        "shop_section_count": len(sections),
    }, sections


def audit(package, offsets, expected_sha256=None):
    if len(offsets) != len(set(offsets)) or not offsets:
        raise ValueError("provide unique candidate offsets")
    package = Path(package)
    size = package.stat().st_size
    if not 19 <= size <= MAX_PACKAGE:
        raise ValueError("package size outside supported audit bounds")
    with package.open("rb") as stream, mmap.mmap(stream.fileno(), 0, access=mmap.ACCESS_READ) as data:
        sha = hashlib.sha256(data).hexdigest()
        if expected_sha256 is not None and sha.lower() != expected_sha256.lower():
            raise ValueError("package SHA-256 does not match requested source")
        header = pck_header(data)
        candidates = []
        parsed = []
        for offset in sorted(offsets):
            info, sections = read_bounded_stream(data, offset, header["index_end_exclusive"])
            candidates.append(info)
            parsed.append(sections)
        for first, second in zip(candidates, candidates[1:]):
            if first["compressed_end_exclusive"] > second["offset"]:
                raise ValueError("overlapping nominated compressed streams")
        changed = None
        only_one = None
        if len(parsed) == 2:
            keys_a, keys_b = set(parsed[0]), set(parsed[1])
            changed = sum(parsed[0][key] != parsed[1][key] for key in keys_a & keys_b)
            only_one = [len(keys_a - keys_b), len(keys_b - keys_a)]
        return {
            "package_size": size,
            "package_sha256": sha,
            "pck_header": header,
            "nominated_streams": candidates,
            "same_named_section_byte_differences": changed,
            "sections_only_in_each_stream": only_one,
            "authoritative_shop_stream": None,
            "shop_path_resolved_by_index": False,
            "shop_sale_protocol_verified": False,
            "client_live_verified": False,
        }


def self_test():
    import unittest

    class AuditTests(unittest.TestCase):
        def make_fixture(self, directory, broken=False):
            first = zlib.compress(b"[Shop_one]\r\n0=first,1,1,3,1,0,0\r\n")
            second = zlib.compress(b"[Shop_one]\r\n0=second,1,1,3,1,0,0\r\n")
            index_end = 40
            payload = b"PCK0" + struct.pack("<HIII", 15, 4, 2, index_end) + b"\0"
            payload += b"\0" * (index_end - len(payload))
            if broken:
                payload = b"NOPE" + payload[4:]
            offset_a = len(payload)
            payload += first
            offset_b = len(payload)
            payload += second
            p = directory / "fixture.package"
            p.write_bytes(payload)
            return p, offset_a, offset_b

        def test_two_candidates_never_select_by_position(self):
            with tempfile.TemporaryDirectory() as temp:
                p, a, b = self.make_fixture(Path(temp))
                result = audit(p, [a, b])
                self.assertEqual(result["same_named_section_byte_differences"], 1)
                self.assertIsNone(result["authoritative_shop_stream"])
                self.assertEqual(len(result["nominated_streams"]), 2)

        def test_rejects_wrong_source_and_index_offsets(self):
            with tempfile.TemporaryDirectory() as temp:
                p, a, b = self.make_fixture(Path(temp))
                with self.assertRaisesRegex(ValueError, "SHA-256"):
                    audit(p, [a], "0" * 64)
                with self.assertRaisesRegex(ValueError, "outside package data"):
                    audit(p, [19])
                with self.assertRaisesRegex(ValueError, "unique"):
                    audit(p, [a, a])
                p, a, b = self.make_fixture(Path(temp), broken=True)
                with self.assertRaisesRegex(ValueError, "PCK0"):
                    audit(p, [a, b])

        def test_rejects_truncated_candidate(self):
            with tempfile.TemporaryDirectory() as temp:
                p, a, b = self.make_fixture(Path(temp))
                p.write_bytes(p.read_bytes()[:a + 3])
                with self.assertRaises(ValueError):
                    audit(p, [a])

    suite = unittest.defaultTestLoader.loadTestsFromTestCase(AuditTests)
    result = unittest.TextTestRunner(verbosity=2).run(suite)
    if not result.wasSuccessful():
        raise SystemExit(1)


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--package", type=Path)
    parser.add_argument("--candidate-offset", action="append", type=int, default=[])
    parser.add_argument("--expected-sha256")
    parser.add_argument("--self-test", action="store_true")
    args = parser.parse_args()
    if args.self_test:
        self_test()
    else:
        if args.package is None or not args.candidate_offset:
            parser.error("--package and --candidate-offset are required")
        print(json.dumps(audit(args.package, args.candidate_offset, args.expected_sha256),
                         ensure_ascii=False, indent=2, sort_keys=True))
