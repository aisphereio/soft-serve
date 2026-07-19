package web

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/aisphereio/soft-serve/pkg/lfs"
	"github.com/aisphereio/soft-serve/pkg/utils"
)

const maxLFSBatchDescriptorBytes = 64 << 10

// Action is the coarse access action required before a protocol request is
// delegated to the Git/LFS handler.
type Action string

const (
	ActionRead  Action = "read"
	ActionWrite Action = "write"
)

const (
	ProtocolGit = "git"
	ProtocolLFS = "git-lfs"
)

// ProtocolRequest is the provider-neutral classification of a Git/LFS HTTP
// request. Embedding applications map Action to their own authorization model.
type ProtocolRequest struct {
	Repository string
	Action     Action
	Protocol   string
}

// DescribeRequest classifies a native Git/LFS request without authenticating
// or authorizing it. Git pack bodies are never read. The bounded JSON body of
// an LFS batch request is restored after its operation is inspected.
func DescribeRequest(r *http.Request) (ProtocolRequest, error) {
	if r == nil {
		return ProtocolRequest{}, fmt.Errorf("soft-serve/web: HTTP request is required")
	}

	repository, route, err := splitProtocolPath(r.URL.Path)
	if err != nil {
		return ProtocolRequest{}, err
	}
	request := ProtocolRequest{Repository: repository}

	switch {
	case route == "/git-upload-pack":
		request.Action = ActionRead
		request.Protocol = ProtocolGit
	case route == "/git-receive-pack":
		request.Action = ActionWrite
		request.Protocol = ProtocolGit
	case route == "/info/refs":
		request.Protocol = ProtocolGit
		switch r.URL.Query().Get("service") {
		case "", "git-upload-pack", "git-upload-archive":
			request.Action = ActionRead
		case "git-receive-pack":
			request.Action = ActionWrite
		default:
			return ProtocolRequest{}, fmt.Errorf("soft-serve/web: unsupported Git service")
		}
	case route == "/info/lfs/objects/batch":
		request.Protocol = ProtocolLFS
		request.Action, err = describeLFSBatch(r)
		if err != nil {
			return ProtocolRequest{}, err
		}
	case strings.HasPrefix(route, "/info/lfs/"):
		request.Protocol = ProtocolLFS
		if r.Method == http.MethodGet && !strings.Contains(route, "/locks") {
			request.Action = ActionRead
		} else {
			request.Action = ActionWrite
		}
	case isReadOnlyGitRoute(route) && (r.Method == http.MethodGet || r.Method == http.MethodHead):
		request.Action = ActionRead
		request.Protocol = ProtocolGit
	default:
		return ProtocolRequest{}, fmt.Errorf("soft-serve/web: unsupported protocol route")
	}

	return request, nil
}

func splitProtocolPath(rawPath string) (string, string, error) {
	clean := "/" + strings.TrimLeft(rawPath, "/")
	if marker := strings.Index(clean, ".git/"); marker >= 0 {
		repository := utils.SanitizeRepo(clean[1 : marker+4])
		if repository == "" {
			return "", "", fmt.Errorf("soft-serve/web: repository is required")
		}
		return repository, clean[marker+4:], nil
	}

	markers := []string{
		"/git-upload-pack",
		"/git-receive-pack",
		"/info/refs",
		"/info/lfs/",
		"/objects/",
		"/HEAD",
	}
	for _, marker := range markers {
		if index := strings.Index(clean, marker); index > 0 {
			repository := utils.SanitizeRepo(clean[1:index])
			if repository == "" {
				break
			}
			return repository, clean[index:], nil
		}
	}

	return "", "", fmt.Errorf("soft-serve/web: cannot determine repository and protocol route")
}

func describeLFSBatch(r *http.Request) (Action, error) {
	if r.Body == nil {
		return "", fmt.Errorf("soft-serve/web: LFS batch body is required")
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, maxLFSBatchDescriptorBytes+1))
	if err != nil {
		return "", fmt.Errorf("soft-serve/web: read LFS batch descriptor: %w", err)
	}
	r.Body = io.NopCloser(bytes.NewReader(body))
	if len(body) > maxLFSBatchDescriptorBytes {
		return "", fmt.Errorf("soft-serve/web: LFS batch descriptor exceeds %d bytes", maxLFSBatchDescriptorBytes)
	}

	var batch struct {
		Operation string `json:"operation"`
	}
	if err := json.Unmarshal(body, &batch); err != nil {
		return "", fmt.Errorf("soft-serve/web: decode LFS batch descriptor: %w", err)
	}
	switch batch.Operation {
	case lfs.OperationDownload:
		return ActionRead, nil
	case lfs.OperationUpload:
		return ActionWrite, nil
	default:
		return "", fmt.Errorf("soft-serve/web: unsupported LFS batch operation")
	}
}

func isReadOnlyGitRoute(route string) bool {
	return route == "/HEAD" ||
		strings.HasPrefix(route, "/objects/")
}
