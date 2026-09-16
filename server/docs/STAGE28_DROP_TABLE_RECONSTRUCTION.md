# Stage 28 — current fee2df drop-table reconstruction

`cmd/protocol-probe/drop_table.go` was absent from the user-supplied editable source baseline.
A semantic reconstruction is now stored at:

`reconstruction/fee2df_overlay/cmd/protocol-probe/drop_table.go.recovered`

It is intentionally excluded from the active build.

## Evidence used

- exact fee2df baseline EXE SHA256 from `docs/CURRENT_AUTHORITY.md`
- current DWARF function signatures, source lines and type layouts
- exact machine code for `loadDropTable` at `0x140305b60`
- exact machine code for `(*dropTable).roll` at `0x140306200`
- inlined DWARF signatures for `(*dropTable).Has` and `entryWeight`
- exact error/constant strings read from referenced virtual addresses
- Go reflection tag strings in the exact binary

Evidence artifacts live in `evidence/stage28/drop_table/`.

## Reconstructed behavior

- JSON maps drop IDs to sections containing Mode, Rolls and Entries.
- empty / `random` mode normalizes to `random`; Rolls <= 0 becomes 1.
- `fixed` mode forces Rolls to 0 and returns every configured entry.
- nil sections, empty entries, blank ConfigID and non-positive Amount are rejected with the
  exact format strings recovered from the binary.
- random mode uses a weighted selection for each roll; Weight <= 0 contributes weight 1.
- output entries initialize only `bagItem.ConfigID` and `bagItem.Amount`; other bag fields are zero.

## Verification boundary

An isolated Go 1.23.2 semantic test covering fixed mode, random defaults and loader validation
passes. This does not prove original source-text identity, full-package build success, or live E2E.
The reconstructed file remains overlay-only until its consumers and dependencies are integrated.
