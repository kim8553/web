#!/usr/bin/env python3
"""Read-only exact-current exchange special-field topology audit.

Counts authored settings and referenced mode-3 shop rows; does NOT infer costs,
item rewards, binding or server transaction behavior.
"""
from __future__ import annotations

import argparse
from collections import Counter, defaultdict
import hashlib
import json
from pathlib import Path

SHOP_SHA256 = 'f42fce7631f7fe302d3fdedad60b12eaa20da12396de412ff0a84edcddc1a6d9'
EXCHANGE_SHA256 = 'ed931884d8a8bb19bad512b5eb571fe8ac97b783449144d4a4834eb050f0dee6'


def parse_ini(text: str) -> dict[str, dict[str, str]]:
    sections: dict[str, dict[str, str]] = {}
    current: str | None = None
    for lineno, original in enumerate(text.splitlines(), 1):
        line = original.strip()
        if not line or line.startswith((';', '#', '//')):
            continue
        if line.startswith('[') and line.endswith(']'):
            current = line[1:-1].strip()
            if not current or current in sections:
                raise ValueError(f'duplicate/empty section at line {lineno}: {current!r}')
            sections[current] = {}
            continue
        if '=' in line and current is not None:
            key, value = line.split('=', 1)
            key = key.strip()
            if not key or key in sections[current]:
                raise ValueError(f'duplicate/empty key in {current} at line {lineno}: {key!r}')
            sections[current][key] = value.strip()
    return sections


def mode3_refs(shop_text: str) -> tuple[Counter[str], int, int]:
    refs: Counter[str] = Counter()
    section = None
    mode3 = 0
    zero_exchange = 0
    for lineno, original in enumerate(shop_text.splitlines(), 1):
        line = original.strip()
        if not line or line.startswith((';', '#', '//')):
            continue
        if line.startswith('[') and line.endswith(']'):
            section = line[1:-1].strip()
            continue
        if '=' not in line or section is None:
            continue
        key, value = line.split('=', 1)
        if not key.strip().isdigit():
            continue
        columns = [v.strip() for v in value.split(',')]
        if len(columns) < 8 or columns[2] != '3':
            continue
        mode3 += 1
        exchange_id = columns[7]
        if not exchange_id.isdecimal():
            raise ValueError(f'mode3 ExchangeData not numeric at {section}:{lineno}: {exchange_id!r}')
        if int(exchange_id) == 0:
            zero_exchange += 1
            continue
        refs[str(int(exchange_id))] += 1
    return refs, mode3, zero_exchange


def analyze(shop_text: str, exchange_text: str) -> dict:
    refs, mode3, zero_exchange = mode3_refs(shop_text)
    sections = parse_ini(exchange_text)
    missing = sorted(set(refs) - set(sections), key=int)
    if missing:
        raise ValueError(f'mode3 references missing ExchangeItem sections: {missing[:12]}')
    field_by_type: dict[str, Counter[str]] = defaultdict(Counter)
    row_by_type: dict[str, Counter[str]] = defaultdict(Counter)
    add_value_pairs: Counter[tuple[str, str]] = Counter()
    type3_with_item = []
    type3_with_addvalue = []
    neither = []
    unknown_type = []
    bound = Counter()
    prop_tokens: Counter[str] = Counter()
    for exchange_id, row_count in refs.items():
        d = sections[exchange_id]
        typ = d.get('Type', '<absent>') or '<empty>'
        if typ not in ('<absent>', '1', '2', '3'):
            unknown_type.append(exchange_id)
        has_item, has_prop, has_add = (bool(d.get(k, '')) for k in ('Item', 'Prop', 'AddValue'))
        shape = ('Item+' if has_item and has_prop else 'Item' if has_item else '') + ('Prop' if has_prop else '') or 'neither'
        field_by_type[typ][shape] += 1
        row_by_type[typ][shape] += row_count
        if has_add:
            add_value_pairs[(typ, d['AddValue'])] += 1
        if typ == '3' and has_item:
            type3_with_item.append(exchange_id)
        if typ == '3' and has_add:
            type3_with_addvalue.append(exchange_id)
        if shape == 'neither':
            neither.append({'id': exchange_id, 'rows': row_count, 'has_addvalue': has_add, 'type': typ})
        if d.get('BindStatus', '') not in ('', '0'):
            bound['definitions'] += 1
            bound['shop_rows'] += row_count
        if has_prop:
            for token in d['Prop'].split(';'):
                if token:
                    prop_tokens[token.split(',', 1)[0]] += 1
    return {
        'scope': 'configuration topology ONLY; client display parsing and server settlement are separate',
        'mode3_rows': mode3,
        'mode3_zero_exchange': zero_exchange,
        'mode3_referenced_rows': sum(refs.values()),
        'referenced_exchange_definitions': len(refs),
        'by_type_definitions': {k: dict(sorted(v.items())) for k, v in sorted(field_by_type.items())},
        'by_type_shop_rows': {k: dict(sorted(v.items())) for k, v in sorted(row_by_type.items())},
        'type3_item_count': len(type3_with_item),
        'type3_addvalue_count': len(type3_with_addvalue),
        'addvalue_by_type_and_authored_text': [
            {'type': typ, 'value': value, 'definitions': count}
            for (typ, value), count in sorted(add_value_pairs.items())
        ],
        'neither_item_nor_prop': sorted(neither, key=lambda v: int(v['id'])),
        'nonzero_bindstatus': dict(bound),
        'prop_names_by_definition_count': dict(sorted(prop_tokens.items())),
        'unknown_type_ids': sorted(unknown_type, key=int),
    }


def read_exact(path: Path, digest: str) -> str:
    data = path.read_bytes()
    actual = hashlib.sha256(data).hexdigest()
    if actual != digest:
        raise ValueError(f'{path.name}: SHA256 mismatch: got {actual}, expected {digest}')
    # Preserve bytes one-to-one; do not replace undecodable GBK values.
    return data.decode('latin-1')


def main() -> None:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--shop', type=Path, required=True)
    parser.add_argument('--exchange', type=Path, required=True)
    args = parser.parse_args()
    print(json.dumps(analyze(read_exact(args.shop, SHOP_SHA256), read_exact(args.exchange, EXCHANGE_SHA256)),
                     ensure_ascii=False, indent=2, sort_keys=True))


if __name__ == '__main__':
    main()
