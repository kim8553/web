#!/usr/bin/env python3
"""Apply ONLY the checked NPC purchase bag-conflict classification and replay."""
from pathlib import Path
import subprocess


def replace_once(path: str, source_sha: str, old: str, new: str) -> None:
    actual_sha = subprocess.check_output(["git", "hash-object", path], text=True).strip()
    if actual_sha != source_sha:
        raise SystemExit(f"REFUSE_SOURCE_CHANGED {path} expected={source_sha} actual={actual_sha}")
    file = Path(path)
    text = file.read_text(encoding="utf-8")
    if text.count(old) != 1:
        raise SystemExit(f"REFUSE_UNEXPECTED_ANCHOR {path} count={text.count(old)}")
    file.write_text(text.replace(old, new, 1), encoding="utf-8")


checked = "server/internal/shopbuyatomic/checked.go"
checked_sha = "cd36b84e2b8190dbee6f2f2c4d2852a7b3ceae9c"
replace_once(
    checked, checked_sha,
    'var ErrWalletChanged = errors.New("shop buy: wallet changed since purchase began")',
    'var ErrWalletChanged = errors.New("shop buy: wallet changed since purchase began")\n\n'
    '// ErrBagChanged identifies ONLY a persisted-bag comparison mismatch.\n'
    '// Other database failures must not trigger a destructive client replay.\n'
    'var ErrBagChanged = errors.New("shop buy: bag changed since purchase began")',
)
# The first edit changes the blob hash; all subsequent edits operate only on
# that already validated in-memory source with individually unique anchors.
file = Path(checked)
text = file.read_text(encoding="utf-8")
old = 'return fmt.Errorf("shop buy: bag changed since purchase began at row %d; reload before retrying", index)'
new = 'return fmt.Errorf("%w at row %d; reload before retrying", ErrBagChanged, index)'
if text.count(old) != 1:
    raise SystemExit("REFUSE_MISSING_BAG_ROW_ANCHOR")
text = text.replace(old, new, 1)
old = 'return fmt.Errorf("shop buy: bag changed since purchase began (rows %d, expected %d); reload before retrying", index, len(expected))'
new = 'return fmt.Errorf("%w (rows %d, expected %d); reload before retrying", ErrBagChanged, index, len(expected))'
if text.count(old) != 1:
    raise SystemExit("REFUSE_MISSING_BAG_COUNT_ANCHOR")
file.write_text(text.replace(old, new, 1), encoding="utf-8")

overlay = "server/cmd/protocol-probe/zz_recovered_overlay.go"
overlay_sha = "8a241f18ef0e31e40388977d416851699f6ae81d"
old = '''\t\tif errors.Is(err, shopbuyatomic.ErrWalletChanged) {
\t\t\tif refreshErr := resyncOrdinaryShopWalletAfterConflict(link, player, currencyStore, roleID, startingWallet); refreshErr != nil {
\t\t\t\treturn true, fmt.Errorf("shop buy wallet conflict: cannot safely refresh session: %w", refreshErr)
\t\t\t}
\t\t}
\t\treturn true, nil'''
new = '''\t\tif errors.Is(err, shopbuyatomic.ErrWalletChanged) {
\t\t\tif refreshErr := resyncOrdinaryShopWalletAfterConflict(link, player, currencyStore, roleID, startingWallet); refreshErr != nil {
\t\t\t\treturn true, fmt.Errorf("shop buy wallet conflict: cannot safely refresh session: %w", refreshErr)
\t\t\t}
\t\t} else if errors.Is(err, shopbuyatomic.ErrBagChanged) {
\t\t\tif refreshErr := resyncOrdinaryShopBagAfterConflict(link, player, bagStore, roleID, startingBag); refreshErr != nil {
\t\t\t\treturn true, fmt.Errorf("shop buy bag conflict: cannot safely refresh session: %w", refreshErr)
\t\t\t}
\t\t}
\t\treturn true, nil'''
replace_once(overlay, overlay_sha, old, new)
print("SHOP_BAG_CONFLICT_SOURCE_LOCK=PASS PURCHASE_ONLY=YES NO_NEW_PACKET=YES")
