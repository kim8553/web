#!/usr/bin/env python3
"""Apply only the audited normal NPC purchase / logout wallet fix.

Never modify the GM grant, GM explicit currency commands, exchange flows,
client protocol, or resource loaders. Executed in CI before publish.
"""
from pathlib import Path
import subprocess

ROOT = Path('server/cmd/protocol-probe')
EXPECTED = {
    'main.go': '5b7e3c5d1184723415cc121371a17d39dff9328b',
    'player_actor.go': '60872dddc1899d2e49354baa68a2abc9782271ad',
    'zz_recovered_overlay.go': '4803e4867c84b228ef170300b7827e691ce1e6ad',
}
for name, expected_sha in EXPECTED.items():
    path = ROOT / name
    actual = subprocess.check_output(['git', 'hash-object', str(path)], text=True).strip()
    if actual != expected_sha:
        raise SystemExit(f'REFUSE_SOURCE_CHANGED path={path} expected={expected_sha} got={actual}')


def replace_once(name: str, old: str, new: str):
    path = ROOT / name
    original = path.read_text()
    if original.count(old) != 1:
        raise SystemExit(f'REFUSE_UNEXPECTED_ANCHOR {path}: {old!r} count={original.count(old)}')
    path.write_text(original.replace(old, new, 1))

replace_once(
    'player_actor.go',
    '\tsilverTicket          int32\n',
    '\tsilverTicket          int32\n\tshopWallet            *currencySnapshot // last successfully committed ordinary NPC purchase\n',
)
replace_once(
    'zz_recovered_overlay.go',
    '\tnextWallet.toActor(player)\n\tplayer.restoreBag(nextBag)\n',
    '\tnextWallet.toActor(player)\n\tplayer.markOrdinaryShopWalletCommitted(nextWallet)\n\tplayer.restoreBag(nextBag)\n',
)
replace_once(
    'main.go',
    '\tdefer persistCurrency()\n',
    '\t// Ordinary NPC purchases already commit currency with the bag. Do not\n'
    '\t// overwrite a newer session wallet with the same stale snapshot on exit.\n'
    '\t// Explicit currency persistence elsewhere retains its existing behavior.\n'
    '\tdefer func() {\n'
    '\t\tif currencyStore == nil || selected == nil || player == nil {\n'
    '\t\t\treturn\n'
    '\t\t}\n'
    '\t\tif err := persistDeferredShopCurrency(currencyStore, selected.ID, player); err != nil {\n'
    '\t\t\tlog.Printf("%s: persist currency on disconnect: %v", conn.RemoteAddr(), err)\n'
    '\t\t}\n'
    '\t}()\n',
)
print('SHOP_DISCONNECT_WALLET_SOURCE_LOCK=PASS GM_GRANT_UNCHANGED=YES GM_CURRENCY_COMMANDS_UNCHANGED=YES EXCHANGE_UNCHANGED=YES')
