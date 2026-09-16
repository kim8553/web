#!/usr/bin/env python3
"""Stage51: recover *embedded chunk-source-labelled* Lua 5.1 BYTECODE, not PCK0 index filenames.

Never execute decoded Lua or game EXEs. Private outputs and original binary inputs
must not be uploaded to a public repository or CI. Source labels are compiler
chunknames, not independently validated PCK0 member paths.
"""
import argparse
import hashlib
import json
from pathlib import Path
import zipfile

EXE_SHA = 'c7f0366026452d525dc0c74b8499aac30fdb69189591420978e12c7094b5fde3'
ARCHIVE_SHA = '42665ddc3b74b92a4be1b06fa07aeb316fcdcec5f07a0ec1b46da12ee54f581d'
SIGNATURE = b'abcd464fghfgfjhf'
SOURCE_PREFIX = b'@G:\\Version\\01_Client\\lua\\'
MAX_RAW = 4 * 1024 * 1024
MAX_PROTOS = 30000

class Rejected(ValueError):
    pass


def digest(data):
    return hashlib.sha256(data).hexdigest()


def key_from_pinned_exe(path):
    exe = path.read_bytes()
    if digest(exe) != EXE_SHA or exe.count(SIGNATURE) != 1:
        raise Rejected('wrong client EXE identity or nonunique key signature')
    start = exe.index(SIGNATURE)
    end = exe.find(b'\x00', start)
    if end < 0 or end - start != 523:
        raise Rejected('verified signature has unexpected key length')
    return exe[start:end]


class Chunk:
    def __init__(self, data, key):
        if (len(data) < 13 or len(data) > MAX_RAW or
                data[:7] != b'\x1bLua\x51\x00\x01' or data[7] != 4 or
                data[8] not in (4, 8) or data[9:12] != b'\x04\x08\x00'):
            raise Rejected('not supported Lua 5.1 bytecode header')
        if not key:
            raise Rejected('empty key')
        self.out = bytearray(data)
        self.key = key
        self.pos = 12
        self.size_t = data[8]
        self.protos = 0
        self.root_source = None
        self.code_count = 0

    def field(self, length):
        if length < 0 or length > len(self.out) - self.pos:
            raise Rejected('field exceeds chunk boundary')
        start = self.pos
        self.pos += length
        k = self.key
        for i in range(length):
            self.out[start+i] ^= k[i % len(k)]
        return self.out[start:self.pos]

    def uint(self, size):
        return int.from_bytes(self.field(size), 'little')

    def string(self):
        count = self.uint(self.size_t)
        if count == 0:
            return None
        if count > len(self.out) - self.pos or count > MAX_RAW:
            raise Rejected('invalid Lua string size')
        payload = bytes(self.field(count))
        if payload[-1] != 0:
            raise Rejected('Lua string lacks trailing zero')
        return payload[:-1]

    def proto(self, depth=0):
        if depth > 80 or self.protos >= MAX_PROTOS:
            raise Rejected('excessive nested prototypes')
        self.protos += 1
        source = self.string()
        if depth == 0:
            self.root_source = source
        self.field(4)     # first line
        self.field(4)     # last line
        self.field(4)     # four 1-byte function metadata fields
        code_len = self.uint(4)
        if code_len > (len(self.out) - self.pos) // 4:
            raise Rejected('invalid instruction count')
        code = self.field(code_len * 4)
        self.code_count += code_len
        for i in range(code_len):
            if (code[i * 4] & 63) > 37:
                raise Rejected('invalid Lua 5.1 opcode')
        constant_count = self.uint(4)
        if constant_count > len(self.out) - self.pos:
            raise Rejected('invalid constant count')
        for _ in range(constant_count):
            tag = self.uint(1)
            if tag == 0:
                continue
            if tag == 1:
                self.field(1)
            elif tag == 3:
                self.field(8)
            elif tag == 4:
                self.string()
            else:
                raise Rejected('invalid constant tag')
        child_count = self.uint(4)
        if child_count > (len(self.out) - self.pos) // 24:
            raise Rejected('invalid nested function count')
        for _ in range(child_count):
            self.proto(depth + 1)
        lines = self.uint(4)
        if lines > (len(self.out) - self.pos) // 4:
            raise Rejected('invalid line info count')
        self.field(lines * 4)
        vars_count = self.uint(4)
        if vars_count > (len(self.out) - self.pos) // 10:
            raise Rejected('invalid local variable count')
        for _ in range(vars_count):
            self.string()
            self.field(8)
        ups = self.uint(4)
        if ups > (len(self.out) - self.pos) // 5:
            raise Rejected('invalid upvalue name count')
        for _ in range(ups):
            self.string()

    def decode(self):
        self.proto()
        if self.pos != len(self.out):
            raise Rejected('bytecode not consumed exactly')
        return bytes(self.out), self.root_source


