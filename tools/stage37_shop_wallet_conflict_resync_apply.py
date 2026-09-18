#!/usr/bin/env python3
"""Apply only a typed NPC purchase wallet conflict and its refresh call site.

Original functions, packet encoding, other currency writers, GM and exchange
behavior are intentionally untouched. Fail when any source blob drifts.
"""
from pathlib import Path
import subprocess

TARGETS = {
    'server/internal/shopbuyatomic/checked.go': 'f8fc05066855aa5cfd4d0b8a43c8479af24e9137',
    'server/cmd/protocol-probe/zz_recovered_overlay.go': '72c888ca89f85268403059ec81cd0ef220f2a506',
    'server/cmd/protocol-probe/player_actor.go': 'f04d2217e1e596b0edec5df65dc57395cf7ec9c1',
}
for name, sha in TARGETS.items():
    current = subprocess.check_output(['git', 'hash-object', name], text=True).strip()
    if current != sha:
        raise SystemExit(f'REFUSE_CHANGED_SOURCE {name}: {current} != {sha}')


def replace_exact(name: str, old: str, new: str) -> None:
    path = Path(name)
    src = path.read_text()
    if src.count(old) != 1:
        raise SystemExit(f'REFUSE_UNEXPECTED_SOURCE {name}: occurrences={src.count(old)}')
    path.write_text(src.replace(old, new, 1))


replace_exact(
    'server/internal/shopbuyatomic/checked.go',
    ')\n\n// LockRoleForBagWrite',
    ')\n\n// ErrWalletChanged identifies an ordinary NPC purchase rejected by the\n'
    '// locked database wallet comparison. Callers must not treat other DB\n'
    '// failures as safe to reconcile.\n'
    'var ErrWalletChanged = errors.New("shop buy: wallet changed since purchase began")\n\n'
    '// LockRoleForBagWrite',
)
replace_exact(
    'server/internal/shopbuyatomic/checked.go',
    'return errors.New("shop buy: wallet changed since purchase began; reload before retrying")',
    'return fmt.Errorf("%w; reload before retrying", ErrWalletChanged)',
)
replace_exact(
    'server/cmd/protocol-probe/zz_recovered_overlay.go',
    '\tif err := persistOrdinaryShopPurchase(bagStore, currencyStore, roleID, nextBag, startingBag, startingWallet, nextWallet); err != nil {\n'
    '\t\tlog.Printf("%s: reject uncommitted shop buy shop=%s item=%s: %v", remote, shopID, item.configID, err)\n'
    '\t\treturn true, nil\n'
    '\t}\n',
    '\tif err := persistOrdinaryShopPurchase(bagStore, currencyStore, roleID, nextBag, startingBag, startingWallet, nextWallet); err != nil {\n'
    '\t\tlog.Printf("%s: reject uncommitted shop buy shop=%s item=%s: %v", remote, shopID, item.configID, err)\n'
    '\t\tif errors.Is(err, shopbuyatomic.ErrWalletChanged) {\n'
    '\t\t\tif refreshErr := resyncOrdinaryShopWalletAfterConflict(link, player, currencyStore, roleID, startingWallet); refreshErr != nil {\n'
    '\t\t\t\treturn true, fmt.Errorf("shop buy wallet conflict: cannot safely refresh session: %w", refreshErr)\n'
    '\t\t\t}\n'
    '\t\t}\n'
    '\t\treturn true, nil\n'
    '\t}\n',
)
replace_exact(
    'server/cmd/protocol-probe/player_actor.go',
    'shopWallet            *currencySnapshot // last successfully committed ordinary NPC purchase',
    'shopWallet            *currencySnapshot // last persisted NPC shop wallet or conflict reconciliation',
)
print('SHOP_WALLET_CONFLICT_SOURCE_LOCK=PASS PURCHASE_ONLY=YES NO_NEW_PACKET=YES')
