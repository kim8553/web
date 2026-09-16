#!/usr/bin/env python3
"""SHA-pinned, read-only fxres record-check audit; does NOT decode private index.

Works alongside stage46_current_bin64_index_gate.py. Synthetic clear-index
fixtures are NOT named members of either real original client package.
"""
import argparse
import hashlib
import json
import struct
from pathlib import Path
import zipfile
import stage46_current_bin64_index_gate as baseline

# Independently observed objdump instruction bytes, AFTER the opaque protected
# section call. Do not infer that this call decrypts the index.
RECORD_CODE = {
    0x14000dc3b:'48 8b d7', 0x14000dc3e:'48 8b ce',
    0x14000dc41:'e8 0a f4 ff ff', 0x14000dc99:'48 8b ce',
    0x14000dc9c:'e8 15 94 1f 00', 0x14000dcb3:'48 8d 47 02',
    0x14000dcb7:'49 3b c5', 0x14000dcc0:'0f b7 07',
    0x14000dcd4:'4d 8d 0c 38', 0x14000dcd8:'4d 3b cd',
    0x14000dce4:'83 c1 e4', 0x14000dce7:'80 7f 1b 00',
    0x14000dcf1:'80 7c 39 1b 00', 0x14000dcfc:'0f b7 47 19',
    0x14000dd00:'3b c1', 0x14000dd0c:'49 8b f9',
    0x14000dd35:'48 8b 56 38', 0x14000dd39:'48 83 c2 1b',
    0x14000dd52:'49 3b ef',
}
DIAGNOSTICS = {
    0x140049538:b'(CPackData::LoadFromFile)read file info size failed',
    0x140049570:b'(CPackData::LoadFromFile)file info size error',
    0x1400495a0:b'(CPackData::LoadFromFile)read file info failed',
    0x1400495d0:b'(CPackData::LoadFromFile)file name error',
    0x140049600:b'(CPackData::LoadFromFile)comment offset error',
}


def file_offset(binary, section, va, size):
    if len(binary)<512 or binary[:2]!=b'MZ':
        raise ValueError('not MZ')
    pe=struct.unpack_from('<I',binary,0x3c)[0]
    if pe>len(binary)-24 or binary[pe:pe+4]!=b'PE\0\0':
        raise ValueError('not PE')
    machine,count=struct.unpack_from('<HH',binary,pe+4)
    optlen=struct.unpack_from('<H',binary,pe+20)[0]
    opt=pe+24
    if machine!=0x8664 or not 0<count<=96 or opt+optlen+40*count>len(binary):
        raise ValueError('PE section limits')
    if struct.unpack_from('<H',binary,opt)[0]!=0x20b:
        raise ValueError('not PE32+')
    base=struct.unpack_from('<Q',binary,opt+24)[0]
    if base!=0x140000000:
        raise ValueError('image-base mismatch')
    matches=[]
    for i in range(count):
        p=opt+optlen+40*i
        name=binary[p:p+8].split(b'\0',1)[0]
        _,rva,raw,off=struct.unpack_from('<IIII',binary,p+8)
        if off>len(binary) or raw>len(binary)-off:
            raise ValueError('PE section outside file')
        rel=va-base-rva
        if name==section and rel>=0 and size<=raw-rel:
            matches.append(off+rel)
    if len(matches)!=1:
        raise ValueError('missing/ambiguous PE section')
    return matches[0]


def postread_audit(binary):
    if (len(binary)>20*1024*1024 or hashlib.sha256(binary).hexdigest()!=baseline.EXPECTED['fxres.exe']
            or baseline.validate_fxres(binary)!=13):
        raise ValueError('Stage39/46 fxres identity mismatch')
    for va,hexbytes in RECORD_CODE.items():
        expected=bytes.fromhex(hexbytes)
        at=file_offset(binary,b'.text',va,len(expected))
        if binary[at:at+len(expected)]!=expected:
            raise ValueError(f'instruction mismatch at 0x{va:x}')
    for va,expected in DIAGNOSTICS.items():
        at=file_offset(binary,b'.rdata',va,len(expected)+1)
        if binary[at:at+len(expected)+1]!=expected+b'\0':
            raise ValueError(f'diagnostic mismatch at 0x{va:x}')
    return {'same_stage39_fxres':True,'record_instruction_sites_verified':len(RECORD_CODE),
            'diagnostic_sites_verified':len(DIAGNOSTICS),
            'clear_record_length_u16_offset':0,'clear_record_name_starts_at_offset':27,
            'name_first_nonzero_and_record_last_nul':True,
            'u16_offset_25_compared_against_record_length_minus_28':True,
            'opaque_call_target':'0x1402070b6','opaque_call_role_known':False,
            'real_index_decrypted_or_parsed':False}


def synthetic_clear_index(index,count):
    """Test-only mirror of observable record boundaries; never call on client data."""
    if not isinstance(index,bytes) or not 0<=count<=128 or len(index)>16384:
        raise ValueError('fixture cap')
    pos=0
    for _ in range(count):
        if len(index)-pos<2:
            raise ValueError('short record length')
        n=struct.unpack_from('<H',index,pos)[0]
        if n<29 or n>len(index)-pos:
            raise ValueError('record bounds')
        rec=index[pos:pos+n]
        if not rec[27] or rec[-1]!=0 or struct.unpack_from('<H',rec,25)[0]>=n-28:
            raise ValueError('record name/offset bounds')
        pos+=n
    if pos!=len(index):
        raise ValueError('unconsumed bytes')
    return count


def self_test():
    baseline.self_test()
    def record(name=b'a\0',offset=0):
        out=bytearray(27+len(name))
        struct.pack_into('<H',out,0,len(out))
        struct.pack_into('<H',out,25,offset)
        out[27:]=name
        return bytes(out)
    a=record()
    assert synthetic_clear_index(a+record(b'bc\0',1),2)==2
    assert synthetic_clear_index(b'',0)==0
    for data,count in [(a[:-1],1),(record(b'\0'),1),(record(b'abX'),1),
                       (record(offset=1),1),(a+b'x',1),(a,2),(b'\x01\0',1),
                       (a,129),(b'x'*16385,0)]:
        try:synthetic_clear_index(data,count)
        except ValueError:pass
        else:raise AssertionError('invalid synthetic index accepted')
    try:postread_audit(b'bad')
    except ValueError:pass
    else:raise AssertionError('non-fxres accepted')
    print('STAGE47_SELF_TEST_PASS: 2 positive and 9 negative synthetic records; wrong fxres rejected')


def audit(zip_path,ini,lua):
    result=baseline.audit(zip_path,ini,lua)
    with zipfile.ZipFile(zip_path) as archive:
        matches=[i for i in archive.infolist() if i.filename.casefold()=='fxres.exe']
        if len(matches)!=1 or matches[0].file_size>20*1024*1024:
            raise ValueError('nonunique/unbounded fxres')
        result['fxres_postread_record_audit']=postread_audit(archive.read(matches[0]))
    result['named_shop_member_verified']=False
    result['purchase_protocol_verified']=False
    return result


if __name__=='__main__':
    p=argparse.ArgumentParser(description=__doc__)
    p.add_argument('--zip',type=Path);p.add_argument('--ini',type=Path)
    p.add_argument('--lua',type=Path);p.add_argument('--self-test',action='store_true')
    a=p.parse_args()
    if a.self_test:self_test()
    else:
        if not all((a.zip,a.ini,a.lua)):
            p.error('--zip --ini --lua required unless --self-test')
        print(json.dumps(audit(a.zip,a.ini,a.lua),indent=2,ensure_ascii=False))
