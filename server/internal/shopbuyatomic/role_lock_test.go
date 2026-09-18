package shopbuyatomic

import (
    "context"
    "errors"
    "strings"
    "testing"

    "github.com/DATA-DOG/go-sqlmock"
)

func TestLockRoleForBagWrite(t *testing.T) {
    for _, tc := range []struct {
        name string
        row bool
        queryErr error
        wantErr bool
    }{
        {name: "valid existing role", row: true},
        {name: "missing role", wantErr: true},
        {name: "read error", queryErr: errors.New("database unavailable"), wantErr: true},
    } {
        t.Run(tc.name, func(t *testing.T) {
            db, mock, err := sqlmock.New()
            if err != nil { t.Fatal(err) }
            defer db.Close()
            mock.ExpectBegin()
            query := mock.ExpectQuery("SELECT role_id FROM roles WHERE role_id = ").WithArgs(uint64(7))
            if tc.queryErr != nil {
                query.WillReturnError(tc.queryErr)
            } else {
                rows := sqlmock.NewRows([]string{"role_id"})
                if tc.row { rows.AddRow(uint64(7)) }
                query.WillReturnRows(rows)
            }
            mock.ExpectRollback()
            tx, err := db.Begin()
            if err != nil { t.Fatal(err) }
            err = LockRoleForBagWrite(context.Background(), tx, 7)
            if (err != nil) != tc.wantErr { t.Fatalf("lock err=%v wantErr=%v", err, tc.wantErr) }
            if tc.wantErr && !strings.Contains(err.Error(), "lock role") { t.Fatalf("missing useful error: %v", err) }
            if err := tx.Rollback(); err != nil { t.Fatal(err) }
            if err := mock.ExpectationsWereMet(); err != nil { t.Fatal(err) }
        })
    }
}

func TestLockRoleForBagWriteRejectsInvalidArguments(t *testing.T) {
    if err := LockRoleForBagWrite(context.Background(), nil, 7); err == nil { t.Fatal("nil transaction accepted") }
}
