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


# 22. Current logicStateFaculty DWARF type is int32, while ActorState.LogicState is uint8.
# The current value is 108; compare the actor byte to that same current constant explicitly.
replace_exact(
    "cmd/protocol-probe/player_progress_test.go",
    "if got := player.actor.Snapshot().LogicState; got != logicStateFaculty {",
    "if got := player.actor.Snapshot().LogicState; got != uint8(logicStateFaculty) {",
)

# 23. Current grantStarterQingGong signature is (conn, roleName, qgLevels). A normal role name
# selects the starter branch and a nil level map exercises the current level-1 fallback.
replace_exact(
    "cmd/protocol-probe/qinggong_records_test.go",
    "if err := grantStarterQingGong(conn); err != nil {",
    'if err := grantStarterQingGong(conn, "测试", nil); err != nil {',
)

# 24. Current roleStore.login authenticates an existing account using passwordCT. It does not
# ProvisionLegacy. Register both JSON-backed accounts with known verifier bytes before logging in.
replace_exact(
    "cmd/protocol-probe/role_store_test.go",
    "\tfirstAccount, firstRole, err := store.login(ctx, \"acct:v1:first\")\n\tif err != nil || firstRole != nil {\n\t\tt.Fatalf(\"first login role=%v error=%v\", firstRole, err)\n\t}\n\tsecondAccount, secondRole, err := store.login(ctx, \"acct:v1:second\")\n\tif err != nil || secondRole != nil {\n\t\tt.Fatalf(\"second login role=%v error=%v\", secondRole, err)\n\t}\n",
    "\tfirstKey := role.AccountKey(\"acct:v1:first\")\n\tfirstVerifier := []byte(\"first-verifier\")\n\tfirstAccount, err := store.register(ctx, firstKey, firstVerifier)\n\tif err != nil {\n\t\tt.Fatal(err)\n\t}\n\t_, firstRole, err := store.login(ctx, firstKey, firstVerifier)\n\tif err != nil || firstRole != nil {\n\t\tt.Fatalf(\"first login role=%v error=%v\", firstRole, err)\n\t}\n\tsecondKey := role.AccountKey(\"acct:v1:second\")\n\tsecondVerifier := []byte(\"second-verifier\")\n\tsecondAccount, err := store.register(ctx, secondKey, secondVerifier)\n\tif err != nil {\n\t\tt.Fatal(err)\n\t}\n\t_, secondRole, err := store.login(ctx, secondKey, secondVerifier)\n\tif err != nil || secondRole != nil {\n\t\tt.Fatalf(\"second login role=%v error=%v\", secondRole, err)\n\t}\n",
)

# Current defaultRoleAppearance only pads to nine fields; it does not inject the legacy default
# faction. Current defaultRoleLocation is selected from appearance[0] (book ID).
replace_exact(
    "cmd/protocol-probe/role_store_test.go",
    "\tif got := firstRole.Appearance.Values[8]; got != defaultRoleFaction {\n\t\tt.Fatalf(\"default faction=%q want=%q\", got, defaultRoleFaction)\n\t}\n\tif got, want := firstRole.Location, defaultRoleLocation(); got.Scene != want.Scene || got.Position != want.Position {",
    "\tif got := firstRole.Appearance.Values[8]; got != \"\" {\n\t\tt.Fatalf(\"current padded faction=%q want empty\", got)\n\t}\n\tif got, want := firstRole.Location, defaultRoleLocation(firstRole.Appearance.Values); got.Scene != want.Scene || got.Position != want.Position {",
)
replace_exact(
    "cmd/protocol-probe/role_store_test.go",
    "_, reloaded, err := store.login(ctx, firstAccount.Key)",
    "_, reloaded, err := store.login(ctx, firstAccount.Key, firstVerifier)",
)
replace_exact(
    "cmd/protocol-probe/role_store_test.go",
    "_, other, err := store.login(ctx, secondAccount.Key)",
    "_, other, err := store.login(ctx, secondAccount.Key, secondVerifier)",
)

# 25. Current handleShortcutCustom has persistence arguments. These record-shape tests deliberately
# use the current no-persistence path (store=nil or roleID=0), which persistShortcuts treats as no-op.
replace_exact(
    "cmd/protocol-probe/shortcut_records_test.go",
    'handleShortcutCustom(conn, player, set, "test")',
    'handleShortcutCustom(conn, player, set, "test", nil, 0)',
    expected=2,
)
replace_exact(
    "cmd/protocol-probe/shortcut_records_test.go",
    'handleShortcutCustom(conn, player, remove, "test")',
    'handleShortcutCustom(conn, player, remove, "test", nil, 0)',
)

# Current EXE/DWARF has no grantHeartBuddhaPalmShortcuts or grantTaiJiQuanGuPuShortcuts.
# Remove only the two tests for those legacy workaround helpers; keep current shortcut/sitcross tests.
shortcut = read("cmd/protocol-probe/shortcut_records_test.go")
start = shortcut.find("func TestGrantHeartBuddhaPalmShortcutsFillSecondNumericPage")
end = shortcut.find("func TestSitcrossUsesCapturedStartArgumentAndNormalPropertyUpdate")
if start < 0 or end < 0 or end <= start:
    raise SystemExit("shortcut_records_test.go: legacy shortcut test block anchors not found")
write("cmd/protocol-probe/shortcut_records_test.go", shortcut[:start] + shortcut[end:])
print("patched cmd/protocol-probe/shortcut_records_test.go: removed non-current shortcut workaround tests")

# 26. Current oneRoleLoginSuccess encodes faction from the supplied appearance and scene from the
# supplied role.Scene. It no longer injects defaultRoleFaction for nil appearance.
replace_exact(
    "cmd/protocol-probe/main_test.go",
    'func TestCreatedRoleList(t *testing.T) {\n\tmsg := oneRoleLoginSuccess(time.Date(2026, 7, 11, 12, 0, 0, 0, time.Local), "本地角色", nil, defaultRoleLocation(nil).Scene)',
    'func TestCreatedRoleList(t *testing.T) {\n\tappearance := []string{"book1", "", "", "", "", "", "", "", defaultRoleFaction}\n\tmsg := oneRoleLoginSuccess(time.Date(2026, 7, 11, 12, 0, 0, 0, time.Local), "本地角色", appearance, defaultRoleLocation(appearance).Scene)',
)
