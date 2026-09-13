from pathlib import Path
import re
import sys

root = Path(sys.argv[1] if len(sys.argv) > 1 else ".")


def read(rel):
    return (root / rel).read_text(encoding="utf-8")


def write(rel, text):
    (root / rel).write_text(text, encoding="utf-8")


def replace_exact(rel, old, new, expected=1):
    text = read(rel)
    count = text.count(old)
    if count != expected:
        raise SystemExit(f"{rel}: expected {expected} occurrence(s), found {count}: {old[:100]!r}")
    write(rel, text.replace(old, new))
    print(f"patched {rel}: {count} replacement(s)")


def regex_once(rel, pattern, repl):
    text = read(rel)
    out, count = re.subn(pattern, repl, text, count=1, flags=re.S)
    if count != 1:
        raise SystemExit(f"{rel}: regex expected 1, found {count}: {pattern}")
    write(rel, out)
    print(f"patched {rel}")


# 1. Current loader is four-return. Legacy loadChengduNPCSpawns is absent from current DWARF.
regex_once("cmd/protocol-probe/npc_catalog.go", r"func loadChengduNPCSpawns\(tableDir, creatorDir string, schema clientdata\.Schema\) \(\[\]npcSpawn, npcCatalogStats, error\) \{.*?\n\}\n", "")
replace_exact("cmd/protocol-probe/npc_catalog.go", "_, stats, loadErr := loadNPCSceneSpawnsFromTables(tables, filepath.Join(creatorRoot, name), schema)", "_, _, stats, loadErr := loadNPCSceneSpawnsFromTables(tables, filepath.Join(creatorRoot, name), schema)")

# 2. Current playerActor DWARF has no activeParry field. These are legacy baseline methods.
regex_once("cmd/protocol-probe/player_actor.go", r"func \(p \*playerActor\) setActiveParry\(value activeParryRequest\) \{.*?\n\}\nfunc \(p \*playerActor\) activeParrySnapshot\(\) activeParryRequest \{.*?\n\}\n", "")

# 3. Exact current ActorState field names.
replace_exact("cmd/protocol-probe/player_actor.go", "state.MaxQGPointAdd", "state.MaxQingGongPointAdd")
replace_exact("cmd/protocol-probe/player_actor.go", "state.MaxQGPoint)", "state.MaxQingGongPoint)")

# 4. Current clientdata.Value is 48 bytes: Type@0, I64@8. Int64Value is inline at decl line 25.
# Current currencyProperties machine code writes tag 4 and the int64 payload at offset 8 for all four currency fields.
replace_exact("internal/clientdata/npc_schema.go", "\tWireInt32      WireType = 3\n\tWireFloat32    WireType = 5", "\tWireInt32      WireType = 3\n\tWireInt64      WireType = 4\n\tWireFloat32    WireType = 5")
replace_exact("internal/clientdata/npc_resolver.go", "\tI32  int32\n\tF32  float32", "\tI32  int32\n\tI64  int64\n\tF32  float32")
replace_exact("internal/clientdata/npc_resolver.go", "func Int32Value(value int32) Value {\n\treturn Value{Type: WireInt32, I32: value}\n}\nfunc Float32Value", "func Int32Value(value int32) Value {\n\treturn Value{Type: WireInt32, I32: value}\n}\nfunc Int64Value(value int64) Value {\n\treturn Value{Type: WireInt64, I64: value}\n}\nfunc Float32Value")
replace_exact("internal/clientdata/npc_resolver.go", "\tcase WireInt32:\n\t\treturn v.I32\n\tcase WireFloat32:", "\tcase WireInt32:\n\t\treturn v.I32\n\tcase WireInt64:\n\t\treturn v.I64\n\tcase WireFloat32:")

# 5. Current S2CFacultyMessage DWARF type is int32 and value is 191.
replace_exact("cmd/protocol-probe/player_progress.go", "const S2CFacultyMessage uint16 = 191", "const S2CFacultyMessage int32 = 191")

# 6. Current openRoleStore uses no-op close closures in both DB and JSON branches.
# JSONRepository.Close is not a current authored method; func1/func2 are exact 5-byte zero-return closures.
replace_exact("cmd/protocol-probe/role_store.go", "close: repository.Close", "close: func() error { return nil }", expected=2)

