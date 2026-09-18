#!/usr/bin/env python3
"""Append a purchase-first regression in the existing isolated-DB integration test."""
import hashlib
from pathlib import Path

path=Path('server/internal/shopbuyatomic/checked_mysql_integration_test.go')
raw=path.read_bytes()
sha=hashlib.sha1(b'blob '+str(len(raw)).encode()+b'\0'+raw).hexdigest()
expected='7d9bee0fa1de5486791beac83785766be36181f0'
if sha!=expected:
    raise SystemExit(f'SHOP_CHECKED_DELETE_TEST_SOURCE_MISMATCH={sha} expected={expected}')
old='''\tif wallet["silver"] != 100 {
\t\tt.Fatalf("rejected purchase altered silver=%d", wallet["silver"])
\t}
}
'''
new='''\tif wallet["silver"] != 100 {
\t\tt.Fatalf("rejected purchase altered silver=%d", wallet["silver"])
\t}

\t// Reverse order: a purchase commits first; an older bag-only snapshot
\t// must NOT delete the purchased item or overwrite the committed bag.
\tconst reverseRole uint64 = 880003
\tif _, err := db.Exec("INSERT INTO roles(role_id) VALUES (?)", reverseRole); err != nil {
\t\tt.Fatal(err)
\t}
\tif _, err := db.Exec("INSERT INTO role_currency(role_id,snapshot) VALUES (?,?)", reverseRole, before); err != nil {
\t\tt.Fatal(err)
\t}
\tif _, err := db.Exec(`INSERT INTO role_bag_items(role_id,seq,slot,config_id,item_type,amount,view_id)
VALUES (?,0,1,'old_item',100,1,1)`, reverseRole); err != nil {
\t\tt.Fatal(err)
\t}
\toriginal := []Row{{Slot: 1, ConfigID: "old_item", ItemType: 100, Amount: 1, ViewID: 1}}
\tpurchased := []Row{original[0], {Slot: 2, ConfigID: "new_item", ItemType: 100, Amount: 1, ViewID: 1}}
\tif err := SaveCheckedBag(purchaseDB, reverseRole, purchased, original, before, after); err != nil {
\t\tt.Fatalf("reverse purchase commit: %v", err)
\t}
\tstaleWriter := []Row{{Slot: 1, ConfigID: "old_item", ItemType: 100, Amount: 2, ViewID: 1}}
\tif err := SaveBagChecked(db, reverseRole, staleWriter, original); err == nil || !strings.Contains(err.Error(), "bag changed") {
\t\tt.Fatalf("stale bag-only writer after committed purchase was not rejected: %v", err)
\t}
\tvar persistedCount int
\tif err := reopened.QueryRow("SELECT COUNT(*) FROM role_bag_items WHERE role_id=?", reverseRole).Scan(&persistedCount); err != nil {
\t\tt.Fatal(err)
\t}
\tif persistedCount != 2 {
\t\tt.Fatalf("stale writer erased purchased bag row: count=%d", persistedCount)
\t}
\tvar bought string
\tif err := reopened.QueryRow("SELECT config_id FROM role_bag_items WHERE role_id=? AND seq=1", reverseRole).Scan(&bought); err != nil {
\t\tt.Fatal(err)
\t}
\tif bought != "new_item" {
\t\tt.Fatalf("stale writer erased purchased item: %q", bought)
\t}
\tvar reverseWalletJSON []byte
\tif err := reopened.QueryRow("SELECT snapshot FROM role_currency WHERE role_id=?", reverseRole).Scan(&reverseWalletJSON); err != nil {
\t\tt.Fatal(err)
\t}
\tvar reverseWallet map[string]int64
\tif err := json.Unmarshal(reverseWalletJSON, &reverseWallet); err != nil {
\t\tt.Fatal(err)
\t}
\tif reverseWallet["silver"] != 70 {
\t\tt.Fatalf("bag-only rejection changed committed wallet: %d", reverseWallet["silver"])
\t}
\t// A writer that actually read the new bag may still update it normally.
\tfresh := []Row{{Slot: 1, ConfigID: "old_item", ItemType: 100, Amount: 2, ViewID: 1}, purchased[1]}
\tif err := SaveBagChecked(db, reverseRole, fresh, purchased); err != nil {
\t\tt.Fatalf("fresh bag-only writer rejected: %v", err)
\t}
\tvar freshAmount int
\tif err := reopened.QueryRow("SELECT amount FROM role_bag_items WHERE role_id=? AND seq=0", reverseRole).Scan(&freshAmount); err != nil {
\t\tt.Fatal(err)
\t}
\tif freshAmount != 2 {
\t\tt.Fatalf("fresh writer amount=%d, want 2", freshAmount)
\t}
}
'''
if raw.count(old.encode())!=1:
    raise SystemExit(f'SHOP_CHECKED_DELETE_TEST_ANCHOR_COUNT={raw.count(old.encode())}')
path.write_bytes(raw.replace(old.encode(),new.encode(),1))
print('SHOP_CHECKED_DELETE_REVERSE_ORDER_MYSQL_TEST_APPLIED=YES')
