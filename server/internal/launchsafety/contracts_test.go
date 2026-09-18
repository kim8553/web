package launchsafety

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func readServerFile(t *testing.T, name string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "..", name))
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func TestPortableLauncherForcesReadOnlyMigrationPath(t *testing.T) {
	bat := readServerFile(t, "启动本地九阴服务.bat")
	runner := readServerFile(t, filepath.Join("migrations", "runner.go"))
	if !strings.Contains(runner, `os.Getenv("NINEYIN_ALLOW_SCHEMA_MIGRATIONS") != "YES"`) || !strings.Contains(runner, "return runner.VerifyApplied(ctx)") {
		t.Fatal("migration runner no longer proves an unset flag selects VerifyApplied; review launcher first")
	}
	envLoad := strings.Index(bat, `if exist "%ROOT%mysql.env"`)
	forceNo := strings.Index(bat, `set "NINEYIN_ALLOW_SCHEMA_MIGRATIONS=NO"`)
	start := strings.Index(bat, `start "Nine Yin Game 19061"`)
	if envLoad < 0 || forceNo <= envLoad || start <= forceNo {
		t.Fatalf("launcher must load mysql.env then force read-only migration mode before starting; env=%d no=%d start=%d", envLoad, forceNo, start)
	}
	if !strings.Contains(bat, `if /I "%NINEYIN_ALLOW_SCHEMA_MIGRATIONS%"=="YES" (`) || strings.Contains(bat, `if /I not "%NINEYIN_ALLOW_SCHEMA_MIGRATIONS%"=="YES" (`) {
		t.Fatal("launcher must reject explicit YES, not require mutating migration approval")
	}
	if !strings.Contains(bat, `if not defined NINEYIN_MYSQL_DSN (`) {
		t.Fatal("launcher must not silently fall back to JSON storage")
	}
}

func TestShopDBInspectionFileIsMetadataOnly(t *testing.T) {
	sqlText := readServerFile(t, filepath.Join("tools", "check-shop-db-columns-readonly.sql"))
	var lines []string
	for _, line := range strings.Split(sqlText, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "--") {
			continue
		}
		lines = append(lines, line)
	}
	content := strings.Join(lines, "\n")
	if !strings.Contains(content, "information_schema.COLUMNS") || !strings.Contains(content, "actual.TABLE_SCHEMA = DATABASE()") {
		t.Fatal("inspector is not scoped to metadata for current database")
	}
	if regexp.MustCompile(`(?i)\b(create|alter|drop|insert|update|delete|replace|truncate|call|grant|revoke|into|outfile)\b`).MatchString(content) {
		t.Fatal("inspector contains a non-read-only SQL keyword")
	}
	for _, name := range []string{"'max_hardiness'", "'role_bag_items'", "'role_currency'", "'schema_migrations'", "'checksum'"} {
		if !strings.Contains(content, name) {
			t.Fatalf("required source-backed schema term missing: %s", name)
		}
	}
}
