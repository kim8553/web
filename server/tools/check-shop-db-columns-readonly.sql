-- READ ONLY: metadata for the database explicitly selected by the operator.
-- Only SELECT from information_schema and SELECT DATABASE(); no player rows,
-- credentials, locks, DDL, migration execution, or automatic schema repair.
-- A clean report DOES NOT prove row data, wallet JSON, migrations, gameplay,
-- or actual player-PC DB compatibility. Inspect every REVIEW value manually.
-- Source: mysqlBagStore.Load/Save, shopbuyatomic.SaveCheckedBag,
-- migrations/runner.go and 0001_normalize_role_repository.sql.

SELECT DATABASE() AS inspected_database,
       CASE WHEN DATABASE() IS NULL THEN 'NO_DATABASE_SELECTED'
            ELSE 'METADATA_ONLY_NOT_LIVE_VERIFIED' END AS inspection_scope;

-- Every participating table must be a BASE TABLE using transactional InnoDB:
-- the purchase transaction locks a role, checks wallet/bag, and rewrites both.
SELECT required.table_name,
       CASE WHEN actual.TABLE_NAME IS NULL THEN 'MISSING_TABLE'
            WHEN actual.TABLE_TYPE <> 'BASE TABLE' THEN 'NOT_BASE_TABLE'
            WHEN actual.ENGINE <> 'InnoDB' OR actual.ENGINE IS NULL THEN 'REVIEW_NON_INNODB_ENGINE'
            ELSE 'INNODB' END AS engine_status,
       actual.TABLE_TYPE AS actual_table_type,
       actual.ENGINE AS actual_engine
FROM (
    SELECT 'roles' AS table_name
    UNION ALL SELECT 'role_bag_items'
    UNION ALL SELECT 'role_currency'
    UNION ALL SELECT 'schema_migrations'
) AS required
LEFT JOIN information_schema.TABLES AS actual
  ON actual.TABLE_SCHEMA = DATABASE() AND actual.TABLE_NAME = required.table_name
ORDER BY required.table_name;

-- Report exact MySQL column metadata, not just presence. MATCH_REFERENCE means
-- matching the existing Go migration or disposable MySQL test declaration,
-- NOT proof that all real values fit or all other schema constraints work.
SELECT required.table_name, required.column_name, required.expected_kind,
       CASE WHEN actual.COLUMN_NAME IS NULL THEN 'MISSING_COLUMN'
            WHEN required.expected_kind = 'unsigned_bigint' AND
                 actual.DATA_TYPE = 'bigint' AND actual.COLUMN_TYPE LIKE '%unsigned%'
                 THEN 'MATCH_REFERENCE'
            WHEN required.expected_kind = 'integer' AND actual.DATA_TYPE IN
                 ('tinyint','smallint','mediumint','int','bigint')
                 THEN 'INTEGER_REVIEW_RANGE_AND_SIGN'
            WHEN required.expected_kind = 'text' AND actual.DATA_TYPE IN
                 ('char','varchar','tinytext','text','mediumtext','longtext')
                 THEN 'TEXT_REVIEW_CAPACITY_AND_CHARSET'
            WHEN required.expected_kind = 'json' AND actual.DATA_TYPE = 'json'
                 THEN 'MATCH_REFERENCE'
            WHEN required.expected_kind = 'json' AND actual.DATA_TYPE IN
                 ('char','varchar','tinytext','text','mediumtext','longtext')
                 THEN 'TEXT_JSON_VALIDITY_UNCHECKED'
            WHEN required.expected_kind = 'binary32' AND
                 actual.DATA_TYPE = 'binary' AND actual.CHARACTER_MAXIMUM_LENGTH = 32
                 THEN 'MATCH_REFERENCE'
            ELSE 'REVIEW_TYPE_MISMATCH' END AS type_status,
       actual.DATA_TYPE AS actual_data_type,
       actual.COLUMN_TYPE AS actual_column_type,
       actual.IS_NULLABLE AS actual_is_nullable,
       actual.CHARACTER_MAXIMUM_LENGTH AS actual_character_maximum_length,
       actual.CHARACTER_SET_NAME AS actual_character_set,
       actual.COLLATION_NAME AS actual_collation,
       actual.COLUMN_KEY AS actual_column_key,
       actual.EXTRA AS actual_extra
