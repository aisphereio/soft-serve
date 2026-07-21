package store

import (
	"context"

	"github.com/aisphereio/soft-serve/pkg/db"
	"github.com/aisphereio/soft-serve/pkg/db/models"
)

// RepositoryCreateExtension runs inside the same database transaction that
// creates the Soft Serve repository metadata. Embedding applications can use
// it to persist business extensions keyed by the canonical repository ID.
type RepositoryCreateExtension func(context.Context, db.Handler, models.Repo) error

type repositoryCreateExtensionsKey struct{}

// WithRepositoryCreateExtension appends a repository creation extension to ctx.
func WithRepositoryCreateExtension(ctx context.Context, extension RepositoryCreateExtension) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	if extension == nil {
		return ctx
	}

	extensions := RepositoryCreateExtensionsFromContext(ctx)
	extensions = append(extensions, extension)
	return context.WithValue(ctx, repositoryCreateExtensionsKey{}, extensions)
}

// RepositoryCreateExtensionsFromContext returns a copy of the configured
// repository creation extensions.
func RepositoryCreateExtensionsFromContext(ctx context.Context) []RepositoryCreateExtension {
	if ctx == nil {
		return nil
	}
	extensions, _ := ctx.Value(repositoryCreateExtensionsKey{}).([]RepositoryCreateExtension)
	return append([]RepositoryCreateExtension(nil), extensions...)
}
