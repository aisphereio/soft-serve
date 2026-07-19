package store

import (
	"context"

	"github.com/aisphereio/soft-serve/pkg/access"
	"github.com/aisphereio/soft-serve/pkg/db"
)

// SettingStore is an interface for managing settings.
type SettingStore interface {
	GetAnonAccess(ctx context.Context, h db.Handler) (access.AccessLevel, error)
	SetAnonAccess(ctx context.Context, h db.Handler, level access.AccessLevel) error
	GetAllowKeylessAccess(ctx context.Context, h db.Handler) (bool, error)
	SetAllowKeylessAccess(ctx context.Context, h db.Handler, allow bool) error
}
