#!/usr/bin/env python3
"""Apply a narrow, fail-closed safety patch to the retained Stage37 buy handler.

The checkout must contain the exact reviewed source blob. The script never
copies an earlier server, changes a packet selector, or edits GM/exchange code.
"""
from pathlib import Path
import hashlib

TARGET = Path('server/cmd/protocol-probe/zz_recovered_overlay.go')
EXPECTED_BLOB = '1700ac2889273ca806494916ddbc6efcf6145f29'


def replace_once(source: str, before: str, after: str) -> str:
    occurrences = source.count(before)
    if occurrences != 1:
        raise SystemExit(f'purchase patch anchor count={occurrences}; refusing source modification')
    return source.replace(before, after, 1)


def main() -> None:
    original = TARGET.read_bytes()
    blob = hashlib.sha1(b'blob ' + str(len(original)).encode() + b'\0' + original).hexdigest()
    if blob != EXPECTED_BLOB:
        raise SystemExit(f'unexpected active shop source blob {blob}; no patch applied')
    text = original.decode('utf-8')
    text = replace_once(
        text,
        '\ttotal := int64(item.price) * int64(amount)\n',
        '\t// Encode the reward before touching the wallet: an unknown bag view,\n'
        '\t// an unrepresentable slot, or a malformed item must not charge.\n'
        '\treward := enrichBagItem(bagItem{ConfigID: item.configID, Amount: amount}, itemCatalog, nil)\n'
        '\tview := bagViewForViewID(reward.ViewID)\n'
        '\tif view == 0 {\n'
        '\t\tlog.Printf("%s: reject shop buy %s item %s: unsupported bag category %d", remote, shopID, item.configID, reward.ViewID)\n'
        '\t\treturn true, nil\n'
        '\t}\n'
        '\tslot := int(bagSlotFor(player, view))\n'
        '\tif slot < 1 || slot > 65535 {\n'
        '\t\tlog.Printf("%s: reject shop buy %s item %s: invalid free slot %d", remote, shopID, item.configID, slot)\n'
        '\t\treturn true, nil\n'
        '\t}\n'
        '\treward.Slot = int32(slot)\n'
        '\titemFrame, err := serverViewAdd(view, uint16(slot), bagItemProps(view, reward))\n'
        '\tif err != nil {\n'
        '\t\treturn true, fmt.Errorf("encode shop reward before wallet debit: %w", err)\n'
        '\t}\n'
        '\ttotal, totalErr := checkedOrdinaryShopTotal(item.price, amount)\n'
        '\tif totalErr != nil {\n'
        '\t\tlog.Printf("%s: reject shop buy %s item %s: %v", remote, shopID, item.configID, totalErr)\n'
        '\t\treturn true, nil\n'
        '\t}\n',
    )
    text = replace_once(
        text,
        '\treward := bagItem{ConfigID: item.configID, Amount: amount}\n'
        '\treward = enrichBagItem(reward, itemCatalog, nil)\n'
        '\tslot := player.addBagItem(reward)\n'
        '\tview := bagViewForViewID(reward.ViewID)\n'
        '\titemFrame, err := serverViewAdd(view, uint16(slot), bagItemProps(view, reward))\n'
        '\tif err != nil {\n'
        '\t\treturn true, err\n'
        '\t}\n',
        '\tslot = player.addBagItem(reward)\n',
    )
    TARGET.write_text(text, encoding='utf-8')
    print('SHOP_PREDEBIT_GUARD_APPLIED=YES')


if __name__ == '__main__':
    main()
