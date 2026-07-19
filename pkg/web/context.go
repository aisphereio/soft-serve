package web

import (
	"context"
	"net/http"

	"charm.land/log/v2"
	"github.com/aisphereio/soft-serve/pkg/backend"
	"github.com/aisphereio/soft-serve/pkg/config"
	"github.com/aisphereio/soft-serve/pkg/db"
	"github.com/aisphereio/soft-serve/pkg/store"
)

// NewContextHandler returns a new context middleware.
// This middleware adds the config, backend, and logger to the request context.
func NewContextHandler(baseContext context.Context) func(http.Handler) http.Handler {
	cfg := config.FromContext(baseContext)
	be := backend.FromContext(baseContext)
	logger := log.FromContext(baseContext).WithPrefix("http")
	dbx := db.FromContext(baseContext)
	datastore := store.FromContext(baseContext)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			ctx = copyProtocolEnvironment(ctx, baseContext)
			ctx = config.WithContext(ctx, cfg)
			ctx = backend.WithContext(ctx, be)
			ctx = log.WithContext(ctx, logger.With(
				"method", r.Method,
				"path", r.URL,
				"addr", r.RemoteAddr,
			))
			ctx = db.WithContext(ctx, dbx)
			ctx = store.WithContext(ctx, datastore)
			r = r.WithContext(ctx)

			next.ServeHTTP(w, r)
		})
	}
}
