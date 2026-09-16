#!/usr/bin/env python3
"""Verify the fixed Sep-15 assistant handoff and extract its source only.

For use in an ephemeral GitHub Actions checkout; never touches the Git tree.
"""
import argparse
import hashlib
import pathlib
import re
import stat
import sys
import zipfile
from collections import defaultdict

EXPECTED_ARCHIVE_SHA256 = 'e898008a0ec1aef39d16663835c9f479daed6e643c4c90252c5adafc3f4ffe1e'
TOP = 'JIUYIN_STAGE37_NEW_CHAT_HANDOFF_20260915_BIND_LUA_CURRENT/'
SOURCE = 'STAGE37_CURRENT_WORKTREE/'
MANIFEST = TOP + 'HANDOFF_MANIFEST_SHA256.txt'
RECORD = re.compile(r'^([0-9a-f]{64})  \./(.+)$')


def verify_and_extract(archive: pathlib.Path, output: pathlib.Path) -> tuple[int, int]:
    if hashlib.sha256(archive.read_bytes()).hexdigest() != EXPECTED_ARCHIVE_SHA256:
        raise ValueError('archive SHA256 mismatch; no files were changed')
    with zipfile.ZipFile(archive) as z:
        if z.testzip() is not None:
            raise ValueError('archive CRC failure')
        files = [i for i in z.infolist() if not i.is_dir()]
        if len(files) != 190 or MANIFEST not in z.namelist():
            raise ValueError('unexpected ZIP file set')
        digest_map = defaultdict(list)
        data = {}
        for info in files:
            if stat.S_ISLNK(info.external_attr >> 16):
                raise ValueError('ZIP symlink forbidden')
            if not info.filename.startswith(TOP) or info.file_size > 12_000_000:
                raise ValueError('unexpected path or file size')
            raw = z.read(info)
            digest_map[hashlib.sha256(raw).hexdigest()].append(info.filename)
            data[info.filename] = raw
        records = []
        for row in data[MANIFEST].decode('utf-8').splitlines():
            match = RECORD.fullmatch(row)
            if not match:
                raise ValueError('invalid manifest line')
            want_sha, relative = match.groups()
            path = pathlib.PurePosixPath(relative)
            if path.is_absolute() or '..' in path.parts or not path.parts or '\\' in relative:
                raise ValueError('manifest path traversal')
            exact = TOP + relative
            candidates = [exact] if exact in data else [n for n in digest_map[want_sha] if n != MANIFEST]
            candidates = [n for n in candidates if hashlib.sha256(data[n]).hexdigest() == want_sha]
            if len(candidates) != 1:
                raise ValueError(f'missing/ambiguous manifest payload: {relative}')
            records.append((relative, candidates[0]))
        used = [name for _, name in records]
        if len(records) != 189 or len(set(used)) != 189 or set(used) != set(data) - {MANIFEST}:
            raise ValueError('manifest/archive file set mismatch')
        source_files = [(p[len(SOURCE):], name) for p, name in records if p.startswith(SOURCE)]
        if len(source_files) != 186:
            raise ValueError('source file count mismatch')
        output.mkdir(parents=True, exist_ok=True)
        for relative, name in source_files:
            target = output.joinpath(*pathlib.PurePosixPath(relative).parts)
            target.parent.mkdir(parents=True, exist_ok=True)
            target.write_bytes(data[name])
        return len(records), len(source_files)


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument('zip_path', type=pathlib.Path)
    parser.add_argument('output_directory', type=pathlib.Path)
    args = parser.parse_args()
    try:
        records, source_files = verify_and_extract(args.zip_path, args.output_directory)
        print(f'ARCHIVE_SHA256=PASS ZIP_CRC=PASS MANIFEST={records}/{records} SOURCE_FILES={source_files}')
        return 0
    except (ValueError, OSError, zipfile.BadZipFile) as exc:
        print(f'HANDOFF_IMPORT_BLOCKED: {exc}', file=sys.stderr)
        return 1


if __name__ == '__main__':
    raise SystemExit(main())
