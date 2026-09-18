"""Synthetic-only regression tests; no client packages are shipped or fetched."""

import hashlib
import struct
import unittest
import zlib

from audit_client_lua_sale_candidates import InvalidChunk, audit, decode_scalar


def masked_size(size, width):
    mask = b"abcd" if width == 4 else b"abcd464f"
    return decode_scalar(size.to_bytes(width, "little"), mask)


def synthetic_luaq(width, words=("sell", "shop"), instruction_raw=b""):
    result = bytearray(b"\x1bLuaQ\x00\x01\x04" + bytes([width]) + b"\x04\x08\x00")
    result += masked_size(0, width)  # source path unset
    result += decode_scalar(struct.pack("<II", 0, 0))  # two zero line numbers
    result += decode_scalar(b"\x00\x00\x00\x02")
    result += decode_scalar(struct.pack("<I", len(instruction_raw) // 4))
    result += instruction_raw
    result += decode_scalar(struct.pack("<I", len(words)))
    for word in words:
        raw = word.encode("ascii") + b"\0"
        result += decode_scalar(b"\x04")
        result += masked_size(len(raw), width)
        result += decode_scalar(raw)
    result += decode_scalar(struct.pack("<I", 0))  # nested functions
    result += decode_scalar(struct.pack("<I", 0))  # lines
    result += decode_scalar(struct.pack("<I", 0))  # locals
    result += decode_scalar(struct.pack("<I", 0))  # upvalues
    return bytes(result)


def synthetic_package(payload):
    header = b"PCK0" + struct.pack("<HHHI", 15, 4, 0x5e61, 1) + struct.pack("<I", 19) + b"\x00"
    assert len(header) == 19
    return header + zlib.compress(payload)


class ReadOnlyLuaCandidateTest(unittest.TestCase):
    def test_both_scalar_widths_find_only_structural_constants(self):
        for width in (4, 8):
            with self.subTest(width=width):
                raw = synthetic_package(synthetic_luaq(width))
                found = audit(raw, hashlib.sha256(raw).hexdigest())
                self.assertEqual(found["counts"]["structurally_valid_luaq_streams"], 1)
                self.assertEqual(found["counts"]["sell_and_shop_prefix_candidates"], 1)
                self.assertEqual(found["sell_and_shop_candidates"][0]["functions"], 1)

    def test_instructions_containing_sell_are_not_a_lua_string(self):
        raw = synthetic_package(synthetic_luaq(4, ("shop",), instruction_raw=b"sell"))
        found = audit(raw, hashlib.sha256(raw).hexdigest())
        self.assertEqual(found["counts"]["structurally_valid_luaq_streams"], 1)
        self.assertNotIn("sell_prefix_candidates", found["counts"])

    def test_corrupt_luaq_does_not_count(self):
        raw = synthetic_package(b"\x1bLuaQ\x00\x01\x04\x04\x04\x08\x00broken")
        found = audit(raw, hashlib.sha256(raw).hexdigest())
        self.assertNotIn("structurally_valid_luaq_streams", found["counts"])

    def test_sha_mismatch_rejects_before_scanning(self):
        raw = synthetic_package(synthetic_luaq(4))
        with self.assertRaisesRegex(InvalidChunk, "SHA-256 mismatch"):
            audit(raw, "0" * 64)

    def test_truncated_zlib_rejected(self):
        raw = synthetic_package(synthetic_luaq(4))[:-5]
        found = audit(raw, hashlib.sha256(raw).hexdigest())
        self.assertNotIn("structurally_valid_luaq_streams", found["counts"])

    def test_invalid_header_rejected(self):
        raw = b"BAD0" + synthetic_package(synthetic_luaq(4))[4:]
        with self.assertRaisesRegex(InvalidChunk, "PCK0"):
            audit(raw, hashlib.sha256(raw).hexdigest())


if __name__ == "__main__":
    unittest.main()
