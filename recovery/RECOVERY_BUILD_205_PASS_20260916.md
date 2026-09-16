# Stage37 Current Recovery — 205-file Production BUILD PASS

Authority branch: `stage37-current-recovery-20260916`
Actions run: `35040325945`
Head commit: `af6bb240cf5b9a2c216685ecf802d1adc77c7723`

## Exact source checkpoint

- source files: 205
- manifest SHA256: `98bfc3ef5c5f022f574ba59320f033bd12329420444fea51ac3fa679ea417926`

## Gates

- exchangecommit: 0 (runtime unit test PASS)
- exchangecommit_race: 0 (runtime race test PASS)
- planner: 0
- migrations: 0
- build: 0
- protocol_compile: 0
- protocol_race_compile: 0
- nonprotocol: 0
- race: 0
- vet: 0
- windows: 0
- protocol_runtime: 125 — NOT RUN because the exact-current resource corpus, including `resources/modern/share/skill/skill_new.ini`, is absent from the GitHub runner. No legacy or synthetic resource was substituted.

## Windows candidate

- SHA256: `0865013eb84e3245c74408ab995883acb6a0648167d5aae954f0b85ac44e71e9`
- PE32+ console x86-64
- Go 1.23.2

## Persistence-first boundary

`internal/exchangecommit` now executes the persistence-first ordering independently of the resource-bound protocol package. Its normal and race tests both PASS. The protocol helper remains dormant and is not wired into C2S `0x4F`.

C2S `0x4F` actual mutation remains OFF. Exact-current `ShowBind` calculation and final persisted result `BindStatus` precedence are still unresolved; no guessed binding rule is enabled.

LIVE/E2E: NOT RUN.
