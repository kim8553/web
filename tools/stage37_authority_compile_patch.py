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


# Current loadNPCSceneSpawnsFromTables returns four values. The legacy
# loadChengduNPCSpawns wrapper does not exist in current DWARF.
regex_once(
    "cmd/protocol-probe/npc_catalog.go",
    r"func loadChengduNPCSpawns\(tableDir, creatorDir string, schema clientdata\.Schema\) \(\[\]npcSpawn, npcCatalogStats, error\) \{.*?\n\}\n",
    "",
)
replace_exact(
    "cmd/protocol-probe/npc_catalog.go",
    "_, stats, loadErr := loadNPCSceneSpawnsFromTables(tables, filepath.Join(creatorRoot, name), schema)",
    "_, _, stats, loadErr := loadNPCSceneSpawnsFromTables(tables, filepath.Join(creatorRoot, name), schema)",
)

# Current playerActor DWARF has no activeParry field; these methods are legacy.
regex_once(
    "cmd/protocol-probe/player_actor.go",
    r"func \(p \*playerActor\) setActiveParry\(value activeParryRequest\) \{.*?\n\}\nfunc \(p \*playerActor\) activeParrySnapshot\(\) activeParryRequest \{.*?\n\}\n",
    "",
)

# Exact current ActorState field names.
replace_exact("cmd/protocol-probe/player_actor.go", "state.MaxQGPointAdd", "state.MaxQingGongPointAdd")
replace_exact("cmd/protocol-probe/player_actor.go", "state.MaxQGPoint)", "state.MaxQingGongPoint)")

# Current clientdata.Value is 48 bytes. DWARF has I64 at offset 8 and an
# inline Int64Value(value int64). Current currencyProperties machine code
# writes tag 4 and the int64 payload at offset 8.
replace_exact(
    "internal/clientdata/npc_schema.go",
    "\tWireInt32      WireType = 3\n\tWireFloat32    WireType = 5",
    "\tWireInt32      WireType = 3\n\tWireInt64      WireType = 4\n\tWireFloat32    WireType = 5",
)
replace_exact(
    "internal/clientdata/npc_resolver.go",
    "\tI32  int32\n\tF32  float32",
    "\tI32  int32\n\tI64  int64\n\tF32  float32",
)
replace_exact(
    "internal/clientdata/npc_resolver.go",
    "func Int32Value(value int32) Value {\n\treturn Value{Type: WireInt32, I32: value}\n}\nfunc Float32Value",
    "func Int32Value(value int32) Value {\n\treturn Value{Type: WireInt32, I32: value}\n}\nfunc Int64Value(value int64) Value {\n\treturn Value{Type: WireInt64, I64: value}\n}\nfunc Float32Value",
)
replace_exact(
    "internal/clientdata/npc_resolver.go",
    "\tcase WireInt32:\n\t\treturn v.I32\n\tcase WireFloat32:",
    "\tcase WireInt32:\n\t\treturn v.I32\n\tcase WireInt64:\n\t\treturn v.I64\n\tcase WireFloat32:",
)

# Current S2CFacultyMessage DWARF type is int32 and value is 191.
replace_exact(
    "cmd/protocol-probe/player_progress.go",
    "const S2CFacultyMessage uint16 = 191",
    "const S2CFacultyMessage int32 = 191",
)

# Current openRoleStore uses two no-op close closures. JSONRepository.Close
# is not a current authored method; current func1/func2 return nil.
replace_exact(
    "cmd/protocol-probe/role_store.go",
    "close: repository.Close",
    "close: func() error { return nil }",
    expected=2,
)
