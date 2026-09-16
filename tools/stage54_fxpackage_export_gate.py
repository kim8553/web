#!/usr/bin/env python3
"""Read-only SHA-pinned PE export/visible-callsite evidence gate (NOT a PCK0 decoder).

The module-name string, PE export names and clear .text callsite do not prove
runtime DLL loading, PCK0 index decryption, or successful game login.
User-provided PE files are NEVER executed, modified or transmitted.
"""
import argparse
import hashlib
import json
import math
import struct
from collections import Counter
from pathlib import Path

HASHES = {
    'fxgame.exe': 'c7f0366026452d525dc0c74b8499aac30fdb69189591420978e12c7094b5fde3',
    'fxpackage.dll': 'ac63e01378e5f46cd11c0f1f840f86bc32594545d7f13c807e81aadace5f1b47',
}
EXPECTED_EXPORTS = {
    'FxModule_GetEntCreator': 0x1c8a,
    'FxModule_GetFuncCreator': 0x15d7,
    'FxModule_GetIntCreator': 0x1aeb,
    'FxModule_GetLogicCreator': 0x139d,
    'FxModule_GetSpace': 0x1582,
    'FxModule_GetType': 0x128a,
    'FxModule_GetVersion': 0x17d5,
    'FxModule_Init': 0x1389,
}
# Original on-disk FxGame .text, not dynamically unpacked game code.
# 0x14001d807: lea rdx, [rip+...] -> literal "PackFileSys".
# 0x14001d816: call 0x14005cc30; the latter compares ASCII case-insensitively.
CALLSITE = {
    0x14001d807: bytes.fromhex('488d1552870400'),
    0x14001d80e: bytes.fromhex('488b8c24a8070000'),
    0x14001d816: bytes.fromhex('e815f40300'),
    0x14001d81b: bytes.fromhex('85c0'),
    0x14005cc80: bytes.fromhex('410fb609'),
    0x14005cc84: bytes.fromhex('4983c101'),
    0x14005cc88: bytes.fromhex('8d41bf'),
    0x14005cc8b: bytes.fromhex('83f819'),
    0x14005cc90: bytes.fromhex('83c120'),
    0x14005cc93: bytes.fromhex('410fb612'),
    0x14005cca6: bytes.fromhex('85c9'),
    0x14005ccaa: bytes.fromhex('3bca'),
    0x14005ccae: bytes.fromhex('2bca'),
}


def check_identity(data, name):
    if hashlib.sha256(data).hexdigest() != HASHES[name]:
        raise ValueError(f'{name}: original SHA-256 mismatch')


def pe(data):
    if len(data) < 512 or data[:2] != b'MZ':
        raise ValueError('not MZ')
    nt = struct.unpack_from('<I', data, 0x3c)[0]
    if nt > len(data)-264 or data[nt:nt+4] != b'PE\0\0':
        raise ValueError('bad NT header')
    machine, count = struct.unpack_from('<HH', data, nt+4)
    opt_size = struct.unpack_from('<H', data, nt+20)[0]
    op = nt+24
    if machine != 0x8664 or not 1 <= count <= 96 or opt_size < 128 or op+opt_size+count*40 > len(data):
        raise ValueError('bad machine/sections/optional header')
    if struct.unpack_from('<H', data, op)[0] != 0x20b:
        raise ValueError('not PE32+')
    base = struct.unpack_from('<Q', data, op+24)[0]
    ep = struct.unpack_from('<I', data, op+16)[0]
    edir, esize = struct.unpack_from('<II', data, op+112)
    sections = []
    for i in range(count):
        q = op+opt_size+i*40
        name = data[q:q+8].split(b'\0', 1)[0].decode('ascii', errors='replace').strip()
        vsize, rva, raw_size, raw_offset = struct.unpack_from('<IIII', data, q+8)
        if raw_offset > len(data) or raw_size > len(data)-raw_offset:
            raise ValueError('PE section outside file')
        sections.append({'name': name, 'rva': rva, 'vsize': vsize,
                         'raw_offset': raw_offset, 'raw_size': raw_size})
    def locate(rva, size=1):
        matches = [s for s in sections if s['rva'] <= rva and rva-s['rva']+size <= s['raw_size']]
        if len(matches) != 1:
            raise ValueError(f'unmapped/ambiguous RVA 0x{rva:x}')
        s = matches[0]
        return s, s['raw_offset']+rva-s['rva']
    return {'base': base, 'entrypoint': ep, 'export_dir': edir, 'export_size': esize,
            'sections': sections, 'locate': locate}


def read_cstr(data, offset, max_len=128):
    end = data.find(b'\0', offset, min(len(data), offset+max_len+1))
    if end < 0:
        raise ValueError('missing NUL or overlong PE export string')
    return data[offset:end].decode('ascii')


