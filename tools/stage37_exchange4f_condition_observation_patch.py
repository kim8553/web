#!/usr/bin/env python3
"""Fail-closed, exactly-once patch of current 0x4F logging only.

This script is CI-only: it does NOT add a purchase authorization, change
bindings, debit, grant, persistence, or any client wire format.
"""
from pathlib import Path
import subprocess

source = Path('server/cmd/protocol-probe/latest_client_shop_exchange_contract.go')
expected_blob = '35add8545f7a006045c231c237c49b47be98bce1'
actual_blob = subprocess.check_output(['git', 'hash-object', str(source)], text=True).strip()
if actual_blob != expected_blob:
    raise SystemExit(f'REJECT: exchange contract source drift: {actual_blob} != {expected_blob}')

before = '''\t\tlog.Printf("%s: current shop exchange buy request shop=%s page=%d pos=%d count=%d authenticated config=%s exchange_data=%d leaves=%d blocked: condition acceptance/cost/bind/commit path unresolved",
\t\t\tremote, request.ShopID, request.Page, request.Position, request.Count, authorized.Selection.Item.configID, authorized.Selection.Item.exchangeData, len(authorized.Capability.Leaves))
\t\treturn true, nil
'''
after = '''\t\t// The 508 evaluator can describe the current player's condition roots,
\t\t// but those roots are not an exact purchase authorization: neither the
\t\t// native Condition/Condition2 acceptance rule nor an atomic player-state
\t\t// snapshot and transaction have been proven. Observe only; never grant.
\t\tconditionAuthority, conditionErr := loadDefaultCurrentShopConditionAuthority()
\t\tif conditionErr != nil || conditionAuthority == nil || conditionAuthority.catalog == nil || player == nil {
\t\t\tlog.Printf("%s: current shop exchange buy request shop=%s page=%d pos=%d count=%d blocked: condition observation unavailable: %v",
\t\t\t\tremote, request.ShopID, request.Page, request.Position, request.Count, conditionErr)
\t\t\treturn true, nil
\t\t}
\t\tevaluator := exactCurrentConditionEvaluator{
\t\t\tcatalog: conditionAuthority.catalog, player: player,
\t\t\tskillMaxTable: conditionAuthority.skillMaxTable,
\t\t}
\t\tdetails, supported, observationErr := evaluateExchangeConditionDetails(
\t\t\tauthorized.Definition.Condition, authorized.Definition.Condition2,
\t\t\tauthorized.Definition.Filters, conditionAuthority.catalog.resolve, evaluator.evaluate,
\t\t)
\t\tif observationErr != nil {
\t\t\tlog.Printf("%s: current shop exchange buy request shop=%s exchange_data=%d blocked: condition observation error: %v",
\t\t\t\tremote, request.ShopID, authorized.Selection.Item.exchangeData, observationErr)
\t\t\treturn true, nil
\t\t}
\t\trootResults := make([]bool, len(details))
\t\tfor i, detail := range details {
\t\t\trootResults[i] = detail.Satisfied
\t\t}
\t\tobservation := summarizeCurrentShopExchangeConditionResults(rootResults, supported)
\t\tlog.Printf("%s: current shop exchange buy request shop=%s page=%d pos=%d count=%d authenticated config=%s exchange_data=%d leaves=%d roots=%d all_supported=%t unsatisfied_or_unresolved=%d blocked: native eligibility/cost/bind/commit path unresolved",
\t\t\tremote, request.ShopID, request.Page, request.Position, request.Count, authorized.Selection.Item.configID,
\t\t\tauthorized.Selection.Item.exchangeData, len(authorized.Capability.Leaves), observation.RootCount,
\t\t\tobservation.AllSupported, observation.UnsatisfiedOrUnresolved)
\t\treturn true, nil
'''
text = source.read_text(encoding='utf-8')
if text.count(before) != 1:
    raise SystemExit(f'REJECT: expected exactly one anchored 0x4F log block; found {text.count(before)}')
source.write_text(text.replace(before, after, 1), encoding='utf-8')
print('PATCHED read-only 0x4F condition observation; mutation remains disabled')
