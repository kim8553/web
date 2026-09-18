"""Offline integrity and restricted-SQL check. No DB/network access."""
from pathlib import Path
import hashlib
import re
import sys

ROOT = Path(__file__).resolve().parent
EXPECTED = 'b5cc5a2f62c05e4efd4d340f86747b589640d24e8fdd4b2193827a027f7df31d'
FILES = ['01_SHOP_SCHEMA_METADATA_ONLY.sql', '02_SHOP_INSERT_CONSTRAINTS_ONLY.sql',
         '03_MIGRATION_LEDGER_IF_PRESENT_ONLY.sql']


def statements(text):
    text = re.sub(r'/\*.*?\*/', ' ', text, flags=re.S)
    text = '\n'.join(line.split('--', 1)[0] for line in text.splitlines())
    return [part.strip() for part in text.split(';') if part.strip()]


def validate_sql(text):
    stmts = statements(text)
    assert stmts, 'empty SQL'
    for stmt in stmts:
        assert re.match(r'^SELECT\s', stmt, re.I), f'not a SELECT: {stmt[:60]}'
        # SELECT can still write through INTO OUTFILE or invoke unsafe routines.
        assert not re.search(r'\bINTO\s+(?:OUTFILE|DUMPFILE)\b|\bFOR\s+UPDATE\b|\b(?:GET_LOCK|LOAD_FILE|SLEEP|BENCHMARK)\s*\(', stmt, re.I)
    return len(stmts)


def main():
    counts = [validate_sql((ROOT / filename).read_text(encoding='utf-8')) for filename in FILES]
    ledger = (ROOT / FILES[2]).read_text(encoding='utf-8')
    assert EXPECTED in ledger, 'embedded migration checksum not found'
    insert = (ROOT / FILES[1]).read_text(encoding='utf-8')
    assert 'BLOCKER_OPTIONAL_NULL_NOT_ALLOWED' in insert
    assert 'BLOCKER_REQUIRED_COLUMN_NOT_INSERTED' in insert
    assert 'COLUMN_DEFAULT IS NULL' in insert
    assert 'role_currency' in insert and 'role_bag_items' in insert
    if len(sys.argv) == 2:
        migration = Path(sys.argv[1]).read_bytes()
        actual = hashlib.sha256(migration).hexdigest()
        assert actual == EXPECTED, f'wrong migration bytes: {actual}'
        print('Exact embedded migration SHA-256: MATCH')
    assert counts == [5, 2, 2], f'unexpected SQL statement counts: {counts}'
    print(f'Static read-only SQL checks PASS: {counts} SELECT statements (total {sum(counts)})')
    print('User MySQL compatibility: NOT RUN / UNKNOWN')


if __name__ == '__main__':
    main()
