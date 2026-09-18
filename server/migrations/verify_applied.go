package migrations

import (
	"context"
	"errors"
	"fmt"
)

// ErrMigrationNotVerified means that the read-only ledger audit did not
// establish an exact match with the migrations embedded in this executable.
// No migration is performed in this case.
var ErrMigrationNotVerified = errors.New("migration: schema not verified; automatic changes disabled")

// VerifyApplied performs only SELECT queries. It does not create a ledger,
// acquire a write lock, or execute any embedded SQL statements. This is an
// immutable-migration ledger check, not a full audit of every application table.
func (runner Runner) VerifyApplied(ctx context.Context) error {
	if runner.DB == nil {
		return fmt.Errorf("%w: nil database", ErrMigrationNotVerified)
	}
	expected, err := Embedded()
	if err != nil {
		return err
	}
	conn, err := runner.DB.Conn(ctx)
	if err != nil {
		return fmt.Errorf("%w: connect: %v", ErrMigrationNotVerified, err)
	}
	defer conn.Close()
	applied, err := loadApplied(ctx, conn)
	if err != nil {
		return fmt.Errorf("%w: read-only migration ledger: %v", ErrMigrationNotVerified, err)
	}
	if len(applied) != len(expected) {
		return fmt.Errorf("%w: ledger has %d versions, binary requires %d", ErrMigrationNotVerified, len(applied), len(expected))
	}
	for _, migration := range expected {
		checksum, ok := applied[migration.Version]
		if !ok {
			return fmt.Errorf("%w: missing version %d", ErrMigrationNotVerified, migration.Version)
		}
		if checksum != migration.Checksum {
			return fmt.Errorf("%w: version %d (%s)", ErrChecksumMismatch, migration.Version, migration.Name)
		}
	}
	return nil
}
