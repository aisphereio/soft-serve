package web

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/aisphereio/soft-serve/pkg/backend"
	"github.com/aisphereio/soft-serve/pkg/config"
	"github.com/aisphereio/soft-serve/pkg/db"
	"github.com/aisphereio/soft-serve/pkg/db/migrate"
	"github.com/aisphereio/soft-serve/pkg/proto"
	"github.com/aisphereio/soft-serve/pkg/store"
	"github.com/aisphereio/soft-serve/pkg/store/database"
)

func TestDescribeRequestClassifiesGitAndLFS(t *testing.T) {
	tests := []struct {
		name         string
		method       string
		target       string
		body         string
		wantAction   Action
		wantProtocol string
	}{
		{
			name:         "advertise upload pack",
			method:       http.MethodGet,
			target:       "/team/demo.git/info/refs?service=git-upload-pack",
			wantAction:   ActionRead,
			wantProtocol: ProtocolGit,
		},
		{
			name:         "upload pack",
			method:       http.MethodPost,
			target:       "/team/demo.git/git-upload-pack",
			wantAction:   ActionRead,
			wantProtocol: ProtocolGit,
		},
		{
			name:         "advertise receive pack",
			method:       http.MethodGet,
			target:       "/team/demo.git/info/refs?service=git-receive-pack",
			wantAction:   ActionWrite,
			wantProtocol: ProtocolGit,
		},
		{
			name:         "receive pack",
			method:       http.MethodPost,
			target:       "/team/demo.git/git-receive-pack",
			wantAction:   ActionWrite,
			wantProtocol: ProtocolGit,
		},
		{
			name:         "download LFS batch",
			method:       http.MethodPost,
			target:       "/team/demo.git/info/lfs/objects/batch",
			body:         `{"operation":"download","objects":[]}`,
			wantAction:   ActionRead,
			wantProtocol: ProtocolLFS,
		},
		{
			name:         "upload LFS batch",
			method:       http.MethodPost,
			target:       "/team/demo.git/info/lfs/objects/batch",
			body:         `{"operation":"upload","objects":[]}`,
			wantAction:   ActionWrite,
			wantProtocol: ProtocolLFS,
		},
		{
			name:         "download LFS object",
			method:       http.MethodGet,
			target:       "/team/demo.git/info/lfs/objects/basic/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			wantAction:   ActionRead,
			wantProtocol: ProtocolLFS,
		},
		{
			name:         "upload LFS object",
			method:       http.MethodPut,
			target:       "/team/demo.git/info/lfs/objects/basic/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			wantAction:   ActionWrite,
			wantProtocol: ProtocolLFS,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.target, strings.NewReader(tt.body))
			described, err := DescribeRequest(req)
			if err != nil {
				t.Fatalf("DescribeRequest() error = %v", err)
			}
			if got, want := described.Repository, "team/demo"; got != want {
				t.Fatalf("Repository = %q, want %q", got, want)
			}
			if got, want := described.Action, tt.wantAction; got != want {
				t.Fatalf("Action = %q, want %q", got, want)
			}
			if got, want := described.Protocol, tt.wantProtocol; got != want {
				t.Fatalf("Protocol = %q, want %q", got, want)
			}
			if tt.body != "" {
				restored, readErr := io.ReadAll(req.Body)
				if readErr != nil {
					t.Fatalf("read restored body: %v", readErr)
				}
				if got := string(restored); got != tt.body {
					t.Fatalf("restored body = %q, want %q", got, tt.body)
				}
			}
		})
	}
}

func TestDescribeRequestRejectsUnknownProtocolRoute(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/team/demo.git/not-a-git-route", nil)
	if _, err := DescribeRequest(req); err == nil {
		t.Fatal("DescribeRequest() error = nil, want error")
	}
}

func TestDescribeRequestRejectsOversizedLFSBatch(t *testing.T) {
	body := `{"operation":"download","padding":"` + strings.Repeat("x", maxLFSBatchDescriptorBytes) + `"}`
	req := httptest.NewRequest(http.MethodPost, "/demo.git/info/lfs/objects/batch", strings.NewReader(body))
	if _, err := DescribeRequest(req); err == nil {
		t.Fatal("DescribeRequest() error = nil, want size error")
	}
}

func TestNewProtocolRouterRejectsUnknownReceivePackWithoutCreatingRepository(t *testing.T) {
	ctx, be := newProtocolTestContext(t)
	handler := NewProtocolRouter(ctx)
	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/missing.git/info/refs?service=git-receive-pack", nil)

	handler.ServeHTTP(recorder, req)

	if got, want := recorder.Code, http.StatusNotFound; got != want {
		t.Fatalf("status = %d, want %d", got, want)
	}
	if _, err := be.Repository(ctx, "missing"); err == nil {
		t.Fatal("unknown receive-pack request created repository")
	}
}

func TestNewProtocolRouterAllowsPreauthorizedFetchWithoutAuthorizationHeader(t *testing.T) {
	ctx, be := newProtocolTestContext(t)
	user, err := be.CreateUser(ctx, "alice", proto.UserOptions{})
	if err != nil {
		t.Fatalf("CreateUser() error = %v", err)
	}
	if _, err := be.CreateRepository(ctx, "demo", user, proto.RepositoryOptions{}); err != nil {
		t.Fatalf("CreateRepository() error = %v", err)
	}

	handler := NewProtocolRouter(ctx)
	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/demo.git/info/refs?service=git-upload-pack", nil)
	handler.ServeHTTP(recorder, req)

	if got, want := recorder.Code, http.StatusOK; got != want {
		t.Fatalf("status = %d, want %d; body=%s", got, want, recorder.Body.String())
	}
}

func newProtocolTestContext(t *testing.T) (context.Context, *backend.Backend) {
	t.Helper()
	tmp := t.TempDir()
	ctx := context.Background()
	cfg := config.DefaultConfig()
	cfg.DataPath = tmp
	cfg.DB.Driver = "sqlite"
	cfg.DB.DataSource = filepath.Join(tmp, "soft-serve.db")
	ctx = config.WithContext(ctx, cfg)
	dbx, err := db.Open(ctx, cfg.DB.Driver, cfg.DB.DataSource)
	if err != nil {
		t.Fatalf("db.Open() error = %v", err)
	}
	t.Cleanup(func() { _ = dbx.Close() })
	if err := migrate.Migrate(ctx, dbx); err != nil {
		t.Fatalf("migrate.Migrate() error = %v", err)
	}
	datastore := database.New(ctx, dbx)
	ctx = db.WithContext(ctx, dbx)
	ctx = store.WithContext(ctx, datastore)
	be := backend.New(ctx, cfg, dbx, datastore)
	ctx = backend.WithContext(ctx, be)
	return ctx, be
}
