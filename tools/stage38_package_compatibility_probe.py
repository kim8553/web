#!/usr/bin/env python3
"""Read-only PCK0 header/config preflight; deliberately DOES NOT unpack or decrypt.

The two external format references are used only for version/header comparisons:
Ersanio/wushu-utils Unpacker.cs (fixed 10-byte header for a legacy layout),
Ekey/WC2.PACKAGE.Tool PackageUnpack.cs (version 20 and patch version 4).
Neither external source is asserted to describe the supplied Snail client format.
"""
import argparse
import configparser
import hashlib
import json
from pathlib import Path
import struct
import zipfile

LEGACY_HEADER = b"PCK0\x0f\x00\x00\x00\x00\x00"
MAX_CONFIG = 64 * 1024
REQUIRED = ("packages.ini", "fxres_packages.ini", "fxpackage.dll", "fxres.exe")


def zip_configs(zip_path):
    with zipfile.ZipFile(zip_path) as archive:
        members = {key: [] for key in REQUIRED}
        for info in archive.infolist():
            if not info.is_dir():
                key = info.filename.replace("\\", "/").split("/")[-1].lower()
                if key in members:
                    members[key].append(info)
        for key in REQUIRED:
            if len(members[key]) != 1:
                raise ValueError(f"expected one {key}; found {len(members[key])}")
        configs = {}
        for key in ("packages.ini", "fxres_packages.ini"):
            member = members[key][0]
            if member.file_size > MAX_CONFIG:
                raise ValueError(f"oversize config {key}")
            raw = archive.read(member)
            config = configparser.ConfigParser(interpolation=None, strict=True)
            config.read_string(raw.decode("utf-8-sig"))
            configs[key] = config
        binaries = {}
        for key in ("fxpackage.dll", "fxres.exe"):
            member = members[key][0]
            if not 1000 < member.file_size <= 20 * 1024 * 1024:
                raise ValueError(f"unexpected binary size {key}")
            data = archive.read(member)
            binaries[key] = {
                "bytes": len(data),
                "sha256": hashlib.sha256(data).hexdigest(),
                "literal_PCK0_occurrences": data.count(b"PCK0"),
            }
    paths = {}
    for name in ("lua", "lua64", "ini", "gui", "share"):
        if not configs["packages.ini"].has_option(name, "File"):
            raise ValueError(f"missing [{name}]/File in packages.ini")
        raw_path = configs["packages.ini"].get(name, "File")
        normalized = raw_path.replace("/", "\\").lower()
        if not normalized.startswith("res\\") or ".." in normalized or not normalized.endswith(".package"):
            raise ValueError(f"invalid package path {name}")
        paths[name] = normalized
    for name in ("ini", "gui", "share"):
        if configs["fxres_packages.ini"].get(name, "File").replace("/", "\\").lower() != paths[name]:
            raise ValueError(f"package list disagreement for {name}")
    if paths["lua"] != "res\\lua.package" or paths["ini"] != "res\\ini.package":
        raise ValueError("supplied package paths do not match expected archive inputs")
    return paths, binaries


def package_header(path):
    with path.open("rb") as f:
        data = f.read(24)
    if len(data) != 24 or data[:4] != b"PCK0":
        raise ValueError(f"{path.name}: not a supported PCK0 header")
    ver, variant = struct.unpack_from("<HH", data, 4)
    # This integer is opaque unless a matching format definition is established.
    opaque8 = struct.unpack_from("<I", data, 8)[0]
    with path.open("rb") as file_stream:
        digest = hashlib.file_digest(file_stream, "sha256").hexdigest()
    return {
        "bytes": path.stat().st_size,
        "sha256": digest,
        "magic": "PCK0",
        "u16_at_0x04": ver,
        "u16_at_0x06": variant,
        "opaque_u32_at_0x08": f"0x{opaque8:08x}",
        "ersanio_legacy_fixed_header_match": data[:10] == LEGACY_HEADER,
        "ekey_wc2_version20_match": ver == 20,
        "index_decrypted": False,
        "contents_extracted": False,
    }


def audit(zip_path, lua_path, ini_path):
    paths, binaries = zip_configs(zip_path)
    return {
        "archive": zip_path.name,
        "declared_package_paths": paths,
        "package_loader_binary_metadata": binaries,
        "packages": {
            "lua": package_header(lua_path),
            "ini": package_header(ini_path),
        },
        "shop_lua_identified": False,
        "shop_ini_identified_in_current_package": False,
        "npc_shop_mapping_verified": False,
        "client_patch_coherence_verified": False,
        "live_or_e2e_verified": False,
    }


def self_test():
    import tempfile
    with tempfile.TemporaryDirectory() as td:
        temp = Path(td)
        raw = b"PCK0\x0f\x00\x04\x00" + b"\xa1\x33\x6f\x48" + b"\x00" * 12
        pack = temp / "example.package"
        pack.write_bytes(raw)
        head = package_header(pack)
        assert head["u16_at_0x04"] == 15 and head["u16_at_0x06"] == 4
        assert not head["ersanio_legacy_fixed_header_match"]
        assert not head["ekey_wc2_version20_match"]
        pack.write_bytes(b"notpck" + b"\x00" * 18)
        try:
            package_header(pack)
        except ValueError:
            pass
        else:
            raise AssertionError("accepted incorrect magic")
        pack.write_bytes(LEGACY_HEADER + b"\x01" * 14)
        assert package_header(pack)["ersanio_legacy_fixed_header_match"]
        paths_ini = "[lua]\nFile=res\\lua.package\n[lua64]\nFile=res\\lua64.package\n[ini]\nFile=res\\ini.package\n[gui]\nFile=res\\gui.package\n[share]\nFile=res\\share.package\n"
        other_ini = "[ini]\nFile=res\\ini.package\n[gui]\nFile=res\\gui.package\n[share]\nFile=res\\share.package\n"
        with zipfile.ZipFile(temp / "sample.zip", "w") as z:
            z.writestr("packages.ini", paths_ini)
            z.writestr("fxres_packages.ini", other_ini)
            z.writestr("fxpackage.dll", b"MZ" + b"\0" * 1998)
            z.writestr("fxres.exe", b"MZ" + b"\0" * 1998)
        paths, _ = zip_configs(temp / "sample.zip")
        assert paths["gui"] == "res\\gui.package"
        with zipfile.ZipFile(temp / "duplicate.zip", "w") as z:
            z.writestr("packages.ini", paths_ini)
            z.writestr("nested/packages.ini", paths_ini)
        try:
            zip_configs(temp / "duplicate.zip")
        except ValueError:
            pass
        else:
            raise AssertionError("accepted duplicate config")
    print("SELF_TEST_PASS")


if __name__ == "__main__":
    ap = argparse.ArgumentParser(description=__doc__)
    ap.add_argument("--zip", type=Path)
    ap.add_argument("--lua", type=Path)
    ap.add_argument("--ini", type=Path)
    ap.add_argument("--self-test", action="store_true")
    args = ap.parse_args()
    if args.self_test:
        self_test()
    else:
        if any(v is None for v in (args.zip, args.lua, args.ini)):
            ap.error("--zip --lua --ini are required")
        print(json.dumps(audit(args.zip, args.lua, args.ini), indent=2, ensure_ascii=False))
