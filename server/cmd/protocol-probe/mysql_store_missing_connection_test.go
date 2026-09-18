package main

import (
	"strings"
	"testing"

	"github.com/local/9yin-go-server/internal/role"
)

// A constructed MySQL store with no DB connection must never acknowledge a
// durable write. This specifically guards GM grant and currency save callers.
func TestMySQLStoreMissingConnectionRejectsWrites(t *testing.T) {
	roleID := role.RoleID(77)
	item := bagItem{Slot: 1, ConfigID: "existing", ItemType: 100, Amount: 1, ViewID: 1}
	for _, tc := range []struct {
		name  string
		write func() error
	}{
		{"bag nil receiver", func() error { var s *mysqlBagStore; return s.Save(roleID, []bagItem{item}) }},
		{"bag missing connection", func() error { return (&mysqlBagStore{}).Save(roleID, []bagItem{item}) }},
		{"currency nil receiver", func() error { var s *mysqlCurrencyStore; return s.Save(roleID, currencySnapshot{Silver: 1}) }},
		{"currency missing connection", func() error { return (&mysqlCurrencyStore{}).Save(roleID, currencySnapshot{Silver: 1}) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.write(); err == nil || !strings.Contains(err.Error(), "database") {
				t.Fatalf("unavailable MySQL connection reported successful save (or unhelpful error): %v", err)
			}
		})
	}
}

func TestGMGrantMissingMySQLConnectionRestoresActor(t *testing.T) {
	player := newPlayerActor("mysql-bag-guard", 0)
	item := bagItem{Slot: 1, ConfigID: "existing", ItemType: 100, Amount: 1, ViewID: 1}
	player.addBagItem(item)
	before := player.bagSnapshot()
	err := grantBagItemDurably(player, &mysqlBagStore{}, role.RoleID(77), bagItem{ConfigID: "new", ItemType: 100, Amount: 1, ViewID: 1})
	if err == nil {
		t.Fatal("GM grant succeeded without a MySQL connection")
	}
	after := player.bagSnapshot()
	if len(after) != len(before) || after[0].ConfigID != before[0].ConfigID || after[0].Amount != before[0].Amount {
		t.Fatalf("GM grant retained unpersisted item: before=%+v after=%+v", before, after)
	}
}

func TestDeferredCurrencyMissingMySQLConnectionFailsClosed(t *testing.T) {
	player := newPlayerActor("mysql-currency-guard", 0)
	player.setSilver(10)
	if err := persistDeferredShopCurrency(&mysqlCurrencyStore{}, role.RoleID(77), player); err == nil {
		t.Fatal("deferred currency save acknowledged without a database")
	}
}

func TestBagMutationMissingMySQLConnectionFailsClosed(t *testing.T) {
	before := []bagItem{{Slot: 1, ConfigID: "old", ItemType: 100, Amount: 1, ViewID: 1}}
	after := []bagItem{{Slot: 1, ConfigID: "new", ItemType: 100, Amount: 1, ViewID: 1}}
	if err := persistBagMutationChecked(&mysqlBagStore{}, role.RoleID(77), before, after); err == nil {
		t.Fatal("bag mutation acknowledged without a database")
	}
}
