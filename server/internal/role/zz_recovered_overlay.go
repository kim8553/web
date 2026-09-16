package role

import (
	"context"
	"errors"
)

func (repository *JSONRepository) CreateAccount(ctx context.Context, key AccountKey) (Account, error) {
	if err := ctx.Err(); err != nil {
		return Account{}, err
	}
	if err := ValidateAccountKey(key); err != nil {
		return Account{}, err
	}
	repository.mu.Lock()
	defer repository.mu.Unlock()
	text := string(key)
	if existing, exists := repository.identities[text]; exists {
		if existing.Status != AccountActive {
			return Account{}, ErrAccountDisabled
		}
		return Account{}, ErrAccountExists
	}
	identity := repository.ensureIdentityLocked(text, false)
	if err := repository.persistLocked(); err != nil {
		delete(repository.identities, text)
		return Account{}, err
	}
	return accountFromJSON(key, identity), nil
}
func (repository *JSONRepository) SetPassword(ctx context.Context, key AccountKey, verifier []byte) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := ValidateAccountKey(key); err != nil {
		return err
	}
	repository.mu.Lock()
	defer repository.mu.Unlock()
	identity, exists := repository.identities[string(key)]
	if !exists {
		return ErrNotFound
	}
	identity.PasswordHash = append([]byte(nil), verifier...)
	repository.identities[string(key)] = identity
	return repository.persistLocked()
}
func (repository *MySQLRepository) SetPassword(ctx context.Context, key AccountKey, verifier []byte) error {
	if err := ValidateAccountKey(key); err != nil {
		return err
	}
	result, err := repository.db.ExecContext(ctx, `UPDATE accounts SET password_hash = ? WHERE account_key = ?`, verifier, []byte(key))
	if err != nil {
		return storageError(err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return storageError(err)
	}
	if affected == 0 {
		return ErrNotFound
	}
	return nil
}
func (repository *MySQLRepository) CreateAccount(ctx context.Context, key AccountKey) (Account, error) {
	if err := ValidateAccountKey(key); err != nil {
		return Account{}, err
	}
	_, err := repository.db.ExecContext(ctx, `INSERT INTO accounts(account_key) VALUES (?)`, []byte(key))
	if err != nil {
		if isDuplicate(err) {
			return Account{}, ErrAccountExists
		}
		return Account{}, storageError(err)
	}
	return repository.FindByKey(ctx, key)
}

var ErrAccountExists = errors.New("role: account already exists")
var ErrPasswordMismatch = errors.New("role: password mismatch")
