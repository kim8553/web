#!/usr/bin/env python3
"""Read-only fxres.exe code fingerprint and PCK0 primary-header topology probe.

The instruction fingerprints are tied to the *supplied* fxres.exe binary. This
script does not infer decryption, parse the raw index, extract files, or execute
any executable. It fails closed if its supported code fingerprint changes.
"""
import argparse
import hashlib
import json
from pathlib import Path
import struct
import zipfile

MAGIC = b"PCK0"
FXRES_NAME = "fxres.exe"
MAX_PACKAGE = 128 * 1024 * 1024
MAX_FXRES = 20 * 1024 * 1024
# Independently located by objdump -D -M intel on the supplied fxres.exe.
# VA -> exact bytes, with only instruction bytes (not guessed symbols).
CODE_FINGERPRINTS = {
    0x14000da21: "81 7c 24 58 50 43 4b 30",     # cmp DWORD PTR [rsp+0x58],0x304b4350
    0x14000da80: "0f b7 4c 24 40",              # movzx ecx, WORD PTR [rsp+0x40]
    0x14000da85: "8d 41 f1",                    # lea eax,[rcx-0xf]
    0x14000dade: "49 8d 56 02",              # lea rdx,[r14+0x2]
    0x14000db30: "42 80 7c 30 0e 00",        # cmp BYTE PTR [rax+r14+0xe],0
    0x14000db72: "41 8b 4e 0a",              # mov ecx,DWORD PTR [r14+0xa]
    0x14000db7a: "8d 42 04",                 # lea eax,[rdx+0x4]
    0x14000dbbb: "41 8b 46 06",              # mov eax,DWORD PTR [r14+0x6]
    0x14000dbc3: "3d 00 00 10 00",           # cmp eax,0x100000
    0x14000dc04: "44 8b e9 44 2b ea 41 83 ed 04", # table_len=field - record_len - 4
    0x14000dcaa: "4d 85 ff",                 # test r15,r15 in entry-count loop
    0x14000dcc0: "0f b7 07",                 # movzx eax, WORD PTR [rdi], index record
    0x14000deeb: "41 8b 46 02",             # mov eax,DWORD PTR [r14+0x2]
}


def file_sha256(path):
    with path.open("rb") as stream:
        return hashlib.file_digest(stream, "sha256").hexdigest()


def validated_fxres_code(zip_path):
    with zipfile.ZipFile(zip_path) as archive:
        members = [i for i in archive.infolist()
                   if not i.is_dir() and i.filename.replace('\\', '/').split('/')[-1].lower() == FXRES_NAME]
        if len(members) != 1 or not 0 < members[0].file_size <= MAX_FXRES:
            raise ValueError("expected exactly one bounded fxres.exe")
        binary = archive.read(members[0])
    if len(binary) < 512 or binary[:2] != b'MZ':
        raise ValueError("not a Windows MZ binary")
    pe = struct.unpack_from('<I', binary, 0x3c)[0]
    if pe > len(binary) - 24 or binary[pe:pe+4] != b'PE\0\0':
        raise ValueError("missing PE signature")
    machine, section_count = struct.unpack_from('<HH', binary, pe+4)
    optional_size = struct.unpack_from('<H', binary, pe+20)[0]
    optional = pe + 24
    if machine != 0x8664 or optional + optional_size > len(binary) or struct.unpack_from('<H', binary, optional)[0] != 0x20b:
        raise ValueError("expected PE32+ x86-64")
    image_base = struct.unpack_from('<Q', binary, optional+24)[0]
    if image_base != 0x140000000 or not 0 < section_count <= 96:
        raise ValueError("different fxres executable layout")
    section_table = optional + optional_size
    if section_table + section_count * 40 > len(binary):
        raise ValueError("truncated PE section table")
    sections = []
    for n in range(section_count):
        at = section_table + n * 40
        name = binary[at:at+8].split(b'\0')[0].decode('ascii', 'replace')
        virtual_size, rva, raw_size, raw_offset = struct.unpack_from('<IIII', binary, at+8)
        flags = struct.unpack_from('<I', binary, at+36)[0]
        if raw_size and (raw_offset > len(binary) or raw_size > len(binary)-raw_offset):
            raise ValueError("out of bounds section")
        sections.append((name, rva, virtual_size, raw_offset, raw_size, flags))
    text = [s for s in sections if s[0] == '.text' and (s[5] & 0x20000000)]
    if len(text) != 1:
        raise ValueError("expected exactly one executable .text")
    _, rva, _, raw_offset, raw_size, _ = text[0]
    for va, fingerprint in CODE_FINGERPRINTS.items():
        expected = bytes.fromhex(fingerprint)
        rel = va - image_base - rva
        if rel < 0 or rel + len(expected) > raw_size or binary[raw_offset+rel:raw_offset+rel+len(expected)] != expected:
            raise ValueError(f"code fingerprint mismatch at 0x{va:x}")
    return {"sha256": hashlib.sha256(binary).hexdigest(),
            "fingerprint_count": len(CODE_FINGERPRINTS),
            "primary_header_read_sites_verified": True}


