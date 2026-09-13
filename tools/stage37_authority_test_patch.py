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


# Current EXE handleActiveParryCustom accepts integral 0xDA settings messages but explicitly
# logs them as ignored settings commands. Current playerActor DWARF has no activeParry field.
write("cmd/protocol-probe/active_parry_test.go", '''package main

import "testing"

func TestActiveParryAcceptsModernIntegralSubtypeAsSettingsCommand(t *testing.T) {
\tplayer := newPlayerActor("招架测试", 0)
\tbefore := player.actor.Snapshot()
\thandled, err := handleActiveParryCustom(player, clientCustomMessage{Values: []clientCustomValue{
\t\t{Type: 2, Int32: clientCustomActiveParry},
\t\t{Type: 5, Float64: 2},
\t\t{Type: 2, Int32: 77},
\t}}, "test")
\tif err != nil || !handled {
\t\tt.Fatalf("handle active parry handled=%t err=%v", handled, err)
\t}
\tif got := player.actor.Snapshot(); got != before {
\t\tt.Fatalf("settings-only active parry mutated actor state: before=%+v after=%+v", before, got)
\t}
}

func TestActiveParryRejectsFractionalSubtypeWithoutMutatingActorState(t *testing.T) {
\tplayer := newPlayerActor("招架测试", 0)
\tbefore := player.actor.Snapshot()
\thandled, err := handleActiveParryCustom(player, clientCustomMessage{Values: []clientCustomValue{
\t\t{Type: 2, Int32: clientCustomActiveParry},
\t\t{Type: 5, Float64: 1.5},
\t\t{Type: 2, Int32: 77},
\t}}, "test")
\tif !handled || err == nil {
\t\tt.Fatalf("fractional subtype handled=%t err=%v", handled, err)
\t}
\tif got := player.actor.Snapshot(); got != before {
\t\tt.Fatalf("fractional subtype mutated actor state: before=%+v after=%+v", before, got)
\t}
}
''')
print("patched cmd/protocol-probe/active_parry_test.go: current settings-only contract")

# Current blocking state is playerActor.parrying via setPlayerBlock(bool,time.Time), not the
# removed legacy activeParry storage methods.
replace_exact("cmd/protocol-probe/combat_ai_test.go", "func TestCombatTickHonorsFreshActiveParry(t *testing.T) {", "func TestCombatTickHonorsCurrentPlayerBlockState(t *testing.T) {")
replace_exact(
    "cmd/protocol-probe/combat_ai_test.go",
    "\tplayer.setActiveParry(activeParryRequest{Subtype: 0, At: now})",
    "\tif !player.setPlayerBlock(true, now) {\n\t\tt.Fatal(\"current block state was not engaged\")\n\t}",
)

# Current oneRoleLoginSuccess signature includes role.Scene and production callers pass the
# selected role's scene. The unit test uses the same current default role scene.
replace_exact(
    "cmd/protocol-probe/main_test.go",
    'oneRoleLoginSuccess(time.Date(2026, 7, 11, 12, 0, 0, 0, time.Local), "本地角色", nil)',
    'oneRoleLoginSuccess(time.Date(2026, 7, 11, 12, 0, 0, 0, time.Local), "本地角色", nil, defaultRoleLocation().Scene)',
)

# Property IDs are current uint16 wire indices. Test byte needles must encode them little-endian;
# direct byte(constant) conversions overflow for indices >255 and never represented the full wire ID.
replace_exact(
    "cmd/protocol-probe/player_progress_test.go",
    "[]byte{byte(propSP), byte(propSP >> 8), 100, 0, 0, 0}",
    "append(binary.LittleEndian.AppendUint16(nil, propSP), 100, 0, 0, 0)",
)
for name in ["propStaticData", "propItemType", "propMaxLevel", "propCurFillValue", "propMaxPowerValue"]:
    replace_exact(
        "cmd/protocol-probe/player_progress_test.go",
        f"{{byte({name}), byte({name} >> 8)}}",
        f"binary.LittleEndian.AppendUint16(nil, {name})",
    )
replace_exact(
    "cmd/protocol-probe/player_progress_test.go",
    "[]byte{byte(propFacultyName), byte(propFacultyName >> 8), 1, 0, 0, 0, 0}",
    "append(binary.LittleEndian.AppendUint16(nil, propFacultyName), 1, 0, 0, 0, 0)",
)

for name in ["propStaticData", "propItemType", "propMaxLevel", "propCurFillValue", "propTotalFillValue", "propPauseTime"]:
    replace_exact(
        "cmd/protocol-probe/skill_view_test.go",
        f"{{byte({name}), byte({name} >> 8)}}",
        f"binary.LittleEndian.AppendUint16(nil, {name})",
    )
replace_exact(
    "cmd/protocol-probe/skill_view_test.go",
    "{byte(propSkillCanUse), byte(propSkillCanUse >> 8), 1}",
    "append(binary.LittleEndian.AppendUint16(nil, propSkillCanUse), 1)",
)