# 7. Current sceneNPCRegistry is 128 bytes and has taskSpawns at offset 120.
replace_exact("cmd/protocol-probe/scene_catalog_registry.go", "\tcatalogs map[string][]npcSpawn\n\tstats    map[string]npcCatalogStats\n}", "\tcatalogs   map[string][]npcSpawn\n\tstats      map[string]npcCatalogStats\n\ttaskSpawns map[string]map[string][]npcSpawn\n}")

# 8. Current transport menu IDs use exact int32 base 860001000; inline transportMenuFuncID is base + index.
scene = read("cmd/protocol-probe/scene_lifecycle.go")
anchor = "type talkMenuItem struct {\n\tfuncID int32\n\ttextID string\n}\n"
if scene.count(anchor) != 1:
    raise SystemExit("scene_lifecycle.go: talkMenuItem anchor mismatch")
insert = anchor + "\nconst transportMenuFuncBase int32 = 860001000\n\nfunc transportMenuFuncID(index int) int32 {\n\treturn transportMenuFuncBase + int32(index)\n}\n"
write("cmd/protocol-probe/scene_lifecycle.go", scene.replace(anchor, insert, 1))
print("patched cmd/protocol-probe/scene_lifecycle.go: transport menu helper")

# Current reconcileViewportLocked returns ([]world.EntityID, error).
replace_exact("cmd/protocol-probe/scene_lifecycle.go", "if err := s.reconcileViewportLocked(); err != nil {\n\t\treturn err\n\t}", "if _, err := s.reconcileViewportLocked(); err != nil {\n\t\treturn err\n\t}")

# 9. Current sendEntryScene(conn, scene, name); reentry passes active role name.
replace_exact("cmd/protocol-probe/scene_transition.go", "sendEntryScene(r.conn, destination.location.Scene)", "sendEntryScene(r.conn, destination.location.Scene, r.activeRole.Name)")

# 10. These legacy shortcut workaround functions are absent from current DWARF.
regex_once("cmd/protocol-probe/shortcut_records.go", r"func grantHeartBuddhaPalmShortcuts\(conn sceneMessageConnection, player \*playerActor\) \(int, error\) \{.*?\n\}\n", "")
regex_once("cmd/protocol-probe/shortcut_records.go", r"func grantTaiJiQuanGuPuShortcuts\(conn sceneMessageConnection, player \*playerActor\) \(int, error\) \{.*?\n\}\n", "")

# 11. effectsHaveKind is an exact current inline helper. Machine code walks 72-byte effects and compares kind byte at offset 0.
skill = read("cmd/protocol-probe/skill_catalog.go")
anchor = "func compileCombatSkill("
idx = skill.find(anchor)
if idx < 0:
    raise SystemExit("skill_catalog.go: compileCombatSkill anchor not found")
helper = "func effectsHaveKind(effects []combatSkillEffect, kind skillEffectKind) bool {\n\tfor _, e := range effects {\n\t\tif e.kind == kind {\n\t\t\treturn true\n\t\t}\n\t}\n\treturn false\n}\n\n"
if "func effectsHaveKind(" in skill:
    raise SystemExit("skill_catalog.go: effectsHaveKind already exists")
write("cmd/protocol-probe/skill_catalog.go", skill[:idx] + helper + skill[idx:])
print("patched cmd/protocol-probe/skill_catalog.go: effectsHaveKind")

# 12. Current skillActionSwitchFrame inlines serverModernCustomIntMessage with messageID 414 and values reset/1223/1.
# Current sceneMessageConnection method set is WriteFrame only.
replace_exact("cmd/protocol-probe/skill_switch.go", 'return serverModernCustomIntMessage(414, []serverCustomValue{customString("reset"), customInt(1223), customInt(1)})', 'return serverModernCustomIntMessage(414, customString("reset"), customInt(1223), customInt(1))')
replace_exact("cmd/protocol-probe/skill_switch.go", "if _, err := link.Write(frame); err != nil {", "if err := link.WriteFrame(frame); err != nil {")
