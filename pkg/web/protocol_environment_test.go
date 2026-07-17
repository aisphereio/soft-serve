package web

import (
	"context"
	"testing"
)

func TestProtocolEnvironmentUsesRequestContext(t *testing.T) {
	type key struct{}
	base := WithProtocolEnvironment(context.Background(), func(ctx context.Context) []string {
		return []string{"SUBJECT=" + ctx.Value(key{}).(string)}
	})
	request := context.WithValue(context.Background(), key{}, "alice")
	request = copyProtocolEnvironment(request, base)

	got := protocolEnvironment(request)
	if len(got) != 1 || got[0] != "SUBJECT=alice" {
		t.Fatalf("protocolEnvironment() = %v, want [SUBJECT=alice]", got)
	}
}
