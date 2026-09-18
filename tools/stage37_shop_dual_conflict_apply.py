#!/usr/bin/env python3
"""Narrow ordinary NPC buy conflict patch; refuse a changed gameplay source."""
from pathlib import Path
import subprocess

path = Path('server/cmd/protocol-probe/zz_recovered_overlay.go')
expected_blob = '8a241f18ef0e31e40388977d416851699f6ae81d'
actual = subprocess.check_output(['git', 'hash-object', str(path)], text=True).strip()
if actual != expected_blob:
    raise SystemExit(f'REFUSE_CHANGED_GAMEPLAY_SOURCE expected={expected_blob} actual={actual}')
text = path.read_text()
old = '''\t\tif errors.Is(err, shopbuyatomic.ErrWalletChanged) {
\t\t\tif refreshErr := resyncOrdinaryShopWalletAfterConflict(link, player, currencyStore, roleID, startingWallet); refreshErr != nil {
\t\t\t\treturn true, fmt.Errorf("shop buy wallet conflict: cannot safely refresh session: %w", refreshErr)
\t\t\t}
\t\t}
'''
new = '''\t\tif errors.Is(err, shopbuyatomic.ErrWalletChanged) {
\t\t\t// Wallet is checked first: this rejection can hide a simultaneous
\t\t\t// bag conflict. Refresh BOTH snapshots, without retrying the buy.
\t\t\tif refreshErr := resyncOrdinaryShopPurchaseAfterWalletConflict(link, player, bagStore, currencyStore, roleID, startingBag, startingWallet); refreshErr != nil {
\t\t\t\treturn true, fmt.Errorf("shop buy wallet conflict: cannot safely refresh session: %w", refreshErr)
\t\t\t}
\t\t}
'''
if text.count(old) != 1:
    raise SystemExit(f'REFUSE_UNEXPECTED_WALLET_CONFLICT_ANCHOR count={text.count(old)}')
path.write_text(text.replace(old, new, 1))
print('SHOP_DUAL_CONFLICT_SOURCE_LOCK=PASS PURCHASE_ONLY=YES NO_NEW_PACKET=YES')
