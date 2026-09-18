-- READ ONLY: INFORMATION_SCHEMA metadata only. No user records, DDL or DML.
-- Source: shopbuyatomic/checked.go saveChecked() actual INSERT column lists.
-- Stage 2: run only after 01 confirms the target database and table names.
-- Optional values written as NULL by ordinaryShopBagRows must be nullable.
SELECT required.table_name, required.column_name,
       CASE WHEN actual.COLUMN_NAME IS NULL THEN 'MISSING_COLUMN'
            WHEN actual.IS_NULLABLE = 'NO' THEN 'BLOCKER_OPTIONAL_NULL_NOT_ALLOWED'
            ELSE 'NULL_ALLOWED' END AS optional_insert_status,
       actual.COLUMN_TYPE AS column_type,
       actual.IS_NULLABLE AS is_nullable
FROM (
    SELECT 'role_bag_items' AS table_name, 'name' AS column_name
    UNION ALL SELECT 'role_bag_items', 'equip_type'
    UNION ALL SELECT 'role_bag_items', 'art_pack'
    UNION ALL SELECT 'role_bag_items', 'hardiness'
    UNION ALL SELECT 'role_bag_items', 'max_hardiness'
) AS required
LEFT JOIN information_schema.COLUMNS AS actual
  ON actual.TABLE_SCHEMA = DATABASE()
 AND actual.TABLE_NAME = required.table_name
 AND actual.COLUMN_NAME = required.column_name
ORDER BY required.column_name;

-- Extra columns not present in the actual two INSERT lists. A NOT NULL column
-- without a default/auto_increment/generated expression can reject a purchase.
-- REVIEW_EXTRA_COLUMN does NOT mean that its constraints are compatible.
SELECT actual.TABLE_NAME AS table_name, actual.COLUMN_NAME AS column_name,
       actual.COLUMN_TYPE AS column_type, actual.IS_NULLABLE AS is_nullable,
       CASE WHEN actual.COLUMN_DEFAULT IS NULL THEN 'NO_DEFAULT'
            ELSE 'HAS_DEFAULT' END AS default_status,
       actual.EXTRA AS extra,
       CASE WHEN actual.IS_NULLABLE = 'NO'
                  AND actual.COLUMN_DEFAULT IS NULL
                  AND actual.EXTRA NOT LIKE '%auto_increment%'
                  AND actual.EXTRA NOT LIKE '%GENERATED%'
            THEN 'BLOCKER_REQUIRED_COLUMN_NOT_INSERTED'
            ELSE 'REVIEW_EXTRA_COLUMN' END AS insert_compatibility_status
FROM information_schema.COLUMNS AS actual
WHERE actual.TABLE_SCHEMA = DATABASE()
  AND (
    (actual.TABLE_NAME = 'role_bag_items'
     AND actual.COLUMN_NAME NOT IN
       ('role_id','seq','slot','config_id','item_type','amount','view_id',
        'name','equip_type','art_pack','hardiness','max_hardiness'))
    OR
    (actual.TABLE_NAME = 'role_currency'
     AND actual.COLUMN_NAME NOT IN ('role_id','snapshot'))
  )
ORDER BY actual.TABLE_NAME, actual.ORDINAL_POSITION;
