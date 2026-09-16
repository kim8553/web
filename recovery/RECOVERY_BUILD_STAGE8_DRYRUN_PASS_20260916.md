# Stage37 Current Recovery — Stage8 0x4F Dry-run Gate Wiring PASS

Authority branch: `stage37-current-recovery-20260916`

Verified GitHub Actions run: `35041998838`
Head commit: `3b2f28649dce4a5a2cf3b603471452e9964c2eb0`
Artifact ID: `10425313914`
Artifact digest: `sha256:80c6adbedcf66ce3e2d6275b4fedd0469975991e72f3b5f34b1b4ceb323b7f83`

## Source integrity

- source files: `210`
- manifest SHA256: `337555d75a71cd347f51aae2361c29fbda47f4a915bdd55c7471a947fc1f68da`

## Gates

`exchangegate`, `exchangegate_race`, `exchangebinding`, `exchangebinding_race`, `exchangecommit`, `exchangecommit_race`, `planner`, `migrations`, `build`, `protocol_compile`, `protocol_race_compile`, `nonprotocol`, `race`, `vet`, and `windows` all exited `0`.

`protocol_runtime=125` remains NOT RUN because exact-current `resources/modern/share/skill/skill_new.ini` is absent. No legacy/synthetic substitution was used.

## Windows candidate

SHA256: `752cad6a3218edc5f6d6f0f7469c385af8c5f614ebc23d5f0de837e8184587fd`

## 0x4F boundary

The real current-shop 0x4F handler now evaluates/logs the transaction gate in dry-run mode. The final binding decision is deliberately the zero-value Unknown decision, so mutation remains blocked. Stage8 CI also fails if the handler contains calls to `stageShopExchangeAtomicReplacement` or `commitShopExchangeReplacementPersistenceFirst`.

Actual inventory mutation remains OFF. LIVE/E2E remains NOT RUN.
