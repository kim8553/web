package main

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"sync"
	"testing"
)

// A stdlib-only fake driver verifies one transaction and injected rollback.
// These tests do not emulate a real MySQL engine or client connection.
type normalShopFakeState struct {
	currency         normalShopPersistedCurrency
	bag              []normalShopPersistedBagRow
	noCurrency       bool
	failOn           string
	zeroCurrencyRows bool
	events           []string
	commit, rollback bool
}

var normalShopFakeOnce sync.Once
var normalShopFakeCurrent *normalShopFakeState

type normalShopFakeDriver struct{}

func (normalShopFakeDriver) Open(string) (driver.Conn, error) {
	return &normalShopFakeConn{state: normalShopFakeCurrent}, nil
}

type normalShopFakeConn struct{ state *normalShopFakeState }

func (*normalShopFakeConn) Prepare(string) (driver.Stmt, error) {
	return nil, errors.New("prepared statements not used in fake")
}
func (c *normalShopFakeConn) Close() error { return nil }
func (c *normalShopFakeConn) Begin() (driver.Tx, error) {
	c.state.events = append(c.state.events, "BEGIN")
	return &normalShopFakeTx{state: c.state}, nil
}
func (c *normalShopFakeConn) ExecContext(_ context.Context, query string, _ []driver.NamedValue) (driver.Result, error) {
	s := c.state
	kind := "OTHER"
	switch {
	case strings.HasPrefix(query, "DELETE FROM role_bag_items"):
		kind = "DELETE_BAG"
	case strings.HasPrefix(query, "INSERT INTO role_bag_items"):
		kind = "INSERT_BAG"
	case strings.HasPrefix(query, "UPDATE role_currency"):
		kind = "UPDATE_CURRENCY"
	}
	s.events = append(s.events, kind)
	if s.failOn == kind {
		return nil, errors.New("injected " + kind + " failure")
	}
	if kind == "UPDATE_CURRENCY" && s.zeroCurrencyRows {
		return driver.RowsAffected(0), nil
	}
	return driver.RowsAffected(1), nil
}
func (c *normalShopFakeConn) QueryContext(_ context.Context, query string, _ []driver.NamedValue) (driver.Rows, error) {
	s := c.state
	if strings.HasPrefix(query, "SELECT snapshot FROM role_currency") {
		s.events = append(s.events, "LOCK_CURRENCY")
		if s.failOn == "LOCK_CURRENCY" {
			return nil, errors.New("injected lock failure")
		}
		var records [][]driver.Value
		if !s.noCurrency {
			encoded, _ := json.Marshal(s.currency)
			records = [][]driver.Value{{encoded}}
		}
		return &normalShopFakeRows{columns: []string{"snapshot"}, values: records}, nil
	}
	if strings.HasPrefix(query, "SELECT config_id, item_type") {
		s.events = append(s.events, "LOCK_BAG")
		if s.failOn == "LOCK_BAG" {
			return nil, errors.New("injected bag lock failure")
		}
		var records [][]driver.Value
		for _, r := range s.bag {
			strVal := func(x string) driver.Value {
				if x == "" {
					return nil
				}
				return x
			}
			intVal := func(x int32) driver.Value {
				if x == 0 {
					return nil
				}
				return int64(x)
			}
			records = append(records, []driver.Value{r.ConfigID, int64(r.ItemType), int64(r.Amount), int64(r.ViewID), strVal(r.Name), strVal(r.EquipType), intVal(r.ArtPack), intVal(r.Hardiness), intVal(r.MaxHardiness), int64(r.Slot)})
		}
		return &normalShopFakeRows{columns: []string{"config_id", "item_type", "amount", "view_id", "name", "equip_type", "art_pack", "hardiness", "max_hardiness", "slot"}, values: records}, nil
	}
	return nil, errors.New("unexpected query " + query)
}

type normalShopFakeRows struct {
	columns []string
	values  [][]driver.Value
	pos     int
}

func (r *normalShopFakeRows) Columns() []string { return r.columns }
func (*normalShopFakeRows) Close() error        { return nil }
func (r *normalShopFakeRows) Next(dst []driver.Value) error {
	if r.pos >= len(r.values) {
		return io.EOF
	}
	copy(dst, r.values[r.pos])
	r.pos++
	return nil
}

