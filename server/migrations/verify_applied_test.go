package migrations

import (
	"context"
	"crypto/sha256"
	"errors"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestUnapprovedStartupVerifiesExistingLedgerWithoutDDL(t *testing.T) {
	t.Setenv("NINEYIN_ALLOW_SCHEMA_MIGRATIONS", "")
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	all, err := Embedded()
	if err != nil {
		t.Fatal(err)
	}
	rows := sqlmock.NewRows([]string{"version", "checksum"})
	for _, m := range all {
		rows.AddRow(m.Version, m.Checksum[:])
	}
	mock.ExpectQuery(regexp.QuoteMeta("SELECT version, checksum FROM schema_migrations ORDER BY version")).WillReturnRows(rows)
	if err := (Runner{DB: db}).Up(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestUnapprovedStartupRejectsMissingLedgerWithoutDDL(t *testing.T) {
	t.Setenv("NINEYIN_ALLOW_SCHEMA_MIGRATIONS", "")
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	mock.ExpectQuery(regexp.QuoteMeta("SELECT version, checksum FROM schema_migrations ORDER BY version")).
		WillReturnError(errors.New("schema_migrations does not exist"))
	err = (Runner{DB: db}).Up(context.Background())
	if !errors.Is(err, ErrMigrationNotVerified) {
		t.Fatalf("error = %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestUnapprovedStartupRejectsIncompleteLedgerWithoutDDL(t *testing.T) {
	t.Setenv("NINEYIN_ALLOW_SCHEMA_MIGRATIONS", "")
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	mock.ExpectQuery(regexp.QuoteMeta("SELECT version, checksum FROM schema_migrations ORDER BY version")).
		WillReturnRows(sqlmock.NewRows([]string{"version", "checksum"}))
	err = (Runner{DB: db}).Up(context.Background())
	if !errors.Is(err, ErrMigrationNotVerified) {
		t.Fatalf("error = %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestUnapprovedStartupRejectsChecksumDriftWithoutDDL(t *testing.T) {
	t.Setenv("NINEYIN_ALLOW_SCHEMA_MIGRATIONS", "")
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	all, err := Embedded()
	if err != nil {
		t.Fatal(err)
	}
	wrong := sha256.Sum256([]byte("wrong"))
	rows := sqlmock.NewRows([]string{"version", "checksum"})
	for _, m := range all {
		rows.AddRow(m.Version, wrong[:])
	}
	mock.ExpectQuery(regexp.QuoteMeta("SELECT version, checksum FROM schema_migrations ORDER BY version")).WillReturnRows(rows)
	err = (Runner{DB: db}).Up(context.Background())
	if !errors.Is(err, ErrChecksumMismatch) {
		t.Fatalf("error = %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
