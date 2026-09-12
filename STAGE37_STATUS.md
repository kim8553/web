# Jiuyin Stage37 build status

Authority server EXE SHA256:
`fee2df844b00e7df07508b9e67f64f0da9df20f00ace09befed790ef28b0d71c`

Latest handoff ZIP SHA256:
`56a49fa0abb9b4aa6192a9e2d93f7e780c29e585cf11ccb3ff0cf9f398955fdb`

Verified reconstruction state before GitHub transfer:

- corrected function accounting: 739 / 739
- authored signature audit: 720 MATCH / 0 MISMATCH
- main.handle: MATCH
- LIVE/E2E: NOT RUN
- full BUILD PASS: NOT ESTABLISHED

The local ChatGPT build environment could not download Go modules because outbound DNS/network access was unavailable. A GitHub Actions dependency probe was added at:

`.github/workflows/jiuyin-stage37-network-probe.yml`

The large Stage37 source bundle itself was not committed by the connector because the connector safety layer rejected the large encoded/archive transfer. Do not treat this branch as containing the complete Stage37 source until the source tree is present.

Next build-contract items identified by compile-only probing:

1. npc_catalog current 4-return contract
2. playerActor.activeParry exact field contract
3. ActorState MaxQingGongPoint / MaxQingGongPointAdd
4. clientdata.Int64Value(int64) exact body
5. S2CFacultyMessage exact type/CALL contract
6. JSONRepository.Close current contract

Do not restart Stage1-36 or main.handle reconstruction. Continue from the compile-contract audit once the latest source tree is present.
