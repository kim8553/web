#!/usr/bin/env python3
"""Read-only Stage27-34 NPC shop log audit. Never infers client rendering or purchase success.

Input logs stay local; JSON output contains aggregate shop IDs and counts, no raw lines,
remote addresses, account identifiers, NPC object IDs, or player data.
"""
import argparse
import json
import re
import sys
from collections import defaultdict
from pathlib import Path

EVENTS = {
    'menu': 'NPC shop menu diagnostic ',
    'selected': 'shop service selected ',
    'catalog': 'shop display catalog ',
    'preflight_rejected': 'shop display preflight rejected ',
    'frames': 'shop display frames ',
    'capacity': 'shop display capacity diagnostic ',
    'exchange_ab': 'shop exchange view AB ',
}
PAIR = re.compile(r'(?<!\S)([A-Za-z_][A-Za-z_0-9]*)=(?:"((?:\\.|[^"\\])*)"|([^\s]+))')

def fields(line):
    # Go %q uses Go escapes; retain literal content without evaluating arbitrary escapes.
    return {m.group(1): (m.group(2) if m.group(2) is not None else m.group(3))
            for m in PAIR.finditer(line)}

def read_logs(paths):
    shops = defaultdict(lambda: {'events': defaultdict(int), 'npc_configs': set(),
                                 'catalog_errors': set(), 'ordinary_counts': set(),
                                 'exchange_counts': set(), 'frame_add_counts': set(),
                                 'preflight_reasons': set()})
    counts = defaultdict(int)
    for path in paths:
        with path.open('r', encoding='utf-8', errors='replace') as stream:
            for line in stream:
                kind = next((kind for kind, prefix in EVENTS.items() if prefix in line), None)
                if kind is None:
                    continue
                attrs = fields(line.split(EVENTS[kind], 1)[1])
                shop = attrs.get('shop', '')
                if not shop:
                    counts['unattributed_event_lines'] += 1
                    continue
                row = shops[shop]
                row['events'][kind] += 1
                counts['recognized_events'] += 1
                if kind in ('menu', 'selected') and attrs.get('npc_config'):
                    row['npc_configs'].add(attrs['npc_config'])
                if kind == 'menu':
                    if attrs.get('catalog_error'):
                        row['catalog_errors'].add(attrs['catalog_error'])
                    for key, dest in (('ordinary', 'ordinary_counts'), ('exchange', 'exchange_counts')):
                        try:
                            row[dest].add(int(attrs[key]))
                        except (ValueError, KeyError):
                            pass
                if kind == 'frames':
                    try:
                        row['frame_add_counts'].add(int(attrs['item_add_sent']))
                    except (ValueError, KeyError):
                        pass
                if kind == 'preflight_rejected' and attrs.get('reason'):
                    row['preflight_reasons'].add(attrs['reason'])
    result = []
    for shop, row in sorted(shops.items()):
        ev = dict(sorted(row['events'].items()))
        if row['catalog_errors']:
            category = 'exact_catalog_error_observed'
        elif ev.get('preflight_rejected'):
            category = 'preflight_rejection_observed'
        elif ev.get('frames'):
            category = 'server_frame_write_completion_logged_client_display_unverified'
        elif ev.get('catalog'):
            category = 'catalog_logged_no_frame_completion_evidence'
        elif ev.get('selected'):
            category = 'shop_selected_no_catalog_evidence'
        elif ev.get('menu') and row['ordinary_counts'] == {0}:
            category = 'menu_catalog_has_zero_ordinary_rows_display_unverified'
        else:
            category = 'insufficient_trace'
        result.append({'shop_id': shop, 'classification': category, 'event_counts': ev,
                       'npc_config_ids': sorted(row['npc_configs']),
                       'catalog_errors': sorted(row['catalog_errors']),
                       'ordinary_counts': sorted(row['ordinary_counts']),
                       'exchange_counts': sorted(row['exchange_counts']),
                       'frame_item_add_counts': sorted(row['frame_add_counts']),
                       'preflight_reasons': sorted(row['preflight_reasons'])})
    return {'source_files': len(paths), 'recognized_events': counts['recognized_events'],
            'unattributed_event_lines': counts['unattributed_event_lines'],
            'shops': result,
            'limits': ['Events are aggregated by exact ShopID, not attributed to an individual concurrent session.',
                       'Absent lines do not prove an absent NPC/shop service or a network failure.',
                       'A completed server write does not prove client receipt or rendering.',
                       'No purchase, currency, inventory, database or reconnect success is inferred.']}

def main(argv=None):
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('logs', nargs='+', type=Path, help='Existing local text logs; never uploaded')
    parser.add_argument('--output', type=Path, help='Optional local JSON output file (default stdout)')
    args = parser.parse_args(argv)
    for file in args.logs:
        if not file.is_file():
            parser.error('input is not a readable file: ' + str(file))
    data = json.dumps(read_logs(args.logs), ensure_ascii=False, indent=2) + '\n'
    if args.output:
        args.output.write_text(data, encoding='utf-8')
    else:
        sys.stdout.write(data)
    return 0

if __name__ == '__main__':
    raise SystemExit(main())