FROM (
    SELECT 'roles' AS table_name, 'role_id' AS column_name, 'unsigned_bigint' AS expected_kind
    UNION ALL SELECT 'role_bag_items', 'role_id', 'unsigned_bigint'
    UNION ALL SELECT 'role_bag_items', 'seq', 'integer'
    UNION ALL SELECT 'role_bag_items', 'slot', 'integer'
    UNION ALL SELECT 'role_bag_items', 'config_id', 'text'
    UNION ALL SELECT 'role_bag_items', 'item_type', 'integer'
    UNION ALL SELECT 'role_bag_items', 'amount', 'integer'
    UNION ALL SELECT 'role_bag_items', 'view_id', 'integer'
    UNION ALL SELECT 'role_bag_items', 'name', 'text'
    UNION ALL SELECT 'role_bag_items', 'equip_type', 'text'
    UNION ALL SELECT 'role_bag_items', 'art_pack', 'integer'
    UNION ALL SELECT 'role_bag_items', 'hardiness', 'integer'
    UNION ALL SELECT 'role_bag_items', 'max_hardiness', 'integer'
    UNION ALL SELECT 'role_currency', 'role_id', 'unsigned_bigint'
    UNION ALL SELECT 'role_currency', 'snapshot', 'json'
    UNION ALL SELECT 'schema_migrations', 'version', 'unsigned_bigint'
    UNION ALL SELECT 'schema_migrations', 'checksum', 'binary32'
) AS required
LEFT JOIN information_schema.COLUMNS AS actual
  ON actual.TABLE_SCHEMA = DATABASE()
 AND actual.TABLE_NAME = required.table_name
 AND actual.COLUMN_NAME = required.column_name
ORDER BY required.table_name, required.column_name;

-- Compare PRIMARY KEY columns in their actual index order. The roles row
-- lock needs role_id; a bag rewrite expects unique (role_id, seq); wallet
-- rewrite expects unique role_id; the migration ledger keys by version.
SELECT required.table_name, required.expected_primary_key,
       COALESCE(keys_found.actual_primary_key, '(none)') AS actual_primary_key,
       CASE WHEN actual.TABLE_NAME IS NULL THEN 'MISSING_TABLE'
            WHEN keys_found.actual_primary_key IS NULL THEN 'MISSING_PRIMARY_KEY'
            WHEN keys_found.actual_primary_key <> required.expected_primary_key
                 THEN 'DIFFERENT_PRIMARY_KEY'
            ELSE 'EXPECTED_PRIMARY_KEY' END AS primary_key_status
FROM (
    SELECT 'roles' AS table_name, 'role_id' AS expected_primary_key
    UNION ALL SELECT 'role_bag_items', 'role_id,seq'
    UNION ALL SELECT 'role_currency', 'role_id'
    UNION ALL SELECT 'schema_migrations', 'version'
) AS required
LEFT JOIN information_schema.TABLES AS actual
  ON actual.TABLE_SCHEMA = DATABASE() AND actual.TABLE_NAME = required.table_name
LEFT JOIN (
    SELECT TABLE_NAME, GROUP_CONCAT(COLUMN_NAME ORDER BY SEQ_IN_INDEX SEPARATOR ',') AS actual_primary_key
    FROM information_schema.STATISTICS
    WHERE TABLE_SCHEMA = DATABASE() AND INDEX_NAME = 'PRIMARY'
    GROUP BY TABLE_NAME
) AS keys_found ON keys_found.TABLE_NAME = required.table_name
ORDER BY required.table_name;

-- Other index metadata is included for manual investigation, not an
-- assertion that some undocumented extra index is required or sufficient.
SELECT TABLE_NAME AS table_name, INDEX_NAME AS index_name,
       NON_UNIQUE AS non_unique, SEQ_IN_INDEX AS seq_in_index,
       COLUMN_NAME AS column_name, INDEX_TYPE AS index_type,
       SUB_PART AS prefix_length
FROM information_schema.STATISTICS
WHERE TABLE_SCHEMA = DATABASE()
  AND TABLE_NAME IN ('roles','role_bag_items','role_currency','schema_migrations')
ORDER BY TABLE_NAME, INDEX_NAME, SEQ_IN_INDEX;
