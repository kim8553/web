# Age of Wushu server reconstruction — repository instructions

This repository is the editable Go-source implementation base. The original user-supplied
`9yin-go-server1` archive is provenance only; do not rewrite history or claim it is the
current Snail server implementation.

## Authority and evidence rules

- **Read `docs/USER_AUTHORITY_AND_WORK_RULES_20260918.md` before further reconstruction:** it pins the user-confirmed `9yin-go-server1.rar` lineage, exact latest user-supplied BIN64 hashes, verified Google Drive `res` folder, working login/map-entry ZIP, DB safety boundary, and the unresolved `FxGameLogic.dll` hash mismatch in the older authority document. Do not silently resolve that mismatch by assuming binary identity.
- Current-client behavior/protocol authority is limited to the exact current Snail client
  binaries and same-client Lua/resources recorded in `docs/CURRENT_AUTHORITY.md`, as
  qualified by the newer verified user-upload hashes in `docs/USER_AUTHORITY_AND_WORK_RULES_20260918.md`.
- The exact `fee2df...` Go server binary is a later implementation reference and reverse-
  engineering target. It is not authority for official Snail gameplay semantics.
- Never import gameplay behavior merely because an older V37/V45/V46/JYZJ server did it.
- No guessed wire fields, packet IDs, formulas, timings, target rules, or status effects.
- Mark unproven behavior as unverified; static/build PASS is not live/E2E PASS.
- Preserve 212 as observe-only until current-client/live evidence justifies a semantic handler.

## Development workflow

- Preserve the user-confirmed working login/map-entry path and existing item/shop systems. Locate needed current-client files proactively in the verified Google Drive `res` folder; confirm each file's content and version before drawing behavioral conclusions. Work on actual confirmed defects, not a replay of previously completed stages.
- Prefer small, evidence-backed changes with tests.
- Keep observer/instrumentation changes non-semantic: no packet mutation, no game-state writes,
  no transport-key mutation, no handler-result changes.
- Record source/protocol provenance for reconstructed files or functions.
- Keep generated binaries, runtime data, logs, and client resources out of Git unless a task
  explicitly requires a versioned fixture.
- Before publication, run `go test ./...` and the Windows amd64 build when the environment
  supports dependency retrieval. Report unavailable checks as unavailable, not PASS.
- Never reset or mutate the user's MySQL schema/data, enable schema migrations, or publish credentials without an audited backup and explicit approval. Use read-only DB inspection first.

## Luna / GitHub continuity

The GitHub development-continuity policy used for this project comes from the user's
`kim8553/web` Luna Chat Coder template. When that skill is copied into this repository,
read `.agents/skills/luna-chat-coder/SKILL.md` before repository development work.
Exact Git commit/PR state is durable source truth; preserve unrelated work.
