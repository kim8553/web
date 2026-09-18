#!/usr/bin/env python3
"""Source-locked candidate patch for the remaining legacy mysqlBagStore.Save.

Purchase role-row lock and tests are already committed. Refuse a source drift.
"""
import hashlib
from pathlib import Path

checked = Path('server/internal/shopbuyatomic/checked.go').read_text()
if 'func LockRoleForBagWrite(' not in checked or 'if err := LockRoleForBagWrite(ctx, tx, roleID); err != nil {' not in checked:
    raise SystemExit('SHOP_WRITER_LOCK_PURCHASE_ANCHOR_MISSING')
path = Path('server/cmd/protocol-probe/zz_recovered_overlay.go')
raw = path.read_bytes()
actual = hashlib.sha1(b'blob ' + str(len(raw)).encode() + b'\0' + raw).hexdigest()
expected = '308c7714307c8eff8f03555531e8c7d5704253e8'
if actual != expected:
    raise SystemExit(f'SHOP_WRITER_LOCK_BLOB_MISMATCH: {actual} != {expected}')
changes = [
    (b'"github.com/local/9yin-go-server/internal/role"\n',
     b'"github.com/local/9yin-go-server/internal/role"\n\t"github.com/local/9yin-go-server/internal/shopbuyatomic"\n'),
    (b'\tdefer tx.Rollback()\n\tif _, err := tx.ExecContext(ctx, "DELETE FROM role_bag_items WHERE role_id = ?", roleID); err != nil {',
     b'\tdefer tx.Rollback()\n\t// Serialize this legacy bag writer with the ordinary purchase transaction.\n\t// This does not repair already-stale caller snapshots.\n\tif err := shopbuyatomic.LockRoleForBagWrite(ctx, tx, uint64(roleID)); err != nil {\n\t\treturn err\n\t}\n\tif _, err := tx.ExecContext(ctx, "DELETE FROM role_bag_items WHERE role_id = ?", roleID); err != nil {'),
]
for old, new in changes:
    if raw.count(old) != 1:
        raise SystemExit(f'SHOP_WRITER_LOCK_ANCHOR_COUNT={raw.count(old)}')
    raw = raw.replace(old, new, 1)
path.write_bytes(raw)
print('SHOP_WRITER_LOCK_OVERLAY_CANDIDATE_APPLIED=YES')
