#!/usr/bin/env python3
# Stage37 latest-client follow-up A/B after LIVE 2026-09-13 21:30~21:31.
#
# Evidence from that LIVE:
# - Client trace logged repeated GameReceiver::ServerViewAdd property error at the exact post-ready bag/view replay boundary.
# - The prior audit showed retail shop rows were visibly accepted using only ConfigID=105(string) + Amount=106(int32).
# - The test bag rows emitted 7,105,106,110; therefore this A/B strips bag rows to the two LIVE-proven shop-visible fields only.
# - Initial scene entry succeeds by promoting an observed 0x0A to compatibility ClientReady, then later receives explicit 0x09.
# - Target re-entry emitted one immediate 0x0A before the client trace reached ExecuteReceiveEntryScene, then another 0x0A ~27s later.
#   The prior immediate 0x0A promotion was therefore mistimed. This probe ignores only the first/too-early 0x0A and promotes
#   a later 0x0A once at least 2 seconds have elapsed since the target re-entry was armed.
#
# This is a compatibility A/B, not exact-authority behavior.
from pathlib import Path
import sys


def replace_once(text, old, new, label):
    count = text.count(old)
    if count != 1:
        raise SystemExit(f"{label}: expected exactly 1 anchor, found {count}")
    return text.replace(old, new, 1)


