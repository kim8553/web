#!/usr/bin/env python3
"""Read-only structural candidate audit for the user-supplied Snail lua[64].package.

This does NOT decode PCK file names, resolve active resource precedence, recover
Lua semantics, or identify a sale packet. Only fully parsed LuaQ zlib streams
are counted. The four-byte constant prefixes are *candidates*, not names.
"""

import argparse
import hashlib
import json
import struct
import sys
import zlib
from collections import Counter
from pathlib import Path

MAX_INFLATED = 5_000_000
MAX_COMPRESSED_SPAN = 1_000_000
MAX_STRING = 1_000_000
MAX_INSTRUCTIONS = 500_000
MAX_CONSTANTS = 300_000
MAX_CHILDREN = 3_000
MAX_DEPTH = 40
PREFIXES = (b"sell", b"shop", b"buy_", b"cust", b"send")


class InvalidChunk(ValueError):
    pass


def decode_scalar(data, key=b"abcd"):
    return bytes(value ^ key[i % len(key)] for i, value in enumerate(data))


class LuaQStructure:
    """Bounded Lua 5.1 structural walk; never execute bytecode or decode paths."""

    def __init__(self, payload):
        if len(payload) < 12 or payload[:8] != b"\x1bLuaQ\x00\x01\x04":
            raise InvalidChunk("unsupported LuaQ header")
        self.size_t = payload[8]
        if self.size_t not in (4, 8) or payload[9:12] != b"\x04\x08\x00":
            raise InvalidChunk("unsupported LuaQ scalar widths")
        self.payload = payload
        self.offset = 12
        self.functions = 0
        self.constants = 0
        self.prefixes = set()
        self.skeleton = []

    def take(self, count):
        if count < 0 or self.offset + count > len(self.payload):
            raise InvalidChunk("truncated structure")
        result = self.payload[self.offset:self.offset + count]
        self.offset += count
        return result

    def scalar(self, count, key=b"abcd"):
        return decode_scalar(self.take(count), key)

    def uint32(self):
        return struct.unpack("<I", self.scalar(4))[0]

    def string(self, *, constant=False):
        # The eight-byte width was validated against both actual package
        # variants. Do not use these masks to claim full string decryption.
        size_mask = b"abcd" if self.size_t == 4 else b"abcd464f"
        size = int.from_bytes(self.scalar(self.size_t, size_mask), "little")
        if size > MAX_STRING:
            raise InvalidChunk("oversized string")
        raw = self.take(size)
        first_four = b""
        if constant and size >= 4:
            # Only the first four decoded bytes are reported. The remaining
            # obfuscated string bytes have NOT been interpreted as plaintext.
            first_four = decode_scalar(raw[:4]).lower()
            if first_four in PREFIXES:
                self.prefixes.add(first_four.decode("ascii"))
        return (size, first_four.hex()) if constant else size

    def proto(self, depth=0):
        if depth > MAX_DEPTH:
            raise InvalidChunk("nested function depth exceeded")
        self.functions += 1
        self.string()  # source name; its path is not resolved
        self.uint32()  # line defined
        self.uint32()  # last line defined
        self.scalar(4)  # nups, params, vararg, stack
        code_size = self.uint32()
        if code_size > MAX_INSTRUCTIONS:
            raise InvalidChunk("oversized instruction table")
        self.take(code_size * 4)  # instructions are deliberately not decoded
        nconst = self.uint32()
        if nconst > MAX_CONSTANTS:
            raise InvalidChunk("oversized constant table")
        self.constants += nconst
        constant_shape = []
        for _ in range(nconst):
            kind = self.scalar(1)[0]
            if kind == 0:
                constant_shape.append((0, 0, ""))
            elif kind == 1:
                self.scalar(1)
                constant_shape.append((1, 0, ""))
            elif kind == 3:
                self.take(8)
                constant_shape.append((3, 0, ""))
            elif kind == 4:
                size, prefix_hex = self.string(constant=True)
                constant_shape.append((4, size, prefix_hex))
            else:
                raise InvalidChunk("unknown Lua constant kind")
        children = self.uint32()
        if children > MAX_CHILDREN:
            raise InvalidChunk("oversized nested function table")
        self.skeleton.append((code_size, nconst, children, tuple(constant_shape)))
        for _ in range(children):
            self.proto(depth + 1)
        lines = self.uint32()
        if lines > MAX_INSTRUCTIONS:
            raise InvalidChunk("oversized line table")
        self.take(lines * 4)
        locals_count = self.uint32()
        if locals_count > MAX_CONSTANTS:
            raise InvalidChunk("oversized local table")
        for _ in range(locals_count):
            self.string()
            self.uint32()
            self.uint32()
        upvalues = self.uint32()
        if upvalues > MAX_CONSTANTS:
            raise InvalidChunk("oversized upvalue table")
        for _ in range(upvalues):
            self.string()

    def verify(self):
        self.proto()
        if self.offset != len(self.payload):
            raise InvalidChunk("trailing bytes after LuaQ structure")
        return self


