#!/usr/bin/env python3
# Stage37 latest-client GM item UI + live scene re-entry compatibility A/B.
#
# Evidence basis:
# - LIVE 2026-09-13 showed GM API give_item exists but the shipped GM HTML has no item-grant section.
# - LIVE scene switch to city05 sent ExitScene/EntryScene (0x0C path), then the latest client emitted 0x0A while
#   runtime.awaitingStableReentryReady was true. Current code discards that 0x0A and waits only for 0x09,
#   leaving the client on a black loading screen.
# - The already-implemented 0x09 stable re-entry path performs the required world.clientReady, deferred
#   player location/vitals, Ready/Activity state transitions, QingGong/shortcut/progress restore and patrol start.
#
# This patch is intentionally narrow:
# 1) expose the already-existing give_item API in the localhost GM HTML;
# 2) when the latest client sends 0x0A during an active target-scene load, run the same stable re-entry
#    completion sequence as the existing 0x09 branch rather than ignoring it.
from pathlib import Path
import sys


def replace_once(text: str, old: str, new: str, label: str) -> str:
    count = text.count(old)
    if count != 1:
        raise SystemExit(f"{label}: expected exactly 1 anchor, found {count}")
    return text.replace(old, new, 1)


def main() -> int:
    if len(sys.argv) != 2:
        raise SystemExit("usage: stage37_latest_client_gm_item_reentry_patch.py <buildtree>")

    root = Path(sys.argv[1])
    probe = root / "cmd" / "protocol-probe"

    gm = probe / "gm_panel.go"
    text = gm.read_text(encoding="utf-8")
    old = '''<section><h2>碎银</h2><input id="silver" type="number" value="99999" min="0"><button onclick="silver()">设置碎银</button></section>\n<section><h2>怒气</h2>'''
    new = '''<section><h2>碎银</h2><input id="silver" type="number" value="99999" min="0"><button onclick="silver()">设置碎银</button></section>\n<section><h2>物品发放</h2><p class="muted">调用服务端现有 <code>give_item</code> API，直接写入当前在线角色背包。先用已知测试物品 <code>Game_item_hp_001</code> 验证。</p><label>ConfigID <input id="item-config" value="Game_item_hp_001"></label><br><label>数量 <input id="item-amount" type="number" value="5" min="1" max="999"></label><button onclick="giveItem()">发到背包</button><span id="command-status" class="muted"></span></section>\n<section><h2>怒气</h2>'''
    text = replace_once(text, old, new, "GM item section")

    old = '''function silver(){command({action:'set_silver',silver:+document.querySelector('#silver').value})}function setSP(){'''
    new = '''function silver(){command({action:'set_silver',silver:+document.querySelector('#silver').value})}function giveItem(){let config=(document.querySelector('#item-config').value||'').trim();let amount=+document.querySelector('#item-amount').value;if(!config)return alert('ConfigID 不能为空');command({action:'give_item',config_id:config,amount:amount,container:'bag'})}function setSP(){'''
    text = replace_once(text, old, new, "GM giveItem JS")
    gm.write_text(text, encoding="utf-8")

    main_go = probe / "main.go"
    text = main_go.read_text(encoding="utf-8")
    old = '''\t\t\tif runtime != nil && runtime.awaitingStableReentryReady {\n\t\t\t\tlog.Printf("%s: ignored early 0x0A while target scene is loading", conn.RemoteAddr())\n\t\t\t\tcontinue\n\t\t\t}\n'''
    new = '''\t\t\tif runtime != nil && runtime.awaitingStableReentryReady {\n\t\t\t\t// Latest-client LIVE evidence: after a GM 0x0C scene re-entry the\n\t\t\t\t// client can emit 0x0A instead of a second explicit 0x09.  Complete\n\t\t\t\t// the same stable re-entry sequence already used by the 0x09 branch.\n\t\t\t\tif err := world.clientReady(); err != nil {\n\t\t\t\t\tlog.Printf("%s: arm latest-client 0x0A re-entry barrier: %v", conn.RemoteAddr(), err)\n\t\t\t\t\treturn\n\t\t\t\t}\n\t\t\t\tif err := sendPlayerLocationAndVitals(link, runtime.player, runtime.activeRole.Location.Position); err != nil {\n\t\t\t\t\tlog.Printf("%s: write latest-client 0x0A deferred player location/vitals: %v", conn.RemoteAddr(), err)\n\t\t\t\t\treturn\n\t\t\t\t}\n\t\t\t\tif err := machine.Ready(); err != nil {\n\t\t\t\t\tlog.Printf("%s: complete latest-client 0x0A stable re-entry ready: %v", conn.RemoteAddr(), err)\n\t\t\t\t\treturn\n\t\t\t\t}\n\t\t\t\tif err := world.clientActivity(); err != nil {\n\t\t\t\t\tlog.Printf("%s: activate latest-client 0x0A stable re-entry objects: %v", conn.RemoteAddr(), err)\n\t\t\t\t\treturn\n\t\t\t\t}\n\t\t\t\tif err := machine.Activity(); err != nil {\n\t\t\t\t\tlog.Printf("%s: complete latest-client 0x0A stable re-entry activity: %v", conn.RemoteAddr(), err)\n\t\t\t\t\treturn\n\t\t\t\t}\n\t\t\t\tif player != nil {\n\t\t\t\t\tlevels := player.qingGongLevelsSnapshot()\n\t\t\t\t\tif grantErr := grantRoleQingGong(link, qinggongStore, selectedRoleID(selected), player.name, levels); grantErr != nil {\n\t\t\t\t\t\tlog.Printf("%s: restore qinggong after latest-client 0x0A re-entry: %v", conn.RemoteAddr(), grantErr)\n\t\t\t\t\t\treturn\n\t\t\t\t\t}\n\t\t\t\t\tif shortcutErr := grantShortcutRows(link, player); shortcutErr != nil {\n\t\t\t\t\t\tlog.Printf("%s: restore shortcuts after latest-client 0x0A re-entry: %v", conn.RemoteAddr(), shortcutErr)\n\t\t\t\t\t\treturn\n\t\t\t\t\t}\n\t\t\t\t\tqingGongGranted = true\n\t\t\t\t\tif progressErr := grantStarterProgress(); progressErr != nil {\n\t\t\t\t\t\tlog.Printf("%s: restore inner-power/attributes after latest-client 0x0A re-entry: %v", conn.RemoteAddr(), progressErr)\n\t\t\t\t\t\treturn\n\t\t\t\t\t}\n\t\t\t\t\tif switchErr := enableSkillActionSwitch(link); switchErr != nil {\n\t\t\t\t\t\tlog.Printf("%s: restore native skill-action switch after latest-client 0x0A re-entry: %v", conn.RemoteAddr(), switchErr)\n\t\t\t\t\t\treturn\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t\tif npcPatrolCount > 0 {\n\t\t\t\t\tworld.startPatrol(npcPatrolCount)\n\t\t\t\t}\n\t\t\t\truntime.awaitingStableReentryReady = false\n\t\t\t\tlog.Printf("%s: promoted target-scene 0x0A to latest-client stable re-entry ready", conn.RemoteAddr())\n\t\t\t\tcontinue\n\t\t\t}\n'''
    text = replace_once(text, old, new, "latest-client 0x0A re-entry promotion")
    main_go.write_text(text, encoding="utf-8")

    test = probe / "latest_client_gm_item_reentry_compat_test.go"
    test.write_text(r'''package main

import (
    "strings"
    "testing"
)

func TestLatestClientGMPanelExposesGiveItem(t *testing.T) {
    for _, token := range []string{"物品发放", "Game_item_hp_001", "action:'give_item'", "config_id:config", "container:'bag'"} {
        if !strings.Contains(gmPanelHTML, token) {
            t.Fatalf("GM panel missing %q", token)
        }
    }
}
''', encoding="utf-8")

    print(f"patched {gm}")
    print(f"patched {main_go}")
    print(f"wrote {test}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
