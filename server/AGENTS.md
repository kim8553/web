# Age of Wushu server reconstruction — repository instructions

This repository is the **continuing editable Go implementation descended from the user-supplied
`9yin-go-server1.rar` server source base**. All server fixes, builds, and test-server packages must
be produced from this source lineage (the current Git branch), not from an unrelated/older server
folder or binary package. The original archive is provenance/source baseline; the current branch is
the evolved editable implementation. Neither the archive nor this repository is authority for
official Snail gameplay semantics.

## Runtime/test-package baseline lock

- Do **not** assume or hard-code an old runtime such as `D:\9yin_server`, V37/V45/V46/JYZJ, or any
  other historical package as the server base.
- A runnable test binary must be built from the current `server/cmd/protocol-probe` tree and run
  against the user's actual extracted/evolved `9yin-go-server1` runtime root, supplied explicitly
  through `NINEYIN_SERVER_ROOT` or a verified path.
- Do not infer MySQL install paths, `mysql.env`, database directories, or credentials from historical
  environments. Inspect only the current `9yin-go-server1` runtime/configuration or user-provided
  current evidence.
- If current-client resources are required but are intentionally absent from Git, report that exact
  dependency. Never silently substitute resources from an older server package.

## Authority and evidence rules

- Current-client behavior/protocol authority is limited to the exact current Snail client
  binaries and same-client Lua/resources recorded in `docs/CURRENT_AUTHORITY.md`.
- The exact `fee2df...` Go server binary is a later implementation reference and reverse-
  engineering target. It is not authority for official Snail gameplay semantics.
- Never import gameplay behavior merely because an older V37/V45/V46/JYZJ server did it.
- No guessed wire fields, packet IDs, formulas, timings, target rules, or status effects.
- Mark unproven behavior as unverified; static/build PASS is not live/E2E PASS.
- Preserve 212 as observe-only until current-client/live evidence justifies a semantic handler.

## Development workflow

- Prefer small, evidence-backed changes with tests.
- Keep observer/instrumentation changes non-semantic: no packet mutation, no game-state writes,
  no transport-key mutation, no handler-result changes.
- Record source/protocol provenance for reconstructed files or functions.
- Keep generated binaries, runtime data, logs, and client resources out of Git unless a task
  explicitly requires a versioned fixture.
- Before publication, run `go test ./...` and the Windows amd64 build when the environment
  supports dependency retrieval. Report unavailable checks as unavailable, not PASS.

## Luna / GitHub continuity

The GitHub development-continuity policy used for this project comes from the user's
`kim8553/web` Luna Chat Coder template. When that skill is copied into this repository,
read `.agents/skills/luna-chat-coder/SKILL.md` before repository development work.
Exact Git commit/PR state is durable source truth; preserve unrelated work.