def audit(data, expected_sha256):
    actual_sha256 = hashlib.sha256(data).hexdigest()
    if actual_sha256 != expected_sha256.lower():
        raise InvalidChunk("package SHA-256 mismatch: no content examined")
    if len(data) < 19 or data[:4] != b"PCK0":
        raise InvalidChunk("not a PCK0 package")
    record_width, variant = struct.unpack_from("<HH", data, 4)
    if (record_width, variant) != (15, 4):
        raise InvalidChunk("unexpected PCK0 header fields")
    declared_entries = struct.unpack_from("<I", data, 10)[0]
    data_start_field = struct.unpack_from("<I", data, 14)[0]
    if not 19 <= data_start_field < len(data):
        raise InvalidChunk("invalid PCK0 data-start field")

    signatures = (b"\x78\x9c", b"\x78\xda", b"\x78\x01")
    counts = Counter()
    candidates = []
    # A raw zlib stream need not be a live PCK directory entry. We do NOT
    # claim that enumerating these offsets recovers PCK file names or order.
    for signature in signatures:
        cursor = data_start_field
        while True:
            offset = data.find(signature, cursor)
            if offset < 0:
                break
            cursor = offset + 1
            try:
                inflater = zlib.decompressobj()
                payload = inflater.decompress(
                    data[offset:offset + MAX_COMPRESSED_SPAN], MAX_INFLATED + 1
                )
                if not inflater.eof or len(payload) > MAX_INFLATED:
                    continue
                if not payload.startswith(b"\x1bLuaQ"):
                    continue
                counts["complete_luaq_zlib_streams"] += 1
                chunk = LuaQStructure(payload).verify()
            except (zlib.error, InvalidChunk, struct.error, RecursionError):
                continue
            counts["structurally_valid_luaq_streams"] += 1
            counts["constant_entries"] += chunk.constants
            found = chunk.prefixes
            if "sell" in found:
                counts["sell_prefix_candidates"] += 1
            if "sell" in found and ("send" in found or "cust" in found):
                counts["sell_with_send_or_cust_prefix_candidates"] += 1
            if "sell" in found and "shop" in found:
                counts["sell_and_shop_prefix_candidates"] += 1
                candidates.append({
                    "compressed_offset": offset,
                    "inflated_bytes": len(payload),
                    "functions": chunk.functions,
                    "constant_entries": chunk.constants,
                    "structure_sha256": hashlib.sha256(
                        json.dumps(chunk.skeleton, separators=(",", ":")).encode("utf-8")
                    ).hexdigest(),
                    "observed_four_byte_prefixes": sorted(found),
                })
    return {
        "package_sha256": actual_sha256,
        "package_bytes": len(data),
        "declared_pck_entries": declared_entries,
        "pck_header_field_at_offset_14": data_start_field,
        "counts": dict(sorted(counts.items())),
        "sell_and_shop_candidates": sorted(candidates, key=lambda x: x["compressed_offset"]),
        "caveat": "Independent zlib/LuaQ structural candidates only; no path mapping, active-file selection, fully decoded strings, sale packet, pricing or LIVE verification.",
    }


def main(argv=None):
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--package", type=Path, required=True)
    parser.add_argument("--expected-sha256", required=True)
    args = parser.parse_args(argv)
    if len(args.expected_sha256) != 64 or any(x not in "0123456789abcdefABCDEF" for x in args.expected_sha256):
        parser.error("--expected-sha256 must be 64 hex characters")
    try:
        data = args.package.read_bytes()
        result = audit(data, args.expected_sha256)
    except (OSError, InvalidChunk) as exc:
        print(f"AUDIT_ERROR: {exc}", file=sys.stderr)
        return 2
    print(json.dumps(result, ensure_ascii=False, indent=2, sort_keys=True))
    return 0


if __name__ == "__main__":
    sys.exit(main())
