"""Synthetic tests: no user client packages, sources or credentials in CI."""
import hashlib
import struct
import unittest

from audit_client_lua_sale_candidates import InvalidChunk, decode_scalar
from audit_client_lua_shop_short_constants import SHORT_MASK, audit_streams
from test_audit_client_lua_sale_candidates import masked_size, synthetic_package


def fixture(width, words):
    chunk = bytearray(b"\x1bLuaQ\x00\x01\x04" + bytes([width]) + b"\x04\x08\x00")
    chunk += masked_size(0, width)
    chunk += decode_scalar(struct.pack("<II", 0, 0))
    chunk += decode_scalar(b"\0\0\0\2")
    chunk += decode_scalar(struct.pack("<I", 0))  # instructions
    chunk += decode_scalar(struct.pack("<I", len(words)))
    for word in words:
        plaintext = word.encode("ascii") + b"\0"
        chunk += decode_scalar(b"\x04")
        chunk += masked_size(len(plaintext), width)
        chunk += decode_scalar(plaintext, SHORT_MASK if len(plaintext) <= 12 else b"abcd")
    for _ in range(4):
        chunk += decode_scalar(struct.pack("<I", 0))  # children, lines, locals, upvalues
    return synthetic_package(bytes(chunk))


class ShopShortConstantAuditTest(unittest.TestCase):
    def test_exact_short_constants_in_both_lua_widths(self):
        for width in (4, 8):
            with self.subTest(width=width):
                data = fixture(width, ("ShopID", "SellPrice0", "SellPrice1", "bSell", "ShopID"))
                result = audit_streams(data, hashlib.sha256(data).hexdigest(), [19])
                self.assertEqual(result["candidates"][0]["complete_allowlisted_short_constants"],
                                 {"ShopID": 2, "SellPrice0": 1, "SellPrice1": 1, "bSell": 1})

    def test_prefix_only_and_long_strings_not_claimed_complete(self):
        data = fixture(4, ("SellPrice0XYZ", "ShopIDother", "selling"))
        found = audit_streams(data, hashlib.sha256(data).hexdigest(), [19])
        self.assertEqual(found["candidates"][0]["complete_allowlisted_short_constants"], {})

    def test_sha_mismatch_fails_closed(self):
        data = fixture(4, ("ShopID",))
        with self.assertRaisesRegex(InvalidChunk, "SHA-256 mismatch"):
            audit_streams(data, "0" * 64, [19])

    def test_invalid_offsets_and_duplicate_rejected(self):
        data = fixture(4, ("ShopID",))
        sha = hashlib.sha256(data).hexdigest()
        for offsets in ([], [18], [19, 19], [len(data)]):
            with self.subTest(offsets=offsets), self.assertRaises(InvalidChunk):
                audit_streams(data, sha, offsets)

    def test_corrupt_structure_rejected(self):
        data = synthetic_package(b"\x1bLuaQ\x00\x01\x04\x04\x04\x08\x00garbage")
        with self.assertRaises(InvalidChunk):
            audit_streams(data, hashlib.sha256(data).hexdigest(), [19])

    def test_malformed_header_rejected(self):
        data = b"BAD0" + fixture(4, ("ShopID",))[4:]
        with self.assertRaisesRegex(InvalidChunk, "PCK0"):
            audit_streams(data, hashlib.sha256(data).hexdigest(), [19])


if __name__ == "__main__":
    unittest.main()
