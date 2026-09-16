#!/usr/bin/env python3
"""Read-only, SHA-pinned PCK0 share.package content correlation (NOT index unpack).

Requires the owner's private share.package and five separately supplied Drive
reference files. Never commit the private output ZIP, package or reference bytes.
Unknown package member names must not be inferred from a content hash match.
"""
import argparse
import hashlib
import json
from pathlib import Path
import struct
import zipfile

import stage48_anonymous_zlib_recovery as stream_probe

SHARE_SIZE = 40680972
SHARE_SHA = '200497852ba3a29279e51f01e2913b5f7260480f2a740a32ebd1680b869844b6'
SHARE_COUNT = 9726
SHARE_INDEX_END = 790911
# Offset/sha pairs observed in THIS exact SHA-pinned package, not guessed names.
OBSERVED = {
    'shopsingle_external_exact': (13782597, '1cfaaba68306726cd8af4bf2847557c434256e5ca18d9af6ae4a1fa5deebc280'),
    'clonestore_external_exact': (3812575, 'b8e0910316455fc0d3ae15a42d1b6ffb32bbbaf808ad640cc12316c5ab140529'),
    'repairaddbag_external_exact': (24207113, 'dd3874943b0472e22ab28fa73b5aee7150c5c84a2e78156f5efebffce4f01ae7'),
    'shopfunc_external_prefix': (25407276, 'ece390caed52f45b7c71544f44642e437fbb55811259c0d7c24ffdd6289fcf5d'),
    'shop_catalog_candidate_A': (33744821, 'd2e058399d4016420846c8df1388c260181b1e81ceee9e51216ca80fcfd4787e'),
    'shop_catalog_candidate_B': (36877029, 'f42fce7631f7fe302d3fdedad60b12eaa20da12396de412ff0a84edcddc1a6d9'),
    'npc_reference_candidate': (25088049, None),
}
REFERENCES = {
    'shopsingle_external_exact': 'shopsingle.ini.bin',
    'clonestore_external_exact': 'clonestore.ini.bin',
    'repairaddbag_external_exact': 'repairaddbag.ini.bin',
    'shopfunc_external_prefix': 'shopfunc.ini.bin',
    'shop_catalog_candidate_A': 'shop.ini.bin',
    'shop_catalog_candidate_B': 'shop.ini.bin',
}
EXACT_KEYS = ('shopsingle_external_exact','clonestore_external_exact','repairaddbag_external_exact')
SHOP_SECTION_BASE = b'[Shop_GB_Yishiting]'


def sha(data):
    return hashlib.sha256(data).hexdigest()


def unique_lines(data, min_length=24):
    return {hashlib.blake2b(line.strip(), digest_size=12).digest()
            for line in data.splitlines() if len(line.strip()) >= min_length}


def overlap(a, b):
    """Exact common long-line counts; NOT proof of identical files/paths."""
    x, y = unique_lines(a), unique_lines(b)
    return {'reference_unique_long_lines': len(x),
            'candidate_unique_long_lines': len(y),
            'exact_common_long_lines': len(x & y)}


def validate_package(path):
    if path.stat().st_size != SHARE_SIZE:
        raise ValueError('unexpected share.package size')
    data=path.read_bytes()
    if sha(data) != SHARE_SHA:
        raise ValueError('unexpected share.package SHA-256')
    if data[:4]!=b'PCK0' or struct.unpack_from('<H',data,4)[0]!=15 or data[18]!=0:
        raise ValueError('unsupported primary package header')
    if struct.unpack_from('<II',data,10)!=(SHARE_COUNT,SHARE_INDEX_END):
        raise ValueError('unexpected declared count/index end')
    return data


def audit(share_file, references_dir):
    data=validate_package(share_file)
    records,tested,total=stream_probe.scan(data,SHARE_INDEX_END,
                                             max_streams=30000,total_limit=512*1024*1024)
    offsets={rec['start']:rec for rec in records}
    refs={name:(references_dir/name).read_bytes() for name in set(REFERENCES.values())}
    if any(not b for b in refs.values()):
        raise ValueError('empty external reference')
    found={}
    private_bytes={}
    for role,(start,expected_sha) in OBSERVED.items():
        if start not in offsets:
            raise ValueError('expected independent checksum-verified stream missing')
        record=offsets[start]
        end,raw=stream_probe.inflate(data,start)
        if end!=record['end'] or len(raw)!=record['raw_size'] or sha(raw)!=record['sha256']:
            raise ValueError('stream changed during independent decompression')
        if expected_sha is not None and sha(raw)!=expected_sha:
            raise ValueError('expected recorded SHA differs')
        assessment={'start':start,'end':end,'raw_bytes':len(raw),'sha256':sha(raw),
                    'original_member_path_verified':False}
        if role in REFERENCES:
            ref=refs[REFERENCES[role]]
            assessment['external_reference_basename']=REFERENCES[role]
            assessment['reference_sha256']=sha(ref)
            assessment['exact_external_content_match']=(raw==ref)
            if role in EXACT_KEYS and raw!=ref:
                raise ValueError(f'{role}: lost verified exact content match')
            if role=='shopfunc_external_prefix':
                if not raw.startswith(ref):
                    raise ValueError('shopfunc full prefix lost')
                assessment['reference_is_full_prefix']=True
                assessment['extra_tail_bytes']=len(raw)-len(ref)
            if role.startswith('shop_catalog_candidate'):
                lines=overlap(ref,raw)
                if lines['exact_common_long_lines'] < 43000:
                    raise ValueError('shop catalog candidate similarity no longer established')
                assessment['long_line_overlap']=lines
                assessment['base_shop_section_present']=SHOP_SECTION_BASE in raw
                assessment['suffix_sections_present']={str(n):raw.count(
                    ('[Shop_GB_Yishiting_%d]'%n).encode()) for n in range(1,6)}
        if role=='npc_reference_candidate':
            assessment['base_id_literal_occurrences']=raw.count(b'Shop_GB_Yishiting')
            if assessment['base_id_literal_occurrences']!=35:
                raise ValueError('NPC reference count differs')
        found[role]=assessment
        private_bytes[role]=raw
    return {'share_package_sha256':SHARE_SHA,'declared_entries_unverified':SHARE_COUNT,
            'verified_anonymous_streams':len(records),'candidate_header_offsets_tested':tested,
            'verified_anonymous_decompressed_bytes':total,'index_entries_decrypted':0,
            'original_member_names_verified':0,'assessments':found,
            'ordinary_shop_selector_verified':False,'purchase_ready':False,'live_e2e':False},private_bytes


