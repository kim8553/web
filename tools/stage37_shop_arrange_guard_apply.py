#!/usr/bin/env python3
"""Apply only the audited ARANGEITEM persistence-order fix to the restored overlay.

The GitHub Actions workflow runs this patch and tests it before committing the
small resulting overlay diff. It must not touch any shop purchase or packet IDs.
"""
from pathlib import Path
import subprocess

path = Path('server/cmd/protocol-probe/zz_recovered_overlay.go')
expected_sha = '4cc38f4c51a293ace27ab1344eb1d465578ce003'
actual_sha = subprocess.check_output(['git', 'hash-object', str(path)], text=True).strip()
if actual_sha != expected_sha:
    raise SystemExit(f'refuse overlay drift: {actual_sha} != {expected_sha}')
text = path.read_text()
start = text.index('func handleArrangeItemCustom(')
end = text.index('\nfunc handleDeleteItemCustom(', start)
block = text[start:end]

def one(old: str, new: str) -> None:
    global block
    if block.count(old) != 1:
        raise SystemExit(f'refuse unexpected arrange function match count {block.count(old)} for {old!r}')
    block = block.replace(old, new, 1)

one(
    '\tarranged, changed := player.arrangeBagView(uint16(srcView))',
    '\tstartingBag := player.bagSnapshot()\n\tarranged, changed := player.arrangeBagView(uint16(srcView))',
)
one(
    '\t\tif err != nil {\n\t\t\treturn true, err\n\t\t}\n\t\tframes = append(frames, frame)',
    '\t\tif err != nil {\n\t\t\tplayer.restoreBag(startingBag)\n\t\t\treturn true, err\n\t\t}\n\t\tframes = append(frames, frame)',
)
one(
    '''\tif err := writeFrames(link, frames...); err != nil {
\t\treturn true, err
\t}
\tif bagStore != nil {
\t\tif err := bagStore.Save(roleID, player.bagSnapshot()); err != nil {
\t\t\tlog.Printf("persist bag after arrange: %v", err)
\t\t}
\t}''',
    '''\t// A stale bag cannot be published to the client or overwrite a newer purchase.
\t// Save before sending the pre-encoded existing frames; a failed save restores
\t// the actor's pre-arrange snapshot and leaves the client view untouched.
\tif err := persistBagMutationChecked(bagStore, roleID, startingBag, player.bagSnapshot()); err != nil {
\t\tplayer.restoreBag(startingBag)
\t\tlog.Printf("%s: reject ARANGEITEM after bag changed: %v", remote, err)
\t\treturn true, nil
\t}
\tif err := writeFrames(link, frames...); err != nil {
\t\treturn true, err
\t}''',
)
path.write_text(text[:start] + block + text[end:])
print('ARRANGE_SCOPE_PATCH=PASS OLD_BLOB=' + expected_sha)
