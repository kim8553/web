# Stage 27 — source-baseline gap report

The exact fee2df DWARF/project-function evidence names 87 project source files. Comparing those
paths to the user-supplied `9yin-go-server1` editable source baseline gives:

- Present in baseline: 53
- Missing from baseline: 34
- Gap list: `evidence/stage27/current_fee2df_files_missing_from_baseline.txt`
- Complete matrix: `evidence/stage27/source_file_gap.csv`

The missing set is concentrated in later persistence/gameplay layers such as bag/equipment,
currency, item catalog, shop buy, quest loader/system, selection, skill book, and newer auth.
This is why the baseline is useful: the later source does not need to be reconstructed from an
empty tree, but the missing/changed files still require exact evidence before semantic reuse.

A separate `go tool nm` comparison under `evidence/stage27/` records only module-qualified
(non-`main`) symbols. It is intentionally not used as a complete function-count metric.

## R6/current-exe overlay recovery

The R6 direct-current-exe reconstruction checkpoint supplies 25 of the 34 baseline-missing
paths. Stage 28 semantically reconstructed `drop_table.go` from exact current DWARF and machine
code, bringing path-level availability to 79/87. Eight source paths remain without a recovered
file:

- `cmd/protocol-probe/equip_catalog.go`
- `cmd/protocol-probe/equip_grant.go`
- `cmd/protocol-probe/fwz_card.go`
- `cmd/protocol-probe/item_catalog.go`
- `cmd/protocol-probe/map_path.go`
- `cmd/protocol-probe/npc_path.go`
- `cmd/protocol-probe/npc_transport.go`
- `internal/auth/field_cipher.go`

`79/87` is only path-level availability. It is **not** an exact-current source-completion
percentage: the 53 baseline files can contain later changes and every reconstructed overlay file
requires evidence/dependency review before promotion.
