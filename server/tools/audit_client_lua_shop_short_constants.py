#!/usr/bin/env python3
"""Read-only, SHA-pinned audit of exact short LuaQ constants in nominated PCK streams.

Only a small allowlist of complete, NUL-terminated ASCII constants is reported.
No PCK path/index mapping, bytecode execution, gameplay or packet inference.
"""

import argparse
import hashlib
import json
import struct
import sys
import zlib
from collections import Counter
from pathlib import Path

from audit_client_lua_sale_candidates import (
    InvalidChunk,
    LuaQStructure,
    MAX_COMPRESSED_SPAN,
    MAX_INFLATED,
    MAX_STRING,
    decode_scalar,
)

# Empirically validated on both supplied packages for complete short values,
# including ShopID, SellPrice0, SellPrice1, index and grid. The extended mask
# has NOT been established for longer strings, source paths or instruction data.
SHORT_MASK = b"abcd464fghfd"
SHORT_LIMIT = len(SHORT_MASK)
IDENTIFIERS = frozenset((
    "ShopID", "ShopType", "shopid", "IsShop", "@ui_shop", "do_shop",
    "SellPrice0", "SellPrice1", "SellPrice2", "SellCount",
    "SellLimit", "BuyLimit", "bSell", "item_obj",
))


class ShopShortConstants(LuaQStructure):
    def __init__(self, payload):
        super().__init__(payload)
        self.identifiers = Counter()

    def string(self, *, constant=False):
        size_mask = b"abcd" if self.size_t == 4 else b"abcd464f"
        size = int.from_bytes(self.scalar(self.size_t, size_mask), "little")
        if size > MAX_STRING:
            raise InvalidChunk("oversized string")
        raw = self.take(size)
        prefix = b""
        if constant and size >= 4:
            prefix = decode_scalar(raw[:4]).lower()
            if prefix in (b"sell", b"shop", b"buy_", b"cust", b"send"):
                self.prefixes.add(prefix.decode("ascii"))
        # Requiring the decoded terminator and an EXACT approved identifier
        # prevents prefix-only hits from being reported as actual full names.
        if constant and 1 <= size <= SHORT_LIMIT:
            plain = bytes(byte ^ SHORT_MASK[i] for i, byte in enumerate(raw))
            if plain.endswith(b"\x00") and all(32 <= byte < 127 for byte in plain[:-1]):
                identifier = plain[:-1].decode("ascii")
                if identifier in IDENTIFIERS:
                    self.identifiers[identifier] += 1
        return (size, prefix.hex()) if constant else size


def audit_streams(data, expected_sha256, offsets):
    if hashlib.sha256(data).hexdigest() != expected_sha256.lower():
        raise InvalidChunk("package SHA-256 mismatch: no content examined")
    if len(data) < 19 or data[:4] != b"PCK0":
        raise InvalidChunk("not a PCK0 package")
    if struct.unpack_from("<HH", data, 4) != (15, 4):
        raise InvalidChunk("unexpected PCK0 header fields")
    data_start = struct.unpack_from("<I", data, 14)[0]
    if not 19 <= data_start < len(data):
        raise InvalidChunk("invalid PCK0 data-start field")
    if not offsets or len(offsets) != len(set(offsets)):
        raise InvalidChunk("offsets must be nonempty and unique")
    entries = []
    for offset in offsets:
        if offset < data_start or offset >= len(data):
            raise InvalidChunk("candidate offset is outside the PCK payload")
        inflater = zlib.decompressobj()
        try:
            payload = inflater.decompress(data[offset:offset + MAX_COMPRESSED_SPAN], MAX_INFLATED + 1)
        except zlib.error as exc:
            raise InvalidChunk("invalid zlib candidate") from exc
        if not inflater.eof or len(payload) > MAX_INFLATED:
            raise InvalidChunk("truncated or oversized zlib candidate")
        chunk = ShopShortConstants(payload).verify()
        entries.append({
            "compressed_offset": offset,
            "functions": chunk.functions,
            "constant_entries": chunk.constants,
            "complete_allowlisted_short_constants": dict(sorted(chunk.identifiers.items())),
        })
    return {
        "package_sha256": hashlib.sha256(data).hexdigest(),
        "pck_declared_entries": struct.unpack_from("<I", data, 10)[0],
        "candidates": entries,
        "limits": "Only NUL-terminated ASCII identifiers of at most 11 characters; no filenames, longer strings, bytecode interpretation, active package selection, packet or sale semantics.",
    }


def main(argv=None):
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--package", required=True, type=Path)
    parser.add_argument("--expected-sha256", required=True)
    parser.add_argument("--offset", required=True, type=int, action="append")
    args = parser.parse_args(argv)
    if len(args.expected_sha256) != 64 or any(c not in "0123456789abcdefABCDEF" for c in args.expected_sha256):
        parser.error("--expected-sha256 must be 64 hexadecimal characters")
    try:
        print(json.dumps(audit_streams(args.package.read_bytes(), args.expected_sha256, args.offset), sort_keys=True, indent=2))
    except (OSError, InvalidChunk) as exc:
        print(f"AUDIT_ERROR: {exc}", file=sys.stderr)
        return 2
    return 0


if __name__ == "__main__":
    sys.exit(main())
