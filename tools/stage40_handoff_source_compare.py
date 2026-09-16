#!/usr/bin/env python3
"""Read-only SHA-256 comparison of a Stage37 handoff and an Actions source manifest.

This checks preservation of paths and bytes, not runtime behavior or merge safety.
It never extracts files, executes binaries, modifies a repository, or enables 0x4F.
"""
from __future__ import annotations

import argparse
import hashlib
import io
import json
from pathlib import Path, PurePosixPath
import re
import sys
import zipfile

ROW = re.compile(r"^([0-9a-f]{64})  (.+)$")
HANDOFF_WORKTREE = "STAGE37_CURRENT_WORKTREE/"
MAX_ARCHIVE_BYTES = 64 * 1024 * 1024
MAX_MANIFEST_BYTES = 1024 * 1024
MAX_HANDOFF_ENTRY_BYTES = 16 * 1024 * 1024


def sha256(value: bytes) -> str:
    return hashlib.sha256(value).hexdigest()


def validate_relative(path: str) -> str:
    if not path or path.startswith("/") or "\\" in path or "\x00" in path:
        raise ValueError("unsafe or empty archive path")
    if any(part in ("", ".", "..") for part in path.split("/")):
        raise ValueError("noncanonical archive path")
    return path


def parse_manifest(text: str, prefix: str = "") -> dict[str, str]:
    rows: dict[str, str] = {}
    for line in text.splitlines():
        match = ROW.fullmatch(line)
        if not match:
            raise ValueError("invalid manifest line")
        digest, source_name = match.groups()
        path = source_name.removeprefix("./")
        if prefix:
            if not path.startswith(prefix):
                continue
            path = path[len(prefix):]
        path = validate_relative(path)
        if path in rows:
            raise ValueError("duplicate normalized manifest path")
        rows[path] = digest
    if not rows:
        raise ValueError("empty manifest")
    return rows


def read_zip(path: Path, expected_hash: str | None) -> zipfile.ZipFile:
    if not path.is_file() or path.stat().st_size > MAX_ARCHIVE_BYTES:
        raise ValueError("missing or oversized archive")
    if expected_hash and sha256(path.read_bytes()) != expected_hash:
        raise ValueError("archive SHA-256 mismatch")
    return zipfile.ZipFile(path)


def handoff_manifest(archive: zipfile.ZipFile) -> tuple[dict[str, str], int]:
    entries = [entry for entry in archive.infolist() if not entry.is_dir()]
    if not entries or len(entries) > 1000:
        raise ValueError("unexpected handoff entry count")
    first = entries[0].filename.split("/", 1)[0] + "/"
    normalized: dict[str, zipfile.ZipInfo] = {}
    for item in entries:
        name = item.filename
        # Some archives store UTF-8 Chinese filenames with the UTF-8 bit unset.
        if not (item.flag_bits & 0x800):
            try:
                candidate = name.encode("cp437").decode("utf-8")
                if candidate.startswith(first):
                    name = candidate
            except (UnicodeEncodeError, UnicodeDecodeError):
                pass
        if not name.startswith(first):
            raise ValueError("handoff entry outside one root directory")
        relative = validate_relative(name[len(first):])
        if relative in normalized:
            raise ValueError("duplicate normalized ZIP member")
        if item.file_size > MAX_HANDOFF_ENTRY_BYTES:
            raise ValueError("oversized handoff member")
        normalized[relative] = item
    manifest_item = normalized.get("HANDOFF_MANIFEST_SHA256.txt")
    if manifest_item is None or manifest_item.file_size > MAX_MANIFEST_BYTES:
        raise ValueError("handoff manifest missing or oversized")
    manifest = parse_manifest(archive.read(manifest_item).decode("utf-8-sig"))
    expected_members = set(manifest) | {"HANDOFF_MANIFEST_SHA256.txt"}
    if set(normalized) != expected_members:
        raise ValueError("handoff has extra or missing manifest entries")
    for name, digest in manifest.items():
        if sha256(archive.read(normalized[name])) != digest:
            raise ValueError("handoff SHA-256 mismatch: " + name)
    worktree = {name[len(HANDOFF_WORKTREE):]: digest for name, digest in manifest.items()
                if name.startswith(HANDOFF_WORKTREE)}
    if not worktree:
        raise ValueError("handoff has no worktree")
    return worktree, len(manifest)


def actions_manifest(archive: zipfile.ZipFile) -> dict[str, str]:
    info = archive.getinfo("generated-manifest.txt")
    if info.file_size > MAX_MANIFEST_BYTES:
        raise ValueError("Actions manifest oversized")
    body = archive.read(info)
    recorded = archive.read("generated-manifest.sha256").decode("utf-8").split()
    if len(recorded) != 2 or not re.fullmatch(r"[0-9a-f]{64}", recorded[0]):
        raise ValueError("invalid Actions manifest checksum")
    if sha256(body) != recorded[0]:
        raise ValueError("Actions manifest SHA-256 mismatch")
    return parse_manifest(body.decode("utf-8"))


def compare(handoff: dict[str, str], actions: dict[str, str]) -> dict[str, object]:
    common = handoff.keys() & actions.keys()
    same = sorted(path for path in common if handoff[path] == actions[path])
    changed = sorted(path for path in common if handoff[path] != actions[path])
    missing = sorted(handoff.keys() - actions.keys())
    added = sorted(actions.keys() - handoff.keys())
    return {"handoff_worktree_files": len(handoff), "actions_source_files": len(actions),
            "unchanged": len(same), "changed": changed,
            "missing_from_actions": missing, "new_in_actions": added,
            "baseline_paths_preserved": not missing,
            "fully_merged_and_live_verified": False}


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("handoff_zip", type=Path)
    parser.add_argument("actions_zip", type=Path)
    parser.add_argument("--handoff-sha256", default=None)
    parser.add_argument("--actions-sha256", default=None)
    options = parser.parse_args()
    try:
        with read_zip(options.handoff_zip, options.handoff_sha256) as old, read_zip(options.actions_zip, options.actions_sha256) as new:
            if old.testzip() is not None or new.testzip() is not None:
                raise ValueError("ZIP CRC failure")
            old_files, checked = handoff_manifest(old)
            new_files = actions_manifest(new)
        result = compare(old_files, new_files)
        result["handoff_manifest_verified"] = checked
        print(json.dumps(result, ensure_ascii=False, indent=2))
        return 0 if result["baseline_paths_preserved"] else 2
    except (ValueError, OSError, KeyError, zipfile.BadZipFile, UnicodeError) as exc:
        print("COMPARE_BLOCKED: " + str(exc), file=sys.stderr)
        return 2


if __name__ == "__main__":
    raise SystemExit(main())
