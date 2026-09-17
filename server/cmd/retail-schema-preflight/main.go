// Command retail-schema-preflight reads schema metadata and aggregate row counts.
// It never executes migrations, DDL, DML, or returns player data or credentials.
package main

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/local/9yin-go-server/migrations"
)

type finding struct {
	Check  string `json:"check"`
	Status string `json:"status"`
	Detail string `json:"detail"`
}
type report struct {
	Database string    `json:"database"`
	Status   string    `json:"status"`
	Findings []finding `json:"findings"`
}

func (r *report) add(check, status, detail string) {
	r.Findings = append(r.Findings, finding{check, status, detail})
	if status == "BLOCKED" {
		r.Status = "BLOCKED"
	}
}

var expected = map[string][]string{
	"roles":             {"role_id", "account_id"},
	"role_currency":     {"role_id", "snapshot"},
	"role_bag_items":    {"role_id", "seq", "slot", "config_id", "item_type", "amount", "view_id", "name", "equip_type", "art_pack", "hardiness", "max_hardiness"},
	"schema_migrations": {"version", "checksum"},
}

func missingColumns(found map[string]map[string]bool, table string, required []string) []string {
	var missing []string
	for _, col := range required {
		if !found[table][col] {
			missing = append(missing, col)
		}
	}
	sort.Strings(missing)
	return missing
}
func inspect(ctx context.Context, db *sql.DB) (report, error) {
	r := report{Status: "PASS", Findings: []finding{}}
	if err := db.QueryRowContext(ctx, "SELECT DATABASE()").Scan(&r.Database); err != nil {
		return r, fmt.Errorf("database selection failed")
	}
	if r.Database == "" {
		r.add("database", "BLOCKED", "No database selected")
		return r, nil
	}
	rows, err := db.QueryContext(ctx, `SELECT table_name, COALESCE(engine, '') FROM information_schema.tables WHERE table_schema = DATABASE() AND table_name IN ('roles','role_currency','role_bag_items','schema_migrations')`)
	if err != nil {
		return r, fmt.Errorf("read-only table metadata query failed")
	}
	engines := make(map[string]string)
	for rows.Next() {
		var table, engine string
		if err = rows.Scan(&table, &engine); err != nil {
			rows.Close()
			return r, fmt.Errorf("scan table metadata failed")
		}
		engines[table] = engine
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return r, fmt.Errorf("read table metadata failed")
	}
	cols, err := db.QueryContext(ctx, `SELECT table_name,column_name FROM information_schema.columns WHERE table_schema = DATABASE() AND table_name IN ('roles','role_currency','role_bag_items','schema_migrations')`)
	if err != nil {
		return r, fmt.Errorf("read-only column metadata query failed")
	}
	found := make(map[string]map[string]bool)
	for cols.Next() {
		var table, col string
		if err = cols.Scan(&table, &col); err != nil {
			cols.Close()
			return r, fmt.Errorf("scan column metadata failed")
		}
		if found[table] == nil {
			found[table] = make(map[string]bool)
		}
		found[table][col] = true
	}
	err = cols.Err()
	cols.Close()
	if err != nil {
		return r, fmt.Errorf("read column metadata failed")
	}
	for _, table := range []string{"roles", "role_currency", "role_bag_items", "schema_migrations"} {
		if _, ok := engines[table]; !ok {
			r.add("table."+table, "BLOCKED", "Required table absent")
			continue
		}
		if !strings.EqualFold(engines[table], "InnoDB") {
			r.add("engine."+table, "BLOCKED", "Expected InnoDB; observed "+engines[table])
		} else {
			r.add("engine."+table, "PASS", "InnoDB")
		}
		if missing := missingColumns(found, table, expected[table]); len(missing) != 0 {
			r.add("columns."+table, "BLOCKED", "Missing required columns: "+strings.Join(missing, ", "))
		} else {
			r.add("columns."+table, "PASS", "Required columns present")
		}
	}
	modern := len(missingColumns(found, "roles", expected["roles"])) == 0
	ledger := len(missingColumns(found, "schema_migrations", expected["schema_migrations"])) == 0
	if !modern && found["roles"]["account_key"] {
		r.add("migration.layout", "BLOCKED", "Legacy roles.account_key layout; no migration is executed by this tool")
	}
	if modern && !ledger {
		r.add("migration.layout", "BLOCKED", "Modern roles table but migration ledger is absent or incomplete; startup may attempt legacy migration")
	}
	if ledger {
		embedded, err := migrations.Embedded()
		if err != nil {
			return r, fmt.Errorf("embedded migration metadata failed")
		}
		for _, m := range embedded {
			var raw []byte
			err = db.QueryRowContext(ctx, "SELECT checksum FROM schema_migrations WHERE version = ?", m.Version).Scan(&raw)
			switch {
			case err == sql.ErrNoRows:
				r.add(fmt.Sprintf("migration.%d", m.Version), "BLOCKED", "Expected migration ledger row absent")
			case err != nil:
				return r, fmt.Errorf("read migration ledger failed")
			case len(raw) != sha256.Size || !equalChecksum(raw, m.Checksum):
				r.add(fmt.Sprintf("migration.%d", m.Version), "BLOCKED", "Migration checksum differs from embedded source")
			default:
				r.add(fmt.Sprintf("migration.%d", m.Version), "PASS", "Embedded checksum "+hex.EncodeToString(m.Checksum[:8]))
			}
		}
	}
	if modern && len(missingColumns(found, "role_currency", expected["role_currency"])) == 0 {
		var total, missing int64
		if err = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM roles").Scan(&total); err != nil {
			return r, fmt.Errorf("read aggregate role count failed")
		}
		if err = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM roles r LEFT JOIN role_currency c ON c.role_id = r.role_id WHERE c.role_id IS NULL").Scan(&missing); err != nil {
			return r, fmt.Errorf("read aggregate missing wallet count failed")
		}
		if missing != 0 {
			r.add("currency.coverage", "BLOCKED", fmt.Sprintf("Roles missing persisted currency row: %d of %d", missing, total))
		} else if total == 0 {
			r.add("currency.coverage", "BLOCKED", "No roles exist; currency initialization cannot be verified")
		} else {
			r.add("currency.coverage", "PASS", fmt.Sprintf("Currency rows present for all %d roles", total))
		}
		var invalid int64
		if err = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM role_currency WHERE snapshot IS NULL OR JSON_VALID(snapshot)=0").Scan(&invalid); err != nil {
			return r, fmt.Errorf("read aggregate currency JSON validity failed")
		}
		if invalid != 0 {
			r.add("currency.json", "BLOCKED", fmt.Sprintf("Invalid/null currency snapshots: %d", invalid))
		} else {
			r.add("currency.json", "PASS", "Persisted currency snapshots are valid JSON")
		}
	}
	return r, nil
}
func equalChecksum(raw []byte, expected [32]byte) bool {
	if len(raw) != len(expected) {
		return false
	}
	for i, b := range raw {
		if b != expected[i] {
			return false
		}
	}
	return true
}
func main() {
	dsn := os.Getenv("NINEYIN_MYSQL_DSN")
	if dsn == "" {
		fmt.Fprintln(os.Stderr, "BLOCKED: NINEYIN_MYSQL_DSN is not configured; no database accessed")
		os.Exit(2)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		fmt.Fprintln(os.Stderr, "BLOCKED: cannot initialize MySQL connection (credentials omitted)")
		os.Exit(2)
	}
	defer db.Close()
	if err = db.PingContext(ctx); err != nil {
		fmt.Fprintln(os.Stderr, "BLOCKED: MySQL connection failed (credentials omitted)")
		os.Exit(2)
	}
	r, err := inspect(ctx, db)
	if err != nil {
		fmt.Fprintln(os.Stderr, "BLOCKED: "+err.Error())
		os.Exit(2)
	}
	out, _ := json.MarshalIndent(r, "", "  ")
	fmt.Println(string(out))
	if r.Status != "PASS" {
		os.Exit(2)
	}
}
