#!/usr/bin/env python3
# Stage37 latest-client initial-player location compatibility A/B probe.
#
# IMPORTANT: This file is NOT part of the exact-current-EXE authority
# reconstruction. It is applied only after the authority patches and the
# existing latest-client Actor2/NPC location compatibility delta.
from pathlib import Path
import sys


def replace_once(text: str, old: str, new: str, label: str) -> str:
    count = text.count(old)
    if count != 1:
        raise SystemExit(f"{label}: expected exactly 1 anchor, found {count}")
    return text.replace(old, new, 1)


def main() -> int:
    if len(sys.argv) != 2:
        raise SystemExit("usage: stage37_latest_client_player_location_compat_patch.py <buildtree>")

    root = Path(sys.argv[1])
    p = root / "cmd" / "protocol-probe" / "main.go"
    text = p.read_text(encoding="utf-8")

    old_spawn = '''\t\t\tlog.Printf("%s: sent PlayerEntry opcode=0x0B object=%#x scene=%s resource=%s", conn.RemoteAddr(), playerObjectID, activeRole.Location.Scene.Config, activeRole.Location.Scene.Resource)\n\t\t\tif err := sendPlayerSpawn(link, player, activeRole.Location.Position, visual); err != nil {\n'''
    new_spawn = '''\t\t\tlog.Printf("%s: sent PlayerEntry opcode=0x0B object=%#x scene=%s resource=%s", conn.RemoteAddr(), playerObjectID, activeRole.Location.Scene.Config, activeRole.Location.Scene.Resource)\n\t\t\tlog.Printf("%s: latest-client player spawn diagnostic scene=%s resource=%s x=%.3f y=%.3f z=%.3f orient=%.3f", conn.RemoteAddr(), activeRole.Location.Scene.Config, activeRole.Location.Scene.Resource, activeRole.Location.Position.X, activeRole.Location.Position.Y, activeRole.Location.Position.Z, activeRole.Location.Position.Orient)\n\t\t\tif err := sendPlayerSpawn(link, player, activeRole.Location.Position, visual); err != nil {\n'''
    text = replace_once(text, old_spawn, new_spawn, "initial spawn diagnostic")

    old_ready = '''\t\t} else if plain[0] == 0x09 {\n\t\t\tlog.Printf("%s: C2S 0x09 len=%d hex=% X", conn.RemoteAddr(), len(plain), plain)\n\t\t\tif runtime != nil && runtime.awaitingStableReentryReady {\n'''
    new_ready = '''\t\t} else if plain[0] == 0x09 {\n\t\t\tlog.Printf("%s: C2S 0x09 len=%d hex=% X", conn.RemoteAddr(), len(plain), plain)\n\t\t\t// LATEST-CLIENT COMPATIBILITY A/B ONLY. The exact-current-EXE initial\n\t\t\t// entry already sent player 0x1F as part of sendPlayerSpawn. The exact\n\t\t\t// stable re-entry path, however, sends player 0x1F again when the target\n\t\t\t// explicit ClientReady arrives. The September client now reaches this\n\t\t\t// boundary after Actor2/NPC location replay but renders the player at an\n\t\t\t// invalid terrain position. Replay only the authoritative player 0x1F\n\t\t\t// once on the first explicit initial-entry 0x09 so LIVE A/B can decide\n\t\t\t// whether initial player-location delivery has the same timing contract.\n\t\t\tif runtime != nil && !runtime.awaitingStableReentryReady && !explicitSceneReady {\n\t\t\t\tposition := runtime.activeRole.Location.Position\n\t\t\t\tif err := link.WriteFrame(serverLocation(playerObjectID, playerOwnerID, worldTransform(position))); err != nil {\n\t\t\t\t\tlog.Printf("%s: write latest-client initial player location replay: %v", conn.RemoteAddr(), err)\n\t\t\t\t\treturn\n\t\t\t\t}\n\t\t\t\tlog.Printf("%s: latest-client player location compatibility replay after explicit ClientReady x=%.3f y=%.3f z=%.3f orient=%.3f", conn.RemoteAddr(), position.X, position.Y, position.Z, position.Orient)\n\t\t\t}\n\t\t\tif runtime != nil && runtime.awaitingStableReentryReady {\n'''
    text = replace_once(text, old_ready, new_ready, "explicit initial ClientReady location replay")

    old_motion = '''\t\t\tif x, y, z, orient, ok := parseMotionPosition(plain); ok {\n\t\t\t\tif selected == nil {\n'''
    new_motion = '''\t\t\tif x, y, z, orient, ok := parseMotionPosition(plain); ok {\n\t\t\t\tlog.Printf("%s: latest-client decoded C2S position x=%.3f y=%.3f z=%.3f orient=%.3f opcode=0x%02X len=%d", conn.RemoteAddr(), x, y, z, orient, plain[0], len(plain))\n\t\t\t\tif selected == nil {\n'''
    text = replace_once(text, old_motion, new_motion, "decoded client position diagnostic")

    p.write_text(text, encoding="utf-8")
    print(f"patched {p}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