def private_archive(path, summary, byte_map):
    if path.exists():
        raise FileExistsError('output already exists; refusing overwrite')
    partial=path.with_suffix(path.suffix+'.partial')
    if partial.exists():
        raise FileExistsError('partial output exists')
    try:
        with zipfile.ZipFile(partial,'x',compression=zipfile.ZIP_DEFLATED,compresslevel=6) as z:
            for role,b in byte_map.items():
                # These are roles/evidence labels, NOT recovered package filenames.
                z.writestr('anonymous_candidates/'+role+'.bin',b)
            z.writestr('manifest.json',json.dumps(summary,ensure_ascii=False,indent=2))
            z.writestr('README_KO.txt',
                'share.package에서 검증된 익명 압축 스트림 7개입니다. 원래 패키지 경로와 파일명은 미확인입니다.\n'
                'external_exact은 별도로 확보한 Drive 파일과 바이트가 같다는 뜻이지 인덱스 이름이 검증됐다는 뜻은 아닙니다.\n'
                'shop_catalog_candidate_A/B 중 어느 파일이 실제 클라이언트 선택 경로인지 확인되지 않았습니다.\n'
                '서버/게임 폴더에 덮어쓰거나 구매 기능을 활성화하지 마세요. 공개 GitHub에 이 ZIP을 올리지 마세요.\n')
        with zipfile.ZipFile(partial) as z:
            if z.testzip() is not None or len(z.namelist())!=len(byte_map)+2:
                raise ValueError('ZIP integrity failed')
            for role,b in byte_map.items():
                if z.read('anonymous_candidates/'+role+'.bin')!=b:
                    raise ValueError('ZIP byte readback mismatch')
        partial.rename(path)
    except BaseException:
        partial.unlink(missing_ok=True)
        raise
    return {'private_zip_bytes':path.stat().st_size,'private_zip_sha256':sha(path.read_bytes()),
            'private_zip_crc':'PASS','anonymous_evidence_members':len(byte_map)}


def self_test():
    assert overlap(b'long_identifier=placeholder\n',b'long_identifier=placeholder\n')['exact_common_long_lines']==1
    assert overlap(b'long_identifier=placeholder\n',b'other_identifier=placeholder\n')['exact_common_long_lines']==0
    assert not b'one'==b'one-tail'
    assert b'one-tail'.startswith(b'one')
    import tempfile
    with tempfile.TemporaryDirectory() as d:
        p=Path(d)/'test.zip'
        r={'synthetic':True}
        got=private_archive(p,r,{'example':b'fixture-data'})
        assert got['anonymous_evidence_members']==1 and got['private_zip_crc']=='PASS'
        with zipfile.ZipFile(p) as z:
            assert z.read('anonymous_candidates/example.bin')==b'fixture-data'
        try:private_archive(p,r,{'example':b'changed'})
        except FileExistsError:pass
        else:raise AssertionError('output overwrite not rejected')
        for bad in (Path(d)/'not-found',p):
            try:validate_package(bad)
            except (FileNotFoundError,ValueError):pass
            else:raise AssertionError('wrong package accepted')
    print('SELF_TEST_PASS: exact/near evidence, ZIP CRC/readback, overwrite/wrong-input rejection')


def main():
    ap=argparse.ArgumentParser(description=__doc__)
    ap.add_argument('--share',type=Path)
    ap.add_argument('--references-dir',type=Path)
    ap.add_argument('--output-private-zip',type=Path)
    ap.add_argument('--self-test',action='store_true')
    a=ap.parse_args()
    if a.self_test:
        self_test();return
    if not a.share or not a.references_dir:
        ap.error('--share and --references-dir required unless --self-test')
    result,raw=audit(a.share,a.references_dir)
    if a.output_private_zip:
        result['private_archive']=private_archive(a.output_private_zip,result,raw)
    print(json.dumps(result,ensure_ascii=False,indent=2))

if __name__=='__main__':main()
