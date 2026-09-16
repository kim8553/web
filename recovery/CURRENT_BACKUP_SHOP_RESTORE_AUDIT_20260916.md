# Stage37: server-backup / empty NPC shop evidence audit (2026-09-16)

Status: READ-ONLY BLOCKER AUDIT. **Not** a shop fix, complete recovery, client protocol confirmation, or LIVE/E2E pass. This document contains no account credentials, database contents, or private Drive file identifiers.

## Authority and build boundary

- Source branch at audit start: `stage37-current-recovery-20260916`, HEAD `b63972ad002604dbb43786b948f7cfaf58b319a9`.
- Stage23 Actions run `35067258652`: completed / success. Artifact `10435251412` (`stage37-current-recovery-verification`) downloaded and its ZIP integrity checked: no compressed-data errors.
- The artifact includes `stage37-current-windows-amd64.exe`, SHA256 `6a8982d9e5e113f216ae33aafc8dbd2bf28f05e5520a3a1e129aecb9b2379099`. `windows.exit=0`, `build.exit=0`. This is a build artifact **without the complete resource and database tree**; do not label it a full server package.
- `protocol_runtime.exit=125`; the artifact says `NOT RUN: exact-current resource corpus absent; no legacy substitution.` No actual latest-client session or normal-shop LIVE/E2E validation was performed here.
- The Stage23 wire gate reports `current_client_selector_verified=0` and `wire_gate_status=BLOCKED_NO_EXACT_CURRENT_CLIENT_AUTHORITY_IN_REPO`; keep the legacy `0x46` candidate fail-closed.

## Connected Drive backup: verified individual files and topology

- The backup root contains `build/`, `resources/`, `mysql-server/`, a root startup `.bat`, a lister `.ps1`, and a MySQL configuration file. Folder existence is **not** proof that every dependent file is present or that a restored PC uses this same tree.
- `build/9yin-game-native-menu.exe` was fetched (13,408,256 bytes; Windows x86-64 PE). SHA256: `fee2df844b00e7df07508b9e67f64f0da9df20f00ace09befed790ef28b0d71c`. This is the older authority binary, **not** the Stage23 Windows build.
- `resources/modern/share/trade/shop.ini` was fetched (2,176,439 bytes), SHA256 `ceb7a3a7e17cbc118ce5a7d3d979dea6120c50cdf6a5823697c05a5b1b9cf5dd`. Read-only line scans find 960 `[Shop_...]` section headers and 47,818 numeric-key lines. These counts prove this backup file is not empty, **not** that the server loads or displays its goods correctly.
- Stage23 `regular_shop_audit.log` instead records `exactCurrentShopINISHA256=f42fce7631f7fe302d3fdedad60b12eaa20da12396de412ff0a84edcddc1a6d9` in the exact-current shop *exchange-condition authority* loader. The Drive backup's `shop.ini` does **not** match this authority. This comparison is a version/integrity gap, **not** a demonstrated cause of an empty *normal* NPC shop.
- `resources/modern/share/skill/skill_new.ini` was fetched (1,677,596 bytes), SHA256 `5d729549b916b4b158b3c87d049391ee3082f2bae038233b6761c7ecada8f83e`. The earlier `STAGE37_LIVE_E2E_PACKAGE.md` launcher contract expects SHA256 `c762e7401c5c1bd4ead6d46b7544b983dd6224bbcfee7516e3bb25975db967c0`; this backup does **not** meet that earlier launcher contract.
- `mysql-server/mysql-26.7.0-winx64/bin/mysqld.exe` exists in backup metadata (53,719,168 bytes); a `data/nineyin/` folder also exists. Their binary digest, data consistency, runtime start, and selected character currency were **not** verified. Database directory, certificates/private keys, and credentials must not be bundled into a public ZIP.
- The backed-up startup script sets server root to its own directory and prefers `build/9yin-game-native-menu.exe`, falling back to the root executable. It expects `loopback-lister.ps1` and `resources/modern/share`. This alone cannot establish which executable or resource tree is present on the user's restored PC.

## Immediate blocker and safe next verification order

1. Establish the **exact executable SHA256**, server root, and loaded `resources/modern/share/trade/shop.ini` SHA256 in the *actual restored PC runtime*, independently of Drive metadata. Do not overwrite the existing database.
2. Obtain the exact-current resource version for the build being tested; require a manifest and per-file SHA256 for the necessary tree. Do not mix old backup resources with exact-current anchors as though identical.
3. Read-only trace the normal NPC shop open/list path: NPC-to-shop ID mapping, resource open/parse result, catalog row count, requested page and server listing-frame emission, followed by client display. This evidence is still absent; **the cause of the reported empty shop is unknown**.
4. Read-only inspect the chosen role's persisted currency and the server's loaded wallet representation; no starter-currency grant, DB reset, or direct DB rewrite. A displayed zero balance alone cannot identify the cause.
5. Only after goods display and a funded test condition are independently established, resume passive purchase-wire comparison. Do not promote the `0x46` selector without exact-current client evidence.

No production source, handler, packet selector, grant path, or database was changed by this audit. No complete server ZIP was produced: an artifact containing only a rebuilt PE and test logs is insufficient, and the complete backup resource corpus has not been individually downloaded, SHA-verified, assembled, and runtime-tested in this environment.