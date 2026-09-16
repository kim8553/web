#!/usr/bin/env python3
"""Read-only offline audit of NPC shop.ini against Stage23 openShopLocked.

No server/client execution, packet inference, database access, or data mutation.
This source-model audit cannot establish LIVE item delivery or UI behavior.
"""
import argparse
import collections
import hashlib
import json
import re
from pathlib import Path

VIEW_PAGE_SIZE = 500  # latest_client_shop_view_contract.go, Stage23
VIEW_CAPACITY = 100   # openShopLocked view construction, Stage23
SUPPORTED_MODES = frozenset((0, 1, 2))


def parse_int32(s):
    s = s.strip()
    if not re.fullmatch(r'[+-]?[0-9]+', s):
        raise ValueError('not a decimal integer')
    n = int(s, 10)
    if not -(1 << 31) <= n < (1 << 31):
        raise ValueError('outside int32')
    return n


def scan(raw, shop_id=None):
    sections = collections.OrderedDict()
    current = None
    max_line_bytes = 0
    for lineno, line in enumerate(raw.splitlines(), 1):
        max_line_bytes = max(max_line_bytes, len(line))
        line = line.decode('latin-1').strip()  # ASCII delimiters; no locale guess
        if not line or line.startswith(('//', ';', '#')):
            continue
        if line.startswith('[') and line.endswith(']'):
            current = line[1:-1]
            sections.setdefault(current, [])
            continue
        if current is None or '=' not in line:
            continue
        key, value = (part.strip() for part in line.split('=', 1))
        if not re.fullmatch(r'[+-]?[0-9]+', key):
            continue
        sections[current].append((lineno, key, value))
    categories = collections.Counter()
    modes = collections.Counter()
    out_of_capacity = 0
    invalid_index = 0
    duplicate_index_sections = 0
    requested = None
    for name, rows in sections.items():
        if not rows:
            categories['no_item_rows'] += 1
            record = dict(status='NO_ITEM_ROWS', total_rows=0, eligible_rows=0)
        else:
            indices = []
            local_modes = collections.Counter()
            error = None
            for lineno, page_text, value in rows:
                parts = value.split(',')
                try:
                    if len(parts) < 7:
                        raise ValueError('fewer than 7 columns')
                    page = parse_int32(page_text)
                    parse_int32(parts[1])  # amount
                    mode = parse_int32(parts[2])
                    parse_int32(parts[3])  # price
                    position = parse_int32(parts[6])
                    if len(parts) > 7 and parts[7].strip():
                        parse_int32(parts[7])  # exchange data
                    if page < 0 or position < 0:
                        raise ValueError('negative page or position')
                except ValueError as exc:
                    error = f'line {lineno}: {exc}'
                    break
                local_modes[mode] += 1
                if mode in SUPPORTED_MODES:
                    idx = page * VIEW_PAGE_SIZE + position + 1
                    indices.append(idx)
                    if idx > VIEW_CAPACITY:
                        out_of_capacity += 1
                    if position >= VIEW_PAGE_SIZE or not 1 <= idx <= 65535:
                        invalid_index += 1
            if error:
                categories['parser_error'] += 1
                record = dict(status='PARSER_ERROR', total_rows=len(rows), error=error)
            else:
                modes.update(local_modes)
                if len(set(indices)) != len(indices):
                    duplicate_index_sections += 1
                if not indices:
                    categories['no_supported_price_mode'] += 1
                    status = 'NO_ITEM_ADD_FRAMES_FROM_MODE_FILTER'
                elif any(i < 1 or i > 65535 for i in indices):
                    categories['invalid_view_index_candidate'] += 1
                    status = 'INVALID_VIEW_INDEX_CANDIDATE'
                else:
                    categories['eligible_rows_present'] += 1
                    status = 'ITEM_ADD_FRAMES_POSSIBLE_NOT_LIVE_VERIFIED'
                record = dict(status=status, total_rows=len(rows),
                              eligible_rows=len(indices), modes=dict(sorted(local_modes.items())),
                              indexes_above_declared_capacity=sum(i > VIEW_CAPACITY for i in indices))
        if name == shop_id:
            requested = record | {'shop_id': name}
    result = {
        'scope': 'OFFLINE_READ_ONLY_STAGE23_SOURCE_MODEL_NOT_LIVE',
        'shop_ini_sha256': hashlib.sha256(raw).hexdigest(),
        'shop_ini_size_bytes': len(raw),
        'section_count': len(sections),
        'max_line_bytes': max_line_bytes,
        'section_categories': dict(sorted(categories.items())),
        'parsed_price_mode_rows': dict(sorted(modes.items())),
        'eligible_rows_outside_declared_view_capacity_100': out_of_capacity,
        'eligible_rows_with_invalid_view_index': invalid_index,
        'sections_with_duplicate_eligible_view_index': duplicate_index_sections,
        'view_capacity_mismatch_is_client_effect_unverified': True,
        'selector_verified': False,
        'live_e2e_verified': False,
    }
    if shop_id is not None:
        result['requested_shop'] = requested or {'shop_id': shop_id, 'status': 'SECTION_NOT_FOUND'}
    return result


def main():
    ap = argparse.ArgumentParser(description=__doc__)
    ap.add_argument('--shop-ini', type=Path, required=True)
    ap.add_argument('--shop-id', help='optional exact NPC shop ID (not an NPC name)')
    args = ap.parse_args()
    if args.shop_id is not None and (len(args.shop_id) > 128 or any(ord(c) < 32 for c in args.shop_id)):
        ap.error('--shop-id must be <= 128 characters without controls')
    print(json.dumps(scan(args.shop_ini.read_bytes(), args.shop_id), ensure_ascii=True, indent=2, sort_keys=True))


if __name__ == '__main__':
    main()
