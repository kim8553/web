# Stage57 — optional missing drop_table.json startup gate

## Authority / lineage

- Editable server lineage: user-provided `9yin-go-server1.rar` -> current `kim8553/web` Go source.
- Branch: `stage37-assistant-handoff-20260917`.
- Current-client behavior authority remains latest Snail client binaries/resources; no old V37/V46/JYZJ gameplay semantics were introduced.

## User runtime evidence

Stage56 on the user's actual Windows runtime progressed past the earlier `playerweapon` and `stringname.idres` startup blockers, then exited at:

`resources/modern/share/item/drop_table.json`

The user-provided original server runtime does not contain that JSON. No replacement drop data was fabricated.

## Code evidence

`defaultDropTablePath` points to `resources/modern/share/item/drop_table.json`.

Current references show the loaded table is consumed by GiftBox/drop resolution (`dropTable.Has`, `openGiftBox`). Normal NPC retail BUY selector `0x46` is not dependent on this table.

## Change

Source commit: `178ceecdfe1b9cb9d985951d58f03794a4a380fb`.

Startup now distinguishes missing-file from other errors:

- `os.ErrNotExist`: create an empty in-memory drop table and continue startup.
- malformed JSON or other I/O error: still fatal.

When the table is absent, GiftBox/drop resolution remains fail-closed because the empty table has no entries. No drop IDs, rewards, probabilities, or current-client semantics are invented.

The local variable was renamed to `dropCatalog` to avoid shadowing the `dropTable` type, and the main `handle(...)` wiring passes that same object.

## Verification

GitHub Actions run `35256977155`: SUCCESS.

Windows runner checks:

- source guard patch: PASS
- `go test -c ./cmd/protocol-probe`: PASS
- Windows `go build ./cmd/protocol-probe`: PASS
- package creation: PASS
- artifact upload: PASS

Test package SHA256:

`dc8065c2fbd6da2dc94f3bc8964df42cbe02f4456a0b7ecf082ca34fb0e296a0`

`stage57-test-server.exe` SHA256:

`130285775de457931b923efafabce0fa7e60c341b219da5a13564ed0ec3a9f6b`

## Verification boundary

Not yet proven after Stage57:

- user Windows runtime reaches the TCP listen state,
- latest current Snail client login,
- NPC normal shop open,
- BUY `0x46` item appears in bag,
- currency deduction is visible,
- reconnect persistence.

GM grant and exchange `0x4F` remain out of scope/hold. Normal sell `0x47` remains unimplemented.
