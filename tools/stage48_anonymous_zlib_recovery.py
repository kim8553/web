#!/usr/bin/env python3
"""Stage48: checksum-verified anonymous zlib stream recovery from SHA-pinned PCK0.

No index decryption, no original member names, and no game binaries executed.
Output ZIP holds PRIVATE USER-SUPPLIED resource bytes; NEVER upload it to public CI/GitHub.
"""
import argparse
import hashlib
import io
import json
from pathlib import Path
import struct
import zipfile
import zlib

SOURCES = {
    "ini": (20578406, "6185812c6153b2a6da56071968dcd1254510a974df06a7e36a0229be2105779a", 18543, 1219224),
    "lua": (21840086, "283c8c245a3fb86af2a7e8c21c53b3de26c30432341dd590c37554adfce04e97", 2286, 187177),
}
RAW_LIMIT = 4 * 1024 * 1024
COMP_LIMIT = 4 * 1024 * 1024
TOTAL_RAW_LIMIT = 256 * 1024 * 1024
MAX_STREAMS = 30000
CHUNK = 16384
HEADERS = tuple(bytes((cmf, flg)) for cmf in range(8, 0x79, 0x10)
                for flg in range(256) if not(flg & 0x20) and ((cmf << 8) | flg) % 31 == 0)


def valid_input(path, kind):
    size, sha, count, index_end = SOURCES[kind]
    if path.stat().st_size != size:
        raise ValueError(f"{kind}: original package size differs")
    data = path.read_bytes()
    if hashlib.sha256(data).hexdigest() != sha:
        raise ValueError(f"{kind}: original package SHA256 differs")
    if data[:4] != b"PCK0" or struct.unpack_from('<H', data, 4)[0] != 15 or data[18] != 0:
        raise ValueError(f"{kind}: unexpected primary header")
    if struct.unpack_from('<II', data, 10) != (count, index_end):
        raise ValueError(f"{kind}: index metadata differs")
    return data, index_end


def inflate(data, start, raw_limit=RAW_LIMIT, comp_limit=COMP_LIMIT):
    """Return (end, raw) only if a complete bounded zlib stream passes Adler-32."""
    decoder = zlib.decompressobj()
    cursor = start
    raw = bytearray()
    while cursor < min(len(data), start + comp_limit):
        end = min(cursor + CHUNK, len(data), start + comp_limit)
        chunk = data[cursor:end]
        try:
            block = decoder.decompress(chunk, raw_limit - len(raw) + 1)
        except zlib.error:
            return None
        raw.extend(block)
        if len(raw) > raw_limit or decoder.unconsumed_tail:
            return None
        if decoder.eof:
            consumed = len(chunk) - len(decoder.unused_data)
            if consumed <= 0:
                return None
            return cursor + consumed, bytes(raw)
        cursor = end
    return None


def candidate_positions(data, lower_bound):
    positions = []
    for header in HEADERS:
        cursor = lower_bound
        while True:
            match = data.find(header, cursor)
            if match < 0:
                break
            positions.append(match)
            cursor = match + 1
    return sorted(set(positions))


def scan(data, lower_bound, max_streams=MAX_STREAMS, total_limit=TOTAL_RAW_LIMIT):
    records = []
    next_allowed = lower_bound
    raw_total = 0
    tested = 0
    for start in candidate_positions(data, lower_bound):
        if start < next_allowed:
            continue
        tested += 1
        result = inflate(data, start)
        if result is None:
            continue
        end, raw = result
        raw_total += len(raw)
        if raw_total > total_limit or len(records) >= max_streams:
            raise ValueError("aggregate extraction limit exceeded")
        records.append({"start": start, "end": end, "raw_size": len(raw),
                        "sha256": hashlib.sha256(raw).hexdigest()})
        next_allowed = end
    return records, tested, raw_total


def make_summary(kind, data, records, tested, raw_total):
    declared = SOURCES[kind][2]
    data_start = SOURCES[kind][3]
    gaps = [records[i]['start'] - records[i - 1]['end'] for i in range(1, len(records))]
    return {"source": kind, "package_sha256": SOURCES[kind][1],
            "declared_entries_not_verified": declared,
            "index_entries_decoded": 0,
            "named_package_members_verified": 0,
            "verified_anonymous_zlib_streams": len(records),
            "candidate_positions_tested": tested,
            "verified_compressed_bytes": sum(r['end'] - r['start'] for r in records),
            "verified_raw_bytes": raw_total,
            "interstream_nonzero_gaps": sum(g != 0 for g in gaps),
            "interstream_unclassified_bytes": sum(gaps),
            "before_first_stream_unclassified_bytes": records[0]['start'] - data_start if records else len(data) - data_start,
            "after_last_stream_unclassified_bytes": len(data) - records[-1]['end'] if records else 0,
            "full_package_unpack": False}


