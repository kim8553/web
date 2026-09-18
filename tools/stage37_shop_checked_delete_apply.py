#!/usr/bin/env python3
"""Patch only the existing DELETEITEM persistence sequence after source check.

No client opcode, item effect, GM grant, or other inventory handler is changed.
"""
import hashlib
from pathlib import Path

path = Path('server/cmd/protocol-probe/zz_recovered_overlay.go')
raw = path.read_bytes()
actual = hashlib.sha1(b'blob ' + str(len(raw)).encode() + b'\0' + raw).hexdigest()
expected = 'bc048c5a6c060a3300475a1176322b82fdaae30c'
if actual != expected:
    raise SystemExit(f'SHOP_CHECKED_DELETE_SOURCE_MISMATCH={actual} expected={expected}')

edits = [
    (
        '''\titem, remaining, consumed, ok := player.decrementBagItem(uint16(srcView), srcPos, amount)
''',
        '''\tstartingBag := player.bagSnapshot()
\titem, remaining, consumed, ok := player.decrementBagItem(uint16(srcView), srcPos, amount)
''',
    ),
    (
        '''\tif err := link.WriteFrame(frame); err != nil {
\t\treturn true, fmt.Errorf("write delete item: %w", err)
\t}
\tif bagStore != nil {
\t\tif err := bagStore.Save(roleID, player.bagSnapshot()); err != nil {
\t\t\tlog.Printf("persist bag after delete: %v", err)
\t\t}
\t}
''',
        '''\t// Persist before publishing the existing frame: rejecting an already-stale
\t// actor bag must not also tell the client that an item was deleted.
\tif err := persistBagMutationChecked(bagStore, roleID, startingBag, player.bagSnapshot()); err != nil {
\t\tplayer.restoreBag(startingBag)
\t\tlog.Printf("%s: reject DELETEITEM after bag changed: %v", remote, err)
\t\treturn true, nil
\t}
\tif err := link.WriteFrame(frame); err != nil {
\t\treturn true, fmt.Errorf("write delete item: %w", err)
\t}
''',
    ),
]
for old, new in edits:
    old, new = old.encode(), new.encode()
    if raw.count(old) != 1:
        raise SystemExit(f'SHOP_CHECKED_DELETE_ANCHOR_COUNT={raw.count(old)} for {old[:75]!r}')
    raw = raw.replace(old, new, 1)
path.write_bytes(raw)
print('SHOP_CHECKED_DELETE_PATCH_APPLIED=YES')
