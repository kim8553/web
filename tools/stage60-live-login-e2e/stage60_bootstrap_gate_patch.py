#!/usr/bin/env python3
from pathlib import Path
import sys


def main() -> int:
    if len(sys.argv) != 2:
        raise SystemExit("usage: stage60_bootstrap_gate_patch.py <repo-root>")
    root = Path(sys.argv[1])
    path = root / "server" / "cmd" / "protocol-probe" / "main.go"
    text = path.read_text(encoding="utf-8")
    old = '''\t\t\tif err := sendPlayerSpawn(link, player, activeRole.Location.Position, visual); err != nil {\n\t\t\t\tlog.Printf("%s: write player spawn chain: %v", conn.RemoteAddr(), err)\n\t\t\t\treturn\n\t\t\t}\n'''
    new = '''\t\t\tif gateLimit, gateEnabled, gateErr := stage60BootstrapGateFromEnv(); gateEnabled {\n\t\t\t\tif gateErr != nil {\n\t\t\t\t\tlog.Printf("%s: STAGE60_BOOTSTRAP_GATE_INVALID: %v", conn.RemoteAddr(), gateErr)\n\t\t\t\t\treturn\n\t\t\t\t}\n\t\t\t\tattempted, sent, gateRunErr := stage60SendPlayerSpawnGated(link, player, activeRole.Location.Position, visual, gateLimit)\n\t\t\t\tif gateRunErr != nil {\n\t\t\t\t\tlog.Printf("%s: STAGE60_BOOTSTRAP_GATE_WRITE_ERROR gate=%d attempted=%d sent=%d err=%v", conn.RemoteAddr(), gateLimit, attempted, sent, gateRunErr)\n\t\t\t\t\treturn\n\t\t\t\t}\n\t\t\t\tlog.Printf("%s: STAGE60_BOOTSTRAP_GATE_HOLD gate=%d attempted=%d sent=%d; post-spawn output intentionally withheld for crash isolation", conn.RemoteAddr(), gateLimit, attempted, sent)\n\t\t\t\tcontinue\n\t\t\t}\n\t\t\tif err := sendPlayerSpawn(link, player, activeRole.Location.Position, visual); err != nil {\n\t\t\t\tlog.Printf("%s: write player spawn chain: %v", conn.RemoteAddr(), err)\n\t\t\t\treturn\n\t\t\t}\n'''
    count = text.count(old)
    if count != 1:
        raise SystemExit(f"choose-role sendPlayerSpawn anchor count={count}, want 1")
    path.write_text(text.replace(old, new, 1), encoding="utf-8")
    print(f"patched {path}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