def safe_embedded_label(source):
    if not isinstance(source, bytes) or not source.startswith(SOURCE_PREFIX):
        raise Rejected('unverified Lua compiler source prefix')
    raw = source[len(SOURCE_PREFIX):]
    try:
        text = raw.decode('utf-8', errors='strict')
    except UnicodeError:
        raise Rejected('Lua chunk source name is not UTF-8')
    sections = text.split('\\')
    if (not sections or any(not s or s in ('.', '..') or '/' in s or ':' in s or '\x00' in s for s in sections)
            or not sections[-1].lower().endswith('.lua') or len(text) > 1024):
        raise Rejected('unsafe or non-Lua compiler chunk source label')
    return '/'.join(sections)


def synthetic_test():
    # Valid standard Lua 5.1 chunk (single RETURN), constructed field by field.
    key = b'abcTESTkey'
    raw = bytearray(b'\x1bLua\x51\x00\x01\x04\x04\x04\x08\x00')
    fields = []
    label = SOURCE_PREFIX + b'form_shop\\test.lua\x00'
    fields.extend([len(label).to_bytes(4, 'little'), label, bytes(4), bytes(4), b'\x00\x00\x00\x02',
                   (1).to_bytes(4,'little'), bytes([30, 0, 0, 1]), bytes(4), bytes(4),
                   bytes(4), bytes(4), bytes(4)])
    for part in fields:
        raw.extend(bytes(b ^ key[i % len(key)] for i, b in enumerate(part)))
    decoded, source = Chunk(bytes(raw), key).decode()
    assert source == label[:-1] and decoded.startswith(b'\x1bLua')
    assert safe_embedded_label(source) == 'form_shop/test.lua'
    for wrong in (raw[:-1], raw[:12] + b'\x00'*len(raw[12:])):
        try:
            Chunk(bytes(wrong), key).decode()
        except Rejected:
            pass
        else:
            raise AssertionError('corrupted synthetic Lua accepted')
    for path in (SOURCE_PREFIX + b'..\\bad.lua', b'@C:\\bad.lua', SOURCE_PREFIX + b'not_lua.txt'):
        try:
            safe_embedded_label(path)
        except Rejected:
            pass
        else:
            raise AssertionError('unsafe synthetic compiler label accepted')
    print('SELF_TEST_PASS: synthetic per-field Lua5.1 decode, truncated/wrong-key and source traversal rejected')


