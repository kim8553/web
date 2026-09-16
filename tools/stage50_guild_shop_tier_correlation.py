#!/usr/bin/env python3
"""Read-only SHA-pinned content correlation, NOT a client NPC shop selector.

Private inputs stay local: original share.package and Stage49 private evidence ZIP.
Outputs metadata only. Never publish game resource bytes or an inferred selector.
"""
import argparse
import hashlib
import json
from pathlib import Path
import re
import zipfile

from stage48_anonymous_zlib_recovery import inflate

PACKAGE_SHA = '200497852ba3a29279e51f01e2913b5f7260480f2a740a32ebd1680b869844b6'
ARCHIVE_SHA = 'dd2fd71461344a06b5610272a9f6ce283b1013cddc4fb9a86f22f82dec2c5a72'
TIER_OFFSET = 20626686
TIER_SHA = '22f04ef1355845f5140d14ce4255683ca7f427e6ac219ad8f617217a32f1e1db'
SHOP_ID = 'Shop_GB_Yishiting'


def sha(data):
    return hashlib.sha256(data).hexdigest()


def parse_sections(data):
    """Return raw section blocks and authored item values; do not infer semantics."""
    pattern = re.compile(rb'(?m)^\[([^\]\r\n]+)\]\r?\n')
    matches = list(pattern.finditer(data))
    result = {}
    for i, match in enumerate(matches):
        name = match.group(1).decode('ascii')
        if name in result:
            raise ValueError('duplicate section')
        end = matches[i + 1].start() if i + 1 < len(matches) else len(data)
        result[name] = data[match.start():end].rstrip(b'\r\n')
    return result


def items(section):
    out = []
    for line in section.splitlines():
        if not line.startswith(b'0='):
            continue
        fields = line[2:].split(b',')
        if len(fields) < 8:
            raise ValueError('unexpected item field count')
        out.append(fields)
    return out


def correlate(tier_data, npc_data, candidate_a, candidate_b):
    tiers = parse_sections(tier_data)
    a, b = parse_sections(candidate_a), parse_sections(candidate_b)
    if set(tiers) != {str(k) for k in range(5)}:
        raise ValueError('five 0-based tier blocks not found')
    lines = npc_data.decode('gb18030', errors='strict').splitlines()
    headings = lines[0].split('\t')
    if len(headings) < 103 or headings[45] != '商店ID' or headings[102] != '所属帮会地块编号':
        raise ValueError('NPC shop and guild plot headers unverified')
    references = []
    for row in lines[1:]:
        fields = row.split('\t')
        if len(fields) > 102 and fields[45] == SHOP_ID:
            references.append((fields[0], fields[102]))
    if len(references) != 35 or len({npc for npc, _ in references}) != 35:
        raise ValueError('expected 35 distinct NPC references not verified')
    evidence = []
    for k in range(5):
        named = f'{SHOP_ID}_{k + 1}'
        if named not in a or named not in b:
            raise ValueError(f'missing catalog section {named}')
        left, right_a, right_b = items(tiers[str(k)]), items(a[named]), items(b[named])
        if not left or len(left) != len(right_a) or len(left) != len(right_b):
            raise ValueError(f'tier {k}: different item counts')
        for ref in [right_a, right_b]:
            if [r[:4] for r in left] != [r[:4] for r in ref]:
                raise ValueError(f'tier {k}: item identity/amount/mode/price mismatch')
            if [r[5] for r in left] != [r[6] for r in ref]:
                raise ValueError(f'tier {k}: item order mismatch')
        if a[named] != b[named]:
            raise ValueError(f'{named}: candidate sections differ in bytes')
        evidence.append({'tier_block': str(k), 'catalog_section': named,
                         'item_count': len(left), 'items_first_four_fields_equal': True,
                         'display_positions_equal': True, 'catalog_A_B_section_bytes_equal': True,
                         'section_sha256': sha(a[named])})
    return {'npc_reference_count': len(references), 'npc_shop_id_column': 45,
            'npc_guild_plot_column': 102,
            'five_content_correlations': evidence,
            'original_index_decrypted': False, 'runtime_tier_selector_verified': False,
            'purchase_ready': False, 'gameplay_modified': False, 'live_e2e_verified': False}


def self_test():
    def fake(items_count, name):
        lines = [f'[{name}]', 'PageInfo=Page1']
        for j in range(items_count):
            lines.append(f'0=ITEM{j},1,1,100,{"1" if name.isdigit() else "0"},{j if name.isdigit() else 1},{0 if name.isdigit() else j},0,0')
        return ('\r\n'.join(lines) + '\r\n').encode()
    tier = b'\r\n'.join(fake(k + 1, str(k)) for k in range(5))
    cat = b'\r\n'.join(fake(k + 1, f'{SHOP_ID}_{k + 1}') for k in range(5))
    header = [''] * 108
    header[45], header[102] = '商店ID', '所属帮会地块编号'
    npc = '\t'.join(header) + '\n' + '\n'.join('\t'.join(['NPC' + str(k)] + [''] * 44 + [SHOP_ID] + [''] * 56 + [str(k + 10)]) for k in range(35))
    npc = npc.encode('gb18030')
    result = correlate(tier, npc, cat, cat)
    assert len(result['five_content_correlations']) == 5
    try:
        correlate(tier, npc, cat.replace(b'ITEM0', b'BAD__'), cat)
    except ValueError:
        pass
    else:
        raise AssertionError('corrupted catalog accepted')
    try:
        correlate(tier, npc.replace(b'\xc9\xcc\xb5\xeaID', b'BAD____'), cat, cat)
    except ValueError:
        pass
    else:
        raise AssertionError('wrong NPC column header accepted')
    print('SELF_TEST_PASS: five content correlations; mutated catalog/header rejected; no selector generated')


def main():
    p = argparse.ArgumentParser(description=__doc__)
    p.add_argument('--share', type=Path)
    p.add_argument('--stage49-private-zip', type=Path)
    p.add_argument('--self-test', action='store_true')
    args = p.parse_args()
    if args.self_test:
        self_test()
        return
    if args.share is None or args.stage49_private_zip is None:
        p.error('--share and --stage49-private-zip are required')
    package = args.share.read_bytes()
    archive_bytes = args.stage49_private_zip.read_bytes()
    if sha(package) != PACKAGE_SHA or sha(archive_bytes) != ARCHIVE_SHA:
        raise ValueError('private input fingerprint mismatch')
    result = inflate(package, TIER_OFFSET)
    if result is None or sha(result[1]) != TIER_SHA:
        raise ValueError('tier stream checksum/fingerprint mismatch')
    with zipfile.ZipFile(args.stage49_private_zip) as f:
        if f.testzip() is not None:
            raise ValueError('private ZIP CRC failure')
        def get(name):
            return f.read('anonymous_candidates/' + name + '.bin')
        summary = correlate(result[1], get('npc_reference_candidate'),
                            get('shop_catalog_candidate_A'), get('shop_catalog_candidate_B'))
    summary['tier_stream_start'] = TIER_OFFSET
    summary['tier_stream_end'] = result[0]
    summary['tier_stream_sha256'] = TIER_SHA
    print(json.dumps(summary, ensure_ascii=False, indent=2))


if __name__ == '__main__':
    main()
