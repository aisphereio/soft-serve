package web

import "context"

type protocolEnvironmentKey struct{}
type protocolDefaultBranchKey struct{}

// ProtocolEnvironment supplies server-controlled environment variables to Git
// protocol subprocesses and repository hooks. Embedders can use it to carry an
// already-authenticated principal across Git's process boundary.
type ProtocolEnvironment func(context.Context) []string

func WithProtocolEnvironment(ctx context.Context, environment ProtocolEnvironment) context.Context {
	return context.WithValue(ctx, protocolEnvironmentKey{}, environment)
}

// WithProtocolDefaultBranch pins HEAD for repositories served by the embedded
// protocol router. This prevents a first feature-branch push from becoming the
// repository's default branch.
func WithProtocolDefaultBranch(ctx context.Context, branch string) context.Context {
	return context.WithValue(ctx, protocolDefaultBranchKey{}, branch)
}

func copyProtocolEnvironment(request, base context.Context) context.Context {
	environment, _ := base.Value(protocolEnvironmentKey{}).(ProtocolEnvironment)
	if environment != nil {
		request = context.WithValue(request, protocolEnvironmentKey{}, environment)
	}
	if branch, _ := base.Value(protocolDefaultBranchKey{}).(string); branch != "" {
		request = context.WithValue(request, protocolDefaultBranchKey{}, branch)
	}
	return request
}

func protocolDefaultBranch(ctx context.Context) string {
	branch, _ := ctx.Value(protocolDefaultBranchKey{}).(string)
	return branch
}

func protocolEnvironment(ctx context.Context) []string {
	environment, _ := ctx.Value(protocolEnvironmentKey{}).(ProtocolEnvironment)
	if environment == nil {
		return nil
	}
	return environment(ctx)
}
