#!/usr/bin/env python3
"""Pinned, two-file normal 0x46 buyer integration; NEVER touch GM or exchange."""
from pathlib import Path
import hashlib

ROOT = Path(__file__).resolve().parents[2]
OVERLAY = ROOT / 'server/cmd/protocol-probe/zz_recovered_overlay.go'
MAIN = ROOT / 'server/cmd/protocol-probe/main.go'
EXPECTED = {
    OVERLAY: '15568d3f43794eb78045a39655e1a4964b8fb736',
    MAIN: '63f8de991249b40b4e95418ebcb497bdc070458a',
}

def blob_sha(data: bytes) -> str:
    return hashlib.sha1(b'blob ' + str(len(data)).encode() + b'\0' + data).hexdigest()

for file, want in EXPECTED.items():
    actual = blob_sha(file.read_bytes())
    if actual != want:
        raise SystemExit(f'ABORT: {file.name} unexpected Git blob {actual} != {want}')

source = OVERLAY.read_text(encoding='utf-8')
start = 'func handleShopBuyCustom(link sceneMessageConnection, player *playerActor, itemCatalog *itemCatalog, bagStore bagStoreIface, currencyStore currencyStoreIface, roleID role.RoleID, custom clientCustomMessage, remote string) (bool, error) {'
end = '\nfunc shopCatalogItems(path, shopID string)'
if source.count(start) != 1 or source.count(end) != 1:
    raise SystemExit('ABORT: old normal-buy handler is not uniquely identifiable')
left, rest = source.split(start, 1)
old_body, right = rest.split(end, 1)
if 'normalShopSafeTotal(item.price, amount)' not in old_body or 'persistBagEquip(bagStore, nil, roleID, player)' not in old_body:
    raise SystemExit('ABORT: old handler content differs')
new_handler = '''func handleShopBuyCustom(link sceneMessageConnection, player *playerActor, world *sceneLifecycle, itemCatalog *itemCatalog, equipCatalog *equipCatalog, bagStore bagStoreIface, currencyStore currencyStoreIface, roleID role.RoleID, custom clientCustomMessage, remote string) (bool, error) {
\treturn normalShopAtomicBuy(link, player, world, itemCatalog, equipCatalog, bagStore, currencyStore, roleID, custom, remote)
}
'''
overlay_new = left + new_handler + end + right

main = MAIN.read_text(encoding='utf-8')
old_call = 'handleShopBuyCustom(link, player, itemCatalog, bagStore, currencyStore, selectedRoleID(selected), custom, conn.RemoteAddr().String())'
new_call = 'handleShopBuyCustom(link, player, world, itemCatalog, equipCatalog, bagStore, currencyStore, selectedRoleID(selected), custom, conn.RemoteAddr().String())'
if main.count(old_call) != 2:
    raise SystemExit(f'ABORT: expected exactly two 0x46 dispatch routes; found {main.count(old_call)}')
main_new = main.replace(old_call, new_call)
if 'case 79:' not in main_new or 'handleShopExchangeContract(' not in main_new:
    raise SystemExit('ABORT: exchange route unexpectedly missing')
OVERLAY.write_text(overlay_new, encoding='utf-8')
MAIN.write_text(main_new, encoding='utf-8')
print('PATCHED: exactly two 0x46 dispatch call sites and one normal-shop handler; GM and 0x4F unchanged')
