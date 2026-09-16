package main

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/local/9yin-go-server/internal/role"
)

func testRoleStore(t *testing.T) *roleStore {
	t.Helper()
	repository, err := role.OpenJSONRepository(filepath.Join(t.TempDir(), "roles.json"))
	if err != nil {
		t.Fatal(err)
	}
	return &roleStore{accounts: repository, roles: repository, close: func() error { return nil }}
}

func TestRoleStoreIsolatesAccountsAndPersistsLocationVersion(t *testing.T) {
	ctx := context.Background()
	store := testRoleStore(t)
	firstKey := role.AccountKey("acct:v1:first")
	firstVerifier := []byte("first-verifier")
	firstAccount, err := store.register(ctx, firstKey, firstVerifier)
	if err != nil {
		t.Fatal(err)
	}
	_, firstRole, err := store.login(ctx, firstKey, firstVerifier)
	if err != nil || firstRole != nil {
		t.Fatalf("first login role=%v error=%v", firstRole, err)
	}
	secondKey := role.AccountKey("acct:v1:second")
	secondVerifier := []byte("second-verifier")
	secondAccount, err := store.register(ctx, secondKey, secondVerifier)
	if err != nil {
		t.Fatal(err)
	}
	_, secondRole, err := store.login(ctx, secondKey, secondVerifier)
	if err != nil || secondRole != nil {
		t.Fatalf("second login role=%v error=%v", secondRole, err)
	}
	firstRole, err = store.create(ctx, firstAccount, "甲", []string{"book5"})
	if err != nil {
		t.Fatal(err)
	}
	if got := firstRole.Appearance.Values[8]; got != "" {
		t.Fatalf("current padded faction=%q want empty", got)
	}
	if got, want := firstRole.Location, defaultRoleLocation(firstRole.Appearance.Values); got.Scene != want.Scene || got.Position != want.Position {
		t.Fatalf("default location=%+v want=%+v", got, want)
	}
	secondRole, err = store.create(ctx, secondAccount, "乙", []string{"book4"})
	if err != nil {
		t.Fatal(err)
	}
	if firstRole.ID == secondRole.ID || firstRole.AccountID == secondRole.AccountID {
		t.Fatalf("identities overlap: first=%#v second=%#v", firstRole, secondRole)
	}

	location := firstRole.Location
	previousVersion := location.Version
	location.Position = role.Position{X: 0, Y: 0, Z: 0, Orient: 0}
	if err := store.saveLocation(ctx, firstRole, location); err != nil {
		t.Fatal(err)
	}
	if firstRole.Location.Version != previousVersion+1 || firstRole.Location.Position != location.Position {
		t.Fatalf("saved location = %#v", firstRole.Location)
	}

	_, reloaded, err := store.login(ctx, firstAccount.Key, firstVerifier)
	if err != nil {
		t.Fatal(err)
	}
	if reloaded == nil || reloaded.Location.Position != location.Position {
		t.Fatalf("reloaded role = %#v", reloaded)
	}
	_, other, err := store.login(ctx, secondAccount.Key, secondVerifier)
	if err != nil {
		t.Fatal(err)
	}
	if other == nil || other.Name != "乙" {
		t.Fatalf("second account role = %#v", other)
	}
}
