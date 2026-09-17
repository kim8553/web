package role

import (
	"context"
	"errors"
	"strings"
)

var ErrAmbiguousLegacyLoginAccount = errors.New("role: ambiguous legacy login account")

// AdoptSoleUnverifiedLoginAccount rebinds the only eligible legacy acct:v1:
// JSON identity to newKey while preserving its account ID, role ID, role data,
// and therefore all role-ID keyed side stores. It is intentionally narrow:
// only active hashed identities that already own a role and have no password
// verifier are eligible. local_default and already-bound accounts are never
// adopted. Multiple eligible candidates fail closed.
func (repository *JSONRepository) AdoptSoleUnverifiedLoginAccount(ctx context.Context, newKey AccountKey, verifier []byte) (Account, bool, error) {
	if err := ctx.Err(); err != nil {
		return Account{}, false, err
	}
	if err := ValidateAccountKey(newKey); err != nil {
		return Account{}, false, err
	}
	if len(verifier) == 0 {
		return Account{}, false, ErrInvalidArgument
	}

	repository.mu.Lock()
	defer repository.mu.Unlock()

	newText := string(newKey)
	if existing, exists := repository.identities[newText]; exists {
		if existing.Status != AccountActive {
			return Account{}, false, ErrAccountDisabled
		}
		return accountFromJSON(newKey, existing), false, nil
	}

	candidateKey := ""
	var candidateIdentity jsonIdentity
	var candidateRole Role
	for key, identity := range repository.identities {
		if key == newText || !strings.HasPrefix(key, "acct:v1:") {
			continue
		}
		if identity.Status != AccountActive || identity.RoleID == 0 || len(identity.PasswordHash) != 0 {
			continue
		}
		value, exists := repository.accounts[key]
		if !exists || strings.TrimSpace(value.Name) == "" {
			continue
		}
		if candidateKey != "" {
			return Account{}, false, ErrAmbiguousLegacyLoginAccount
		}
		candidateKey = key
		candidateIdentity = identity
		candidateRole = clone(value)
	}
	if candidateKey == "" {
		return Account{}, false, nil
	}

	originalIdentity := candidateIdentity
	candidateIdentity.PasswordHash = append([]byte(nil), verifier...)
	candidateIdentity.AccountVersion++
	if candidateIdentity.AccountVersion == 0 {
		candidateIdentity.AccountVersion = 1
	}

	delete(repository.identities, candidateKey)
	delete(repository.accounts, candidateKey)
	repository.identities[newText] = candidateIdentity
	repository.accounts[newText] = candidateRole
	if err := repository.persistLocked(); err != nil {
		delete(repository.identities, newText)
		delete(repository.accounts, newText)
		repository.identities[candidateKey] = originalIdentity
		repository.accounts[candidateKey] = candidateRole
		return Account{}, false, err
	}
	return accountFromJSON(newKey, candidateIdentity), true, nil
}
