package main

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/local/9yin-go-server/internal/role"
)

func TestRoleStoreLoginAdoptsSoleUnverifiedJSONIdentity(t *testing.T) {
	ctx := context.Background()
	repository, err := role.OpenJSONRepository(filepath.Join(t.TempDir(), "roles.json"))
	if err != nil {
		t.Fatal(err)
	}
	oldKey := role.AccountKey("acct:v1:112cd36f1e1d2bac408e0428aabef6e399080c2bc33407de6677e60dec97c2a6")
	// Only the e9339acb7891 prefix was observed in the Stage58 LIVE log; the
	// remaining suffix is synthetic test data, not an exact captured identity.
	newKey := role.AccountKey("acct:v1:e9339acb78910000000000000000000000000000000000000000000000000000")
	if err := repository.Save(ctx, oldKey, role.Role{
		Name:       "商弘",
		Appearance: []string{"book5", "cloth_b0001"},
		Scene:      role.Scene{Config: `ini\scene\city05_ChengDu`, Resource: "city05"},
		Position:   role.Position{X: 612.8513, Y: 23.163153, Z: 706.49054, Orient: 0.7674046},
	}); err != nil {
		t.Fatal(err)
	}
	before, err := repository.FindByKey(ctx, oldKey)
	if err != nil {
		t.Fatal(err)
	}
	store := &roleStore{accounts: repository, roles: repository, close: func() error { return nil }}
	verifier := []byte("stage59-current-client-password-verifier")
	account, snapshot, err := store.login(ctx, newKey, verifier)
	if err != nil {
		t.Fatal(err)
	}
	if account.ID != before.ID {
		t.Fatalf("account ID changed: got=%d want=%d", account.ID, before.ID)
	}
	if snapshot == nil || snapshot.Name != "商弘" || snapshot.AccountID != before.ID {
		t.Fatalf("adopted snapshot=%#v", snapshot)
	}
	if _, err := repository.FindByKey(ctx, oldKey); !errors.Is(err, role.ErrNotFound) {
		t.Fatalf("old identity still resolves: %v", err)
	}
	if _, _, err := store.login(ctx, newKey, verifier); err != nil {
		t.Fatalf("second login with rebound identity failed: %v", err)
	}
	if _, _, err := store.login(ctx, newKey, []byte("wrong-verifier")); !errors.Is(err, role.ErrPasswordMismatch) {
		t.Fatalf("wrong verifier err=%v, want password mismatch", err)
	}
}
