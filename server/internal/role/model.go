package role

import (
	"context"
	"errors"
)

type AccountKey string
type Scene struct {
	Config   string `json:"config"`
	Resource string `json:"resource"`
}
type Position struct {
	X      float32 `json:"x"`
	Y      float32 `json:"y"`
	Z      float32 `json:"z"`
	Orient float32 `json:"orient"`
}
type Role struct {
	Name       string   `json:"name"`
	Appearance []string `json:"appearance,omitempty"`
	Scene      Scene    `json:"scene"`
	Position   Position `json:"position"`
}

var (
	ErrInvalidAccountKey = errors.New("role: empty account key")
	ErrRoleNotFound      = errors.New("role: account has no role")
)

type Repository interface {
	Load(ctx context.Context, account AccountKey) (Role, bool, error)
	Save(ctx context.Context, account AccountKey, value Role) error
	UpdatePosition(ctx context.Context, account AccountKey, position Position) error
}
