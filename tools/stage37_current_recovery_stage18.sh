#!/usr/bin/env bash
set -euo pipefail

ROOT="$(pwd)"
BUILD="$ROOT/buildtree"

# Stage18-A is a read-only reconstructed-source audit. It does not promote the
# candidate ordinary-shop selector and does not alter production mutation
# wiring. The purpose is to extract the exact current reconstructed live-state
# helpers before implementing the pure purchase staging adapter.
bash tools/stage37_current_recovery_stage17.sh

grep -F 'current_client_selector_verified=0' "$ROOT/stage17_regular_shop_atomic_persistence_status.log"
grep -F 'wire_gate_status=BLOCKED_NO_EXACT_CURRENT_CLIENT_AUTHORITY_IN_REPO' "$ROOT/stage17_regular_shop_atomic_persistence_status.log"
grep -F 'production_handler_wiring=OFF' "$ROOT/stage17_regular_shop_atomic_persistence_status.log"

cat > "$ROOT/stage18_regular_shop_state_audit_status.log" <<'STATUS'
stage18_scope=ordinary_npc_shop_live_state_helper_audit_only
current_client_selector_verified=0
wire_gate_status=BLOCKED_NO_EXACT_CURRENT_CLIENT_AUTHORITY_IN_REPO
production_handler_wiring=OFF
legacy_or_historical_selector_promotion=OFF
gm_grant_changes=0
source_mutation=NONE
STATUS

python3 - "$BUILD/cmd/protocol-probe/zz_recovered_overlay.go" > "$ROOT/stage18_regular_shop_state_helpers.log" <<'PY'
import re
import sys
from pathlib import Path

path = Path(sys.argv[1])
text = path.read_text(encoding="utf-8")
patterns = [
    ("addBagItem", r"func\s+\(p\s+\*playerActor\)\s+addBagItem\s*\("),
    ("bagSnapshot", r"func\s+\(p\s+\*playerActor\)\s+bagSnapshot\s*\("),
    ("currencySnapshot", r"func\s+\(p\s+\*playerActor\)\s+currencySnapshot\s*\("),
    ("addGold", r"func\s+\(p\s+\*playerActor\)\s+addGold\s*\("),
    ("addSilver", r"func\s+\(p\s+\*playerActor\)\s+addSilver\s*\("),
    ("addSilverCard", r"func\s+\(p\s+\*playerActor\)\s+addSilverCard\s*\("),
]

def extract(start):
    brace = text.find("{", start)
    if brace < 0:
        raise RuntimeError("opening brace not found")
    depth = 0
    in_str = False
    in_rune = False
    esc = False
    i = brace
    while i < len(text):
        ch = text[i]
        if in_str:
            if esc:
                esc = False
            elif ch == "\\":
                esc = True
            elif ch == '"':
                in_str = False
        elif in_rune:
            if esc:
                esc = False
            elif ch == "\\":
                esc = True
            elif ch == "'":
                in_rune = False
        else:
            if ch == '"':
                in_str = True
            elif ch == "'":
                in_rune = True
            elif ch == "{":
                depth += 1
            elif ch == "}":
                depth -= 1
                if depth == 0:
                    return text[start:i+1]
        i += 1
    raise RuntimeError("unterminated function")

for name, pattern in patterns:
    m = re.search(pattern, text)
    print(f"===== {name} =====")
    if not m:
        print("NOT_FOUND")
        continue
    print(extract(m.start()))
    print()

for helper in ["nextBagSlotFrom", "bagViewForViewID"]:
    m = re.search(r"func\s+" + re.escape(helper) + r"\s*\(", text)
    print(f"===== {helper} =====")
    if not m:
        print("NOT_FOUND")
    else:
        print(extract(m.start()))
        print()
PY

for required in '===== addBagItem =====' '===== bagSnapshot =====' '===== currencySnapshot =====' '===== addGold =====' '===== addSilver =====' '===== addSilverCard ====='; do
  grep -F "$required" "$ROOT/stage18_regular_shop_state_helpers.log" >/dev/null
done
if grep -A1 '^===== addBagItem =====$' "$ROOT/stage18_regular_shop_state_helpers.log" | grep -F 'NOT_FOUND' >/dev/null; then
  echo 'addBagItem not found in reconstructed source' >&2
  exit 97
fi

# Stage18-A does not add buildtree source files; Stage17 regression remains the
# compile/test authority while this audit is read-only.
source_files=$(find "$BUILD" -type f ! -name 'ci.real.mod' ! -name 'ci.real.sum' ! -name 'protocol-probe' | wc -l | tr -d ' ')
echo "stage18_audit_source_files=$source_files"
if [ "$source_files" != "219" ]; then
  echo "unexpected Stage18 audit source count: $source_files" >&2
  exit 96
fi
