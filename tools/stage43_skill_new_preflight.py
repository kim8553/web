#!/usr/bin/env python3
"""Read-only skill_new.ini intake. Never uploads or prints game resource contents."""
from __future__ import annotations

import argparse
from collections import Counter
import hashlib
import json
from pathlib import Path

MAX_GO_SCANNER_TOKEN_BYTES = 4 * 1024 * 1024
MAX_INTAKE_BYTES = 64 * 1024 * 1024


def audit(data: bytes) -> dict:
    if len(data) > MAX_INTAKE_BYTES:
        raise ValueError("file exceeds read-only intake limit")
    section: bytes | None = None
    headers: Counter[bytes] = Counter()
    fields: dict[bytes, dict[bytes, bytes]] = {}
    ignored_nonfield_lines = 0
    longest_line = 0
    for raw in data.splitlines():
        longest_line = max(longest_line, len(raw))
        line = raw.strip()
        if line.startswith(b"\xef\xbb\xbf"):
            line = line[3:].strip()
        if not line or line.startswith((b";", b"#")):
            continue
        if line.startswith(b"[") and line.endswith(b"]"):
            section = line[1:-1].strip()
            headers[section] += 1
            fields.setdefault(section, {})
            continue
        if section is None:
            continue
        key, sep, value = line.partition(b"=")
        if not sep:
            ignored_nonfield_lines += 1
            continue
        fields[section][key.strip().lower()] = value.strip()
    scripts: Counter[str] = Counter()
    eligible = 0
    missing_static = 0
    nonnumeric_static = 0
    for section_fields in fields.values():
        script = section_fields.get(b"script", b"")
        if script.lower() == b"skillnormal":
            scripts["SkillNormal"] += 1
            eligible += 1
        elif script.lower() == b"skilllock":
            scripts["SkillLock"] += 1
            eligible += 1
        else:
            scripts["other_or_missing"] += 1
        if script.lower() in (b"skillnormal", b"skilllock"):
            static = section_fields.get(b"staticdata", b"")
            if not static:
                missing_static += 1
            elif not static.isdigit():
                nonnumeric_static += 1
    try:
        data.decode("utf-8")
        valid_utf8 = True
    except UnicodeDecodeError:
        valid_utf8 = False
    blockers = []
    if not fields:
        blockers.append("no_sections")
    if longest_line > MAX_GO_SCANNER_TOKEN_BYTES:
        blockers.append("exceeds_go_scanner_token_limit")
    if b"\x00" in data:
        blockers.append("nul_bytes")
    # Duplicate INI headers are permitted by the current Go parser but must
    # be reported; reporting them is NOT grounds for inventing a merge rule.
    return {
        "sha256": hashlib.sha256(data).hexdigest(),
        "bytes": len(data),
        "line_count": len(data.splitlines()),
        "section_headers": sum(headers.values()),
        "unique_sections": len(fields),
        "duplicate_section_names": sum(count > 1 for count in headers.values()),
        "go_skill_index_candidates": eligible,
        "scripts": dict(sorted(scripts.items())),
        "missing_staticdata_in_candidates": missing_static,
        "nonnumeric_staticdata_in_candidates": nonnumeric_static,
        "ignored_nonfield_lines": ignored_nonfield_lines,
        "max_line_bytes": longest_line,
        "utf8_valid": valid_utf8,
        "nul_bytes": data.count(b"\x00"),
        "scanner_preflight_pass": not blockers,
        "scanner_blockers": blockers,
        "full_combat_compile_verified": False,
        "exact_current_client_provenance_verified": False,
        "live_e2e_verified": False,
    }


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("path", type=Path, help="local skill_new.ini; not uploaded")
    args = parser.parse_args()
    try:
        result = audit(args.path.read_bytes())
    except (OSError, ValueError) as exc:
        parser.exit(2, f"preflight unavailable: {exc}\n")
    print(json.dumps(result, ensure_ascii=True, sort_keys=True, indent=2))
    return 0 if result["scanner_preflight_pass"] else 1


if __name__ == "__main__":
    raise SystemExit(main())
