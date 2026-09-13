#!/usr/bin/env python3
# Stage37 latest-client legacy born02 spawn compatibility remap.
#
# Evidence basis:
# - LIVE 2026-09-13 position discriminator proved the September client applies
#   player ServerLocation 0x1F and visibly relocates from the legacy persisted
#   born02 position 693.908,24.694,404.350,3.611 to the exact recovered
#   current-EXE book8 born02 position 905.069,10.810,196.980,1.610.
# - This compatibility overlay does NOT claim the book8 coordinate is the
#   authoritative latest book5 spawn. It remaps only the exact legacy position
#   that was proven to land inside invalid terrain on the latest client.
# - Persistence is intentionally unchanged in this build.
from pathlib import Path
import sys


def replace_once(text: str, old: str, new: str, label: str) -> str:
    count = text.count(old)
    if count != 1:
        raise SystemExit(f"{label}: expected exactly 1 anchor, found {count}")
    return text.replace(old, new, 1)


def main() -> int:
    if len(sys.argv) != 2:
        raise SystemExit("usage: stage37_latest_client_legacy_born02_remap_patch.py <buildtree>")

    root = Path(sys.argv[1])
    p = root / "cmd" / "protocol-probe" / "main.go"
    text = p.read_text(encoding="utf-8")

    old_spawn = '''\t\t\tlog.Printf("%s: sent PlayerEntry opcode=0x0B object=%#x scene=%s resource=%s", conn.RemoteAddr(), playerObjectID, activeRole.Location.Scene.Config, activeRole.Location.Scene.Resource)\n\t\t\tlog.Printf("%s: latest-client player spawn diagnostic scene=%s resource=%s x=%.3f y=%.3f z=%.3f orient=%.3f", conn.RemoteAddr(), activeRole.Location.Scene.Config, activeRole.Location.Scene.Resource, activeRole.Location.Position.X, activeRole.Location.Position.Y, activeRole.Location.Position.Z, activeRole.Location.Position.Orient)\n\t\t\tif err := sendPlayerSpawn(link, player, activeRole.Location.Position, visual); err != nil {\n'''
    new_spawn = '''\t\t\tlog.Printf("%s: sent PlayerEntry opcode=0x0B object=%#x scene=%s resource=%s", conn.RemoteAddr(), playerObjectID, activeRole.Location.Scene.Config, activeRole.Location.Scene.Resource)\n\t\t\tlegacyBorn02 := activeRole.Location.Scene.Resource == "born02" &&\n\t\t\t\tactiveRole.Location.Position.X > 693.907 && activeRole.Location.Position.X < 693.909 &&\n\t\t\t\tactiveRole.Location.Position.Y > 24.693 && activeRole.Location.Position.Y < 24.695 &&\n\t\t\t\tactiveRole.Location.Position.Z > 404.349 && activeRole.Location.Position.Z < 404.351\n\t\t\tif legacyBorn02 {\n\t\t\t\told := activeRole.Location.Position\n\t\t\t\tactiveRole.Location.Position.X = 905.069\n\t\t\t\tactiveRole.Location.Position.Y = 10.810\n\t\t\t\tactiveRole.Location.Position.Z = 196.980\n\t\t\t\tactiveRole.Location.Position.Orient = 1.610\n\t\t\t\tlog.Printf("%s: latest-client LEGACY-BORN02-REMAP initial spawn old=(%.3f,%.3f,%.3f,%.3f) new=(%.3f,%.3f,%.3f,%.3f) persistence=unchanged", conn.RemoteAddr(), old.X, old.Y, old.Z, old.Orient, activeRole.Location.Position.X, activeRole.Location.Position.Y, activeRole.Location.Position.Z, activeRole.Location.Position.Orient)\n\t\t\t}\n\t\t\tlog.Printf("%s: latest-client player spawn diagnostic scene=%s resource=%s x=%.3f y=%.3f z=%.3f orient=%.3f", conn.RemoteAddr(), activeRole.Location.Scene.Config, activeRole.Location.Scene.Resource, activeRole.Location.Position.X, activeRole.Location.Position.Y, activeRole.Location.Position.Z, activeRole.Location.Position.Orient)\n\t\t\tif err := sendPlayerSpawn(link, player, activeRole.Location.Position, visual); err != nil {\n'''
    text = replace_once(text, old_spawn, new_spawn, "initial legacy born02 remap")

    old_ready = '''\t\t\tif runtime != nil && !runtime.awaitingStableReentryReady && !explicitSceneReady {\n\t\t\t\tposition := runtime.activeRole.Location.Position\n\t\t\t\tif err := link.WriteFrame(serverLocation(playerObjectID, playerOwnerID, worldTransform(position))); err != nil {\n'''
    new_ready = '''\t\t\tif runtime != nil && !runtime.awaitingStableReentryReady && !explicitSceneReady {\n\t\t\t\tposition := runtime.activeRole.Location.Position\n\t\t\t\tif runtime.activeRole.Location.Scene.Resource == "born02" &&\n\t\t\t\t\tposition.X > 693.907 && position.X < 693.909 &&\n\t\t\t\t\tposition.Y > 24.693 && position.Y < 24.695 &&\n\t\t\t\t\tposition.Z > 404.349 && position.Z < 404.351 {\n\t\t\t\t\tposition.X = 905.069\n\t\t\t\t\tposition.Y = 10.810\n\t\t\t\t\tposition.Z = 196.980\n\t\t\t\t\tposition.Orient = 1.610\n\t\t\t\t\tlog.Printf("%s: latest-client LEGACY-BORN02-REMAP ClientReady replay new=(%.3f,%.3f,%.3f,%.3f) runtime-persistence=unchanged", conn.RemoteAddr(), position.X, position.Y, position.Z, position.Orient)\n\t\t\t\t}\n\t\t\t\tif err := link.WriteFrame(serverLocation(playerObjectID, playerOwnerID, worldTransform(position))); err != nil {\n'''
    text = replace_once(text, old_ready, new_ready, "clientready legacy born02 remap")

    p.write_text(text, encoding="utf-8")
    print(f"patched {p}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
