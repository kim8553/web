-- READ ONLY: read only schema_migrations metadata, never account/player rows.
-- PRECONDITION: run 01 first; run this only if schema_migrations is a real table.
-- Based on Go migrations.Embedded(): SHA-256 of entire exact migration file.
-- Source migration: migrations/0001_normalize_role_repository.sql (version 1).
-- If this table is absent, DO NOT run this file; record MISSING_TABLE from 01.
SELECT DATABASE() AS inspected_database,
       expected.version AS required_version,
       expected.sha256 AS expected_sha256,
       LOWER(HEX(actual.checksum)) AS observed_sha256,
       CASE WHEN actual.version IS NULL THEN 'MISSING_VERSION'
            WHEN LOWER(HEX(actual.checksum)) <> expected.sha256 THEN 'CHECKSUM_MISMATCH'
            ELSE 'CHECKSUM_MATCH_REFERENCE' END AS ledger_status
FROM (
    SELECT 1 AS version,
           'b5cc5a2f62c05e4efd4d340f86747b589640d24e8fdd4b2193827a027f7df31d' AS sha256
) AS expected
LEFT JOIN schema_migrations AS actual ON actual.version = expected.version;

-- The runner rejects extra ledger versions even when version 1 matches.
-- Ledger versions/checksums are migration metadata, not character rows.
SELECT version AS recorded_version,
       CASE WHEN version = 1 THEN 'EXPECTED_VERSION'
            ELSE 'EXTRA_VERSION_REJECTED_BY_RUNNER' END AS version_status
FROM schema_migrations
ORDER BY version;
