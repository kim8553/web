from pathlib import Path
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
        raise SystemExit(f"{rel}: expected {expected} occurrence(s), found {count}: {old[:120]!r}")
    write(rel, text.replace(old, new))
    print(f"patched {rel}: {count} replacement(s)")


# 28. Current clientdata.WireType.String has an exact WireInt64 branch.
# The current EXE jump table maps WireType=4 to the static string "int64".
replace_exact(
    "internal/clientdata/npc_schema.go",
    '\tcase WireInt32:\n\t\treturn "int32"\n\tcase WireFloat32:',
    '\tcase WireInt32:\n\t\treturn "int32"\n\tcase WireInt64:\n\t\treturn "int64"\n\tcase WireFloat32:',
)

# 29. Current main.appendNPCProperty has an explicit WireType=4 jump-table arm.
# Machine code at 0x140363072 grows the byte slice by 8 and stores Value.I64 as one
# little-endian 64-bit word. Preserve the exact bit pattern by converting int64 to uint64.
replace_exact(
    "cmd/protocol-probe/npc_catalog.go",
    "\tcase clientdata.WireInt32:\n\t\tvalue := uint32(property.Value.I32)\n\t\tmsg = append(msg, byte(value), byte(value>>8), byte(value>>16), byte(value>>24))\n\tcase clientdata.WireFloat32:",
    "\tcase clientdata.WireInt32:\n\t\tvalue := uint32(property.Value.I32)\n\t\tmsg = append(msg, byte(value), byte(value>>8), byte(value>>16), byte(value>>24))\n\tcase clientdata.WireInt64:\n\t\tmsg = binary.LittleEndian.AppendUint64(msg, uint64(property.Value.I64))\n\tcase clientdata.WireFloat32:",
)

# 30. Current main.compileSkillActions passes the one-byte static separator ';' (0x3B)
# to strings.genSplit at both split call sites. The recovered overlay had ',' at both sites,
# causing every current zhaoshi action row to be rejected as a one-field record.
rel = "cmd/protocol-probe/skill_catalog.go"
text = read(rel)
start = text.find("func compileSkillActions(")
end = text.find("func compileExactSkillBuff(", start)
if start < 0 or end < 0 or end <= start:
    raise SystemExit(f"{rel}: compileSkillActions anchors not found")
segment = text[start:end]
old = 'strings.Split(field.value, ",")'
count = segment.count(old)
if count != 2:
    raise SystemExit(f"{rel}: compileSkillActions expected 2 comma split sites, found {count}")
segment = segment.replace(old, 'strings.Split(field.value, ";")')
write(rel, text[:start] + segment + text[end:])
print(f"patched {rel}: 2 current semicolon split sites")
