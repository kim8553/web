-- READ-ONLY metadata inspection for the existing database selected by the operator.
-- No CREATE/ALTER/DROP/INSERT/UPDATE/DELETE, no player rows or credentials read.
-- Required columns are sourced from cmd/protocol-probe/zz_recovered_overlay.go
-- mysqlBagStore.Load/Save, internal/shopbuyatomic/checked.go, and
-- migrations/runner.go loadApplied. A zero-row MISSING result is NOT proof
-- of complete DB compatibility, correct types/indexes, or live functionality.
SELECT DATABASE() AS inspected_database;
SELECT required.table_name, required.column_name,
       CASE WHEN actual.COLUMN_NAME IS NULL THEN 'MISSING' ELSE 'PRESENT' END AS column_status
FROM (
    SELECT 'roles' AS table_name, 'role_id' AS column_name
    UNION ALL SELECT 'role_bag_items', 'role_id'
    UNION ALL SELECT 'role_bag_items', 'seq'
    UNION ALL SELECT 'role_bag_items', 'slot'
    UNION ALL SELECT 'role_bag_items', 'config_id'
    UNION ALL SELECT 'role_bag_items', 'item_type'
    UNION ALL SELECT 'role_bag_items', 'amount'
    UNION ALL SELECT 'role_bag_items', 'view_id'
    UNION ALL SELECT 'role_bag_items', 'name'
    UNION ALL SELECT 'role_bag_items', 'equip_type'
    UNION ALL SELECT 'role_bag_items', 'art_pack'
    UNION ALL SELECT 'role_bag_items', 'hardiness'
    UNION ALL SELECT 'role_bag_items', 'max_hardiness'
    UNION ALL SELECT 'role_currency', 'role_id'
    UNION ALL SELECT 'role_currency', 'snapshot'
    UNION ALL SELECT 'schema_migrations', 'version'
    UNION ALL SELECT 'schema_migrations', 'checksum'
) AS required
LEFT JOIN information_schema.COLUMNS AS actual
    ON actual.TABLE_SCHEMA = DATABASE()
   AND actual.TABLE_NAME = required.table_name
   AND actual.COLUMN_NAME = required.column_name
ORDER BY required.table_name, required.column_name;
