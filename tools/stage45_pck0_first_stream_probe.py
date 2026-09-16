#!/usr/bin/env python3
"""Fail-closed, read-only PCK0 header and FIRST ANONYMOUS zlib-stream audit.

Offsets are limited to the primary-header layout verified by Stage39's fxres
machine-code fingerprint. This tool DOES NOT decode the index, identify file
names, or extract a package member. Never upload input packages to public CI.
"""
import argparse
import hashlib
import json
from pathlib import Path
import struct
import tempfile
import zlib

MAX_PACKAGE = 128 * 1024 * 1024
MAX_COMPRESSED = 8 * 1024 * 1024
MAX_OUTPUT = 4 * 1024 * 1024
CHUNK_SIZE = 4096


def inspect(path: Path, *, max_compressed=MAX_COMPRESSED, max_output=MAX_OUTPUT):
    length = path.stat().st_size
    if not 19 <= length <= MAX_PACKAGE:
        raise ValueError("package size outside audited bounds")
    with path.open("rb") as handle:
        header = handle.read(19)
        if header[:4] != b"PCK0" or struct.unpack_from("<H", header, 4)[0] != 15 or header[18] != 0:
            raise ValueError("unknown PCK0 header layout")
        count, first_data = struct.unpack_from("<II", header, 10)
        if not 0 < count <= 0x100000 or not 19 <= first_data < length:
            raise ValueError("header count or index extent outside bounds")
        if length - first_data < 2:
            raise ValueError("no compressed payload")
        handle.seek(first_data)
        decompressor = zlib.decompressobj()
        digest = hashlib.sha256()
        output_size = 0
        consumed = 0
        magic = b""
        while not decompressor.eof:
            if consumed >= max_compressed:
                raise ValueError("compressed first stream exceeds cap")
            piece = handle.read(min(CHUNK_SIZE, max_compressed - consumed))
            if not piece:
                raise ValueError("truncated first zlib stream")
            consumed += len(piece)
            try:
                decoded = decompressor.decompress(piece, max_output + 1 - output_size)
            except zlib.error as err:
                raise ValueError("invalid first zlib stream") from err
            if output_size + len(decoded) > max_output or decompressor.unconsumed_tail:
                raise ValueError("decompressed first stream exceeds cap")
            output_size += len(decoded)
            digest.update(decoded)
            if len(magic) < 8:
                magic += decoded[:8-len(magic)]
        consumed -= len(decompressor.unused_data)
        if consumed <= 0:
            raise ValueError("empty compressed stream")
    with path.open("rb") as handle:
        package_hash = hashlib.file_digest(handle, "sha256").hexdigest()
    kind = ("ini_section_prefix" if magic.startswith(b"[") else
            "lua51_bytecode_prefix" if magic.startswith(b"\x1bLuaQ") else
            "unknown")
    return {
        "package_size": length,
        "package_sha256": package_hash,
        "primary_header_length": 15,
        "declared_entry_count_not_verified": count,
        "index_start": 19,
        "index_end_exclusive": first_data,
        "index_decrypted": False,
        "index_paths_verified": 0,
        "named_members_extracted": 0,
        "first_anonymous_zlib_stream": {
            "start": first_data,
            "compressed_length": consumed,
            "uncompressed_length": output_size,
            "uncompressed_sha256": digest.hexdigest(),
            "prefix_classification_only": kind,
            "checksum_verified_by_zlib_eof": True,
            "file_name_verified": False,
        },
        "shop_catalog_or_purchase_verified": False,
        "live_e2e_verified": False,
    }


def self_test():
    def fixture(payload=b"[fixture]\n", *, index=b"opaque", count=1,
                bad_magic=False, bad_end=False, trunc=False, corrupt=False):
        header = ((b"XXXX" if bad_magic else b"PCK0") +
                  struct.pack("<HIII", 15, 4, count, 19 + len(index) + (100000 if bad_end else 0)) + b"\0")
        stream = bytearray(zlib.compress(payload))
        if trunc:
            stream = stream[:-2]
        if corrupt:
            stream[-1] ^= 0x80
        return header + index + stream
    with tempfile.TemporaryDirectory() as folder:
        path = Path(folder) / "fixture.package"
        path.write_bytes(fixture())
        got = inspect(path)
        assert got["index_end_exclusive"] == 25
        assert got["first_anonymous_zlib_stream"]["compressed_length"] == len(zlib.compress(b"[fixture]\n"))
        assert got["index_paths_verified"] == got["named_members_extracted"] == 0
        path.write_bytes(fixture(payload=b"\x1bLuaQ" + b"\0" * 10))
        assert inspect(path)["first_anonymous_zlib_stream"]["prefix_classification_only"] == "lua51_bytecode_prefix"
        for kwargs in ({"bad_magic": True}, {"bad_end": True}, {"count": 0},
                       {"trunc": True}, {"corrupt": True},
                       {"payload": b"A" * (MAX_OUTPUT + 1)}):
            path.write_bytes(fixture(**kwargs))
            try:
                inspect(path)
            except ValueError:
                pass
            else:
                raise AssertionError(f"invalid fixture accepted: {kwargs.keys()}")
        path.write_bytes(fixture(payload=bytes(range(256))*16))
        try:
            inspect(path, max_compressed=16)
        except ValueError:
            pass
        else:
            raise AssertionError("compressed input cap ignored")
    print("SELF_TEST_PASS: valid INI/Lua prefixes; bad magic/count/bounds/truncation/checksum/bomb/compressed cap rejected")


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--ini", type=Path)
    parser.add_argument("--lua", type=Path)
    parser.add_argument("--self-test", action="store_true")
    args = parser.parse_args()
    if args.self_test:
        self_test()
    else:
        if args.ini is None or args.lua is None:
            parser.error("--ini and --lua must both be supplied")
        print(json.dumps({"ini": inspect(args.ini), "lua": inspect(args.lua)}, indent=2))