def recover(exe_path, anonymous_zip, output_zip):
    if output_zip.exists() or output_zip.with_name(output_zip.name + '.partial').exists():
        raise FileExistsError('output exists; refusing overwrite')
    key = key_from_pinned_exe(exe_path)
    if digest(anonymous_zip.read_bytes()) != ARCHIVE_SHA:
        raise Rejected('Stage48 archive fingerprint mismatch')
    temp = output_zip.with_name(output_zip.name + '.partial')
    manifest = {'notice': 'Embedded compiler chunk source label, NOT recovered PCK0 index path; bytecode NOT Lua source.',
                'origin': 'SHA-pinned original fxgame.exe and Stage48 PRIVATE anonymous stream ZIP',
                'members': [], 'rejected_non_lua': []}
    used = set()
    duplicates = 0
    try:
        with zipfile.ZipFile(anonymous_zip) as source_zip, temp.open('xb') as handle, zipfile.ZipFile(handle,'w', compression=zipfile.ZIP_DEFLATED, compresslevel=6) as result_zip:
            if source_zip.testzip() is not None:
                raise Rejected('Stage48 ZIP CRC mismatch')
            index = json.loads(source_zip.read('manifest.json'))
            lua = [m for m in index['members'] if m['source']=='lua']
            for ordinal, member in enumerate(lua):
                blob = source_zip.read(member['anonymous_path'])
                if digest(blob) != member['sha256'] or len(blob) != member['raw_size']:
                    raise Rejected('Stage48 anonymous member changed')
                try:
                    decoded, chunkname = Chunk(blob, key).decode()
                    relative = safe_embedded_label(chunkname)
                except Rejected as exc:
                    manifest['rejected_non_lua'].append({'source_start': member['start'], 'reason': str(exc)})
                    continue
                output_name = 'compiler_chunk_labels/' + relative + '.luac'
                if output_name in used:
                    duplicates += 1
                    output_name = f'duplicate_compiler_chunk_labels/ordinal_{ordinal:05d}/' + relative + '.luac'
                    if output_name in used:
                        raise Rejected('nonunique collision-safe output path')
                used.add(output_name)
                result_zip.writestr(output_name, decoded)
                manifest['members'].append({'bytecode_path': output_name,
                                            'compiler_chunk_label': chunkname.decode('utf-8'),
                                            'source_stream_offset': member['start'],
                                            'source_stream_sha256': member['sha256'],
                                            'decoded_sha256': digest(decoded),
                                            'decoded_bytes': len(decoded)})
            manifest['summary'] = {'lua_chunks_recovered': len(manifest['members']),
                                   'non_lua_or_rejected': len(manifest['rejected_non_lua']),
                                   'duplicate_source_labels_preserved': duplicates,
                                   'package_index_decrypted': False,
                                   'named_pck0_members_verified': 0,
                                   'lua_code_executed': False,
                                   'server_gameplay_modified': False,
                                   'live_e2e': False}
            result_zip.writestr('manifest.json', json.dumps(manifest, ensure_ascii=False, indent=2))
            result_zip.writestr('README_KO.txt', '최신 클라이언트 Lua 5.1 복호화 바이트코드입니다. 경로는 컴파일러에 내장된 chunkname으로 원본 PCK0 인덱스 파일명이 아닙니다. Lua 원본 소스(.lua)로 오해하지 마세요. 클라이언트 설치 폴더에 덮어쓰지 마세요.\n')
        with zipfile.ZipFile(temp) as check:
            if check.testzip() is not None or len(check.namelist()) != len(used) + 2:
                raise Rejected('output ZIP CRC/count failed')
        temp.rename(output_zip)
    except BaseException:
        temp.unlink(missing_ok=True)
        raise
    return {'output_zip': str(output_zip), 'zip_bytes':output_zip.stat().st_size,
            'zip_sha256':digest(output_zip.read_bytes()), 'zip_crc':'PASS',
            **manifest['summary']}


def main():
    p=argparse.ArgumentParser(description=__doc__)
    p.add_argument('--fxgame',type=Path)
    p.add_argument('--stage48-private-zip',type=Path)
    p.add_argument('--output-zip',type=Path)
    p.add_argument('--self-test',action='store_true')
    a=p.parse_args()
    if a.self_test:
        synthetic_test();return
    if not (a.fxgame and a.stage48_private_zip and a.output_zip):
        p.error('private --fxgame, --stage48-private-zip, --output-zip are required')
    print(json.dumps(recover(a.fxgame,a.stage48_private_zip,a.output_zip),ensure_ascii=False,indent=2))

if __name__=='__main__':main()
