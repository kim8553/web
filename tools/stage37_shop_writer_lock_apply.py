#!/usr/bin/env python3
"""Narrow, blob-locked role-row serialization of existing bag writers.

Do not replace the original inventory system. Refuse unexpected source edits.
"""
import hashlib
from pathlib import Path


def patch(path, sha, edits):
    p = Path(path)
    raw = p.read_bytes()
    actual = hashlib.sha1(b'blob ' + str(len(raw)).encode() + b'\0' + raw).hexdigest()
    if actual != sha:
        raise SystemExit(f'SHOP_WRITER_LOCK_BLOB_MISMATCH {path}: {actual} != {sha}')
    for old, new in edits:
        old, new = old.encode(), new.encode()
        count = raw.count(old)
        if count != 1:
            raise SystemExit(f'SHOP_WRITER_LOCK_ANCHOR_COUNT {path}: {count}')
        raw = raw.replace(old, new, 1)
    p.write_bytes(raw)


patch('server/internal/shopbuyatomic/checked.go', 'de1e46e67d688cb325a8c6306f1983c43965e732', [
    ('// SaveChecked retains the wallet-only check for existing callers.',
     '''// LockRoleForBagWrite serializes all participating bag writes for a real
// migrated role. Unlike a bag-row lock, this also works for an empty bag;
// unlike a currency-row lock, it works before a wallet snapshot is created.
// It does not establish optimistic freshness of a legacy caller's snapshot.
func LockRoleForBagWrite(ctx context.Context, tx *sql.Tx, roleID uint64) error {
    if tx == nil || roleID == 0 {
        return errors.New("shop bag: missing transaction or role")
    }
    var lockedID uint64
    if err := tx.QueryRowContext(ctx, "SELECT role_id FROM roles WHERE role_id = ? FOR UPDATE", roleID).Scan(&lockedID); err != nil {
        return fmt.Errorf("lock role for bag write: %w", err)
    }
    if lockedID != roleID {
        return errors.New("shop bag: locked wrong role")
    }
    return nil
}

// SaveChecked retains the wallet-only check for existing callers.'''),
    ('\tdefer tx.Rollback()\n\tvar storedJSON []byte',
     '\tdefer tx.Rollback()\n\tif checkBag {\n\t\tif err := LockRoleForBagWrite(ctx, tx, roleID); err != nil {\n\t\t\treturn err\n\t\t}\n\t}\n\tvar storedJSON []byte'),
])
patch('server/cmd/protocol-probe/zz_recovered_overlay.go', '308c7714307c8eff8f03555531e8c7d5704253e8', [
    ('"github.com/local/9yin-go-server/internal/role"\n',
     '"github.com/local/9yin-go-server/internal/role"\n\t"github.com/local/9yin-go-server/internal/shopbuyatomic"\n'),
    ('\tdefer tx.Rollback()\n\tif _, err := tx.ExecContext(ctx, "DELETE FROM role_bag_items WHERE role_id = ?", roleID); err != nil {',
     '\tdefer tx.Rollback()\n\t// Use the same existing roles row lock as ordinary NPC purchases.\n\t// This prevents overlapping DELETE/INSERT bag transactions; it does not\n\t// make an already-stale legacy caller snapshot fresh.\n\tif err := shopbuyatomic.LockRoleForBagWrite(ctx, tx, uint64(roleID)); err != nil {\n\t\treturn err\n\t}\n\tif _, err := tx.ExecContext(ctx, "DELETE FROM role_bag_items WHERE role_id = ?", roleID); err != nil {'),
])
patch('server/internal/shopbuyatomic/checked_bag_test.go', '074c63917da4f0b3e46bbe1aedb1dca2e6627a14', [
    ('\t\t\tmock.ExpectBegin()\n\t\t\tmock.ExpectQuery("SELECT snapshot FROM role_currency")',
     '\t\t\tmock.ExpectBegin()\n\t\t\tmock.ExpectQuery("SELECT role_id FROM roles").WithArgs(uint64(7)).WillReturnRows(sqlmock.NewRows([]string{"role_id"}).AddRow(uint64(7)))\n\t\t\tmock.ExpectQuery("SELECT snapshot FROM role_currency")'),
])
patch('server/internal/shopbuyatomic/checked_mysql_integration_test.go', '6fda258f10e4d1d24af887a2c856806f048be204', [
    ('\tif _, err := db.Exec(`CREATE TABLE role_bag_items (',
     '\tif _, err := db.Exec(`CREATE TABLE roles (role_id BIGINT UNSIGNED NOT NULL PRIMARY KEY) ENGINE=InnoDB`); err != nil {\n\t\tt.Fatal(err)\n\t}\n\tif _, err := db.Exec(`CREATE TABLE role_bag_items ('),
    ('\tconst id uint64 = 880001\n\tif _, err := db.Exec(`INSERT INTO role_currency',
     '\tconst id uint64 = 880001\n\tif _, err := db.Exec(`INSERT INTO roles(role_id) VALUES (?)`, id); err != nil {\n\t\tt.Fatal(err)\n\t}\n\tif _, err := db.Exec(`INSERT INTO role_currency'),
])
print('SHOP_WRITER_LOCK_PATCH_APPLIED=YES')
