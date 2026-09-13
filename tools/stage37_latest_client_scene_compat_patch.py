#!/usr/bin/env python3
# Stage37 latest-client scene readiness compatibility probe, revision 1.
from pathlib import Path
import sys


def main() -> int:
    if len(sys.argv) != 2:
        raise SystemExit("usage: stage37_latest_client_scene_compat_patch.py <buildtree>")
    root = Path(sys.argv[1])
    p = root / "cmd" / "protocol-probe" / "scene_lifecycle.go"
    text = p.read_text(encoding="utf-8")

    old1 = '''\tnewPatrol := make([]worldcore.EntityID, 0, len(delta.Adds))\n\tfor _, addition := range delta.Adds {\n'''
    new1 = '''\t// LATEST-CLIENT COMPATIBILITY A/B ONLY. This is deliberately not part of\n\t// the exact-current-EXE authority reconstruction. Historical successful\n\t// LIVE captures show that the client receives an immediate ServerLocation\n\t// after each newly materialized Actor2 and five delayed location replays\n\t// before it emits the final stage_main ClientReady. The September live\n\t// client reaches the same pre-ready traffic but stalls before that boundary.\n\tlocationReplays := make([]sceneLocationReplay, 0, len(delta.Adds))\n\tnewPatrol := make([]worldcore.EntityID, 0, len(delta.Adds))\n\tfor _, addition := range delta.Adds {\n'''
    if old1 not in text:
        raise SystemExit("compat patch anchor #1 not found")
    text = text.replace(old1, new1, 1)

    old2 = '''\t\tif err := s.conn.WriteFrame(payload); err != nil {\n\t\t\treturn nil, fmt.Errorf("add scene object %d: %w", id, err)\n\t\t}\n\t\tlog.Printf("%s: sent modern scene object id=%d via ServerAddObject", s.remote, id)\n'''
    new2 = '''\t\tif err := s.conn.WriteFrame(payload); err != nil {\n\t\t\treturn nil, fmt.Errorf("add scene object %d: %w", id, err)\n\t\t}\n\t\tif entity.ownerID != 0 {\n\t\t\tlocation := serverLocation(entity.id, entity.ownerID, entity.transform)\n\t\t\tif err := s.conn.WriteFrame(location); err != nil {\n\t\t\t\treturn nil, fmt.Errorf("latest-client compat locate scene object %d: %w", id, err)\n\t\t\t}\n\t\t\tlocationReplays = append(locationReplays, sceneLocationReplay{\n\t\t\t\tid: uint32(id), payload: append([]byte(nil), location...),\n\t\t\t})\n\t\t}\n\t\tlog.Printf("%s: sent modern scene object id=%d via ServerAddObject + latest-client location compatibility", s.remote, id)\n'''
    if old2 not in text:
        raise SystemExit("compat patch anchor #2 not found")
    text = text.replace(old2, new2, 1)

    old3 = '''\tif err := s.viewport.Commit(delta); err != nil {\n\t\treturn nil, err\n\t}\n\treturn newPatrol, nil\n'''
    new3 = '''\tif err := s.viewport.Commit(delta); err != nil {\n\t\treturn nil, err\n\t}\n\tif len(locationReplays) != 0 {\n\t\ts.scheduleLocationReplaysLocked(locationReplays)\n\t\tlog.Printf("%s: latest-client scene compatibility armed location replays objects=%d delays=%v", s.remote, len(locationReplays), actor2LocationReplayDelays)\n\t}\n\treturn newPatrol, nil\n'''
    if old3 not in text:
        raise SystemExit("compat patch anchor #3 not found")
    text = text.replace(old3, new3, 1)

    p.write_text(text, encoding="utf-8")
    print(f"patched {p}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