def main():
    if len(sys.argv) != 2:
        raise SystemExit("usage: stage37_latest_client_bag_105106_reentry_late0a_patch.py <buildtree>")
    root = Path(sys.argv[1])
    probe = root / "cmd" / "protocol-probe"

    # Bag: keep only the two low fields already proven visible in retail shop rows.
    helper = probe / "latest_client_bag_view_ordinal_compat.go"
    text = helper.read_text(encoding="utf-8")
    old = '''\tif haveMaxAmount {\n\t\tresult = append(result, viewInt(110, maxAmount))\n\t}\n\treturn result\n}\n'''
    new = '''\tif haveMaxAmount {\n\t\tresult = append(result, viewInt(110, maxAmount))\n\t}\n\t// 2026-09-13 LIVE A/B: FxNet2 reports ServerViewAdd property error for the\n\t// four-field bag row.  Only 105/106 are independently LIVE-proven visible\n\t// in retail shop rows, so publish exactly those two in this probe.\n\tfiltered := result[:0]\n\tfor _, property := range result {\n\t\tif property.index == 105 || property.index == 106 {\n\t\t\tfiltered = append(filtered, property)\n\t\t}\n\t}\n\treturn filtered\n}\n'''
    text = replace_once(text, old, new, "bag 105/106 filter")
    helper.write_text(text, encoding="utf-8")

    # Re-entry: record when the target re-entry is armed.
    trans = probe / "scene_transition.go"
    text = trans.read_text(encoding="utf-8")
    old = '''\tawaitingStableReentryReady bool\n}\n'''
    new = '''\tawaitingStableReentryReady bool\n\treentryStartedAt           time.Time\n}\n'''
    text = replace_once(text, old, new, "sceneRuntime reentryStartedAt")
    old = '''\tr.npcCatalog = catalog\n\tr.awaitingStableReentryReady = true\n\treturn nil\n}\n'''
    new = '''\tr.npcCatalog = catalog\n\tr.awaitingStableReentryReady = true\n\tr.reentryStartedAt = time.Now()\n\treturn nil\n}\n'''
    text = replace_once(text, old, new, "arm reentry timestamp")
    trans.write_text(text, encoding="utf-8")

    # Promote only a later re-entry 0x0A, not the immediate pre-ExecuteReceiveEntryScene one.
    main_go = probe / "main.go"
    text = main_go.read_text(encoding="utf-8")
    old = '''\t\t\tif runtime != nil && runtime.awaitingStableReentryReady {\n\t\t\t\tlog.Printf("%s: ignored early 0x0A while target scene is loading", conn.RemoteAddr())\n\t\t\t\tcontinue\n\t\t\t}\n'''
    new = '''\t\t\tif runtime != nil && runtime.awaitingStableReentryReady {\n\t\t\t\telapsed := time.Since(runtime.reentryStartedAt)\n\t\t\t\tif elapsed < 2*time.Second {\n\t\t\t\t\tlog.Printf("%s: ignored too-early target-scene 0x0A elapsed=%s", conn.RemoteAddr(), elapsed)\n\t\t\t\t\tcontinue\n\t\t\t\t}\n\t\t\t\t// LIVE ordering: the first target 0x0A preceded client ExecuteReceiveEntryScene,\n\t\t\t\t// while a later 0x0A arrived after the client had been in target loading for\n\t\t\t\t// tens of seconds.  Complete the already-recovered stable re-entry sequence\n\t\t\t\t// only on that later boundary.\n\t\t\t\tif err := world.clientReady(); err != nil {\n\t\t\t\t\tlog.Printf("%s: arm late-0x0A re-entry barrier: %v", conn.RemoteAddr(), err)\n\t\t\t\t\treturn\n\t\t\t\t}\n\t\t\t\tif err := sendPlayerLocationAndVitals(link, runtime.player, runtime.activeRole.Location.Position); err != nil {\n\t\t\t\t\tlog.Printf("%s: write late-0x0A deferred player location/vitals: %v", conn.RemoteAddr(), err)\n\t\t\t\t\treturn\n\t\t\t\t}\n\t\t\t\tif err := machine.Ready(); err != nil {\n\t\t\t\t\tlog.Printf("%s: complete late-0x0A stable re-entry ready: %v", conn.RemoteAddr(), err)\n\t\t\t\t\treturn\n\t\t\t\t}\n\t\t\t\tif err := world.clientActivity(); err != nil {\n\t\t\t\t\tlog.Printf("%s: activate late-0x0A stable re-entry objects: %v", conn.RemoteAddr(), err)\n\t\t\t\t\treturn\n\t\t\t\t}\n\t\t\t\tif err := machine.Activity(); err != nil {\n\t\t\t\t\tlog.Printf("%s: complete late-0x0A stable re-entry activity: %v", conn.RemoteAddr(), err)\n\t\t\t\t\treturn\n\t\t\t\t}\n\t\t\t\tif player != nil {\n\t\t\t\t\tlevels := player.qingGongLevelsSnapshot()\n\t\t\t\t\tif grantErr := grantRoleQingGong(link, qinggongStore, selectedRoleID(selected), player.name, levels); grantErr != nil {\n\t\t\t\t\t\tlog.Printf("%s: restore qinggong after late-0x0A re-entry: %v", conn.RemoteAddr(), grantErr)\n\t\t\t\t\t\treturn\n\t\t\t\t\t}\n\t\t\t\t\tif shortcutErr := grantShortcutRows(link, player); shortcutErr != nil {\n\t\t\t\t\t\tlog.Printf("%s: restore shortcuts after late-0x0A re-entry: %v", conn.RemoteAddr(), shortcutErr)\n\t\t\t\t\t\treturn\n\t\t\t\t\t}\n\t\t\t\t\tqingGongGranted = true\n\t\t\t\t\tif progressErr := grantStarterProgress(); progressErr != nil {\n\t\t\t\t\t\tlog.Printf("%s: restore inner-power/attributes after late-0x0A re-entry: %v", conn.RemoteAddr(), progressErr)\n\t\t\t\t\t\treturn\n\t\t\t\t\t}\n\t\t\t\t\tif switchErr := enableSkillActionSwitch(link); switchErr != nil {\n\t\t\t\t\t\tlog.Printf("%s: restore native skill-action switch after late-0x0A re-entry: %v", conn.RemoteAddr(), switchErr)\n\t\t\t\t\t\treturn\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t\tif npcPatrolCount > 0 {\n\t\t\t\t\tworld.startPatrol(npcPatrolCount)\n\t\t\t\t}\n\t\t\t\truntime.awaitingStableReentryReady = false\n\t\t\t\truntime.reentryStartedAt = time.Time{}\n\t\t\t\tlog.Printf("%s: promoted later target-scene 0x0A to stable re-entry ready elapsed=%s", conn.RemoteAddr(), elapsed)\n\t\t\t\tcontinue\n\t\t\t}\n'''
    text = replace_once(text, old, new, "late target 0x0A promotion")
    main_go.write_text(text, encoding="utf-8")

    test = probe / "latest_client_bag_105106_reentry_late0a_test.go"
    test.write_text(r'''package main

import "testing"

func TestLatestClientBagWirePropertiesFollowupUsesOnly105And106(t *testing.T) {
    got := latestClientBagWireProperties([]serverViewProperty{
        viewString(7, "Game_item_hp_001"),
        viewString(105, "Game_item_hp_001"),
        viewInt(106, 5),
        viewInt(110, 30),
    })
    if len(got) != 2 || got[0].index != 105 || got[1].index != 106 {
        t.Fatalf("followup bag wire indexes=%v", []uint16{got[0].index, got[1].index})
    }
}
''', encoding="utf-8")

    print(f"patched {helper}")
    print(f"patched {trans}")
    print(f"patched {main_go}")
    print(f"wrote {test}")

if __name__ == '__main__':
    main()
