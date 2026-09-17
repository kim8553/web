#!/usr/bin/env python3
"""Apply a pinned, fail-closed 0x4F preflight patch to the EXACT current source.

This script is not a runtime resource extractor. It never changes transaction,
557 binding or grant/debit semantics. It checks the three source blob IDs before
changing anything, and aborts on absent/duplicate anchors.
"""
from __future__ import annotations

import hashlib
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent
EXPECTED = {
    "server/cmd/protocol-probe/scene_lifecycle.go": "ac3b032953b29de613ef752f0f87b2173e3d1edb",
    "server/cmd/protocol-probe/main.go": "5b7e3c5d1184723415cc121371a17d39dff9328b",
    "server/cmd/protocol-probe/latest_client_shop_exchange_contract.go": "e136bcff713b2f9f60904e91df5c3e9444c4e1a3",
}


def git_blob_hash(data: bytes) -> str:
    return hashlib.sha1(b"blob " + str(len(data)).encode() + b"\0" + data).hexdigest()


def replace_once(source: str, before: str, after: str, label: str) -> str:
    count = source.count(before)
    if count != 1:
        raise RuntimeError(f"{label}: expected exactly one source anchor, found {count}")
    return source.replace(before, after, 1)


def patch_scene(source: str) -> str:
    source = replace_once(source,
        "\tlastMenuMovie       bool\n",
        "\tlastMenuMovie       bool\n\tactiveShop          currentShopExchangeSession\n",
        "scene field")
    source = replace_once(source,
        "\ts.lastMenuMovie = false\n\ts.epoch++\n",
        "\ts.lastMenuMovie = false\n\ts.activeShop = currentShopExchangeSession{}\n\ts.epoch++\n",
        "scene begin clears shop")
    source = replace_once(source,
        "\tdelete(s.entities, entityID)\n\tdelete(s.combatActors, entityID)\n",
        "\tif s.activeShop.NPCObjectID == id {\n\t\ts.activeShop = currentShopExchangeSession{}\n\t}\n\tdelete(s.entities, entityID)\n\tdelete(s.combatActors, entityID)\n",
        "NPC despawn clears shop")
    source = replace_once(source,
        "func (s *sceneLifecycle) objectRequest(request clientObjectRequest) (string, error) {\n\ts.mu.Lock()\n\tdefer s.mu.Unlock()\n\tif s.closed {\n\t\treturn \"\", nil\n\t}\n\tentity, exists := s.entities[worldcore.EntityID(request.ObjectID)]\n",
        "func (s *sceneLifecycle) objectRequest(request clientObjectRequest) (string, error) {\n\ts.mu.Lock()\n\tdefer s.mu.Unlock()\n\tif s.closed {\n\t\treturn \"\", nil\n\t}\n\t// A new NPC interaction invalidates the previous shop, even when the\n\t// client keeps an old shop window open. No speculative timeout is used.\n\ts.activeShop = currentShopExchangeSession{}\n\tentity, exists := s.entities[worldcore.EntityID(request.ObjectID)]\n",
        "new NPC interaction clears shop")
    source = replace_once(source,
        "func (s *sceneLifecycle) openSelectedServiceLocked(entity sceneEntity, service npcService) error {\n\tswitch service.mark {\n",
        "func (s *sceneLifecycle) openSelectedServiceLocked(entity sceneEntity, service npcService) error {\n\t// Called under s.mu; retain authorization only for a successfully\n\t// opened shop service, never for a menu advertisement alone.\n\ts.activeShop = currentShopExchangeSession{}\n\tswitch service.mark {\n",
        "opening a service invalidates previous shop")
    source = replace_once(source,
        "\t\tif err := s.openShopLocked(service.value); err != nil {\n\t\t\treturn err\n\t\t}\n\tcase 0x1002:\n",
        "\t\tif err := s.openShopLocked(service.value); err != nil {\n\t\t\treturn err\n\t\t}\n\t\ts.activeShop = currentShopExchangeSession{ShopID: service.value, NPCObjectID: entity.id, NPCOwnerID: entity.ownerID, NPCConfigID: entity.configID, SceneEpoch: s.epoch}\n\tcase 0x1002:\n",
        "record only successful NPC shop open")
    return source


