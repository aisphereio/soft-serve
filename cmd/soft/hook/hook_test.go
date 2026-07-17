package hook

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/aisphereio/soft-serve/pkg/hooks"
)

type rejectingHooks struct {
	err       error
	preArgs   []hooks.HookArg
	updateArg hooks.HookArg
}

func (h *rejectingHooks) PreReceive(_ context.Context, _, _ io.Writer, _ string, args []hooks.HookArg) error {
	h.preArgs = append([]hooks.HookArg(nil), args...)
	return h.err
}

func (h *rejectingHooks) Update(_ context.Context, _, _ io.Writer, _ string, arg hooks.HookArg) error {
	h.updateArg = arg
	return h.err
}

func (*rejectingHooks) PostReceive(context.Context, io.Writer, io.Writer, string, []hooks.HookArg) {
}

func (*rejectingHooks) PostUpdate(context.Context, io.Writer, io.Writer, string, ...string) {
}

func TestRunInternalHookPropagatesPreReceiveError(t *testing.T) {
	wantErr := errors.New("main branch requires publish")
	hks := &rejectingHooks{err: wantErr}
	stdin := strings.NewReader(
		"0000000000000000000000000000000000000000 aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa refs/heads/feature\n" +
			"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb refs/heads/main\n",
	)

	err := runInternalHook(context.Background(), hks, hooks.PreReceiveHook, "demo", stdin, io.Discard, io.Discard, nil)
	if !errors.Is(err, wantErr) {
		t.Fatalf("runInternalHook() error = %v, want %v", err, wantErr)
	}
	if got, want := len(hks.preArgs), 2; got != want {
		t.Fatalf("pre-receive args = %d, want %d", got, want)
	}
}

func TestRunInternalHookPropagatesUpdateError(t *testing.T) {
	wantErr := errors.New("tag rewrite requires manage")
	hks := &rejectingHooks{err: wantErr}
	args := []string{
		"refs/tags/v1.0.0",
		"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
	}

	err := runInternalHook(context.Background(), hks, hooks.UpdateHook, "demo", strings.NewReader(""), io.Discard, io.Discard, args)
	if !errors.Is(err, wantErr) {
		t.Fatalf("runInternalHook() error = %v, want %v", err, wantErr)
	}
	if got, want := hks.updateArg.RefName, "refs/tags/v1.0.0"; got != want {
		t.Fatalf("update ref = %q, want %q", got, want)
	}
}
