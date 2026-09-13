#!/usr/bin/env python3
# Latest-client A/B after 2026-09-13 21:11 LIVE result.
#
# Evidence boundary:
# - give_item reached the server, was persisted, and a ViewAdd was emitted, but the client bag stayed empty.
# - the current grant path re-CREATEs already-existing starter views without deleting the tool bag first.
# - historical protocol evidence used DELETE_VIEW -> CREATE_VIEW -> authoritative replay for a live bag refresh.
# - latest target-scene switch sent ExitScene/EntryScene + player AddObject, then the client never emitted a
#   target ClientReady (0x09/0x0A). Initial entry, which works, sends player location/vitals before ClientReady;
#   latest-client NPC loading also proved an immediate ServerLocation after AddObject can be required.
#
# This probe is intentionally narrow:
# 1) expose existing give_item UI;
# 2) for a normal GM bag-item grant, delete only tool-bag View 2 immediately before the existing authoritative
#    grantBagItems replay (the replay itself recreates View 2 with current ordinal-safe properties);
# 3) after target EntryScene + player AddObject, send player 0x1F/vital immediately and still wait for the real
#    target 0x09. No 0x0A promotion is included in this probe.
from pathlib import Path
import sys


def replace_once(text, old, new, label):
    count = text.count(old)
    if count != 1:
        raise SystemExit(f"{label}: expected exactly 1 anchor, found {count}")
    return text.replace(old, new, 1)


def main():
    if len(sys.argv) != 2:
        raise SystemExit("usage: stage37_latest_client_bag_refresh_reentry_sync_patch.py <buildtree>")
    root = Path(sys.argv[1])
    probe = root / "cmd" / "protocol-probe"

    gm = probe / "gm_panel.go"
    text = gm.read_text(encoding="utf-8")
    old = '''<section><h2>碎银</h2><input id="silver" type="number" value="99999" min="0"><button onclick="silver()">设置碎银</button></section>\n<section><h2>怒气</h2>'''
    new = '''<section><h2>碎银</h2><input id="silver" type="number" value="99999" min="0"><button onclick="silver()">设置碎银</button></section>\n<section><h2>物品发放</h2><p class="muted">调用服务端现有 <code>give_item</code> API。测试物品：<code>Game_item_hp_001</code></p><label>ConfigID <input id="item-config" value="Game_item_hp_001"></label><br><label>数量 <input id="item-amount" type="number" value="5" min="1" max="999"></label><button onclick="giveItem()">发到背包</button><span id="command-status" class="muted"></span></section>\n<section><h2>怒气</h2>'''
    text = replace_once(text, old, new, "GM item section")
    old = '''function silver(){command({action:'set_silver',silver:+document.querySelector('#silver').value})}function setSP(){'''
    new = '''function silver(){command({action:'set_silver',silver:+document.querySelector('#silver').value})}function giveItem(){let config=(document.querySelector('#item-config').value||'').trim();let amount=+document.querySelector('#item-amount').value;if(!config)return alert('ConfigID 不能为空');command({action:'give_item',config_id:config,amount:amount,container:'bag'})}function setSP(){'''
    text = replace_once(text, old, new, "GM giveItem JS")
    gm.write_text(text, encoding="utf-8")

    main_go = probe / "main.go"
    text = main_go.read_text(encoding="utf-8")
    old = '''\t\t\tplayer.addBagItem(bagItem)\n\t\t\tif saveErr := bagStore.Save(selectedRoleID(selected), player.bagSnapshot()); saveErr != nil {\n\t\t\t\tlog.Printf("%s: persist bag after give: %v", conn.RemoteAddr(), saveErr)\n\t\t\t}\n\t\t\tif err := grantBagItems(link, player, itemCatalog, equipCatalog, conn.RemoteAddr().String()); err != nil {\n'''
    new = '''\t\t\tplayer.addBagItem(bagItem)\n\t\t\tif saveErr := bagStore.Save(selectedRoleID(selected), player.bagSnapshot()); saveErr != nil {\n\t\t\t\tlog.Printf("%s: persist bag after give: %v", conn.RemoteAddr(), saveErr)\n\t\t\t}\n\t\t\t// Latest-client LIVE A/B: a live View 2 already exists here.  Historical\n\t\t\t// bag lifecycle evidence refreshes a live bag with DELETE_VIEW before\n\t\t\t// CREATE_VIEW + authoritative replay.  Delete only the normal tool bag;\n\t\t\t// grantBagItems below recreates it using the current ordinal-safe encoder.\n\t\t\tif err := link.WriteFrame(serverDeleteView(2)); err != nil {\n\t\t\t\tgmSession.setAction("刷新背包失败：" + err.Error())\n\t\t\t\treturn err\n\t\t\t}\n\t\t\tlog.Printf("%s: latest-client BAG-REFRESH deleted live View=2 before authoritative replay", conn.RemoteAddr())\n\t\t\tif err := grantBagItems(link, player, itemCatalog, equipCatalog, conn.RemoteAddr().String()); err != nil {\n'''
    text = replace_once(text, old, new, "normal GM item live bag refresh")
    main_go.write_text(text, encoding="utf-8")

    trans = probe / "scene_transition.go"
    text = trans.read_text(encoding="utf-8")
    old = '''\tif err := sendPlayerAddObject(r.conn, r.player, destination.location.Position, resolveRoleVisual(r.activeRole.Appearance.Values)); err != nil {\n\t\treturn fmt.Errorf("write player re-entry archive: %w", err)\n\t}\n\tclear(r.activeNPCs)\n'''
    new = '''\tif err := sendPlayerAddObject(r.conn, r.player, destination.location.Position, resolveRoleVisual(r.activeRole.Appearance.Values)); err != nil {\n\t\treturn fmt.Errorf("write player re-entry archive: %w", err)\n\t}\n\t// Latest-client re-entry A/B: the working initial-entry chain supplies the\n\t// authoritative player location/vitals before the explicit ClientReady.  The\n\t// failed target-scene chain stopped after AddObject and never produced 0x09.\n\t// Replay only already-recovered player state here, then still require the real\n\t// target ClientReady before materializing target NPCs.\n\tif err := sendPlayerLocationAndVitals(r.conn, r.player, destination.location.Position); err != nil {\n\t\treturn fmt.Errorf("write early target player location/vitals: %w", err)\n\t}\n\tlog.Printf("%s: latest-client REENTRY-EARLY-PLAYER-SYNC scene=%s resource=%s x=%.3f y=%.3f z=%.3f orient=%.3f", r.conn.RemoteAddr(), destination.location.Scene.Config, destination.location.Scene.Resource, destination.location.Position.X, destination.location.Position.Y, destination.location.Position.Z, destination.location.Position.Orient)\n\tclear(r.activeNPCs)\n'''
    text = replace_once(text, old, new, "target re-entry early player sync")
    trans.write_text(text, encoding="utf-8")

    test = probe / "latest_client_bag_refresh_reentry_sync_test.go"
    test.write_text(r'''package main

import (
    "strings"
    "testing"
)

func TestLatestClientGMPanelStillExposesGiveItem(t *testing.T) {
    for _, token := range []string{"物品发放", "Game_item_hp_001", "action:'give_item'", "config_id:config", "container:'bag'"} {
        if !strings.Contains(gmPanelHTML, token) {
            t.Fatalf("GM panel missing %q", token)
        }
    }
}
''', encoding="utf-8")

    print(f"patched {gm}")
    print(f"patched {main_go}")
    print(f"patched {trans}")
    print(f"wrote {test}")

if __name__ == '__main__':
    main()
