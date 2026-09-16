#!/usr/bin/env python3
"""Read-only provenance guard: compare legacy Shop.xml IDs with authoritative shop.ini.

No conversion, auto-mapping, packet inference, or mutation is performed.
"""
import argparse
import json
from pathlib import Path
import re
import xml.etree.ElementTree as ET

SECTION = re.compile(rb"(?m)^[ \t]*\[([\x20-\x7e]+)\][ \t]*\r?$")
ENCODING = re.compile(br"<\?xml[^>]*encoding=[\"']([^\"']+)[\"']", re.I)


def read_ini_sections(path: Path) -> set[str]:
    raw = path.read_bytes()
    names = {m.group(1).decode("ascii") for m in SECTION.finditer(raw)}
    if not names:
        raise ValueError("No valid ASCII INI section headers; cannot compare")
    return names


def read_legacy_xml_ids(path: Path) -> set[str]:
    raw = path.read_bytes()
    match = ENCODING.search(raw[:160])
    declared = match.group(1).decode("ascii").lower() if match else "utf-8"
    codec = "gb18030" if declared in ("gb2312", "gbk", "gb18030") else declared
    if codec not in ("utf-8", "gb18030"):
        raise ValueError("Unsupported XML encoding")
    root = ET.fromstring(raw.decode(codec))
    if root.tag != "Object":
        raise ValueError("Expected legacy XML Object root")
    ids = {p.attrib["ID"] for p in root.findall("./Property") if p.attrib.get("ID")}
    if not ids:
        raise ValueError("No XML Property IDs; cannot compare")
    return ids


def audit(authoritative_ini: Path, legacy_xml: Path, shop_id: str) -> dict:
    current = read_ini_sections(authoritative_ini)
    legacy = read_legacy_xml_ids(legacy_xml)
    return {
        "authoritative_ini_section_count": len(current),
        "suspicious_nested_bracket_section_count": sum("[" in x or "]" in x for x in current),
        "legacy_xml_shop_count": len(legacy),
        "exact_id_intersection_count": len(current & legacy),
        "queried_shop_id": shop_id,
        "authoritative_exact_section_found": shop_id in current,
        "legacy_exact_id_found": shop_id in legacy,
        "authoritative_suffix_candidate_count": sum(x.startswith(shop_id + "_") for x in current),
        "auto_map_allowed": False,
        "legacy_xml_authority_allowed": False,
        "live_client_render_verified": False,
        "purchase_or_bag_verified": False,
    }


def self_test() -> None:
    from tempfile import TemporaryDirectory
    with TemporaryDirectory() as d:
        ini, xml = Path(d) / "shop.ini", Path(d) / "Shop.xml"
        ini.write_bytes(b"[Shop_A]\r\n[Shop_GB_Yishiting_1]\r\n[Shop_GB_Yishiting_2]\r\n")
        xml.write_bytes(b'<?xml version="1.0" encoding="gb2312"?><Object><Property ID="OldShop"/><Property ID="Shop_A"/></Object>')
        result = audit(ini, xml, "Shop_GB_Yishiting")
        assert result["authoritative_ini_section_count"] == 3
        assert result["legacy_xml_shop_count"] == 2
        assert result["exact_id_intersection_count"] == 1
        assert result["authoritative_exact_section_found"] is False
        assert result["authoritative_suffix_candidate_count"] == 2
        assert result["auto_map_allowed"] is False
        assert result["legacy_xml_authority_allowed"] is False
        ini.write_bytes(b"not an ini")
        try:
            audit(ini, xml, "Shop_GB_Yishiting")
        except ValueError:
            pass
        else:
            raise AssertionError("Malformed INI was accepted")
    print("SELF_TEST_PASS")


def main() -> None:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--authoritative-ini", type=Path)
    parser.add_argument("--legacy-xml", type=Path)
    parser.add_argument("--shop-id", default="Shop_GB_Yishiting")
    parser.add_argument("--self-test", action="store_true")
    args = parser.parse_args()
    if args.self_test:
        self_test()
        return
    if not args.authoritative_ini or not args.legacy_xml:
        parser.error("--authoritative-ini and --legacy-xml are required")
    print(json.dumps(audit(args.authoritative_ini, args.legacy_xml, args.shop_id), ensure_ascii=False, indent=2))


if __name__ == "__main__":
    main()
