#!/usr/bin/env python3
"""Patch only audited Stage52 files; no migrations or user DB writes."""
from pathlib import Path
import re
import subprocess

root = Path(__file__).resolve().parents[2]
source = root / "server/cmd/retail-schema-preflight/main.go"
fixture = root / "server/cmd/retail-schema-preflight/integration_test.go"
expected = {
    source: "cc5b9845b7c551aa872ba22c0de8c9b1b71a5b7a",
    fixture: "4a6279ce7f4fd0f12fa0cf680c0d4d36953eb1eb",
}
for path, sha in expected.items():
    actual = subprocess.check_output(["git", "hash-object", str(path)], cwd=root, text=True).strip()
    if actual != sha:
        raise SystemExit(f"REFUSE changed source: {path.name} blob={actual}")

old = 'JSON_VALID(snapshot)=0'
new = 'JSON_VALID(CAST(snapshot AS CHAR CHARACTER SET utf8mb4))=0'
text = source.read_text(encoding="utf-8")
if text.count(old) != 1:
    raise SystemExit("REFUSE unexpected currency JSON query")
source.write_text(text.replace(old, new), encoding="utf-8")

text = fixture.read_text(encoding="utf-8")
pattern = r'^\s*`INSERT INTO role_currency\(role_id, snapshot\) VALUES .*\n'
text, changed = re.subn(pattern, "\n", text, count=1, flags=re.MULTILINE)
if changed != 1:
    raise SystemExit("REFUSE unexpected fixture SQL literal")
anchor = '\tembedded, err := migrations.Embedded()'
if text.count(anchor) != 1:
    raise SystemExit("REFUSE unexpected embedded migration anchor")
insert = '''\t// Bind JSON as bytes: avoid dialect-dependent SQL string escaping.\n\tif _, err := db.ExecContext(ctx, "INSERT INTO role_currency(role_id, snapshot) VALUES (?, ?)", 1, []byte(`{"silver":100,"gold":5,"silver_card":0,"silver_ticket":0}`)); err != nil {\n\t\tt.Fatal(err)\n\t}\n'''
fixture.write_text(text.replace(anchor, insert + anchor), encoding="utf-8")
print("PATCHED: production BLOB JSON validation plus parameterized disposable fixture only")
