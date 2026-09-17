# Stage56 — optional string-name startup gate

## Authority and lineage

- Editable server lineage remains the user-supplied `9yin-go-server1.rar` -> current `kim8553/web` Go source.
- Branch: `stage37-assistant-handoff-20260917`.
- Source fix commit: `97f71463d99a5980a4d8dd1c4dba49cc44683120`.
- Latest Snail current client/resources remain protocol/gameplay authority. No proprietary client resource bytes are committed here.

## User runtime evidence that triggered Stage56

Stage55 passed the previous missing `resources/modern/share/ini/effect/playerweapon` startup gate, then the user's Windows runtime exited at:

`load string names: open .../resources/modern/text/stringname.idres: The system cannot find the file specified.`

This means Stage55 was not a successful server boot and no gameplay E2E conclusion can be drawn from it.

## Current-source usage proof

The Stage56 Windows workflow enumerated all current `protocol-probe` references to `stringNames`, `loadStringNames`, `defaultStringNameINI`, and `gmDisplayName` before patching. The loaded map is passed to `serveGM` and used by `gmDisplayName` for GM catalog display labels. No normal shop BUY `0x46` gameplay/persistence path consumes the map.

Therefore a missing localization file must not prevent the current normal-retail test server from starting. This does not prove that the localization resource is unimportant to the final project; it only proves that it is not a runtime prerequisite for the current BUY test path.

## Current-resource cross-check

Private read-only analysis of the authenticated latest Drive `text.package` (`PCK0`) found one unique zlib payload matching several NPC-name keys. Evidence only, no resource bytes are published:

- decompressed size: `2,473,572` bytes
- parsed non-comment `key=value` entries: about `87,913`
- SHA256: `6104223891b746cfedbe98a17a56e0237a5e8e85516b6956188c1b19c4e431e5`
- package offset of that zlib stream: `21,850,335`

A separate historical text segmentation workbook in the connected Drive labels `stringname` as the NPC-name text segment. This corroborates the payload classification, but it is not treated as proof of the current encrypted PCK filename index. The server does not bundle or publish this payload.

## Source change

`server/cmd/protocol-probe/main.go` now handles `loadStringNames(defaultStringNameINI)` as follows:

- `os.ErrNotExist`: create an empty display-name map, log a warning, continue startup.
- any other error: retain fail-closed `log.Fatalf` behavior.
- file present and readable: preserve the normal loader and success log.

This is deliberately narrower than silently ignoring all localization failures.

## Verification

GitHub Actions run `35255541068`:

- string-name reference inventory: PASS
- source patch + gofmt: PASS
- `go test -c ./cmd/protocol-probe` on Windows: PASS
- Windows `go build ./cmd/protocol-probe`: PASS
- source commit/push: PASS
- Stage56 Windows package creation: PASS

Package:

- `STAGE56_9YIN_GO_SERVER1_TEST_SERVER_WINDOWS_20260918.zip`
- SHA256: `5898fbdc5ff63abe040f84d91b771d85854c1909401289d09556779addf33491`
- `stage56-test-server.exe` SHA256: `0699ba1b3c2a35b7afa42b4f47d8a0a288e253ed93cee43bda0a6ea1d9add329`

## Honest verification boundary

Not yet verified after this change:

- user's Windows Stage56 process remains running after all startup initialization,
- latest Snail client login,
- NPC shop open,
- normal BUY `0x46` live dispatch,
- bag display update,
- currency deduction,
- reconnect persistence.

The next required evidence is the user's Stage56 startup console. Only after the server remains running should current-client BUY E2E be attempted.

GM item grant and exchange `0x4F` remain on HOLD. Normal SELL `0x47` remains unimplemented.
