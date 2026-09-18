#!/usr/bin/env python3
"""Narrow, source-locked repair of the existing bag-to-bag MOVEITEM path.

Run only on the audited Stage37 overlay blob. No packet layouts or resource
semantics are added; the current existing frames are kept intact.
"""
from pathlib import Path
import hashlib
import re

path = Path('server/cmd/protocol-probe/zz_recovered_overlay.go')
raw = path.read_bytes()
blob_sha = hashlib.sha1(b'blob ' + str(len(raw)).encode() + b'\0' + raw).hexdigest()
expected = '851420924a5bd5fe54e08c40eb45a81ee154dbee'
if blob_sha != expected:
    raise SystemExit(f'refuse unknown overlay source: {blob_sha} != {expected}')
text = raw.decode('utf-8')
start_marker = 'func applyBagMove('
end_marker = '\nfunc weaponModeFor('
if text.count(start_marker) != 1 or text.count(end_marker) != 1:
    raise SystemExit('refuse unknown bag move function boundary')
prefix, remainder = text.split(start_marker, 1)
function, suffix = remainder.split(end_marker, 1)
function = start_marker + function

def change(old: str, new: str):
    global function
    if function.count(old) != 1:
        raise SystemExit(f'refuse unexpected bag move source anchor count {function.count(old)}: {old!r}')
    function = function.replace(old, new, 1)

change(') (bool, error) {\n\titem, ok := player.takeBagItem(srcView, srcPos)',
       ') (bool, error) {\n\tstartingBag := player.bagSnapshot()\n\titem, ok := player.takeBagItem(srcView, srcPos)')
# The source slot is vacant after takeBagItem; use it for the displaced item
# even when the swap crosses two different bag views. The old local Slot=0
# survived after addBagItem assigned a slot to its private value copy.
change('\t\t\tdisp.ViewID = srcViewID\n\t\t\tdisp.Slot = 0',
       '\t\t\tdisp.ViewID = srcViewID\n\t\t\tdisp.Slot = srcPos')
change('\t\tplayer.addBagItem(disp)\n\t\tdisplaced = &disp',
       '\t\tdisp.Slot = int32(player.addBagItem(disp))\n\t\tdisplaced = &disp')
# Roll back actor state on PRE-COMMIT frame encoding errors. Never roll back
# RAM after commit if transport delivery fails: the database is definitive.
write_marker = '\tif err := writeFrames(link, frames...); err != nil {'
if function.count(write_marker) != 1:
    raise SystemExit('refuse unknown bag move publish boundary')
prepublish, postpublish = function.split(write_marker, 1)
error_returns = re.compile(r'(?m)^(\t+)return true, err$')
prepublish, count = error_returns.subn(
    lambda match: match.group(1) + 'player.restoreBag(startingBag)\n' + match.group(1) + 'return true, err',
    prepublish,
)
if count != 3:
    raise SystemExit(f'refuse unknown number of pre-commit frame failures: {count}')
function = prepublish + write_marker + postpublish
change('\tframes = append(frames, row)\n\tif err := writeFrames(link, frames...); err != nil {',
       '\tframes = append(frames, row)\n'
       '\t// Reject a stale bag before publishing frames or overwriting an NPC purchase.\n'
       '\tif err := persistBagMutationChecked(bagStore, roleID, startingBag, player.bagSnapshot()); err != nil {\n'
       '\t\tplayer.restoreBag(startingBag)\n'
       '\t\tlog.Printf("%s: reject MOVEITEM after bag changed: %v", remote, err)\n'
       '\t\treturn true, nil\n'
       '\t}\n'
       '\tif err := writeFrames(link, frames...); err != nil {')
change('\t}\n\tpersistBagEquip(bagStore, nil, roleID, player)\n\tif displaced != nil {',
       '\t}\n\tif displaced != nil {')
new_text = prefix + function + end_marker + suffix
if new_text == text:
    raise SystemExit('refuse empty patch')
path.write_text(new_text, encoding='utf-8')
print('BAG_MOVE_SOURCE_LOCK=PASS CROSS_VIEW_SLOT=FIXED STALE_BAG_GUARD=ADDED FRAMES_AFTER_COMMIT=YES')
