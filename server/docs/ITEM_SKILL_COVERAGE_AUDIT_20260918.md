# Item and skill coverage: source-backed baseline (2026-09-18)

Authority: `9yin-go-server1.rar` lineage, current `stage37-restored-20260918` Go source, and the specific original resource bytes in the isolated test fixture. Do not substitute JYZJ/V37/V46 gameplay, fabricate client packets, or replace the user's resource tree or database.

## Catalog audit: definitions are **not** functional completion

The opt-in `TestItemSkillCoverageAudit` uses the server's own item/equipment/skill loaders, not a second custom INI parser. Run it with `JIUYIN_COVERAGE_AUDIT=1` and a complete, separately mounted `NINEYIN_SERVER_ROOT`; optionally set `JIUYIN_COVERAGE_CSV` to an **unused** path for a per-ID CSV ledger. The test creates the CSV exclusively (never overwrites an existing file), and does not contact MySQL or a client.

Observed on the isolated resource fixture for this task:

| Metric | Measured | Interpretation |
| --- | ---: | --- |
| `tool_item.ini` catalog entries | 76,186 | Parsed definitions, **not** 76,186 working items |
| `equipment.ini` catalog entries | 28,856 | Separate definitions; do not add as unique playable items without deduplication |
| `skill_new.ini` SkillNormal / SkillLock entries | 17,466 | Indexed definitions, **not** implemented skills |
| Level-one combat skill definitions compiled | 4,652 | Source compiler emitted definitions; game execution unverified |
| Level-one skill compile failures | 12,814 | All classified `missing_player_action` in this fixture; some may be NPC-only or otherwise not player-castable |
| Tool entries with `FuncBuffer=buf_ride_yufeng` | 7 | Original resource matches existing ready-buff StaticData `10532`; source dispatch corrected |
| Tool entries with `FuncBuffer=buf_ride_yufeng_jinwu` | 1 | Different resource ready-buff StaticData `31118`; **not** mapped to ordinary effect |
| Tool entries with nonempty `ToolUseEffect` | 1,271 | Metadata presence alone says nothing about an implemented effect; may overlap other handlers |

The accompanying per-ID CSV has 122,508 rows of tool, equipment and indexed skill records. Its `client_live_e2e=not_tested` column is deliberate. `definition_loaded=yes` and `level_one_compilation=compiled` are **not** claims of use, persistence, damage, animations, targeting, or latest-client compatibility. These counts may change with another resource revision.

## Confirmed item-use defect and narrow repair

Previous `applyUseBuffItem` decremented every `FuncBuffer` item **before** checking for the only wired effect. Unknown buffers therefore lost a unit and published an inventory-change frame although the code logged `not wired yet`. Non-potion `ToolUseEffect`-only items similarly passed the consumable gate, then decreased quantity with no effect handler. Both paths now reject use without changing the bag, sending view frames, or invoking bag persistence. This does not implement their unknown effects.

The old wired identifier `buff_ride_yufeng` did not match the original `tool_item.ini`: seven ordinary windrunner items use `buf_ride_yufeng` and reference `WindReadyBuff=buf_ride_yufeng_ready`; `buff_new.ini` assigns that ready buff StaticData `10532`, matching the existing `startYufengReady` code. The handler now recognizes only `buf_ride_yufeng`. The separate `buf_ride_yufeng_jinwu` uses ready buff StaticData `31118` and remains blocked pending actual code/client evidence; no alias, effect, or fallback was invented.

## Verification and limitations

- Regression tests reproduce the two inventory-loss defects on the original handler, then pass after the guard: unknown `FuncBuffer`, non-potion `ToolUseEffect`, known Yufeng quantity/effect/persistence call, original INI/ready-buff identity, top-level `USEITEM` dispatch for `ride_windrunner_001`, and distinct Jinwu refusal. Bag persistence is a spy, **not** live MySQL.
- Selected existing combat-skill compilation, permission, cooldown and targeting tests plus shop bag-move and dual-conflict resync pass on the isolated fixture. `go vet` for the protocol, internal packages and migrations passed, and Windows amd64 compilation passed with VCS stamping disabled for the transferred sandbox checkout.
- The unrestricted `go test ./cmd/protocol-probe ./internal/... ./migrations` run was **not green**: 13 `cmd/protocol-probe` tests failed in the intentionally incomplete fixture. Logs show absent shop, neigong, description, qgskill and scene/NPC creator data; the other listed packages and migrations passed. Do **not** misreport a full suite PASS or infer that the source fix caused these fixture-dependent failures.
- No user's MySQL was accessed, no migrations were permitted, and no existing item/shop implementation was rebuilt. The fixture is a partial collection of Drive-sourced resources; neither a Windows LIVE boot nor latest Snail client game E2E was performed. No overall completion percentage or release date is established.

Follow-up requires sourcing the remaining current-version resource files and comparing the missing action IDs with **actual** player-action definitions and client evidence. For item completion, verify every functional category end-to-end (grant/buy, bag replication, equip/use, effect, persistence, relog). Do not convert catalog counts into a completion percentage.