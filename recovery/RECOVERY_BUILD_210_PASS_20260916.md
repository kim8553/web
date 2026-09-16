# Stage37 Current Recovery — 210-file Transaction Gate PASS

Authority branch: `stage37-current-recovery-20260916`

Verified GitHub Actions run: `35041631604`
Head commit: `4b7c975d40cf6223807c0a5eeb71d1eefdb646bf`
Artifact ID: `10425965179`
Artifact digest: `sha256:8bf7282b957655c1ef42993b68cc3f8177f84190dc98a03f14c671cbdc80b37d`

## Source integrity

- source files: `210`
- generated manifest SHA256: `99f1b40a4a042e51f8e4c36255710094910aa05ce80853920be26f39c3d3e2dd`

## Transaction safety gates

All of the following exited `0`:

- `exchangegate`
- `exchangegate_race`
- `exchangebinding`
- `exchangebinding_race`
- `exchangecommit`
- `exchangecommit_race`
- `planner`
- `migrations`
- `build`
- `protocol_compile`
- `protocol_race_compile`
- `nonprotocol`
- `race`
- `vet`
- `windows`

`protocol_runtime=125` remains **NOT RUN** because exact-current `resources/modern/share/skill/skill_new.ini` is absent. No synthetic/legacy replacement was used.

## Windows candidate

- SHA256: `acb87df83ad8193027b58c3669d6b87ef112fd141f3fb2940dddb01ec7cfbb22`
- type: `PE32+ executable (console) x86-64, for MS Windows, 15 sections`

## Gate semantics

`internal/exchangegate` allows mutation only when condition support/satisfaction, property support/satisfaction, materials, capacity, final binding, and persistence readiness are all explicitly true. Its zero value is blocked. It collects independent blocking reasons instead of inventing a gameplay precedence rule.

C2S `0x4F` actual mutation remains OFF. LIVE/E2E remains NOT RUN.