def produce_archive(destination, pairs):
    if destination.exists():
        raise FileExistsError(f"refusing to overwrite {destination}")
    temp = destination.with_name(destination.name + ".partial")
    if temp.exists():
        raise FileExistsError(f"refusing to overwrite {temp}")
    manifest = {"notice": "ANONYMOUS zlib streams, NOT named package files; index not decrypted.",
                "source_packages": {}, "members": []}
    try:
        with temp.open('xb') as stream, zipfile.ZipFile(stream, 'w', compression=zipfile.ZIP_DEFLATED, compresslevel=6, allowZip64=True) as archive:
            for kind, data, records, summary in pairs:
                manifest['source_packages'][kind] = summary
                for ordinal, record in enumerate(records):
                    path = f"{kind}/stream_{ordinal:05d}_at_{record['start']:08d}.bin"
                    result = inflate(data, record['start'])
                    if result is None or result[0] != record['end'] or len(result[1]) != record['raw_size'] or hashlib.sha256(result[1]).hexdigest() != record['sha256']:
                        raise ValueError(f"{kind}: changed stream during archive write")
                    archive.writestr(path, result[1])
                    manifest['members'].append({"anonymous_path": path, "source": kind, **record})
            archive.writestr('README_KO.txt', "익명 zlib 스트림 복구본입니다. 원본 package 파일명/경로 및 파일별 연결은 미확인입니다.\n"
                             "이 자료를 shop.ini 등으로 임의로 이름 붙이거나 게임 파일에 덮어쓰지 마세요.\n"
                             "manifest.json의 start/end는 원본 패키지 바이트 오프셋이며 체크섬 확인 스트림만 담았습니다.\n")
            archive.writestr('manifest.json', json.dumps(manifest, ensure_ascii=False, separators=(',', ':')))
        with zipfile.ZipFile(temp) as check:
            if check.testzip() is not None:
                raise ValueError('output ZIP CRC validation failed')
            if len(check.namelist()) != len(manifest['members']) + 2:
                raise ValueError('output ZIP member count mismatch')
        temp.rename(destination)
    except BaseException:
        temp.unlink(missing_ok=True)
        raise
    return {"zip_path": str(destination), "zip_bytes": destination.stat().st_size,
            "zip_sha256": hashlib.sha256(destination.read_bytes()).hexdigest(),
            "anonymous_members": len(manifest['members']), "zip_crc": "PASS"}


def self_test():
    raw_a = b'[example]\nname=test\n' * 15
    raw_b = b'\x1bLuaQ' + bytes(range(128))
    a = zlib.compress(raw_a)
    b = zlib.compress(raw_b, 0)  # tests non-0x789c header
    data = b'PCK0' + bytes(20) + a + b'\x99\xf0\x07' + b
    records, _, total = scan(data, 24, max_streams=5, total_limit=1024)
    assert len(records) == 2 and total == len(raw_a) + len(raw_b)
    assert records[0]['start'] == 24 and records[0]['end'] == 24 + len(a)
    assert records[1]['start'] == 27 + len(a) and records[1]['end'] == len(data)
    assert inflate(a[:-1], 0) is None
    corrupted = bytearray(a); corrupted[-1] ^= 0xff
    assert inflate(corrupted, 0) is None
    assert inflate(a, 0, raw_limit=16) is None
    assert inflate(a, 0, comp_limit=3) is None
    assert not scan(b'no zlib here', 0, max_streams=5, total_limit=1024)[0]
    print('SELF_TEST_PASS: both zlib header variants, gaps, truncated/checksum/bomb/compressed-cap/empty rejected')


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--ini', type=Path)
    parser.add_argument('--lua', type=Path)
    parser.add_argument('--output-zip', type=Path, help='PRIVATE ZIP, never commit/upload to public CI')
    parser.add_argument('--self-test', action='store_true')
    args = parser.parse_args()
    if args.self_test:
        self_test()
        return
    if not args.ini or not args.lua:
        parser.error('--ini and --lua required')
    pairs = []
    for kind, path in [('ini', args.ini), ('lua', args.lua)]:
        data, start = valid_input(path, kind)
        records, tested, raw_total = scan(data, start)
        pairs.append((kind, data, records, make_summary(kind, data, records, tested, raw_total)))
    result = {"observations": [p[3] for p in pairs],
              "shop_ini_identified": False, "client_package_index_decrypted": False,
              "gameplay_modified": False, "live_e2e_verified": False}
    if args.output_zip:
        result['private_archive'] = produce_archive(args.output_zip, pairs)
    print(json.dumps(result, ensure_ascii=False, indent=2))


if __name__ == '__main__':
    main()
