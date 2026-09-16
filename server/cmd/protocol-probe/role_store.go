package main

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/local/9yin-go-server/internal/auth"
	"github.com/local/9yin-go-server/internal/role"
)

const defaultRoleFaction = "force_yihua"

type roleStore struct {
	accounts role.AccountRepository
	roles    role.RoleRepository
	close    func() error
}

func openRoleStore(db *sql.DB) (*roleStore, error) {
	if db != nil {
		repository := role.NewMySQLRepository(db, role.MySQLConfig{})
		return &roleStore{accounts: repository, roles: repository, close: func() error { return nil }}, nil
	}
	repository, err := role.OpenJSONRepository(runtimeProjectPath("data", "roles.json"))
	if err != nil {
		return nil, err
	}
	return &roleStore{accounts: repository, roles: repository, close: func() error { return nil }}, nil
}
func (store *roleStore) Close() error {
	if store == nil || store.close == nil {
		return nil
	}
	return store.close()
}
func (store *roleStore) login(ctx context.Context, key role.AccountKey, passwordCT []byte) (role.Account, *role.RoleSnapshot, error) {
	account, err := store.accounts.FindByKey(ctx, key)
	if err != nil {
		return role.Account{}, nil, err
	}
	if !auth.VerifyPasswordVerifier(passwordCT, account.PasswordHash) {
		return role.Account{}, nil, role.ErrPasswordMismatch
	}
	summaries, err := store.roles.ListByAccount(ctx, account.ID)
	if err != nil {
		return account, nil, err
	}
	if len(summaries) == 0 {
		return account, nil, nil
	}
	if len(summaries) != 1 || summaries[0].Slot != 0 {
		return account, nil, fmt.Errorf("role store: unsupported role layout for account %d", account.ID)
	}
	snapshot, err := store.roles.LoadOwned(ctx, account.ID, summaries[0].ID)
	if err != nil {
		return account, nil, err
	}
	return account, &snapshot, nil
}
func (store *roleStore) create(ctx context.Context, account role.Account, name string, appearance []string) (*role.RoleSnapshot, error) {
	appearance = defaultRoleAppearance(appearance)
	created, err := store.roles.Create(ctx, account.ID, role.NewRole{Name: name, Appearance: role.Appearance{FormatVersion: 1, Values: appearance}, Location: defaultRoleLocation(appearance)})
	if err != nil {
		return nil, err
	}
	return &created, nil
}
func defaultRoleAppearance(appearance []string) []string {
	values := append([]string(nil), appearance...)
	for len(values) < 9 {
		values = append(values, "")
	}
	return values
}
func roleFaction(appearance []string) string {
	if len(appearance) > 8 {
		return appearance[8]
	}
	return ""
}
func (store *roleStore) saveLocation(ctx context.Context, snapshot *role.RoleSnapshot, location role.Location) error {
	if snapshot == nil {
		return role.ErrNotFound
	}
	next, err := store.roles.SaveLocation(ctx, snapshot.ID, snapshot.Location.Version, location)
	if err != nil {
		return err
	}
	location.Version = next
	snapshot.Location = location
	return nil
}
func (store *roleStore) saveFaction(ctx context.Context, snapshot *role.RoleSnapshot, faction string) error {
	if snapshot == nil {
		return role.ErrNotFound
	}
	values := append([]string(nil), snapshot.Appearance.Values...)
	for len(values) < 9 {
		values = append(values, "")
	}
	values[8] = faction
	appearance := role.Appearance{FormatVersion: snapshot.Appearance.FormatVersion, Values: values}
	if appearance.FormatVersion == 0 {
		appearance.FormatVersion = 1
	}
	next, err := store.roles.SaveAppearance(ctx, snapshot.ID, snapshot.Appearance.Version, appearance)
	if err != nil {
		return err
	}
	appearance.Version = next
	snapshot.Appearance = appearance
	return nil
}
func defaultRoleLocation(appearance []string) role.Location {
	bookID := ""
	if len(appearance) > 0 {
		bookID = appearance[0]
	}
	switch bookID {
	case "book1":
		return role.Location{Scene: role.Scene{Config: `ini\scene\born04_QianDengZhen`, Resource: "born04"}, Position: role.Position{X: 874.933, Y: -32.392, Z: 746.688, Orient: 0.335}, Version: 1}
	case "book2":
		return role.Location{Scene: role.Scene{Config: `ini\scene\born03_YanYuZhuang`, Resource: "born03"}, Position: role.Position{X: 460.629, Y: 12.811, Z: 659.731, Orient: 5.58}, Version: 1}
	case "book4":
		return role.Location{Scene: role.Scene{Config: `ini\scene\city05_ChengDu`, Resource: "city05"}, Position: role.Position{X: 915.753, Y: 31.192, Z: 538.193, Orient: 6.17}, Version: 1}
	case "book5":
		return role.Location{Scene: role.Scene{Config: `ini\scene\born02_ERenGU`, Resource: "born02"}, Position: role.Position{X: 693.908, Y: 24.694, Z: 404.35, Orient: 3.611}, Version: 1}
	case "book8":
		return role.Location{Scene: role.Scene{Config: `ini\scene\born02_ERenGU`, Resource: "born02"}, Position: role.Position{X: 905.069, Y: 10.81, Z: 196.98, Orient: 1.61}, Version: 1}
	default:
		return role.Location{Scene: role.Scene{Config: `ini\scene\born04_QianDengZhen`, Resource: "born04"}, Position: role.Position{X: 874.933, Y: -32.392, Z: 746.688, Orient: 0.335}, Version: 1}
	}
}
