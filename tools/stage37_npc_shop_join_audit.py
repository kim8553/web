#!/usr/bin/env python3
"""Read-only offline join: NPC table's authored ShopID -> shop.ini.

Models Stage23 openShopLocked price modes 0/1/2, NOT actual client display.
Never accesses DB, network, player data, or user machine.
"""
import argparse
import collections
import hashlib
import json
import re
from pathlib import Path

_INT32 = re.compile(rb'[+-]?[0-9]+\Z')
_SUPPORTED = frozenset((0, 1, 2))


def int32(raw):
    raw = raw.strip()
    if not _INT32.fullmatch(raw):
        raise ValueError('not a decimal integer')
    n = int(raw)
    if not -(1 << 31) <= n < 1 << 31:
        raise ValueError('outside int32')
    return n


def ascii_id(raw, label):
    if not raw or len(raw) > 128 or not re.fullmatch(rb'[A-Za-z0-9_:\-.]+', raw):
        raise ValueError(f'invalid {label}: must be 1..128 ASCII ID characters')
    return raw.decode('ascii')


def shop_sections(raw):
    """Extract exact section names and numeric-key row modes; expose errors."""
    result = {}
    current = None
    for lineno, raw_line in enumerate(raw.splitlines(), 1):
        line = raw_line.strip()
        if not line or line.startswith((b'//', b';', b'#')):
            continue
        if line.startswith(b'[') and line.endswith(b']'):
            # Keep the malformed-but-observed leading '[' in one backup header:
            # Stage23 matches the entire '[ShopID]' string exactly.
            section_id = line[1:-1]
            if not section_id or len(section_id) > 128 or any(b < 32 or b > 126 for b in section_id):
                raise ValueError(f'invalid shop section header on line {lineno}')
            current = section_id.decode('ascii')
            if current in result:
                raise ValueError(f'duplicate shop section at line {lineno}: {current}')
            result[current] = {'modes': collections.Counter(), 'errors': []}
            continue
        if current is None or b'=' not in line:
            continue
        key, value = (part.strip() for part in line.split(b'=', 1))
        if not _INT32.fullmatch(key):
            continue
        try:
            fields = value.split(b',')
            if len(fields) < 7:
                raise ValueError('fewer than seven columns')
            page = int32(key)
            int32(fields[1])
            mode = int32(fields[2])
            int32(fields[3])
            position = int32(fields[6])
            if len(fields) > 7 and fields[7].strip():
                int32(fields[7])
            if page < 0 or position < 0:
                raise ValueError('negative page or position')
        except ValueError as exc:
            result[current]['errors'].append(f'line {lineno}: {exc}')
        else:
            result[current]['modes'][mode] += 1
    return result


def npc_rows(raw):
    lines = raw.splitlines()
    if len(lines) < 5:
        raise ValueError('NPC table has fewer than five metadata lines')
    columns = lines[2].split(b'\t')
    if columns.count(b'ID') != 1 or columns.count(b'ShopID') != 1:
        raise ValueError('NPC schema must have unique ID and ShopID fields')
    width, id_col, shop_col = len(columns), columns.index(b'ID'), columns.index(b'ShopID')
    for idx, metadata in enumerate(lines[:5], 1):
        if len(metadata.split(b'\t')) != width:
            raise ValueError(f'NPC metadata line {idx}: mismatched column count')
    rows = {}
    empty_id_rows = 0
    duplicate_identical_rows = 0
    for lineno, raw_line in enumerate(lines[5:], 6):
        if not raw_line.strip():
            continue
        cells = raw_line.split(b'\t')
        if len(cells) != width:
            raise ValueError(f'NPC line {lineno}: expected {width} tab-separated columns, got {len(cells)}')
        npc_raw = cells[id_col].strip()
        shop_raw = cells[shop_col].strip()
        if not npc_raw:
            if shop_raw:
                raise ValueError(f'NPC line {lineno}: ShopID without NPC ID')
            empty_id_rows += 1
            continue
        npc = ascii_id(npc_raw, 'NPC ID')
        shop = ascii_id(shop_raw, 'ShopID') if shop_raw else None
        if npc in rows:
            if rows[npc] != shop:
                raise ValueError(f'NPC line {lineno}: conflicting duplicate NPC ID: {npc}')
            duplicate_identical_rows += 1
            continue
        rows[npc] = shop
    return rows, width, empty_id_rows, duplicate_identical_rows, len(lines) - 5


def classify(shop, sections):
    if shop not in sections:
        return 'SECTION_NOT_FOUND'
    info = sections[shop]
    if info['errors']:
        return 'SHOP_ROW_PARSE_ERROR'
    if not info['modes']:
        return 'NO_ITEM_ROWS'
    if set(info['modes']).intersection(_SUPPORTED):
        return 'NORMAL_MODE_ROWS_PRESENT_NOT_LIVE_VERIFIED'
    return 'NO_NORMAL_MODE_ITEM_ADD_FRAMES'


def audit(npc_raw, shop_raw, npc_id=None):
    rows, width, empty_ids, duplicate_ids, physical_rows = npc_rows(npc_raw)
    sections = shop_sections(shop_raw)
    stats = collections.Counter()
    references = collections.defaultdict(list)
    for npc, shop in rows.items():
        if not shop:
            continue
        references[shop].append(npc)
        stats[classify(shop, sections)] += 1
    output = {
        'scope': 'READ_ONLY_BACKUP_SNAPSHOT_SOURCE_MODEL_NOT_LIVE',
        'npc_file_sha256': hashlib.sha256(npc_raw).hexdigest(),
        'shop_file_sha256': hashlib.sha256(shop_raw).hexdigest(),
        'npc_schema_columns': width,
        'npc_data_lines': physical_rows,
        'npc_blank_id_lines_excluded': empty_ids,
        'npc_identical_duplicate_id_lines_excluded': duplicate_ids,
        'npc_unique_nonblank_ids': len(rows),
        'npc_rows_with_shop_id': sum(map(len, references.values())),
        'distinct_shop_ids_referenced': len(references),
        'npc_shop_classification': dict(sorted(stats.items())),
        'shop_sections': len(sections),
        'catalog_sections_not_referenced_by_this_npc_table': len(sections.keys() - references.keys()),
        'legacy_0x46_selector_verified': False,
        'production_handler_enabled': False,
        'live_e2e_verified': False,
    }
    if npc_id is not None:
        if npc_id not in rows:
            output['requested_npc'] = {'npc_id': npc_id, 'status': 'NPC_ID_NOT_IN_THIS_TABLE'}
        else:
            shop = rows[npc_id]
            info = sections.get(shop) if shop else None
            output['requested_npc'] = {
                'npc_id': npc_id,
                'shop_id': shop,
                'status': classify(shop, sections) if shop else 'NPC_WITHOUT_SHOP_ID',
                'price_mode_counts': dict(sorted(info['modes'].items())) if info and not info['errors'] else None,
                'shop_row_parse_errors': info['errors'][:3] if info else [],
            }
    return output


def main():
    ap = argparse.ArgumentParser(description=__doc__)
    ap.add_argument('--npc-table', required=True, type=Path)
    ap.add_argument('--shop-ini', required=True, type=Path)
    ap.add_argument('--npc-id', help='exact ASCII NPC ID from inspected table')
    args = ap.parse_args()
    if args.npc_id is not None:
        ascii_id(args.npc_id.encode('utf-8'), 'NPC ID')
    print(json.dumps(audit(args.npc_table.read_bytes(), args.shop_ini.read_bytes(), args.npc_id),
                     sort_keys=True, ensure_ascii=True, indent=2))


if __name__ == '__main__':
    main()
