#!/usr/bin/env python3
"""Offline, READ-ONLY inventory of authored mode-3 exchange data.

Does not infer purchase settlement, debit direction, binding, quantity scaling,
item category, price, or transaction safety. Filenames are prior SHA labels.
Input must be the two previously authenticated current share.package payloads.
"""
import argparse
from collections import Counter
import hashlib
import json
from pathlib import Path

HASHES = {
    'shop.ini': 'f42fce7631f7fe302d3fdedad60b12eaa20da12396de412ff0a84edcddc1a6d9',
    'exchangeitem.ini': 'ed931884d8a8bb19bad512b5eb571fe8ac97b783449144d4a4834eb050f0dee6',
}


def sections(data):
    """Parse authored ASCII section identifiers and field keys, not encoded comments."""
    found = {}
    name = None
    for line in data.decode('latin-1').splitlines():
        line = line.strip()
        if line.startswith('[') and line.endswith(']'):
            name = line[1:-1]
            if name in found:
                raise ValueError('duplicate exchange section: ' + name)
            found[name] = {}
        elif name is not None and '=' in line and not line.startswith((';', '#', '//')):
            key, value = line.split('=', 1)
            key = key.strip()
            if key in found[name]:
                raise ValueError(f'duplicate exchange field {name}.{key}')
            found[name][key] = value.strip()
    return found


def references(data):
    """Visit every numeric shop row, preserving repeated row keys and sections."""
    counts = Counter()
    row_count = mode3_count = referenced_count = 0
    section = ''
    first = {}
    for line in data.decode('latin-1').splitlines():
        line = line.strip()
        if line.startswith('[') and line.endswith(']'):
            section = line[1:-1]
            continue
        if '=' not in line or line.startswith((';', '#', '//')):
            continue
        key, value = line.split('=', 1)
        if not key.strip().isdecimal():
            continue
        row_count += 1
        parts = value.split(',')
        if len(parts) < 8 or parts[2].strip() != '3':
            continue
        mode3_count += 1
        ident = parts[7].strip()
        if not ident.isdecimal():
            raise ValueError(f'malformed mode-3 ExchangeData in {section} row {key}')
        if int(ident) <= 0:
            continue
        referenced_count += 1
        ident = str(int(ident))
        counts[ident] += 1
        first.setdefault(ident, {'shop': section, 'row_key': key.strip()})
    return counts, first, (row_count, mode3_count, referenced_count)


def audit(shop, exchange):
    definitions = sections(exchange)
    counts, samples, totals = references(shop)
    missing = sorted(set(counts) - set(definitions))
    if missing:
        raise ValueError('referenced exchange sections missing: ' + ', '.join(missing[:10]))
    categories = Counter()
    category_rows = Counter()
    types = Counter()
    type_rows = Counter()
    nonempty_addvalue = []
    bad_item = []
    bad_prop = []
    special = []
    for ident, n in counts.items():
        d = definitions[ident]
        has_item, has_prop = bool(d.get('Item')), bool(d.get('Prop'))
        category = ('item+prop' if has_item and has_prop else
                    'item-only' if has_item else 'prop-only' if has_prop else 'neither')
        categories[category] += 1
        category_rows[category] += n
        kind = d.get('Type', '<absent>')
        types[kind] += 1
        type_rows[kind] += n
        if d.get('AddValue'):
            nonempty_addvalue.append({'id': ident, 'shop_rows': n, 'type': kind})
        if not has_item and not has_prop:
            special.append({'id': ident, 'shop_rows': n,
                            'has_add_value': bool(d.get('AddValue')),
                            'has_condition': bool(d.get('Condition') or d.get('Condition2')),
                            'empty_section': not bool(d), 'first_reference': samples[ident]})
        for field, issues in [('Item', bad_item), ('Prop', bad_prop)]:
            value = d.get(field, '')
            if not value:
                continue
            malformed = [part for part in value.split(';') if len(part.split(',')) != 2
                         or not part.split(',')[0].strip()
                         or not part.split(',')[-1].strip().isdigit()]
            if malformed:
                issues.append({'id': ident, 'shop_rows': n,
                               'trailing_semicolon': value.endswith(';'),
                               'malformed_token_count': len(malformed),
                               'first_reference': samples[ident]})
    positive = [ident for ident in counts if int(definitions[ident].get('BindStatus', '0') or '0') > 0]
    return {
        'scope': 'authoritative INI syntax only; no exchange mutation or native rule inferred',
        'numeric_listing_rows': totals[0], 'mode3_rows': totals[1],
        'mode3_nonzero_exchange_rows': totals[2],
        'referenced_unique_exchange_data': len(counts),
        'authored_exchange_sections': len(definitions),
        'field_presence_unique': dict(sorted(categories.items())),
        'field_presence_shop_rows': dict(sorted(category_rows.items())),
        'type_unique': dict(sorted(types.items())),
        'type_shop_rows': dict(sorted(type_rows.items())),
        'addvalue_unique': len(nonempty_addvalue),
        'addvalue_shop_rows': sum(x['shop_rows'] for x in nonempty_addvalue),
        'positive_bind_unique': len(positive),
        'positive_bind_shop_rows': sum(counts[x] for x in positive),
        'no_item_no_prop': sorted(special, key=lambda x: int(x['id'])),
        'malformed_item_pair_lists': sorted(bad_item, key=lambda x: int(x['id'])),
        'malformed_prop_pair_lists': sorted(bad_prop, key=lambda x: int(x['id'])),
        'go_current_validator_trailing_semicolon_must_remain_fail_closed': True,
    }


def self_test():
    sample_shop = b'[a]\n0=prize,1,3,0,0,0,0,7\n0=prize,1,3,0,0,0,1,8\n0=prize,1,3,0,0,0,2,9\n'
    sample_ex = b'[7]\nItem=a,1;\n[8]\nProp=CapitalType2,100\n[9]\n'
    result = audit(sample_shop, sample_ex)
    assert result['mode3_nonzero_exchange_rows'] == 3
    assert result['field_presence_unique'] == {'item-only': 1, 'neither': 1, 'prop-only': 1}
    assert len(result['malformed_item_pair_lists']) == 1
    assert result['no_item_no_prop'][0]['empty_section']
    for broken in [b'[7]\n[7]\n', b'[7]\nItem=a,1\nItem=b,1\n']:
        try:
            audit(sample_shop, broken)
        except ValueError:
            pass
        else:
            raise AssertionError('duplicate/missing sections accepted')
    print('SELF_TEST_PASS: repeated shop keys preserved; field classes; trailing delimiter; duplicate/missing fail-closed')


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--shop', type=Path)
    parser.add_argument('--exchange', type=Path)
    parser.add_argument('--self-test', action='store_true')
    args = parser.parse_args()
    if args.self_test:
        self_test()
        return
    if args.shop is None or args.exchange is None:
        parser.error('verified --shop and --exchange payloads required')
    payloads = {'shop.ini': args.shop.read_bytes(), 'exchangeitem.ini': args.exchange.read_bytes()}
    for name, payload in payloads.items():
        got = hashlib.sha256(payload).hexdigest()
        if got != HASHES[name]:
            raise ValueError(f'{name}: exact-current SHA256 mismatch: {got}')
    print(json.dumps(audit(payloads['shop.ini'], payloads['exchangeitem.ini']),
                     sort_keys=True, indent=2, ensure_ascii=False))


if __name__ == '__main__':
    main()