def inspect_package(path):
    length = path.stat().st_size
    if not 19 <= length <= MAX_PACKAGE:
        raise ValueError(f"{path.name}: out of package size bounds")
    with path.open('rb') as stream:
        first = stream.read(19)
    if first[:4] != MAGIC:
        raise ValueError(f"{path.name}: invalid magic")
    record_len = struct.unpack_from('<H', first, 4)[0]
    # The observed program accepts 0x000f..0x040f; this probe only knows the
    # observed length-15 record layout, and rejects every other record length.
    if record_len != 15:
        raise ValueError(f"{path.name}: primary record length is not the observed 15")
    if first[18] != 0:
        raise ValueError(f"{path.name}: missing NUL in 15-byte record")
    raw_field_6 = struct.unpack_from('<I', first, 6)[0]
    declared_count = struct.unpack_from('<I', first, 10)[0]
    end = struct.unpack_from('<I', first, 14)[0]
    if not 0 < declared_count <= 0x100000:
        raise ValueError(f"{path.name}: invalid declared entry count")
    if end < 4 + record_len or end > length:
        raise ValueError(f"{path.name}: index range outside file")
    return {"bytes": length, "sha256": file_sha256(path),
            "primary_record_length": record_len,
            "uninterpreted_u32_at_offset_6": f"0x{raw_field_6:08x}",
            "declared_entry_count": declared_count,
            "index_start_offset": 4 + record_len,
            "index_end_offset_exclusive": end,
            "raw_index_byte_length": end - (4 + record_len),
            "index_decrypted": False,
            "index_entries_parsed": 0,
            "files_extracted": 0}


def audit(zip_path, lua, ini):
    return {"source": "user-uploaded client ZIP + private Drive package copies",
            "fxres": validated_fxres_code(zip_path),
            "packages": {"lua": inspect_package(lua), "ini": inspect_package(ini)},
            "ordinary_shop_lua_verified": False,
            "npc_shop_mapping_verified": False,
            "client_screen_verified": False,
            "purchase_or_bag_verified": False,
            "live_e2e_verified": False}


def self_test():
    import tempfile
    with tempfile.TemporaryDirectory() as td:
        p = Path(td) / 'fixture.package'
        def fixture(count=2, end=64, magic=MAGIC, suffix=0):
            return magic + struct.pack('<HIII', 15, 0x12000004, count, end) + bytes([suffix]) + bytes(70)
        p.write_bytes(fixture())
        got = inspect_package(p)
        assert got['declared_entry_count'] == 2
        assert got['index_start_offset'] == 19 and got['index_end_offset_exclusive'] == 64
        assert got['raw_index_byte_length'] == 45 and got['files_extracted'] == 0
        for bad in (fixture(magic=b'PACK'), fixture(count=0), fixture(count=0x100001),
                    fixture(end=18), fixture(end=200), fixture(suffix=1),
                    MAGIC + struct.pack('<H', 16) + bytes(80)):
            p.write_bytes(bad)
            try: inspect_package(p)
            except ValueError: pass
            else: raise AssertionError('bad fixture accepted')
    print('SELF_TEST_PASS')


if __name__ == '__main__':
    ap = argparse.ArgumentParser(description=__doc__)
    ap.add_argument('--zip', type=Path)
    ap.add_argument('--lua', type=Path)
    ap.add_argument('--ini', type=Path)
    ap.add_argument('--self-test', action='store_true')
    args = ap.parse_args()
    if args.self_test:
        self_test()
    else:
        if not all((args.zip,args.lua,args.ini)):
            ap.error('--zip --lua --ini required unless --self-test')
        print(json.dumps(audit(args.zip,args.lua,args.ini),ensure_ascii=False,indent=2))
