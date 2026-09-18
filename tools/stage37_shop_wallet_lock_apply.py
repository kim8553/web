#!/usr/bin/env python3
"""Apply only the reviewed wallet-compare call site to the exact Stage37 overlay."""
from pathlib import Path
import hashlib

path = Path('server/cmd/protocol-probe/zz_recovered_overlay.go')
original = path.read_bytes()
# GitHub blob SHA, not a raw-file SHA256.
blob_sha = hashlib.sha1(b'blob ' + str(len(original)).encode() + b'\0' + original).hexdigest()
expected = '7f44c22ad739da7a49383d04564d27da43c1edf6'
if blob_sha != expected:
    raise SystemExit(f'STOP: overlay blob changed: {blob_sha}, expected {expected}')
source = original.decode('utf-8')
changes = [
    ('\tnextWallet := currencySnapshot{}\n\tnextWallet.fromActor(player)\n\tswitch item.priceMode {',
     '\tstartingWallet := currencySnapshot{}\n\tstartingWallet.fromActor(player)\n\tnextWallet := startingWallet\n\tswitch item.priceMode {'),
    ('\tif err := persistOrdinaryShopPurchase(bagStore, currencyStore, roleID, nextBag, nextWallet); err != nil {',
     '\tif err := persistOrdinaryShopPurchase(bagStore, currencyStore, roleID, nextBag, startingWallet, nextWallet); err != nil {'),
]
for before, after in changes:
    if source.count(before) != 1 or after in source:
        raise SystemExit(f'STOP: expected exactly one untouched purchase anchor: {before[:60]!r}')
    source = source.replace(before, after, 1)
path.write_text(source, encoding='utf-8')
print('SHOP_WALLET_COMPARE_PATCH_APPLIED=YES')