def export_table(data, info):
    locate = info['locate']
    _, off = locate(info['export_dir'], 40)
    row = struct.unpack_from('<IIHHIIIIIII', data, off)
    _, _, _, _, name_rva, base, nf, nn, functions, names, ordinals = row
    if not (1 <= nf <= 256 and 1 <= nn <= nf):
        raise ValueError('export count out of bounds')
    _, foff = locate(functions, nf*4)
    _, noff = locate(names, nn*4)
    _, ooff = locate(ordinals, nn*2)
    dllname = read_cstr(data, locate(name_rva)[1])
    out = {}
    for i in range(nn):
        n_rva = struct.unpack_from('<I', data, noff+4*i)[0]
        name = read_cstr(data, locate(n_rva)[1])
        ordinal = struct.unpack_from('<H', data, ooff+2*i)[0]
        if ordinal >= nf or name in out:
            raise ValueError('invalid/duplicate export ordinal/name')
        rva = struct.unpack_from('<I', data, foff+4*ordinal)[0]
        section, fileoff = locate(rva, 24)
        out[name] = {'ordinal': base+ordinal, 'rva': rva,
                     'section_name': section['name'] or '(unnamed)',
                     'raw_offset': fileoff}
    return dllname, out


def entropy(data):
    counts = Counter(data)
    return round(-sum((c/len(data))*math.log2(c/len(data)) for c in counts.values()), 4)


def audit(fxgame, fxpackage):
    check_identity(fxgame, 'fxgame.exe')
    check_identity(fxpackage, 'fxpackage.dll')
    game = pe(fxgame)
    pkg = pe(fxpackage)
    if game['base'] != 0x140000000 or pkg['base'] != 0x180000000:
        raise ValueError('unexpected image base')
    dll_name, exports = export_table(fxpackage, pkg)
    if dll_name != 'FxPackage.dll' or set(exports) != set(EXPECTED_EXPORTS):
        raise ValueError('unexpected DLL/export set')
    for name, rva in EXPECTED_EXPORTS.items():
        if exports[name]['rva'] != rva or exports[name]['section_name'] != '(unnamed)':
            raise ValueError(f'unexpected DLL export RVA/section: {name}')
    entry_section, entry_offset = pkg['locate'](pkg['entrypoint'], 16)
    if entry_section['name'] != '.boot' or pkg['entrypoint'] != 0x6bd058:
        raise ValueError('unexpected module entrypoint')
    for va, expected in CALLSITE.items():
        sec, p = game['locate'](va-game['base'], len(expected))
        if sec['name'] != '.text' or fxgame[p:p+len(expected)] != expected:
            raise ValueError(f'FxGame .text instruction mismatch at 0x{va:x}')
    s, lit = game['locate'](0x65f60, len(b'PackFileSys\0'))
    if s['name'] != '.rdata' or fxgame[lit:lit+12] != b'PackFileSys\0':
        raise ValueError('PackFileSys literal mismatch')
    module_list = b'FxTool.dll;FxNet2.dll;FxGui.dll;FxRender.dll;FxTerrain.dll;FxSpecial.dll;FxWorld.dll;FxSound.dll;FxGameLogic.dll;FxModel.dll;FxModelAdv.dll;FxPackage.dll;'
    if fxgame.count(module_list) != 1:
        raise ValueError('FxGame module string missing/duplicated')
    first = next(s for s in pkg['sections'] if s['rva'] == 0x1000)
    raw = fxpackage[first['raw_offset']:first['raw_offset']+min(131072, first['raw_size'])]
    return {
        'input_sha256_verified': True,
        'fxgame_module_list_contains_fxpackage': True,
        'fxgame_clear_text_packfilesys_case_insensitive_comparison_instructions_verified': len(CALLSITE),
        'fxpackage_export_name_and_rva_pairs_verified': len(exports),
        'fxpackage_exports': exports,
        'fxpackage_entrypoint': {'rva': pkg['entrypoint'], 'section': entry_section['name'], 'raw_offset': entry_offset},
        'fxpackage_first_code_section_entropy_first_131072_bytes': entropy(raw),
        'actual_module_loaded_at_runtime': None,
        'decoded_index_entries': 0,
        'actual_index_algorithm_known': False,
        'complete_unpack': False,
    }


def self_test():
    for sample in (b'', b'MZ', b'MZ'+b'\0'*510):
        try: pe(sample)
        except ValueError: pass
        else: raise AssertionError('malformed synthetic PE accepted')
    try: check_identity(b'wrong', 'fxpackage.dll')
    except ValueError: pass
    else: raise AssertionError('wrong SHA accepted')
    print('STAGE54_SELF_TEST_PASS: 3 malformed PE and one wrong SHA rejected')


def main():
    parser=argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--fxgame', type=Path)
    parser.add_argument('--fxpackage', type=Path)
    parser.add_argument('--self-test', action='store_true')
    a=parser.parse_args()
    if a.self_test:
        self_test(); return
    if not (a.fxgame and a.fxpackage):
        parser.error('both --fxgame and --fxpackage are required')
    print(json.dumps(audit(a.fxgame.read_bytes(),a.fxpackage.read_bytes()),indent=2,ensure_ascii=False))

if __name__ == '__main__':
    main()