type normalShopFakeTx struct{ state *normalShopFakeState }

func (tx *normalShopFakeTx) Commit() error {
	tx.state.events = append(tx.state.events, "COMMIT")
	if tx.state.failOn == "COMMIT" {
		return errors.New("injected commit failure")
	}
	tx.state.commit = true
	return nil
}
func (tx *normalShopFakeTx) Rollback() error {
	tx.state.events = append(tx.state.events, "ROLLBACK")
	tx.state.rollback = true
	return nil
}

func normalShopFakeDB(t *testing.T, state *normalShopFakeState) *sql.DB {
	t.Helper()
	normalShopFakeOnce.Do(func() { sql.Register("normal-shop-stage49-fake", normalShopFakeDriver{}) })
	normalShopFakeCurrent = state
	db, err := sql.Open("normal-shop-stage49-fake", "")
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { db.Close() })
	return db
}
func TestNormalShopMySQLSnapshotsAtomicity(t *testing.T) {
	prior := []normalShopPersistedBagRow{{ConfigID: "item_a", ItemType: 1, Amount: 1, ViewID: 3, Slot: 1}}
	next := []normalShopPersistedBagRow{prior[0], {ConfigID: "item_b", ItemType: 1, Amount: 2, ViewID: 3, Slot: 2}}
	before := normalShopPersistedCurrency{Silver: 100, Gold: 3}
	after := normalShopPersistedCurrency{Silver: 90, Gold: 3}
	cases := []struct {
		name, fail       string
		currency         normalShopPersistedCurrency
		bag              []normalShopPersistedBagRow
		missing, zero    bool
		wantCommit       bool
		wantBeforeWrites bool
	}{
		{name: "success_one_transaction", currency: before, bag: prior, wantCommit: true},
		{name: "currency_stale", currency: normalShopPersistedCurrency{Silver: 99, Gold: 3}, bag: prior, wantBeforeWrites: true},
		{name: "bag_stale", currency: before, bag: []normalShopPersistedBagRow{{ConfigID: "item_a", ItemType: 1, Amount: 2, ViewID: 3, Slot: 1}}, wantBeforeWrites: true},
		{name: "missing_currency", currency: before, bag: prior, missing: true, wantBeforeWrites: true},
		{name: "bag_insert_failure", currency: before, bag: prior, fail: "INSERT_BAG"},
		{name: "currency_update_failure", currency: before, bag: prior, fail: "UPDATE_CURRENCY"},
		{name: "unchanged_free_currency_row", currency: before, bag: prior, zero: true, wantCommit: true},
		{name: "bag_lock_failure", currency: before, bag: prior, fail: "LOCK_BAG", wantBeforeWrites: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			state := &normalShopFakeState{currency: tc.currency, bag: tc.bag, failOn: tc.fail, noCurrency: tc.missing, zeroCurrencyRows: tc.zero}
			db := normalShopFakeDB(t, state)
			err := normalShopCommitMySQLSnapshots(context.Background(), db, 17, before, after, prior, next)
			if tc.wantCommit != (err == nil) {
				t.Fatalf("commit=%v err=%v events=%v", tc.wantCommit, err, state.events)
			}
			if state.commit != tc.wantCommit {
				t.Fatalf("commit=%v events=%v", state.commit, state.events)
			}
			if !tc.wantCommit && !state.rollback {
				t.Fatalf("failure without rollback: %v", state.events)
			}
			if len(state.events) == 0 || state.events[0] != "BEGIN" {
				t.Fatalf("missing transaction: %v", state.events)
			}
			if tc.wantBeforeWrites {
				for _, event := range state.events {
					if event == "DELETE_BAG" || event == "INSERT_BAG" || event == "UPDATE_CURRENCY" {
						t.Fatalf("stale state attempted mutation: %v", state.events)
					}
				}
			}
			if tc.wantCommit && state.events[len(state.events)-1] != "COMMIT" {
				t.Fatalf("commit ordering: %v", state.events)
			}
		})
	}
}
