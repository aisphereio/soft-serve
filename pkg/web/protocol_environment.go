package web

import "context"

type protocolEnvironmentKey struct{}

// ProtocolEnvironment supplies server-controlled environment variables to Git
// protocol subprocesses and repository hooks. Embedders can use it to carry an
// already-authenticated principal across Git's process boundary.
type ProtocolEnvironment func(context.Context) []string

func WithProtocolEnvironment(ctx context.Context, environment ProtocolEnvironment) context.Context {
	return context.WithValue(ctx, protocolEnvironmentKey{}, environment)
}

func copyProtocolEnvironment(request, base context.Context) context.Context {
	environment, _ := base.Value(protocolEnvironmentKey{}).(ProtocolEnvironment)
	if environment == nil {
		return request
	}
	return context.WithValue(request, protocolEnvironmentKey{}, environment)
}

func protocolEnvironment(ctx context.Context) []string {
	environment, _ := ctx.Value(protocolEnvironmentKey{}).(ProtocolEnvironment)
	if environment == nil {
		return nil
	}
	return environment(ctx)
}
