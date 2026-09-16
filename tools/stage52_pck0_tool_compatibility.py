#!/usr/bin/env python3
"""Stage52 read-only SHA-pinned PCK0 TOC compatibility gate, NOT a decryptor.

Public tool handles only local owner-provided package paths, prints metadata only.
No external tool or client game executable is run; no game resource bytes emitted.
Third-party parser models are restricted to directly observed header/offset rules.
"""
import argparse
import hashlib
import json
import struct
from pathlib import Path

EXPECTED = {
    'ini': {'size':20578406, 'sha256':'6185812c6153b2a6da56071968dcd1254510a974df06a7e36a0229be2105779a','entries':18543,'index_end':1219224},
    'lua': {'size':21840086, 'sha256':'283c8c245a3fb86af2a7e8c21c53b3de26c30432341dd590c37554adfce04e97','entries':2286,'index_end':187177},
    'share': {'size':40680972, 'sha256':'200497852ba3a29279e51f01e2913b5f7260480f2a740a32ebd1680b869844b6','entries':9726,'index_end':790911},
}
LEGACY_TEN = b'PCK0\x0f\x00\x00\x00\x00\x00'


def gate(package: bytes, expected: dict) -> dict:
    if len(package) != expected['size'] or hashlib.sha256(package).hexdigest() != expected['sha256']:
        raise ValueError('package identity differs from verified owner copy')
    if len(package) < 47 or package[:4] != b'PCK0' or package[18] != 0:
        raise ValueError('invalid PCK0 header')
    version, flags = struct.unpack_from('<HH',package,4)
    entries, index_end = struct.unpack_from('<II',package,10)
    if (version,entries,index_end)!=(15,expected['entries'],expected['index_end']):
        raise ValueError('unexpected version, entry count or index end')
    if not 19 <= index_end < len(package):
        raise ValueError('index boundary outside package')
    first_length = struct.unpack_from('<H',package,19)[0]
    first_offset_32 = struct.unpack_from('<I',package,21)[0]
    first_offset_64 = struct.unpack_from('<Q',package,21)[0]
    first_raw_size = struct.unpack_from('<I',package,29)[0]
    first_comp_size = struct.unpack_from('<I',package,33)[0]
    # The three public tools' *observed* original-index expectations only:
    # Wushu.Utils.Package/Unpacker.cs requires 10-byte fixed legacy header.
    # russell662/JiuYinUnpackTool/src/pck.rs rejects flags !=0.
    # AOWPackageExtractor/Program.cs reads first 32-bit data offset raw at index+2.
    # AOWPackageExtractorCpp/AOWPackageExtractorCpp.cpp reads raw TOC; no transform.
    old_header_ok = package[:10] == LEGACY_TEN
    raw_range_32 = first_offset_32 <= len(package) and first_comp_size <= len(package)-first_offset_32
    raw_range_64 = first_offset_64 <= len(package) and first_comp_size <= len(package)-first_offset_64
    # We do not attempt to decode filename bytes, interpret nonce, or infer crypto.
    return {
        'original_package_sha256_verified': True,
        'pck_header': {'version_u16':version,'flags_u16_uninterpreted':flags,
                       'declared_entries_unverified':entries,'index_start':19,
                       'index_end':index_end,'index_bytes':index_end-19},
        'legacy_public_tool_checks': {
            'Ersanio_wushu_utils_header_exact_match':old_header_ok,
            'russell662_JiuYinUnpackTool_flags_zero_required':flags == 0,
            'ramazanaktolu_AOWPackageExtractor_first_raw_index_data_range_valid':raw_range_32,
            'ramazanaktolu_AOWPackageExtractorCpp_first_raw_index_data_range_valid_under_le_u32':raw_range_32,
        },
        'first_raw_record_probe_NOT_DECODED': {'u16_record_length':first_length,
            'u32_candidate_offset':first_offset_32,'u64_candidate_offset':first_offset_64,
            'u32_candidate_uncompressed_size':first_raw_size,'u32_candidate_compressed_size':first_comp_size,
            'raw_u64_range_valid':raw_range_64},
        'actual_index_decoder_implemented':False,
        'named_members_verified':0,
        'complete_unpack':False,
    }


def self_test():
    # Synthetic short clear index with a valid single zlib member: only metadata
    # checks here; no synthetic pass may be counted as actual package extraction.
    import zlib
    raw=b'[test]\nvalue=1\n'
    comp=zlib.compress(raw)
    name=b'x.ini\0'
    record_len=27+len(name)
    index_end=19+record_len
    header=b'PCK0'+struct.pack('<H',15)+b'\0'*4+struct.pack('<II',1,index_end)+b'\0'
    record=struct.pack('<HQII',record_len,index_end,len(raw),len(comp))+b'\0'*7+struct.pack('<H',0)+name
    blob=header+record+comp
    ex={'size':len(blob),'sha256':hashlib.sha256(blob).hexdigest(),'entries':1,'index_end':index_end}
    p=gate(blob,ex)
    assert all(p['legacy_public_tool_checks'].values())
    assert p['first_raw_record_probe_NOT_DECODED']['raw_u64_range_valid']
    modified=bytearray(blob);modified[6]=4
    ex4={**ex,'sha256':hashlib.sha256(modified).hexdigest()}
    p4=gate(bytes(modified),ex4)
    assert not p4['legacy_public_tool_checks']['Ersanio_wushu_utils_header_exact_match']
    assert not p4['legacy_public_tool_checks']['russell662_JiuYinUnpackTool_flags_zero_required']
    # Header flag alone does not imply record bytes are unintelligible.
    assert p4['first_raw_record_probe_NOT_DECODED']['raw_u64_range_valid']
    for tampered in (blob[:-1],blob[:19]+bytes((blob[19]^1,))+blob[20:]):
        try:gate(tampered,ex)
        except ValueError: pass
        else:raise AssertionError('tampered input accepted')
    print('STAGE52_SELF_TEST_PASS: synthetic legacy and flag-4 comparison, no inferred encryption, two tamper rejections')


def main():
    ap=argparse.ArgumentParser(description=__doc__)
    for kind in EXPECTED: ap.add_argument('--'+kind,type=Path)
    ap.add_argument('--self-test',action='store_true')
    a=ap.parse_args()
    if a.self_test:self_test();return
    result={}
    for name, expected in EXPECTED.items():
        path=getattr(a,name)
        if path is None:ap.error('all three original --ini --lua --share required')
        result[name]=gate(path.read_bytes(),expected)
    print(json.dumps(result,indent=2,ensure_ascii=False))

if __name__=='__main__':main()
