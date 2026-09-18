#!/usr/bin/env python3
"""Narrow ordinary NPC buy conflict patch; refuse a changed gameplay source."""
from pathlib import Path
import subprocess

path = Path('server/cmd/protocol-probe/zz_recovered_overlay.go')
expected_blob = '282a20af6a2efe63090c9f6622bfca598fef54e7'
actual = subprocess.check_output(['git', 'hash-object', str(path)], text=True).strip()
if actual != expected_blob:
    raise SystemExit(f'REFUSE_CHANGED_GAMEPLAY_SOURCE expected={expected_blob} actual={actual}')
text = path.read_text()
old = '\t\t\tif refreshErr := resyncOrdinaryShopWalletAfterConflict(link, player, currencyStore, roleID, startingWallet); refreshErr != nil {'
new = '''\t\t\t// The wallet check runs before the bag check, so wallet conflict
\t\t\t// can mask a simultaneous bag conflict. Never retry the purchase.
\t\t\tif refreshErr := resyncOrdinaryShopPurchaseAfterWalletConflict(link, player, bagStore, currencyStore, roleID, startingBag, startingWallet); refreshErr != nil {'''
if text.count(old) != 1:
    raise SystemExit(f'REFUSE_UNEXPECTED_WALLET_CONFLICT_ANCHOR count={text.count(old)}')
path.write_text(text.replace(old, new, 1))
print('SHOP_DUAL_CONFLICT_SOURCE_LOCK=PASS PURCHASE_ONLY=YES NO_NEW_PACKET=YES')
