#!/usr/bin/env python3
"""Read-only Stage46 identity/index gate. NEVER decrypts, extracts or executes game files.

Use only with the user's private local inputs; publish only this tool and aggregate
metadata, never package contents, game executable bytes or keys.
"""
import argparse
import hashlib
import json
from pathlib import Path
import struct
import tempfile
import zipfile

EXPECTED = {
    'fxgame.exe': 'c7f0366026452d525dc0c74b8499aac30fdb69189591420978e12c7094b5fde3',
    'fxgamelogic.dll': '16cf49b652c39b63ba0767780cc39f32ea1e50bdc683da9d5e7b2697cf95c3e8',
    'fxnet2.dll': '0443d9f401fdcc6a92a391869c898780ffac93399913dc9a82c7cac8cf6c3fde',
    'fxcore.dll': 'ce0da0f52a80db5033e177a59be868c44b0baeca4a71226b0ccb2aca76cae724',
    'fxres.exe': '07ae76288148132995538488f12e2214fbecfdc0f18bdc2dd3189093e3e9fa9c',
}
PACKAGE_SHA = {
    'ini': '6185812c6153b2a6da56071968dcd1254510a974df06a7e36a0229be2105779a',
    'lua': '283c8c245a3fb86af2a7e8c21c53b3de26c30432341dd590c37554adfce04e97',
}
# Earlier Stage39 verified these instruction bytes on SHA-256 07ae...fxres.exe.
# They identify the existing header path, NOT a decryption implementation.
FXRES_FINGERPRINTS = {
    0x14000da21: '81 7c 24 58 50 43 4b 30',
    0x14000da80: '0f b7 4c 24 40',
    0x14000da85: '8d 41 f1',
    0x14000dade: '49 8d 56 02',
    0x14000db30: '42 80 7c 30 0e 00',
    0x14000db72: '41 8b 4e 0a',
    0x14000db7a: '8d 42 04',
    0x14000dbbb: '41 8b 46 06',
    0x14000dbc3: '3d 00 00 10 00',
    0x14000dc04: '44 8b e9 44 2b ea 41 83 ed 04',
    0x14000dcaa: '4d 85 ff',
    0x14000dcc0: '0f b7 07',
    0x14000deeb: '41 8b 46 02',
}


def sha_file(path):
    with path.open('rb') as f:
        return hashlib.file_digest(f, 'sha256').hexdigest()


def validate_fxres(binary):
    if len(binary) < 512 or binary[:2] != b'MZ':
        raise ValueError('fxres: missing MZ')
    pe = struct.unpack_from('<I', binary, 0x3c)[0]
    if pe > len(binary) - 24 or binary[pe:pe+4] != b'PE\0\0':
        raise ValueError('fxres: missing PE')
    machine, sections = struct.unpack_from('<HH', binary, pe+4)
    optional_len = struct.unpack_from('<H', binary, pe+20)[0]
    opt = pe+24
    if machine != 0x8664 or not 0 < sections <= 96 or opt + optional_len + sections*40 > len(binary):
        raise ValueError('fxres: unexpected PE layout')
    if struct.unpack_from('<H', binary, opt)[0] != 0x20b:
        raise ValueError('fxres: not PE32+')
    base = struct.unpack_from('<Q', binary, opt+24)[0]
    if base != 0x140000000:
        raise ValueError('fxres: unexpected image base')
    text = []
    for i in range(sections):
        p = opt+optional_len+i*40
        name = binary[p:p+8].split(b'\0')[0]
        virt, rva, raw, off = struct.unpack_from('<IIII', binary, p+8)
        characteristics = struct.unpack_from('<I', binary, p+36)[0]
        if off > len(binary) or raw > len(binary)-off:
            raise ValueError('fxres: section out of file')
        if name == b'.text' and characteristics & 0x20000000:
            text.append((rva,raw,off))
    if len(text) != 1:
        raise ValueError('fxres: expected one executable .text')
    rva,raw,off=text[0]
    for va, code in FXRES_FINGERPRINTS.items():
        expected=bytes.fromhex(code)
        rel=va-base-rva
        if rel < 0 or rel+len(expected)>raw or binary[off+rel:off+rel+len(expected)]!=expected:
            raise ValueError(f'fxres: instruction bytes differ at 0x{va:x}')
    return len(FXRES_FINGERPRINTS)


