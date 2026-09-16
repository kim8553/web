#!/usr/bin/env python3
"""Read-only, bounded client ZIP audit for Stage37 shop UI string anchors.

Static RIP-relative references are *candidates*, not proved packet handlers.
Does not execute or extract any Windows binary, and never changes the source ZIP.
"""
import argparse
import hashlib
import json
from pathlib import Path
import re
import struct
import zipfile

MEMBERS = ("fxgame.exe", "fxgamelogic.dll", "fxnet2.dll", "fxcore.dll")
ANCHORS = (
    b"form_stage_main\\form_shop\\form_shop\x00",
    b"open_shop\x00",
    b"on_open_shop_exchange_form\x00",
    b"on_single_shop_info\x00",
)
# REX.W + LEA/MOV r64, [RIP + disp32]; mask/modrm narrowed to RIP addressing.
RIP_INSN = re.compile(rb"[\x48\x49\x4c\x4d][\x8b\x8d][\x05\x0d\x15\x1d\x25\x2d\x35\x3d].{4}", re.S)


def pe_sections(data: bytes):
    if len(data) < 0x100 or data[:2] != b"MZ":
        raise ValueError("not MZ")
    pe = struct.unpack_from("<I", data, 0x3c)[0]
    if pe + 24 > len(data) or data[pe:pe + 4] != b"PE\0\0":
        raise ValueError("not PE")
    section_count = struct.unpack_from("<H", data, pe + 6)[0]
    optional_len = struct.unpack_from("<H", data, pe + 20)[0]
    optional = pe + 24
    if optional + optional_len > len(data) or struct.unpack_from("<H", data, optional)[0] != 0x20b:
        raise ValueError("not PE32+")
    base = struct.unpack_from("<Q", data, optional + 24)[0]
    sections = []
    start = optional + optional_len
    if section_count > 96 or start + section_count * 40 > len(data):
        raise ValueError("bad section table")
    for i in range(section_count):
        off = start + i * 40
        name = data[off:off + 8].split(b"\0")[0].decode("ascii", "replace")
        vsize, rva, raw_size, raw_off = struct.unpack_from("<IIII", data, off + 8)
        if raw_size and (raw_off > len(data) or raw_size > len(data) - raw_off):
            raise ValueError("out-of-bounds PE section")
        sections.append((name, rva, vsize, raw_off, raw_size))
    return base, sections


def offset_to_va(offset: int, base: int, sections):
    for _, rva, _, raw_off, raw_size in sections:
        if raw_off <= offset < raw_off + raw_size:
            return base + rva + offset - raw_off
    raise ValueError("string outside mapped PE sections")


def static_rip_references(data: bytes, target_va: int, base: int, sections):
    hits = []
    for name, rva, _, raw_off, raw_size in sections:
        if name != ".text":
            continue
        for m in RIP_INSN.finditer(data, raw_off, raw_off + raw_size):
            ip = base + rva + m.start() - raw_off
            displacement = struct.unpack_from("<i", data, m.start() + 3)[0]
            if ip + 7 + displacement == target_va:
                hits.append(f"0x{ip:x}")
    return sorted(set(hits))


def inspect_dll(data: bytes):
    base, sections = pe_sections(data)
    result = {}
    for anchor in ANCHORS:
        offsets = [m.start() for m in re.finditer(re.escape(anchor), data)]
        if len(offsets) != 1:
            raise ValueError(f"anchor cardinality {anchor!r}: {len(offsets)}; no safe attribution")
        address = offset_to_va(offsets[0], base, sections)
        result[anchor[:-1].decode("ascii")] = {
            "file_offset": f"0x{offsets[0]:x}",
            "virtual_address": f"0x{address:x}",
            "static_rip_relative_references": static_rip_references(data, address, base, sections),
        }
    return result


def audit_zip(path: Path):
    result = {"source": path.name, "members": {}, "shop_anchors": {}}
    with zipfile.ZipFile(path) as z:
        index = {}
        for info in z.infolist():
            if info.is_dir():
                continue
            name = info.filename.replace("\\", "/").split("/")[-1].lower()
            if name in MEMBERS:
                index.setdefault(name, []).append(info)
        for name in MEMBERS:
            hits = index.get(name, [])
            if len(hits) != 1:
                raise ValueError(f"expected exactly one {name}, found {len(hits)}")
            info = hits[0]
            if not 0 < info.file_size <= 100 * 1024 * 1024:
                raise ValueError(f"unexpected member size for {name}")
            with z.open(info) as reader:
                data = reader.read(info.file_size + 1)
            if len(data) != info.file_size:
                raise ValueError(f"size mismatch for {name}")
            result["members"][name] = {"bytes": len(data), "sha256": hashlib.sha256(data).hexdigest()}
            if name == "fxgamelogic.dll":
                result["shop_anchors"] = inspect_dll(data)
    result["packet_selector_verified"] = False
    result["purchase_handler_verified"] = False
    result["live_or_e2e_verified"] = False
    return result


def self_test():
    # synthetic instruction: LEA RCX,[RIP+0x19] at VA 0x1000 -> target 0x1020.
    insn = b"\x48\x8d\x0d\x19\x00\x00\x00"
    m = RIP_INSN.fullmatch(insn)
    assert m and struct.unpack_from("<i", insn, 3)[0] + 0x1000 + 7 == 0x1020
    assert not RIP_INSN.fullmatch(b"\x48\x8d\x01\x19\x00\x00\x00")
    try:
        pe_sections(b"MZ" + bytes(256))
    except ValueError:
        pass
    else:
        raise AssertionError("accepted malformed PE")
    print("SELF_TEST_PASS")


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--zip", type=Path, help="original bin64 ZIP; read-only")
    parser.add_argument("--self-test", action="store_true")
    args = parser.parse_args()
    if args.self_test:
        self_test()
    else:
        if not args.zip:
            parser.error("--zip is required unless --self-test")
        print(json.dumps(audit_zip(args.zip), indent=2, ensure_ascii=False))
