#!/usr/bin/env bash
set -euo pipefail
ROOT="$(pwd)"

# Stage23 adds ONLY separate read-only capture/analysis utilities. Stage22's
# production code, fail-closed handler and exact-current wire gate are unchanged.
bash tools/stage37_current_recovery_stage22.sh

grep -Fx 'current_client_selector_verified=0' "$ROOT/stage22_shop_wire_trace_status.log"
grep -Fx 'safe_production_handler_wiring=OFF' "$ROOT/stage22_shop_wire_trace_status.log"
grep -Fx 'legacy_candidate_mutation_route=DISABLED_FAIL_CLOSED' "$ROOT/stage22_shop_wire_trace_status.log"

# Exact Stage22 manifest observed in the successful Stage22 Actions artifact.
expected_manifest='bc2aeaf77ce9d9b198d725356b976599063b12c3324fc548e41298c8c7e46d9a'
actual_manifest="$(awk '{print $1}' "$ROOT/generated-manifest.sha256")"
echo "stage23_expected_stage22_manifest_sha256=$expected_manifest"
echo "stage23_actual_stage22_manifest_sha256=$actual_manifest"
test "$actual_manifest" = "$expected_manifest"
python3 -m unittest discover -s tests -p 'test_stage37_shop_wire_diff.py' -v > "$ROOT/stage23_shop_wire_diff_test.log" 2>&1
echo 0 > "$ROOT/stage23_shop_wire_diff_test.exit"

python3 - <<'PY' > "$ROOT/stage23_shop_wire_capture_audit.log"
from pathlib import Path
import ast
py = Path('tools/stage37_shop_wire_diff.py').read_text(encoding='utf-8')
ps = Path('tools/stage37_capture_shop_wire.ps1').read_text(encoding='utf-8')
source = ast.parse(py)
assert any(isinstance(n, ast.FunctionDef) and n.name == 'compare' for n in ast.walk(source))
for token in ('selector_promotion_allowed', 'handler_enable_allowed', 'UNVERIFIED_OBSERVATIONAL_DIFFERENCE_ONLY'):
    assert token in py, token
for token in ('SHOP_WIRE_OBSERVE opcode=', 'FileAccess]::Read', 'FileShare]::ReadWrite', 'maxAppendedBytes', 'WriteAllLines'):
    assert token in ps, token
for token in ('Start-Process', 'Stop-Process', 'New-Service', 'Set-ItemProperty', 'Invoke-WebRequest'):
    assert token not in ps, token
assert 'raw' not in ps.lower().split('$header = [regex]', 1)[1].split('$filtered =', 1)[0]
assert 'NINEYIN_SHOP_WIRE_TRACE' not in ps  # no server environment or process mutation
print('stage23_capture_audit=PASS')
print('capture_behavior=READ_ONLY_LOG_WINDOW_SANITIZED_HEADER_OUTPUT')
print('powershell_execution=NOT_RUN_ON_LINUX')
print('selector_promotion=FORBIDDEN')
print('existing_stage22_production_binary=UNCHANGED')
PY
cat > "$ROOT/stage23_shop_wire_diff_status.log" <<'STATUS'
stage23_scope=offline_baseline_vs_purchase_wire_observation_comparison
capture_script=WINDOWED_LOG_READ_ONLY_SANITIZED_HEADER_ONLY
capture_script_windows_runtime=NOT_RUN_ON_LINUX
capture_script_max_bytes=8388608
comparison_max_input_bytes=16777216
comparison_result=UNVERIFIED_OBSERVATIONAL_CANDIDATES_ONLY
baseline_zero_messages=ACCEPTED_WITH_WARNING
purchase_zero_messages=REJECTED
selector_promotion_allowed=NO
production_handler_wiring=OFF
legacy_candidate_mutation_route=DISABLED_FAIL_CLOSED
gm_grant_changes=0
stage22_production_source_changes=NONE
protocol_runtime=NOT_RUN_EXACT_RESOURCE_ABSENT
live_e2e=NOT_RUN
STATUS

echo 'stage23_shop_wire_diff_test=0'
echo 'stage23_capture_audit=PASS'
echo 'stage23_production_source_manifest_unchanged=PASS'
echo "stage23_windows_exe_sha256=$(awk '{print $1}' "$ROOT/stage37-current-windows-amd64.sha256")"
