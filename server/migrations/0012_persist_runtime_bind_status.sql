-- migrate:statement
CREATE PROCEDURE nineyin_m12_bind_status()
BEGIN
  IF (SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = DATABASE() AND table_name = 'role_bag_items') <> 1 THEN
    SIGNAL SQLSTATE '45000' SET MESSAGE_TEXT = 'M12 preflight: role_bag_items is required';
  END IF;
  IF (SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = DATABASE() AND table_name = 'role_equip_items') <> 1 THEN
    SIGNAL SQLSTATE '45000' SET MESSAGE_TEXT = 'M12 preflight: role_equip_items is required';
  END IF;
  IF (SELECT COUNT(*) FROM information_schema.columns WHERE table_schema = DATABASE() AND table_name = 'role_bag_items' AND column_name = 'bind_status') = 0 THEN
    ALTER TABLE role_bag_items ADD COLUMN bind_status TINYINT UNSIGNED NOT NULL DEFAULT 0;
  END IF;
  IF (SELECT COUNT(*) FROM information_schema.columns WHERE table_schema = DATABASE() AND table_name = 'role_equip_items' AND column_name = 'bind_status') = 0 THEN
    ALTER TABLE role_equip_items ADD COLUMN bind_status TINYINT UNSIGNED NOT NULL DEFAULT 0;
  END IF;
END

-- migrate:statement
CALL nineyin_m12_bind_status()

-- migrate:statement
DROP PROCEDURE nineyin_m12_bind_status
