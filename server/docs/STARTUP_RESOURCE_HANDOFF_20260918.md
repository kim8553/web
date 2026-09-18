# 2026-09-18 startup resource repair handoff

Authority: original `9yin-go-server1.rar` lineage; editable source on `stage37-restored-20260918`. Do not replace current Go logic with old RAR binaries, JYZJ, V37/V46 or old client DLLs. Latest user BIN64 ZIP was separately hashed; original and client bytes were not modified.

## Verified code paths / first actual failure

- `server/cmd/protocol-probe/runtime_paths.go`: `NINEYIN_SERVER_ROOT` takes precedence; otherwise executable directory/current directory and ancestors are inspected. A nested `9yin-go-server1/9yin-go-server1` installation is not intrinsically a path bug: launcher location and `resources/modern/share` must be compared.
- `server/cmd/protocol-probe/zz_recovered_overlay.go`: `loadEquipCatalog` reads `resources/modern/share/ini/effect/playerweapon` and startup fails if the folder is absent; the 2026-09-18 user log shows this failure before GM/game listeners exist.
- `server/cmd/protocol-probe/main.go`: an unset `NINEYIN_MYSQL_DSN` selects legacy JSON storage. A configured MySQL connection calls `migrations.Runner.Up` BEFORE item/equipment/string/drop resource catalog loading. Resource failure could therefore follow DB migration if explicitly authorized. Never run this against a user's DB without independent schema review, backup and explicit approval.
- `resources/modern/share/map/path` absence in that user log caused an NPC patrol fallback and was not the fatal startup error. No fabricated folder is needed.

## Confirmed Drive backups (not proof of latest-client compatibility)

- `resources/modern/share/ini/effect/playerweapon`: 16 original INIs from https://drive.google.com/drive/folders/1EQVXknAw8CStJ4skVkp7OsIYvJ3DYotq ; modified 2023-07-10. The 16-file source package and its per-file SHA256 manifest are also retained separately.
- `resources/modern/share/item/drop_table.json`: 1354 bytes, SHA256 `1772db61dcfa37f907e0759d9ccb07ab865aaaa8cdd1109b38ff265d69d9975b`, modified 2026-08-19, https://drive.google.com/file/d/15hW7bXdtECApFExiXDsWsDWVg6JFdk2n . JSON parsed and root section keys inspected; runtime loader integration not run.
- `resources/modern/text/stringname.idres`: 6,830,432 bytes, SHA256 `e6a4f1f5493e97903b0e306c83f10ca8c03afff43c4ef0acef5b7e216c183aaa`, modified 2026-08-16, https://drive.google.com/file/d/1mSbI6IGQf-U3P7WzWyqMH0hLztWYgy8v . Decoded as UTF-8 with BOM; runtime loader integration not run.
- All three paths were located by traversing actual `9yin_server/resources/modern` subfolders: a filename search alone failed to surface the last two. Do not report that these files are absent.
- Other original resource paths are found in https://drive.google.com/drive/folders/1MsjhX8VQRJzEo3ihdT7OCojwEmU8lEvJ , subject to per-file version and loader checks. Do not copy an entire historical resource tree into a current installation without verification.

Local 18-file candidate resource ZIP: `JIUYIN_RUNTIME_RESOURCES_DRIVE_BACKUP_20260918.zip`, SHA256 `eafc00d089bfb271995ff29869463fd22e71ae04366b09ef5069f3d22f929f71` (provided in the project conversation, not tracked in Git). Includes relative paths, per-file SHA256 manifest and `INSTALL_MISSING_ONLY.ps1` (default read-only; no overwrite). Does **not** contain the executable, DB, `mysql.env`, user `data`, BIN64, all required runtime resources, or proof of matching current client.

## GitHub changes and verification

- `server/verify-runtime-resources.ps1`: read-only check of known Go loader paths and optional MySQL DSN requirement. Neither creates dummy data nor contacts DB.
- `server/启动本地九阴服务.bat`: resolves root from its own location, reads existing local `mysql.env` without printing its DSN, rejects accidental JSON fallback, invokes preflight before DB gate. It refuses MySQL startup unless migration authorization has been separately obtained. Existing user data/resources are not overwritten by this launcher.
- `.github/workflows/stage37-startup-resource-preflight.yml`: Windows PowerShell syntax, nested-root missing-resource diagnostics, launcher gate order, isolated MySQL purchase regression, Go vet and Windows amd64 compilation.
- CI for source commit `fc45288e6962ff38e683281e651e78ca21c0a5e8`: https://github.com/kim8553/web/actions/runs/35329950645 , completed SUCCESS. Specifically: Windows missing-resource negative test and script checks PASS; isolated dual-conflict real MySQL test PASS; unit purchase tests PASS; vet PASS; Win64 binary compilation PASS. Bag/wallet real MySQL tests were SKIPPED in that particular run; the CI does not establish real game server boot or live integration.

## 2026-09-18 continuation: default read-only migration gate (source commit `5ccdc2666423f3bbf63736c518e2d97a926dca95`)

- `server/migrations/runner.go`: an executable launched directly, bypassing the `.bat`, now calls `VerifyApplied` instead of automatic `Runner.Up` DDL unless `NINEYIN_ALLOW_SCHEMA_MIGRATIONS` is exactly `YES`. Do **not** set that flag in a user's existing installation without separate approval; with `YES`, the original mutating migration path still exists and runs before resources load.
- `server/migrations/verify_applied.go`: only SELECTs existing `schema_migrations`; checks that applied versions and SHA256 checksums exactly match the embedded migrations. Missing ledger, missing version, extra version, or checksum mismatch stops startup without running SQL migrations. A matching ledger is NOT proof that every live table and data row actually matches or is compatible with current Go runtime.
- `server/migrations/runner_test.go`, `verify_applied_test.go`, `verify_applied_mysql_test.go`: original lock tests explicitly opt into the approved path. New sqlmock regression tests assert no DDL and fail-closed behavior. The real MySQL test only runs with `JIUYIN_TEST_MIGRATION_MYSQL_CI=1` and a disposable CI DSN and confirms a missing ledger stays missing after startup attempt; it never uses the user's `NINEYIN_MYSQL_DSN`.
- Source commit CI: https://github.com/kim8553/web/actions/runs/35331200600 , completed SUCCESS. All 8 migration tests passed, including the real disposable MySQL no-ledger/no-creation test. Isolated dual-conflict NPC purchase MySQL test passed, mock/unit purchase tests passed, Go vet and Windows amd64 compilation passed, Windows resource negative tests passed. Other bag-only and wallet-only real MySQL tests were SKIPPED in this run; do not call them PASS.

## Still blocked / do not misstate

1. The resource ZIP has not been proven byte-version compatible with the latest Snail client. Complete startup resource inventory and loader parsing of the backups remain unverified.
2. No verified Windows server startup with these resource bytes, HTTP listener 19062 or game TCP listener 19061. No user PC MySQL verification, no latest-client LIVE/E2E shop purchase.
3. No read-only audit of the user's full table schema or `schema_migrations` ledger; nothing was executed against that database. Explicitly authorized `YES` retains mutating SQL and the pre-resource timing risk, so do not ask the user to set the flag. The portable batch currently still refuses startup without separate approval, including a ledger-matching read-only path.
4. Prior NPC shop purchase gameplay implementation remains intact. No NPC selling, GM item grants, or exchange changes are authorized by this startup task.
