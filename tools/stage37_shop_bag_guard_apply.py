#!/usr/bin/env python3
"""Apply one SHA-locked change to the existing NPC shop buy handler."""
import hashlib
from pathlib import Path

path = Path('server/cmd/protocol-probe/zz_recovered_overlay.go')
raw = path.read_bytes()
blob_sha = hashlib.sha1(b'blob ' + str(len(raw)).encode() + b'\0' + raw).hexdigest()
expected_sha = 'ab2c3a436847a198a4d2c2bb009363228f0b93f3'
if blob_sha != expected_sha:
    raise SystemExit(f'SHOP_BAG_GUARD_BLOB_MISMATCH: {blob_sha} != {expected_sha}')
old = (b'\tnextBag := append(player.bagSnapshot(), reward)\n'
       b'\tif err := persistOrdinaryShopPurchase(bagStore, currencyStore, roleID, nextBag, startingWallet, nextWallet); err != nil {')
new = (b'\tstartingBag := player.bagSnapshot()\n'
       b'\tnextBag := append(startingBag, reward)\n'
       b'\tif err := persistOrdinaryShopPurchase(bagStore, currencyStore, roleID, nextBag, startingBag, startingWallet, nextWallet); err != nil {')
if raw.count(old) != 1:
    raise SystemExit(f'SHOP_BAG_GUARD_ANCHOR_COUNT={raw.count(old)}')
path.write_bytes(raw.replace(old, new, 1))
print('SHOP_BAG_GUARD_PATCH_APPLIED=YES')
