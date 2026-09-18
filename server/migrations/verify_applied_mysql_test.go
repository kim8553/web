package migrations

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"testing"

	_ "github.com/go-sql-driver/mysql"
)

// Run exclusively against the disposable CI MySQL service. Never enable this
// test using an existing installation or a user's NINEYIN_MYSQL_DSN.
func TestUnapprovedStartupMissingLedgerDoesNotCreateTablesRealMySQL(t *testing.T) {
	if os.Getenv("JIUYIN_TEST_MIGRATION_MYSQL_CI") != "1" {
		t.Skip("disposable migration MySQL CI not enabled")
	}
	dsn := os.Getenv("JIUYIN_TEST_SHOP_MYSQL_DSN")
	if dsn == "" {
		t.Fatal("disposable CI DSN is not configured")
	}
	t.Setenv("NINEYIN_ALLOW_SCHEMA_MIGRATIONS", "")
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	ctx := context.Background()
	if err := db.PingContext(ctx); err != nil {
		t.Fatal(err)
	}
	const ledgerCount = `SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = DATABASE() AND table_name = 'schema_migrations'`
	var before int
	if err := db.QueryRowContext(ctx, ledgerCount).Scan(&before); err != nil {
		t.Fatal(err)
	}
	if before != 0 {
		t.Fatal("expected a disposable database without a migration ledger")
	}
	if err := (Runner{DB: db}).Up(ctx); !errors.Is(err, ErrMigrationNotVerified) {
		t.Fatalf("unapproved migration error = %v, want ErrMigrationNotVerified", err)
	}
	var after int
	if err := db.QueryRowContext(ctx, ledgerCount).Scan(&after); err != nil {
		t.Fatal(err)
	}
	if after != 0 {
		t.Fatalf("unapproved startup created migration ledger: before=%d after=%d", before, after)
	}
}
