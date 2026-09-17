#!/usr/bin/env python3
"""Apply one pinned, auditable normal-shop arithmetic guard to current Go source.

Never touches GM grant, exchange (0x4F), item semantics or private client data.
Intentionally refuses to edit an unknown source revision.
"""
import hashlib
from pathlib import Path

ROOT = Path(__file__).resolve().parents[2]
TARGET = ROOT / 'server/cmd/protocol-probe/zz_recovered_overlay.go'
EXPECTED_GIT_BLOB = '1700ac2889273ca806494916ddbc6efcf6145f29'
ANCHOR = b'\ttotal := int64(item.price) * int64(amount)\n\tswitch item.priceMode {'
REPLACEMENT = b'''\ttotal, safeCost := normalShopSafeTotal(item.price, amount)
\tif !safeCost {
\t\tlog.Printf("%s: shop buy %s item %s blocked: invalid price=%d quantity=%d or total exceeds actor int32 currency range", remote, shopID, item.configID, item.price, amount)
\t\treturn true, nil
\t}
\tswitch item.priceMode {'''


def git_blob_sha(data: bytes) -> str:
    return hashlib.sha1(b'blob ' + str(len(data)).encode('ascii') + b'\0' + data).hexdigest()


def main() -> None:
    original = TARGET.read_bytes()
    actual = git_blob_sha(original)
    if actual != EXPECTED_GIT_BLOB:
        raise SystemExit(f'FAIL CLOSED: production blob changed: {actual}; expected {EXPECTED_GIT_BLOB}')
    if original.count(ANCHOR) != 1:
        raise SystemExit(f'FAIL CLOSED: ambiguous or missing purchase anchor: {original.count(ANCHOR)}')
    revised = original.replace(ANCHOR, REPLACEMENT, 1)
    if revised.count(REPLACEMENT) != 1 or revised.count(ANCHOR) != 0:
        raise SystemExit('FAIL CLOSED: patch postconditions failed')
    TARGET.write_bytes(revised)
    print('PATCHED current normal shop price * amount guard in one pinned handler')
    print('OLD_BLOB', actual, 'NEW_BLOB', git_blob_sha(revised))


if __name__ == '__main__':
    main()
