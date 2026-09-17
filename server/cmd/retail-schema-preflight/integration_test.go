//go:build mysql_integration

package main

import (
	"context"
	"database/sql"
	"os"
	"testing"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/local/9yin-go-server/migrations"
)

// This test alone writes DDL/DML, and only to the specifically named CI-only fixture.
// The production retail-schema-preflight executable has SELECT statements only.
func TestRetailSchemaPreflightDisposableMySQL(t *testing.T) {
	if os.Getenv("NINEYIN_STAGE52_DISPOSABLE") != "YES_DROP_STAGE52_TABLES" {
		t.Skip("disposable fixture authorization absent")
	}
	dsn := os.Getenv("NINEYIN_STAGE52_TEST_MYSQL_DSN")
	config, err := mysql.ParseDSN(dsn)
	if err != nil || config.DBName != "jiuyin_stage52" {
		t.Fatal("only the dedicated jiuyin_stage52 fixture is permitted")
	}
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		t.Fatal(err)
	}
	for _, table := range []string{"role_bag_items", "role_currency", "schema_migrations", "roles"} {
		// Table identifiers come only from the hardcoded fixture list above.
		if _, err := db.ExecContext(ctx, "DROP TABLE IF EXISTS "+table); err != nil {
			t.Fatal(err)
		}
	}
	for _, statement := range []string{
		`CREATE TABLE roles (role_id BIGINT UNSIGNED PRIMARY KEY, account_id BIGINT UNSIGNED NOT NULL) ENGINE=InnoDB`,
		`CREATE TABLE role_currency (role_id BIGINT UNSIGNED PRIMARY KEY, snapshot BLOB NOT NULL) ENGINE=InnoDB`,
		`CREATE TABLE role_bag_items (role_id BIGINT UNSIGNED NOT NULL, seq INT NOT NULL, slot INT NOT NULL, config_id VARCHAR(128) NOT NULL, item_type INT NOT NULL, amount INT NOT NULL, view_id INT NOT NULL, name VARCHAR(128) NULL, equip_type VARCHAR(128) NULL, art_pack INT NULL, hardiness INT NULL, max_hardiness INT NULL, PRIMARY KEY(role_id,seq)) ENGINE=InnoDB`,
		`CREATE TABLE schema_migrations (version BIGINT UNSIGNED PRIMARY KEY, checksum BINARY(32) NOT NULL) ENGINE=InnoDB`,
		`INSERT INTO roles(role_id, account_id) VALUES (1,1)`,
		`INSERT INTO role_currency(role_id, snapshot) VALUES (1,'{"silver":100,"gold":5,"silver_card":0,"silver_ticket":0}')`,
	} {
		if _, err := db.ExecContext(ctx, statement); err != nil {
			t.Fatal(err)
		}
	}
	embedded, err := migrations.Embedded()
	if err != nil || len(embedded) == 0 {
		t.Fatalf("embedded migrations unavailable: %v", err)
	}
	for _, migration := range embedded {
		if _, err := db.ExecContext(ctx, `INSERT INTO schema_migrations(version,checksum) VALUES (?,?)`, migration.Version, migration.Checksum[:]); err != nil {
			t.Fatal(err)
		}
	}
	check := func(label, want string) {
		t.Helper()
		r, err := inspect(ctx, db)
		if err != nil || r.Status != want {
			t.Fatalf("%s: expected %s, report=%+v, error=%v", label, want, r, err)
		}
	}
	check("complete fixture", "PASS")
	if _, err := db.ExecContext(ctx, `ALTER TABLE role_bag_items DROP COLUMN max_hardiness`); err != nil {
		t.Fatal(err)
	}
	check("missing bag column", "BLOCKED")
	if _, err := db.ExecContext(ctx, `ALTER TABLE role_bag_items ADD COLUMN max_hardiness INT NULL`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `DELETE FROM schema_migrations WHERE version = ?`, embedded[0].Version); err != nil {
		t.Fatal(err)
	}
	check("missing migration ledger", "BLOCKED")
	if _, err := db.ExecContext(ctx, `INSERT INTO schema_migrations(version,checksum) VALUES (?,?)`, embedded[0].Version, embedded[0].Checksum[:]); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `DELETE FROM role_currency WHERE role_id = 1`); err != nil {
		t.Fatal(err)
	}
	check("uninitialized role currency", "BLOCKED")
	if _, err := db.ExecContext(ctx, `INSERT INTO role_currency(role_id,snapshot) VALUES (1,'{not-json')`); err != nil {
		t.Fatal(err)
	}
	check("corrupt currency JSON", "BLOCKED")
}