def patch_main(source: str) -> str:
    return replace_once(source,
        "\t\t\tcase 64, 69, 79:\n\t\t\t\tif _, handleErr := handleShopExchangeContract(link, player, custom, conn.RemoteAddr().String()); handleErr != nil {\n\t\t\t\t\tlog.Printf(\"%s: handle current shop exchange contract: %v\", conn.RemoteAddr(), handleErr)\n\t\t\t\t}\n\t\t\tcase 70:\n",
        "\t\t\tcase 64, 69:\n\t\t\t\tif _, handleErr := handleShopExchangeContract(link, player, custom, conn.RemoteAddr().String()); handleErr != nil {\n\t\t\t\t\tlog.Printf(\"%s: handle current shop exchange contract: %v\", conn.RemoteAddr(), handleErr)\n\t\t\t\t}\n\t\t\tcase 79:\n\t\t\t\trequest, matched, parseErr := parseShopExchangeBuyRequest(custom)\n\t\t\t\tif parseErr != nil || !matched || player == nil {\n\t\t\t\t\tlog.Printf(\"%s: current shop exchange buy blocked: malformed request or missing scene player: %v\", conn.RemoteAddr(), parseErr)\n\t\t\t\t\tbreak\n\t\t\t\t}\n\t\t\t\tif !world.currentShopExchangeSessionAllows(request.ShopID) {\n\t\t\t\t\tlog.Printf(\"%s: current shop exchange buy blocked: no matching live NPC shop service shop=%s\", conn.RemoteAddr(), request.ShopID)\n\t\t\t\t\tbreak\n\t\t\t\t}\n\t\t\t\tif _, handleErr := handleShopExchangeContract(link, player, custom, conn.RemoteAddr().String()); handleErr != nil {\n\t\t\t\t\tlog.Printf(\"%s: handle current shop exchange contract: %v\", conn.RemoteAddr(), handleErr)\n\t\t\t\t}\n\t\t\tcase 70:\n",
        "0x4F dispatcher session preflight")


def patch_contract(source: str) -> str:
    return replace_once(source,
        "func resolveDefaultCurrentShopExchangeBuyAuthority(request shopExchangeBuyRequest) (currentShopExchangeBuyAuthority, bool, error) {\n\tauthority, err := loadDefaultCurrentShopConditionAuthority()\n\tif err != nil {\n\t\treturn currentShopExchangeBuyAuthority{}, false, err\n\t}\n\treturn resolveAuthorizedCurrentShopExchangeBuyAuthority(defaultShopINIPath, defaultExchangeItemINIPath, authority, request)\n}\n",
        "func resolveDefaultCurrentShopExchangeBuyAuthority(request shopExchangeBuyRequest) (currentShopExchangeBuyAuthority, bool, error) {\n\t// sync.Once authenticates only the first load; recheck every request\n\t// against the same exact-current five-file fingerprint set.\n\tif err := recheckCurrentShopExchangeResources(); err != nil {\n\t\treturn currentShopExchangeBuyAuthority{}, false, err\n\t}\n\tauthority, err := loadDefaultCurrentShopConditionAuthority()\n\tif err != nil {\n\t\treturn currentShopExchangeBuyAuthority{}, false, err\n\t}\n\tresolved, selected, err := resolveAuthorizedCurrentShopExchangeBuyAuthority(defaultShopINIPath, defaultExchangeItemINIPath, authority, request)\n\tif err != nil || !selected {\n\t\treturn resolved, selected, err\n\t}\n\tif err := recheckCurrentShopExchangeResources(); err != nil {\n\t\treturn currentShopExchangeBuyAuthority{}, false, err\n\t}\n\treturn resolved, true, nil\n}\n",
        "cached shop authority revalidation")


def main() -> None:
    originals = {}
    for relpath, wanted in EXPECTED.items():
        raw = (ROOT / relpath).read_bytes()
        got = git_blob_hash(raw)
        if got != wanted:
            raise RuntimeError(f"{relpath}: source blob mismatch {got} != {wanted}; abort without writing")
        originals[relpath] = raw.decode("utf-8")
    patches = {
        "server/cmd/protocol-probe/scene_lifecycle.go": patch_scene(originals["server/cmd/protocol-probe/scene_lifecycle.go"]),
        "server/cmd/protocol-probe/main.go": patch_main(originals["server/cmd/protocol-probe/main.go"]),
        "server/cmd/protocol-probe/latest_client_shop_exchange_contract.go": patch_contract(originals["server/cmd/protocol-probe/latest_client_shop_exchange_contract.go"]),
    }
    # No source is written until every exact anchor in every file has passed.
    for relpath, content in patches.items():
        (ROOT / relpath).write_text(content, encoding="utf-8", newline="")
        print(f"PATCHED {relpath}")


if __name__ == "__main__":
    main()
