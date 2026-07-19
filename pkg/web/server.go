package web

import (
	"context"
	"net/http"

	"charm.land/log/v2"
	"github.com/aisphereio/soft-serve/pkg/config"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
)

// NewRouter returns a new HTTP router.
func NewRouter(ctx context.Context) http.Handler {
	logger := log.FromContext(ctx).WithPrefix("http")
	router := mux.NewRouter()

	// Health routes
	HealthController(ctx, router)

	// Git routes
	GitController(ctx, router)

	router.PathPrefix("/").HandlerFunc(renderNotFound)

	// Context handler
	// Adds context to the request
	h := NewLoggingMiddleware(router, logger)
	h = NewContextHandler(ctx)(h)
	h = handlers.CompressHandler(h)
	h = handlers.RecoveryHandler()(h)

	cfg := config.FromContext(ctx)

	h = handlers.CORS(handlers.AllowedHeaders(cfg.HTTP.CORS.AllowedHeaders),
		handlers.AllowedOrigins(cfg.HTTP.CORS.AllowedOrigins),
		handlers.AllowedMethods(cfg.HTTP.CORS.AllowedMethods),
	)(h)

	return h
}

// NewProtocolRouter returns an embeddable Git Smart HTTP/LFS router. The
// embedding application must authenticate and authorize the structured result
// of DescribeRequest before invoking this handler. This router deliberately
// skips Soft Serve's user/token/collaborator authorization and never creates a
// repository from receive-pack traffic.
func NewProtocolRouter(ctx context.Context) http.Handler {
	logger := log.FromContext(ctx).WithPrefix("http.protocol")
	router := mux.NewRouter()
	GitProtocolController(ctx, router)
	router.PathPrefix("/").HandlerFunc(renderNotFound)

	h := NewLoggingMiddleware(router, logger)
	h = NewContextHandler(ctx)(h)
	return h
}