def inspect_package(path, expected_sha=None):
    size=path.stat().st_size
    if not 19 <= size <= 128*1024*1024:
        raise ValueError('package: size outside bounded scope')
    digest=sha_file(path)
    if expected_sha is not None and digest != expected_sha:
        raise ValueError('package: SHA-256 differs from Stage45 authority')
    with path.open('rb') as f:
        b=f.read(46)
    if b[:4] != b'PCK0':
        raise ValueError('package: PCK0 magic absent')
    length=struct.unpack_from('<H',b,4)[0]
    field6=struct.unpack_from('<I',b,6)[0]
    count=struct.unpack_from('<I',b,10)[0]
    end=struct.unpack_from('<I',b,14)[0]
    if length != 15 or b[18] != 0 or not 0<count<=0x100000 or not 19<=end<=size:
        raise ValueError('package: unsupported primary header or invalid index boundary')
    if len(b)<19+18:
        raise ValueError('package: insufficient first 27-byte-record probe')
    # The upstream Rust pck.rs accepts index flags == 0, interpreting bytes 6:8.
    # Here both inputs contain 4; never force the plaintext parser to proceed.
    plain_flag=struct.unpack_from('<H',b,6)[0]
    candidate_record_size=struct.unpack_from('<H',b,19)[0]
    candidate_data_offset=struct.unpack_from('<Q',b,21)[0]
    candidate_comp_size=struct.unpack_from('<I',b,33)[0]
    return {
        'size_bytes':size,'sha256':digest,'primary_record_length_not_version':length,
        'uninterpreted_u32_at_file_offset_6':f'0x{field6:08x}',
        'u16_at_file_offset_6_seen_by_upstream_plain_parser':plain_flag,
        'upstream_plain_parser_requires_u16_zero':True,
        'upstream_plain_parser_compatible':plain_flag==0,
        'declared_entry_count_not_verified':count,
        'index_start':19,'index_end_exclusive':end,
        'raw_index_bytes':end-19,
        'naive_plain_first_record_length':candidate_record_size,
        'naive_plain_first_data_offset_in_bounds':candidate_data_offset<=size and candidate_comp_size<=size-candidate_data_offset,
        'index_decrypted':False,'index_entries_parsed':0,'named_resources_extracted':0,
    }


def audit(zip_path, ini, lua):
    if not 0<zip_path.stat().st_size<=256*1024*1024:
        raise ValueError('ZIP outside bounded scope')
    with zipfile.ZipFile(zip_path) as z:
        members=z.infolist()
        if not 0<len(members)<=256 or sum(i.file_size for i in members)>512*1024*1024:
            raise ValueError('ZIP entry or decompressed-byte cap exceeded')
        names=[i.filename.casefold() for i in members]
        if len(set(names))!=len(names) or any('/' in n or '\\' in n or '..' in n for n in names):
            raise ValueError('ZIP duplicate/path-containing member')
        if z.testzip() is not None:
            raise ValueError('ZIP CRC test failed')
        verified={}
        for name,sha in EXPECTED.items():
            if name not in names:
                raise ValueError(f'ZIP: missing {name}')
            info=members[names.index(name)]
            if not 0<info.file_size<=64*1024*1024:
                raise ValueError(f'ZIP: unexpected {name} size')
            binary=z.read(info)
            actual=hashlib.sha256(binary).hexdigest()
            if actual!=sha:
                raise ValueError(f'ZIP: {name} SHA differs from Stage46 observed client')
            verified[name]={'bytes':len(binary),'sha256':actual}
            if name=='fxres.exe':
                verified[name]['stage39_instruction_fingerprints_matching']=validate_fxres(binary)
    return {'read_only':True,'zip_sha256':sha_file(zip_path),'zip_crc_all_members_pass':True,
            'zip_members':len(members),'five_client_binary_identities':verified,
            'private_packages':{'ini':inspect_package(ini,PACKAGE_SHA['ini']),
                                'lua':inspect_package(lua,PACKAGE_SHA['lua'])},
            'index_decode_method_known':False,'named_shop_resource_verified':False,
            'npc_shop_live_e2e':False}


def self_test():
    with tempfile.TemporaryDirectory() as td:
        p=Path(td)/'fixture.package'
        def fixture(field6=0x33a10004,count=2,end=70,magic=b'PCK0',terminator=0):
            return magic+struct.pack('<HIII',15,field6,count,end)+bytes([terminator])+bytes(100)
        p.write_bytes(fixture())
        out=inspect_package(p)
        assert out['u16_at_file_offset_6_seen_by_upstream_plain_parser']==4
        assert not out['upstream_plain_parser_compatible']
        assert out['index_entries_parsed']==0 and not out['index_decrypted']
        for bad in (fixture(magic=b'FAIL'),fixture(count=0),fixture(count=0x100001),
                    fixture(end=18),fixture(end=1000),fixture(terminator=1),
                    b'PCK0'+struct.pack('<H',16)+bytes(110)):
            p.write_bytes(bad)
            try:inspect_package(p)
            except ValueError:pass
            else:raise AssertionError('invalid primary header accepted')
    print('SELF_TEST_PASS: plaintext incompatibility + seven invalid-header variants')


if __name__=='__main__':
    parser=argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--zip',type=Path)
    parser.add_argument('--ini',type=Path)
    parser.add_argument('--lua',type=Path)
    parser.add_argument('--self-test',action='store_true')
    args=parser.parse_args()
    if args.self_test:self_test()
    else:
        if not all((args.zip,args.ini,args.lua)):
            parser.error('--zip --ini --lua required except for --self-test')
        print(json.dumps(audit(args.zip,args.ini,args.lua),indent=2,ensure_ascii=False))
