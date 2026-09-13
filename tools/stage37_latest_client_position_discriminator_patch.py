#!/usr/bin/env python3
# Stage37 latest-client player-position discriminator A/B probe.
#
# DIAGNOSTIC ONLY. This does not change the exact-current-EXE authority default
# role location or persistence. It changes only the one latest-client-only
# post-explicit-ClientReady player ServerLocation replay added by the prior A/B.
# The alternate coordinate is not guessed: it is the exact recovered current-EXE
# defaultRoleLocation for book8, which uses the same born02_ERenGU / born02 scene.
from pathlib import Path
import sys


def main() -> int:
    if len(sys.argv) != 2:
        raise SystemExit("usage: stage37_latest_client_position_discriminator_patch.py <buildtree>")
    root = Path(sys.argv[1])
    p = root / "cmd" / "protocol-probe" / "main.go"
    text = p.read_text(encoding="utf-8")

    old = '''\t\t\tif runtime != nil && !runtime.awaitingStableReentryReady && !explicitSceneReady {\n\t\t\t\tposition := runtime.activeRole.Location.Position\n\t\t\t\tif err := link.WriteFrame(serverLocation(playerObjectID, playerOwnerID, worldTransform(position))); err != nil {\n\t\t\t\t\tlog.Printf("%s: write latest-client initial player location replay: %v", conn.RemoteAddr(), err)\n\t\t\t\t\treturn\n\t\t\t\t}\n\t\t\t\tlog.Printf("%s: latest-client player location compatibility replay after explicit ClientReady x=%.3f y=%.3f z=%.3f orient=%.3f", conn.RemoteAddr(), position.X, position.Y, position.Z, position.Orient)\n\t\t\t}\n'''
    new = '''\t\t\tif runtime != nil && !runtime.awaitingStableReentryReady && !explicitSceneReady {\n\t\t\t\tposition := runtime.activeRole.Location.Position\n\t\t\t\t// LATEST-CLIENT POSITION DISCRIMINATOR A/B ONLY. Do not persist or\n\t\t\t\t// alter runtime.activeRole. The exact recovered current-EXE book8\n\t\t\t\t// default uses the same born02 scene at this distinct location. If\n\t\t\t\t// the latest client consumes player 0x1F here, camera/minimap position\n\t\t\t\t// must visibly change. If it does not, raw spawn-coordinate choice is\n\t\t\t\t// not the immediate blocker and main-player materialization/LocatePlayer\n\t\t\t\t// handling remains the higher-priority path.\n\t\t\t\tif runtime.activeRole.Location.Scene.Resource == "born02" {\n\t\t\t\t\tposition.X = 905.069\n\t\t\t\t\tposition.Y = 10.810\n\t\t\t\t\tposition.Z = 196.980\n\t\t\t\t\tposition.Orient = 1.610\n\t\t\t\t}\n\t\t\t\tif err := link.WriteFrame(serverLocation(playerObjectID, playerOwnerID, worldTransform(position))); err != nil {\n\t\t\t\t\tlog.Printf("%s: write latest-client position discriminator replay: %v", conn.RemoteAddr(), err)\n\t\t\t\t\treturn\n\t\t\t\t}\n\t\t\t\tlog.Printf("%s: latest-client POSITION-DISCRIMINATOR replay after explicit ClientReady x=%.3f y=%.3f z=%.3f orient=%.3f persisted_x=%.3f persisted_y=%.3f persisted_z=%.3f persisted_orient=%.3f", conn.RemoteAddr(), position.X, position.Y, position.Z, position.Orient, runtime.activeRole.Location.Position.X, runtime.activeRole.Location.Position.Y, runtime.activeRole.Location.Position.Z, runtime.activeRole.Location.Position.Orient)\n\t\t\t}\n'''
    count = text.count(old)
    if count != 1:
        raise SystemExit(f"position discriminator anchor: expected exactly 1, found {count}")
    text = text.replace(old, new, 1)
    p.write_text(text, encoding="utf-8")
    print(f"patched {p}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
