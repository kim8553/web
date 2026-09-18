#!/usr/bin/env python3
"""Apply only the reviewed ordinary-shop persistence change, fail closed on drift."""
from pathlib import Path
import subprocess

path = Path('server/cmd/protocol-probe/zz_recovered_overlay.go')
expected_blob = '18765d66dd58830094c9e3c345298148f6cf9ed1'
actual_blob = subprocess.check_output(['git', 'hash-object', str(path)], text=True).strip()
if actual_blob != expected_blob:
    raise SystemExit(f'SHOP_ATOMIC_BLOCKED_SOURCE_DRIFT expected={expected_blob} actual={actual_blob}')
source = path.read_text(encoding='utf-8')
start_marker = 'func handleShopBuyCustom('
end_marker = '\nfunc shopCatalogItems('
if source.count(start_marker) != 1 or source.count(end_marker) != 1:
    raise SystemExit('SHOP_ATOMIC_BLOCKED_HANDLER_BOUNDARY')
head, tail = source.split(start_marker, 1)
body, after = tail.split(end_marker, 1)
old_start = '\tswitch item.priceMode {\n'
old_end = '\tlog.Printf("%s: shop buy %s item %s x%d capital=%d price=%d -> bag view=%d slot=%d", remote, shopID, item.configID, amount, item.priceMode, item.price, view, slot)\n'
if body.count(old_start) != 1 or body.count(old_end) != 1:
    raise SystemExit('SHOP_ATOMIC_BLOCKED_PURCHASE_BLOCK_DRIFT')
before, remainder = body.split(old_start, 1)
old_middle, end = remainder.split(old_end, 1)
if 'persistBagEquip(bagStore, nil, roleID, player)' not in old_middle or 'writeFrames(link, frames...)' not in old_middle:
    raise SystemExit('SHOP_ATOMIC_BLOCKED_UNEXPECTED_ORIGINAL_SAVE')
replacement = '''\tnextWallet := currencySnapshot{}
\tnextWallet.fromActor(player)
\tswitch item.priceMode {
\tcase 0:
\t\tif int64(nextWallet.Gold) < total {
\t\t\tlog.Printf("%s: shop buy %s item %s needs gold %d, has %d", remote, shopID, item.configID, total, nextWallet.Gold)
\t\t\treturn true, nil
\t\t}
\t\tnextWallet.Gold -= int32(total)
\tcase 1:
\t\tif int64(nextWallet.Silver) < total {
\t\t\tlog.Printf("%s: shop buy %s item %s needs silver %d, has %d", remote, shopID, item.configID, total, nextWallet.Silver)
\t\t\treturn true, nil
\t\t}
\t\tnextWallet.Silver -= int32(total)
\tcase 2:
\t\tif int64(nextWallet.SilverCard) < total {
\t\t\tlog.Printf("%s: shop buy %s item %s needs silverCard %d, has %d", remote, shopID, item.configID, total, nextWallet.SilverCard)
\t\t\treturn true, nil
\t\t}
\t\tnextWallet.SilverCard -= int32(total)
\t}
\tcurrencyFrame, err := ordinaryShopCurrencyFrame(nextWallet)
\tif err != nil {
\t\treturn true, fmt.Errorf("encode shop wallet before persistence: %w", err)
\t}
\tnextBag := append(player.bagSnapshot(), reward)
\tif err := persistOrdinaryShopPurchase(bagStore, currencyStore, roleID, nextBag, nextWallet); err != nil {
\t\tlog.Printf("%s: reject uncommitted shop buy shop=%s item=%s: %v", remote, shopID, item.configID, err)
\t\treturn true, nil
\t}
\t// The database commit is definitive. Publish the same snapshots only after
\t// the complete bag and wallet have been durably committed together.
\tnextWallet.toActor(player)
\tplayer.restoreBag(nextBag)
\tif err := writeFrames(link, currencyFrame, itemFrame); err != nil {
\t\treturn true, fmt.Errorf("publish committed shop purchase: %w", err)
\t}
'''
updated = head + start_marker + before + replacement + old_end + end + end_marker + after
if updated == source:
    raise SystemExit('SHOP_ATOMIC_BLOCKED_NO_CHANGE')
path.write_text(updated, encoding='utf-8')
print('SHOP_ATOMIC_PATCH_APPLIED=YES')
