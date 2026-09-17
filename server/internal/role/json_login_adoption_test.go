package role

import (
	"bytes"
	"context"
	"errors"
	"path/filepath"
	"testing"
)

func stage59Role(name string) Role {
	return Role{
		Name:       name,
		Appearance: []string{"book5", "cloth_b0001"},
		Scene:      Scene{Config: `ini\scene\city05_ChengDu`, Resource: "city05"},
		Position:   Position{X: 612.8513, Y: 23.163153, Z: 706.49054, Orient: 0.7674046},
	}
}

func TestAdoptSoleUnverifiedLoginAccountPreservesIdentityAndRole(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "roles.json")
	repository, err := OpenJSONRepository(path)
	if err != nil {
		t.Fatal(err)
	}
	oldKey := AccountKey("acct:v1:112cd36f1e1d2bac408e0428aabef6e399080c2bc33407de6677e60dec97c2a6")
	// Only the e9339acb7891 prefix was observed in the Stage58 LIVE log; the
	// remaining test suffix is synthetic and must not be treated as captured
	// current-client identity evidence.
	newKey := AccountKey("acct:v1:e9339acb78910000000000000000000000000000000000000000000000000000")
	if err := repository.Save(ctx, oldKey, stage59Role("商弘")); err != nil {
		t.Fatal(err)
	}
	beforeAccount, err := repository.FindByKey(ctx, oldKey)
	if err != nil {
		t.Fatal(err)
	}
	beforeRoles, err := repository.ListByAccount(ctx, beforeAccount.ID)
	if err != nil || len(beforeRoles) != 1 {
		t.Fatalf("before roles=%v err=%v", beforeRoles, err)
	}
	verifier := []byte("current-client-password-verifier")
	account, adopted, err := repository.AdoptSoleUnverifiedLoginAccount(ctx, newKey, verifier)
	if err != nil {
		t.Fatal(err)
	}
	if !adopted {
		t.Fatal("legacy account was not adopted")
	}
	if account.ID != beforeAccount.ID {
		t.Fatalf("account id changed: got=%d want=%d", account.ID, beforeAccount.ID)
	}
	if !bytes.Equal(account.PasswordHash, verifier) {
		t.Fatalf("password verifier mismatch: %x", account.PasswordHash)
	}
	if _, err := repository.FindByKey(ctx, oldKey); !errors.Is(err, ErrNotFound) {
		t.Fatalf("old key still resolves: %v", err)
	}
	afterAccount, err := repository.FindByKey(ctx, newKey)
	if err != nil {
		t.Fatal(err)
	}
	afterRoles, err := repository.ListByAccount(ctx, afterAccount.ID)
	if err != nil || len(afterRoles) != 1 {
		t.Fatalf("after roles=%v err=%v", afterRoles, err)
	}
	if afterRoles[0].ID != beforeRoles[0].ID || afterRoles[0].Name != "商弘" {
		t.Fatalf("role identity changed: before=%v after=%v", beforeRoles[0], afterRoles[0])
	}

	reopened, err := OpenJSONRepository(path)
	if err != nil {
		t.Fatal(err)
	}
	persisted, err := reopened.FindByKey(ctx, newKey)
	if err != nil {
		t.Fatal(err)
	}
	if persisted.ID != beforeAccount.ID || !bytes.Equal(persisted.PasswordHash, verifier) {
		t.Fatalf("reopened account=%#v", persisted)
	}
	persistedRoles, err := reopened.ListByAccount(ctx, persisted.ID)
	if err != nil || len(persistedRoles) != 1 || persistedRoles[0].ID != beforeRoles[0].ID {
		t.Fatalf("reopened roles=%v err=%v", persistedRoles, err)
	}
}

func TestAdoptSoleUnverifiedLoginAccountFailsClosedWhenAmbiguous(t *testing.T) {
	ctx := context.Background()
	repository, err := OpenJSONRepository(filepath.Join(t.TempDir(), "roles.json"))
	if err != nil {
		t.Fatal(err)
	}
	first := AccountKey("acct:v1:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	second := AccountKey("acct:v1:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb")
	if err := repository.Save(ctx, first, stage59Role("甲")); err != nil {
		t.Fatal(err)
	}
	if err := repository.Save(ctx, second, stage59Role("乙")); err != nil {
		t.Fatal(err)
	}
	newKey := AccountKey("acct:v1:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc")
	if _, adopted, err := repository.AdoptSoleUnverifiedLoginAccount(ctx, newKey, []byte("verifier")); !errors.Is(err, ErrAmbiguousLegacyLoginAccount) || adopted {
		t.Fatalf("adopted=%v err=%v", adopted, err)
	}
	if _, err := repository.FindByKey(ctx, first); err != nil {
		t.Fatalf("first candidate mutated: %v", err)
	}
	if _, err := repository.FindByKey(ctx, second); err != nil {
		t.Fatalf("second candidate mutated: %v", err)
	}
	if _, err := repository.FindByKey(ctx, newKey); !errors.Is(err, ErrNotFound) {
		t.Fatalf("new key unexpectedly created: %v", err)
	}
}

func TestAdoptSoleUnverifiedLoginAccountIgnoresLocalDefaultAndBoundAccounts(t *testing.T) {
	ctx := context.Background()
	repository, err := OpenJSONRepository(filepath.Join(t.TempDir(), "roles.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := repository.Save(ctx, LegacyDefaultAccount, stage59Role("本地")); err != nil {
		t.Fatal(err)
	}
	bound := AccountKey("acct:v1:dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd")
	if err := repository.Save(ctx, bound, stage59Role("绑定")); err != nil {
		t.Fatal(err)
	}
	if err := repository.SetPassword(ctx, bound, []byte("existing-verifier")); err != nil {
		t.Fatal(err)
	}
	newKey := AccountKey("acct:v1:eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee")
	if _, adopted, err := repository.AdoptSoleUnverifiedLoginAccount(ctx, newKey, []byte("new-verifier")); err != nil || adopted {
		t.Fatalf("adopted=%v err=%v", adopted, err)
	}
	if _, err := repository.FindByKey(ctx, newKey); !errors.Is(err, ErrNotFound) {
		t.Fatalf("new key unexpectedly created: %v", err)
	}
}
