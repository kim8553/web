#!/usr/bin/env python3
"""Compare a separately supplied server shop.ini with two nominated package streams.

Read-only provenance audit. Never infer archive path, loader precedence, or the
active client file from content similarity or stream offset.
"""
import argparse
import hashlib
import json
import mmap
from pathlib import Path
import re
import tempfile
import zlib

import audit_share_shop_pck as package_audit

MAX_LOOSE = 6 * 1024 * 1024
HEADER = re.compile(rb"(?m)^\[([^\]\r\n]+)\]\r?$")


def sections(raw):
    matches = list(HEADER.finditer(raw))
    if not matches or matches[0].start() != 0:
        raise ValueError("shop data missing initial section")
    result = {}
    for n, match in enumerate(matches):
        name = match.group(1)
        if name in result:
            raise ValueError("duplicate section header")
        end = matches[n + 1].start() if n + 1 < len(matches) else len(raw)
        result[name] = raw[match.start():end]
    return result


def stream_bytes(data, start, index_end):
    if not index_end <= start < len(data):
        raise ValueError("candidate offset outside package data")
    decompressor = zlib.decompressobj()
    raw = bytearray()
    cursor = start
    while cursor < len(data):
        chunk = data[cursor:min(cursor + package_audit.CHUNK, len(data))]
        try:
            part = decompressor.decompress(chunk, MAX_LOOSE + 1 - len(raw))
        except zlib.error as exc:
            raise ValueError("invalid nominated zlib stream") from exc
        raw.extend(part)
        if len(raw) > MAX_LOOSE or decompressor.unconsumed_tail:
            raise ValueError("nominated stream exceeds size limit")
        cursor += len(chunk) - len(decompressor.unused_data) if decompressor.eof else len(chunk)
        if decompressor.eof:
            return bytes(raw), cursor
    raise ValueError("truncated nominated zlib stream")


def audit(package_path, loose_path, offsets, package_sha, loose_sha):
    if len(offsets) != 2 or len(set(offsets)) != 2:
        raise ValueError("exactly two distinct nominated offsets required")
    package_path = Path(package_path)
    loose_path = Path(loose_path)
    if not 0 < loose_path.stat().st_size <= MAX_LOOSE:
        raise ValueError("loose shop file size outside audit limit")
    loose_raw = loose_path.read_bytes()
    loose_digest = hashlib.sha256(loose_raw).hexdigest()
    if loose_digest != loose_sha.lower():
        raise ValueError("loose shop SHA-256 mismatch")
    package_report = package_audit.audit(package_path, offsets, package_sha)
    reference = sections(loose_raw)
    candidates = []
    with package_path.open("rb") as source, mmap.mmap(source.fileno(), 0, access=mmap.ACCESS_READ) as data:
        index_end = package_report["pck_header"]["index_end_exclusive"]
        for candidate in package_report["nominated_streams"]:
            raw, end = stream_bytes(data, candidate["offset"], index_end)
            if (end != candidate["compressed_end_exclusive"] or
                    hashlib.sha256(raw).hexdigest() != candidate["uncompressed_sha256"]):
                raise ValueError("nominated stream integrity disagreement")
            parsed = sections(raw)
            shared = set(reference) & set(parsed)
            different = sorted(name.decode("utf-8", "backslashreplace") for name in shared if reference[name] != parsed[name])
            candidates.append({
                "offset": candidate["offset"],
                "uncompressed_sha256": candidate["uncompressed_sha256"],
                "section_count": len(parsed),
                "shared_section_count": len(shared),
                "different_section_count": len(different),
                "different_section_names": different,
                "only_in_loose_count": len(set(reference) - set(parsed)),
                "only_in_stream_count": len(set(parsed) - set(reference)),
            })
    return {
        "loose_shop_sha256": loose_digest,
        "loose_section_count": len(reference),
        "package_sha256": package_report["package_sha256"],
        "nominated_streams": candidates,
        "server_default_path_content_verified_on_user_pc": False,
        "shop_path_resolved_by_package_index": False,
        "client_active_stream": None,
        "resources_replaced": False,
    }


def self_test():
    import unittest

    class Tests(unittest.TestCase):
        def fixture(self, root):
            first = b"[Shop_one]\n0=first\n[Shop_two]\n0=same\n"
            second = b"[Shop_one]\n0=second\n[Shop_two]\n0=same\n"
            index_end = 40
            import struct
            package = b"PCK0" + struct.pack("<HIII", 15, 4, 2, index_end) + b"\x00"
            package += b"\x00" * (index_end - len(package))
            a = len(package)
            package += zlib.compress(first)
            b = len(package)
            package += zlib.compress(second)
            p, l = root / "example.package", root / "shop.ini"
            p.write_bytes(package)
            l.write_bytes(first)
            return p, l, (a, b), hashlib.sha256(package).hexdigest(), hashlib.sha256(first).hexdigest()

        def test_distinct_sources_without_precedence(self):
            with tempfile.TemporaryDirectory() as tmp:
                p, l, offsets, ps, ls = self.fixture(Path(tmp))
                outcome = audit(p, l, offsets, ps, ls)
                self.assertIsNone(outcome["client_active_stream"])
                self.assertFalse(outcome["shop_path_resolved_by_package_index"])
                self.assertEqual([s["different_section_count"] for s in outcome["nominated_streams"]], [0, 1])

        def test_rejects_unverified_source_or_offsets(self):
            with tempfile.TemporaryDirectory() as tmp:
                p, l, offsets, ps, ls = self.fixture(Path(tmp))
                with self.assertRaisesRegex(ValueError, "SHA-256"):
                    audit(p, l, offsets, ps, "0" * 64)
                with self.assertRaisesRegex(ValueError, "SHA-256"):
                    audit(p, l, offsets, "0" * 64, ls)
                with self.assertRaisesRegex(ValueError, "two distinct"):
                    audit(p, l, (offsets[0], offsets[0]), ps, ls)
    result = unittest.TextTestRunner(verbosity=2).run(unittest.defaultTestLoader.loadTestsFromTestCase(Tests))
    if not result.wasSuccessful():
        raise SystemExit(1)


if __name__ == "__main__":
    ap = argparse.ArgumentParser(description=__doc__)
    ap.add_argument("--package", type=Path)
    ap.add_argument("--loose-shop", type=Path)
    ap.add_argument("--candidate-offset", action="append", type=int, default=[])
    ap.add_argument("--expected-package-sha256")
    ap.add_argument("--expected-loose-sha256")
    ap.add_argument("--self-test", action="store_true")
    args = ap.parse_args()
    if args.self_test:
        self_test()
    else:
        if not all((args.package, args.loose_shop, args.expected_package_sha256, args.expected_loose_sha256)):
            ap.error("--package --loose-shop and both --expected-*-sha256 are required")
        print(json.dumps(audit(args.package, args.loose_shop, args.candidate_offset,
                               args.expected_package_sha256, args.expected_loose_sha256),
                         indent=2, ensure_ascii=False, sort_keys=True))
